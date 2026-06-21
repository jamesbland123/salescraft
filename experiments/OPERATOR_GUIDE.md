# Experiment Operator Guide

This guide captures operational lessons from running Salescraft autonomous SDLC
experiments with an AI outer operator. It is for the operator, not for the model
being evaluated.

The intended workflow is one human bootstrap prompt. After that, the AI operator
should have enough local instructions, commands, and guardrails to keep the
experiment moving, monitor it, fix setup-only blockers, and report meaningful
status without the human driving each step.

The operator may fix the experiment harness, host environment, logging, tmux
supervision, and reproducibility controls. The operator must not help the model
under test build the app.

## Unattended Permissions

Assume the human will not be available to approve prompts after the bootstrap
command. The watcher and AI operator must therefore be launched with all
permissions they need up front.

For the watcher, this means launching it from a normal host terminal, not from an
approval-gated sandbox:

```bash
tmux new-session -d -s salescraft-watch -c /Users/james/dev/salescraft './experiments/scripts/watch-experiment.sh --config experiments/configs/phase1-codex-gpt.json'
```

For an AI operator, this means starting the operator in the tool's no-approval,
full-access mode. For Codex, use the equivalent of:

```bash
codex exec --dangerously-bypass-approvals-and-sandbox < operator-prompt.md
```

Use the equivalent dangerous/no-approval mode for Claude Code, OpenCode, or any
other operator tool. If the operator is approval-gated, it may stall when it
needs to install tools, start Podman, inspect tmux, access logs, or push commits.

The evaluated model is separate from the operator. Its permissions are declared
in the experiment config. For Codex trials, the config must include the
no-approval/full-access flags needed for the model to make commits and install
packages inside the disposable trial workspace.

Do not use unattended mode to bypass reproducibility checks. A dirty golden repo,
existing trial workspace, existing artifact directory, or failed `podman ps`
should still stop the watcher before a trial starts.

## Operator Scope

The AI operator owns the outer loop:

- preflight the golden repo and host environment
- launch the trial in tmux
- poll the run until it completes or blocks
- classify stops as setup issues or app-building behavior
- fix setup issues only
- bump and commit `trial_id` before fresh restarts
- preserve artifacts and trial workspaces for later inspection

Allowed:

- install or repair host tools required by the experiment
- start, stop, or inspect tmux sessions
- start and verify Podman infrastructure
- bump `trial_id` and commit experiment config changes
- inspect runner logs, process state, artifacts, and `BUILD_STATE.md`
- fix runner bugs, config bugs, or environment gaps

Not allowed:

- edit generated app code in `experiments/trials/{trial_id}`
- tell the model under test how to fix app build failures
- mark build items complete for the model
- clean trial workspaces before manual inspection is done

## Required Tools

Install and verify these before starting a long run:

```bash
git --version
tmux -V
go version
pnpm --version
podman --version
podman-compose --version
./experiments/runner/salescraft-exp --help
```

Useful operator tools:

- `tmux`: long-running supervision and attachable console
- `podman`, `podman-compose`: required by Salescraft service verification
- `ps`, `lsof`: process and port/socket checks
- `tail`, `sed`, `find`: artifact and log inspection
- `git`: config commits and golden repo cleanliness
- `curl`: socket/API smoke checks when diagnosing Podman

## Preflight Checklist

Run these from the golden repo before `prepare` or `trial`:

```bash
git status --short
./experiments/runner/salescraft-exp --help
podman machine list
podman ps
podman-compose --version
```

Requirements:

- `git status --short` is empty
- `trial_id` is unique
- `experiments/trials/{trial_id}` does not exist
- `experiments/artifacts/{trial_id}` does not exist
- `podman ps` succeeds before any trial that may reach Podman work
- the runner binary is rebuilt if runner code changed

If `podman ps` fails, do not start a trial. Fix Podman first.

## Tmux Lessons

Use tmux for experiments that may run for hours. A plain detached command can
make the session disappear when the command exits. An interactive shell fallback
can also exit unexpectedly when stdin is not usable. Prefer a hard keepalive at
the end of the tmux command.

Start an experiment session:

```bash
tmux new-session -d -s salescraft-exp -c /Users/james/dev/salescraft \
  "zsh -lc 'set +e; ./experiments/runner/salescraft-exp trial --config experiments/configs/phase1-codex-gpt.json; status=\$?; printf \"\\n[salescraft-exp] exited with status %s\\n\" \"\$status\"; echo \"Workspace/artifacts retained. Leave this pane open for inspection.\"; while true; do sleep 3600; done'"
```

For unattended operation, prefer the watcher. It runs as an outer tmux session
and starts the experiment in its own tmux session:

```bash
tmux new-session -d -s salescraft-watch -c /Users/james/dev/salescraft './experiments/scripts/watch-experiment.sh --config experiments/configs/phase1-codex-gpt.json'
```

Attach to the watcher with `tmux attach -t salescraft-watch`. Attach to the
experiment with `tmux attach -t salescraft-exp`.

Attach:

```bash
tmux attach -t salescraft-exp
```

Detach without stopping the run:

```text
Ctrl-b d
```

Poll from another terminal:

```bash
tmux list-panes -t salescraft-exp -F '#{pane_current_command} #{pane_dead} #{pane_dead_status}'
tmux capture-pane -t salescraft-exp -p -S -120
```

If the session disappears, inspect artifacts instead of assuming the run is
still active:

```bash
find experiments/artifacts/phase1-codex-gpt-run13 -maxdepth 1 -type f -print | sort
sed -n '1,220p' experiments/trials/phase1-codex-gpt-run13/BUILD_STATE.md
```

## Podman Lessons

The app build plan expects Podman, not Docker Desktop. On macOS, installing the
CLI is not enough; a Podman VM must be initialized, started, and reachable.

Install:

```bash
brew install podman podman-compose
```

Initialize and start:

```bash
podman machine init --cpus 2 --memory 4096 --disk-size 30 --timezone UTC podman-machine-default
podman machine start
podman ps
```

`podman ps` is the real readiness check. `podman machine start` can report
success while the host-side proxy is still not usable.

Common failure modes:

- `podman-compose` missing: install it before restarting the trial.
- `podman ps` cannot connect: the VM or proxy is not ready; do not start the trial.
- `Last Up` says `Never`: confirm with `podman ps`; `machine list` can lag or be misleading.
- CoreOS emergency mode or Ignition failure: the VM did not initialize cleanly.
- SSH handshake resets on the Podman port: the host proxy is listening, but the VM is not usable.

Useful diagnostics:

```bash
podman machine list
podman machine inspect
podman system connection list
lsof -nP -iTCP:<podman-ssh-port>
tail -120 /var/folders/q_/syv6sqxn219fkz9vmmzykpy00000gn/T/podman/podman-machine-default.log
```

If a newly-created Podman machine boots into emergency mode, remove and recreate
it before retrying:

```bash
podman machine stop
podman machine rm -f podman-machine-default
podman machine init --cpus 2 --memory 4096 --disk-size 30 --timezone UTC podman-machine-default
podman machine start
podman ps
```

Do not launch the experiment until `podman ps` succeeds.

## Trial Restart Rules

When a setup issue blocks a run:

1. Inspect the failed trial artifacts and `BUILD_STATE.md`.
2. Decide whether the failure is experiment setup or app-building behavior.
3. Fix only setup issues.
4. Bump `trial_id`.
5. Commit the config bump and any runner/setup documentation changes.
6. Start a fresh trial in tmux.

Do not reuse a failed trial workspace for comparable data.

## One-Prompt AI Operation

The human should be able to start an operator session with one prompt. The AI
operator should then:

1. Read this guide and `experiments/README.md`.
2. Discover the active config and `trial_id`.
3. Run preflight checks.
4. Fix missing host setup that is required by the experiment.
5. Commit config or runner/setup documentation changes before `prepare`.
6. Start the trial under tmux with a keepalive wrapper.
7. Poll the tmux pane and process table regularly.
8. Inspect artifacts when the run stops.
9. Continue only when the next action is setup-level and reproducibility-safe.
10. Stop and report when the result is a genuine app-building outcome or an external blocker.

The AI operator should not wait for human confirmation for routine inspection,
trial-id bumps, commits of experiment setup changes, or relaunching after a
confirmed setup-only failure. It should ask before destructive cleanup, OS-level
privileged changes, or anything that would alter generated app code.

## Bootstrap Prompt

Use this prompt when asking an AI assistant to supervise experiments:

```text
You are the outer experiment operator for the Salescraft autonomous SDLC
evaluation in /Users/james/dev/salescraft.

Goal:
Run and monitor the configured experiment. Preserve reproducibility and collect
artifacts. Do not help the model under test build the app. I want to give this
prompt once; after that, keep operating autonomously until the experiment
completes, blocks on app-building behavior, or reaches a real external blocker.
You should be running with no approval prompts and full shell/filesystem/network
access. If you are approval-gated, report that immediately before starting the
experiment because the human may not be present to approve actions later.

First read:
- experiments/README.md
- experiments/OPERATOR_GUIDE.md

Allowed work:
- fix experiment runner bugs, config issues, host environment problems, tmux
  supervision, Podman setup, logging, and artifact collection
- inspect tmux panes, processes, runner logs, artifacts, and BUILD_STATE.md
- bump trial_id and commit config/setup changes before preparing a new trial
- install missing host dependencies when they are required by the experiment

Prohibited work:
- do not edit generated app code in experiments/trials/{trial_id}
- do not tell the evaluated model how to solve app implementation failures
- do not mark build items complete for the evaluated model
- do not clean trial workspaces unless manual testing is done and I explicitly ask

Autonomy:
- do not stop after a plan; run the needed checks and commands
- do not rely on future human approvals; preflight permissions before the run
- poll the tmux session and artifacts regularly
- if a setup-only issue stops the run, fix it, bump trial_id, commit the config,
  and restart a fresh trial
- keep trial workspaces and artifacts intact
- give concise status updates with what is running, what changed, and where logs are

Before starting:
1. Check git status in the golden repo.
2. Confirm the runner binary exists and is current if runner code changed.
3. Confirm the config trial_id is unique.
4. Confirm podman, podman-compose, and `podman ps` work.
5. Start the trial in tmux with a keepalive loop after command exit.

Use this tmux pattern:
tmux new-session -d -s salescraft-watch -c /Users/james/dev/salescraft './experiments/scripts/watch-experiment.sh --config experiments/configs/phase1-codex-gpt.json'

Monitor frequently:
- tmux capture-pane -t salescraft-watch -p -S -120
- tmux capture-pane -t salescraft-exp -p -S -120
- tmux list-panes -t salescraft-exp -F '#{pane_current_command} #{pane_dead} #{pane_dead_status}'
- ps -axo pid,etime,command
- inspect experiments/artifacts/{trial_id} and BUILD_STATE.md when the run stops

If the run stops:
Classify the stop as either experiment setup or app-building behavior. Fix only
experiment setup. If a fresh run is needed, bump trial_id, commit the config, and
start a new tmux session.
```

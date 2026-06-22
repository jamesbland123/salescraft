# Salescraft Experiment Runner

This directory contains the fixed outer runner for autonomous SDLC experiments.
The runner resets Salescraft to a pinned baseline, invokes one selected tool
inside a disposable workspace, verifies the result with fixed commands, and
archives the generated artifacts.

For long-running supervision, tmux usage, Podman setup, and the AI operator
prompt, see [OPERATOR_GUIDE.md](OPERATOR_GUIDE.md).

The runner is a control. Codex, Claude Code, OpenCode, Aider, OpenHands, and
similar systems are experimental tools that run inside the workspace created by
the runner.

## Directory Layout

```text
experiments/
|-- README.md
|-- configs/
|   |-- phase1-aider-sonnet.json
|   |-- phase1-claude-code-opus48.json
|   |-- phase1-claude-code-sonnet.json
|   |-- phase1-codex-gpt.json
|   |-- phase1-opencode-sonnet.json
|   `-- phase1-openhands-sonnet.json
|-- runner/
|   |-- go.mod
|   `-- cmd/salescraft-exp/main.go
|-- trials/      # generated, ignored by git
|-- artifacts/   # generated, ignored by git
`-- results.sqlite # reserved for later scoring/indexing
```

## Build the Runner

```bash
cd experiments/runner
go build ./cmd/salescraft-exp
```

This creates `experiments/runner/salescraft-exp`.

## Trial Lifecycle

A normal trial has four automated steps:

```bash
./experiments/runner/salescraft-exp prepare --config experiments/configs/phase1-codex-gpt.json
./experiments/runner/salescraft-exp run --config experiments/configs/phase1-codex-gpt.json
./experiments/runner/salescraft-exp verify --config experiments/configs/phase1-codex-gpt.json
./experiments/runner/salescraft-exp archive --config experiments/configs/phase1-codex-gpt.json
./experiments/runner/salescraft-exp evaluate --config experiments/configs/phase1-codex-gpt.json
```

Or run the full automated lifecycle:

```bash
./experiments/runner/salescraft-exp trial --config experiments/configs/phase1-codex-gpt.json
```

For unattended operation, run the watcher in tmux. It starts the experiment in
a separate tmux session, polls it, and keeps the experiment pane alive after
runner exit. Start this from a normal host shell, or from an AI operator that
was launched with no approval prompts and full shell/filesystem/network access:

```bash
tmux new-session -d -s salescraft-watch -c /Users/james/dev/salescraft './experiments/scripts/watch-experiment.sh --config experiments/configs/phase1-codex-gpt.json'
```

For unattended runs where the operator is allowed to repair the local Podman VM,
add `--repair-podman`:

```bash
tmux new-session -d -s salescraft-watch -c /Users/james/dev/salescraft './experiments/scripts/watch-experiment.sh --config experiments/configs/phase1-codex-gpt.json --repair-podman'
```

Watch the operator:

```bash
tmux attach -t salescraft-watch
```

Watch the experiment directly:

```bash
tmux attach -t salescraft-exp
```

For multi-hour runs where you want a durable AI operator, start a third tmux
session. This is separate from the evaluated model. It periodically snapshots
tmux, process, Podman, and artifact state, then runs one bounded noninteractive
Codex operator turn:

```bash
tmux new-session -d -s salescraft-operator -c /Users/james/dev/salescraft './experiments/scripts/operator-loop.sh --config experiments/configs/phase1-codex-gpt.json --interval 900'
```

Watch the operator loop:

```bash
tmux attach -t salescraft-operator
```

Use this only for the outer operator. Do not use `salescraft-operator` as the
evaluated tool session. The operator prompt limits action to setup, host
environment, tmux, Podman, runner/config, logging, and artifact handling.

`trial` does not clean up the workspace. It leaves
`experiments/trials/{trial_id}` available so you can launch, inspect, and test
the generated app after archival. When manual testing is complete, remove the
workspace explicitly:

```bash
./experiments/runner/salescraft-exp clean --config experiments/configs/phase1-codex-gpt.json
```

If the config has already been bumped to the next run, use `--trial-id` to
evaluate or re-archive a previous retained workspace without editing the config:

```bash
./experiments/runner/salescraft-exp evaluate --config experiments/configs/phase1-codex-gpt.json --trial-id phase1-codex-gpt-run13
```

The runner streams tool and verification output to the console while also
writing the same output to artifact logs. Long-running commands emit a
`[salescraft-exp] command: still running...` heartbeat every 30 seconds.
Claude Code print-mode defaults to buffered text output, which can make tmux
look idle even while the model is working. Claude Code trial configs use
`--output-format=stream-json --include-partial-messages` so tmux shows realtime
model/tool activity comparable to Codex's verbose stderr stream.

## Reset Model

Each trial starts from a clean local clone at `baseline_ref`.

```text
golden repo commit
  -> disposable clone in experiments/trials/{trial_id}
  -> selected tool runs only inside that clone
  -> fixed verification commands run after each iteration
  -> runner repeats until BUILD_STATE has no next eligible work, a blocker appears, or max_iterations is reached
  -> artifacts are archived
  -> clone is retained for manual testing
  -> clone is removed later with explicit clean
```

The runner accepts annotated second-level build-state headings such as
`## Next Eligible (Phase 4 - Domain Features)`. These annotations are for human
readability and must not cause the runner to treat a partially complete build
as finished.

Do not run Codex, Claude Code, OpenCode, or other tools directly in the golden
repo when collecting comparable experiment data.

## Preflight Rules

`prepare` refuses to run if:

- the golden repo has uncommitted changes
- the trial workspace already exists
- the artifact directory for the trial already exists
- `baseline_ref` cannot be resolved

This makes accidental reruns visible. To repeat a trial, create a new
`trial_id`, or intentionally remove the old trial workspace and artifacts.

## Configuration

Trial configs are JSON so the runner can use only the Go standard library.
Every config should declare the independent variable being changed and all
controls being held fixed.

Important fields:

- `trial_id`: unique ID for this run
- `allowed_variable`: the only variable intended to change in this trial group
- `baseline_ref`: pinned git ref or SHA for the golden source
- `tool.command`: executable for the tool under test
- `tool.args`: arguments passed to that tool
- `models`: explicit model IDs by phase
- `verification.commands`: fixed post-run verification commands
- `cache_policy`: `cold`, `warm`, or another declared policy

Commands are arrays, not shell strings. If a tool needs shell behavior, call a
shell explicitly, for example:

```json
["zsh", "-lc", "codex exec -c 'model_provider=\"amazon-bedrock\"' -c 'sandbox_workspace_write.network_access=true' -c 'shell_environment_policy.inherit=\"all\"' --model openai.gpt-5.5 < PROMPT.md"]
```

Using explicit arrays makes command capture more reproducible.

The configs include command lines for Codex, Claude Code, OpenCode, Aider, and
OpenHands. Treat these files as the canonical trial declarations. If a tool
command changes, update the config and commit it before running `prepare`.

Codex uses GPT-family models exposed through Bedrock. The Codex config
explicitly pins `model_provider` to `amazon-bedrock` and uses the Bedrock-visible
model slug `openai.gpt-5.5`. It is therefore a native Bedrock toolchain baseline
(`toolchain` variable), not a pure tool-only comparison against Sonnet-backed
tools. For a pure tool comparison, include only tools that can run the same
fixed model.

The runner injects per-trial package-manager cache paths under
`/tmp/salescraft-exp/{trial_id}` so Corepack, npm, and pnpm do not write into the
user home directory. For tools with their own execution sandbox, the tool config
must also allow package-manager network access when `package_install` is enabled.

## Artifacts

Each trial writes:

```text
experiments/artifacts/{trial_id}/
|-- manifest.json
|-- input-digest.json
|-- tool-stdout.log
|-- tool-stderr.log
|-- tool-result.json
|-- verify-log.txt
|-- verify-result.json
|-- final-status.txt
|-- evaluation-log.txt
|-- evaluation-result.json
|-- final-report.md
|-- final-diff.patch
`-- generated-repo.tar.gz
```

`manifest.json` records the trial declaration. `input-digest.json` hashes the
fixed experiment inputs so mismatched prompts, specs, or protocols are easy to
detect. `final-status.txt` is a semantic summary with the trial outcome,
archive time, build completion/blocker fields, next eligible work, and final
workspace git status. A clean generated workspace is recorded as
`workspace_git_status: clean` rather than an empty file.

`evaluate` is the independent post-run evaluator. It does not invoke the model
under test. It reruns the fixed verification commands, launches the generated
web app for browser route/workflow checks, scores static DDD/domain-language
fit against the research rubric, collects runner iteration/timing data,
summarizes workspace inventory and package scripts, and writes:

- `evaluation-log.txt`: raw evaluator command output
- `evaluation-result.json`: structured data for later scoring/comparison
- `final-report.md`: human-readable report for the trial
- `judge-brief.md`: read-only evidence packet for an independent LLM judge
- `judge-prompt.md`: exact prompt sent to the judge
- `judge-report.md`: independent LLM judge critique
- `judge-result.json`: structured judge command/verdict metadata

The final report is organized around the research questions from
`experiments/evaluation/research-rubric.md`, including quality score,
functional/browser evidence, DDD adherence, completion, timing, and residual
risk notes. It includes an evaluator verdict and critical findings so a
browser or workflow failure is recorded as an app quality finding, not confused
with a harness failure. Browser checks use the generated app's local Next.js
runtime and Playwright from the trial workspace when available. They run
headless and write screenshots under
`experiments/artifacts/{trial_id}/browser-screenshots/`.

For Codex trials, `evaluate` also parses the per-iteration `tokens used`
summary from tool stderr logs and reports total observed tokens plus an
iteration breakdown. USD cost still requires a separate provider pricing/rate
table.

The browser evaluator currently checks:

- core routes required by the committed UI spec
- navigation integrity from the authenticated shell
- login form basics
- estimate builder acceptance surface
- relationship intelligence acceptance surface
- bid response acceptance surface

These browser checks are acceptance-surface checks, not exhaustive manual QA.
The current quality score is deterministic and provisional: fixed verification,
browser checks, static DDD/domain-language scanning, completion count, and
basic heuristic scores for security, documentation, and performance.

The LLM judge step is a separate read-only Codex invocation using the
`models.judge` value from the trial config. For Codex-based judging this should
be an OpenAI model ID, for example `openai.gpt-5.5`. The judge receives the
brief, deterministic report, browser JSON, and rubric as prompt evidence and
must return a first-line verdict of `pass`, `marginal`, or `fail`. The runner
does not allow this judge to repair the generated app; it only captures the
critique and folds the verdict into the final report.

## Comparing Tools

When comparing tools, keep these fixed unless the tool itself is the declared
variable:

- baseline git SHA
- prompt and spec files
- build plan
- agent operating model
- evaluation protocol
- model IDs
- sampling settings
- loop strategy
- context strategy
- capability profile
- verification commands
- cache policy

When comparing models, keep the tool fixed.

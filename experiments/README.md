# Salescraft Experiment Runner

This directory contains the fixed outer runner for autonomous SDLC experiments.
The runner resets Salescraft to a pinned baseline, invokes one selected tool
inside a disposable workspace, verifies the result with fixed commands, and
archives the generated artifacts.

The runner is a control. Codex, Claude Code, OpenCode, Aider, OpenHands, and
similar systems are experimental tools that run inside the workspace created by
the runner.

## Directory Layout

```text
experiments/
|-- README.md
|-- configs/
|   `-- phase1-codex-sonnet.example.json
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

A normal trial has five steps:

```bash
./experiments/runner/salescraft-exp prepare --config experiments/configs/phase1-codex-sonnet.example.json
./experiments/runner/salescraft-exp run --config experiments/configs/phase1-codex-sonnet.example.json
./experiments/runner/salescraft-exp verify --config experiments/configs/phase1-codex-sonnet.example.json
./experiments/runner/salescraft-exp archive --config experiments/configs/phase1-codex-sonnet.example.json
./experiments/runner/salescraft-exp clean --config experiments/configs/phase1-codex-sonnet.example.json
```

Or run the full lifecycle:

```bash
./experiments/runner/salescraft-exp trial --config experiments/configs/phase1-codex-sonnet.example.json
```

## Reset Model

Each trial starts from a clean git worktree at `baseline_ref`.

```text
golden repo commit
  -> disposable worktree in experiments/trials/{trial_id}
  -> selected tool runs only inside that worktree
  -> fixed verification commands run
  -> artifacts are archived
  -> worktree is removed
```

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
["zsh", "-lc", "codex exec < PROMPT.md"]
```

Using explicit arrays makes command capture more reproducible.

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
|-- final-diff.patch
`-- generated-repo.tar.gz
```

`manifest.json` records the trial declaration. `input-digest.json` hashes the
fixed experiment inputs so mismatched prompts, specs, or protocols are easy to
detect.

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


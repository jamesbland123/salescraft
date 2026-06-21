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
|   |-- phase1-aider-sonnet.json
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
```

Or run the full automated lifecycle:

```bash
./experiments/runner/salescraft-exp trial --config experiments/configs/phase1-codex-gpt.json
```

`trial` does not clean up the workspace. It leaves
`experiments/trials/{trial_id}` available so you can launch, inspect, and test
the generated app after archival. When manual testing is complete, remove the
workspace explicitly:

```bash
./experiments/runner/salescraft-exp clean --config experiments/configs/phase1-codex-gpt.json
```

The runner streams tool and verification output to the console while also
writing the same output to artifact logs. Long-running commands emit a
`[salescraft-exp] command: still running...` heartbeat every 30 seconds.

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

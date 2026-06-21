# Agent Operating Model

This document defines how autonomous AI builders operate inside this repo. It exists so application quality is not accidentally determined by each model inventing its own workflow.

`PROMPT.md` is the short entry point. `BUILD_PLAN.md` defines the build order. `spec/*.md` defines the product.

## Capability Profile

The default Salescraft build profile is `standard-autonomous`.

| Profile | Subagents | Skills | MCP | Browser | Package Install | Use |
|---------|-----------|--------|-----|---------|-----------------|-----|
| `minimal` | No | No | No | No | Yes | Baseline single-agent build |
| `standard-autonomous` | Yes | Existing only | Existing only | Yes | Yes | Default build profile |
| `enhanced` | Yes | Create or update allowed | Add MCP allowed | Yes | Yes | Toolchain experimentation |
| `constrained` | No | No | No | No | No | Raw model behavior |

Unless an evaluation protocol says otherwise, use `standard-autonomous`.

## General Rules

- The parent agent owns the final codebase state, verification, and commit.
- Read only the specs needed for the selected work item, plus authoritative resolution docs.
- Prefer existing repo patterns over new abstractions.
- Do not create new frameworks, packages, services, skills, or MCP servers unless the selected capability profile allows it.
- Do not hide failures. Record blocked work with the exact command and short error summary.
- Use the filesystem and `BUILD_STATE.md` as durable memory.

## Required Artifacts

Maintain these artifacts during autonomous builds:

- `BUILD_STATE.md`: phase, completed items, in-progress items, blocked items, next eligible work.
- `VERIFY_LOG.md`: append-only summary of verification commands, pass/fail result, and notable errors.
- `ACCEPTANCE_TRACE.md`: maps completed work items to acceptance criteria and tests.

Create `VERIFY_LOG.md` and `ACCEPTANCE_TRACE.md` when the first implemented feature needs them. Do not create empty placeholder files.

## Subagent Rules

Use subagents only when work can proceed safely in parallel.

- Maximum 3 subagents per iteration.
- Each subagent must work in non-overlapping directories.
- Shared schema, Prisma schema, root config, package manager files, and barrel exports are parent-agent-owned unless a subagent is explicitly assigned them.
- Parent agent must merge, inspect, and verify all subagent output.
- Subagents must not commit independently.
- Subagents must not change the build plan, agent operating model, or product specs unless specifically assigned documentation work.

### Subagent Input Contract

Every subagent receives:

```text
Build item: {item-id}
Capability profile: {profile}
Relevant specs:
- {spec file}

Allowed paths:
- {directory or file list}

Do not edit:
- {shared or restricted paths}

Requirements:
- {work item acceptance notes}

Expected output:
- Files changed
- Tests added or updated
- Verification attempted
- Open risks or blockers
```

### Subagent Output Contract

Every subagent returns:

```text
Summary:
- {what changed}

Files changed:
- {path}

Tests:
- {tests added}
- {commands run}

Risks:
- {known issues or "none"}
```

## Skills Policy

Skills are reusable instructions or workflows available to the agent.

- In `standard-autonomous`, use existing skills only when directly relevant.
- Do not create or update skills unless using the `enhanced` profile or an experiment explicitly varies skill creation.
- If a skill is used, mention it in the work summary and record any generated artifacts.
- Skill output must still pass the same repo verification gates.

## MCP Policy

MCP servers expose external capabilities such as browser control, databases, docs, or code tools.

- In `standard-autonomous`, use only MCP servers already available in the environment.
- Do not add or configure new MCP servers unless the `enhanced` profile is active.
- MCP usage must be task-relevant. Do not use MCP tools to bypass repo conventions.
- If a browser MCP is used for frontend work, verify the actual rendered page after the relevant dev server is running.

## Tool Use Policy

- Use fast local search tools first when available.
- Prefer structured parsers and package scripts over ad hoc text manipulation.
- Use package-manager commands from the repo root unless a spec says otherwise.
- Request user approval for network installs, destructive operations, or privileged actions when required by the environment.
- Do not discard unrelated changes.

## Planning and Handoff

For each work item, the parent agent should maintain a short plan:

1. Specs read.
2. Files expected to change.
3. Implementation steps.
4. Tests and verification commands.
5. Risks.

For multi-agent or multi-phase handoff, use this format:

```text
Work item:
Status:
Completed:
Changed files:
Verification:
Remaining work:
Blockers:
Next recommended step:
```

## Evaluation Controls

When this repo is used for model/toolchain evaluation, the trial runner must declare:

- Capability profile.
- Model assignments by phase.
- Orchestrator.
- Context-passing strategy.
- Loop strategy and max iterations.
- Allowed tools, skills, and MCP servers.
- Budget limits.
- Whether new skills or MCP servers may be created.

Without those declarations, the trial is not comparable to other runs.


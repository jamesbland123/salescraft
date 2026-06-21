# Evaluation Protocol

This document defines the controls needed when Salescraft is used to evaluate autonomous SDLC model and toolchain performance.

The product spec defines the destination. `BUILD_PLAN.md` defines the route. `AGENT_OPERATING_MODEL.md` defines the vehicle and driving rules. This protocol defines what is fixed and what is varied during experiments.

## Trial Declaration

Every trial must declare:

| Field | Required | Example |
|-------|----------|---------|
| Trial ID | Yes | `phase1-sonnet-standard-run01` |
| Application spec version | Yes | Git commit SHA |
| Build plan version | Yes | Git commit SHA |
| Capability profile | Yes | `standard-autonomous` |
| Orchestrator | Yes | Claude Code, Aider, OpenHands |
| Planning model | Yes | Claude Sonnet |
| Code model | Yes | Claude Sonnet |
| Test model | Yes | Claude Sonnet or separate reviewer |
| Loop strategy | Yes | test-driven loop, max 5 |
| Context strategy | Yes | full history, summarized handoff, artifact-only |
| Allowed tools | Yes | filesystem, shell, browser |
| Allowed skills | Yes | existing only, none, create allowed |
| Allowed MCP servers | Yes | browser, docs, database |
| Budget | Yes | USD and token cap |

## Capability Variables

The following are experimental variables, not background details:

- Subagent availability.
- Number of concurrent subagents.
- Skill availability.
- Skill creation or update permissions.
- MCP server availability.
- Browser/render verification.
- Package installation permissions.
- Context handoff shape.
- Retry and repair loop limits.

If any of these change between trials, record the change as an independent variable.

## Fixed Controls

Hold these constant unless the experiment explicitly varies them:

- Product specs and resolved ambiguities.
- Build plan.
- Capability profile.
- Temperature and sampling settings.
- Execution environment.
- Verification commands.
- Scoring rubric.
- Judge model.
- Cost accounting rules.

## Required Logs

Each trial must preserve:

- Final generated codebase.
- `BUILD_STATE.md`.
- `VERIFY_LOG.md`.
- `ACCEPTANCE_TRACE.md`.
- Prompt and handoff artifacts.
- Tool call summary.
- Token and cost log.
- Test, lint, typecheck, and build outputs.
- Security scan output if run.

## Scoring Notes

Score the build outcome, not just the final answer. A trial that produces attractive code but cannot install, build, or run should fail the relevant execution criteria.

Quality scoring should include:

- Functional correctness.
- Completeness against specs.
- DDD and domain language adherence.
- Test quality.
- Code quality.
- Security.
- Documentation.
- Cost and time to completion.


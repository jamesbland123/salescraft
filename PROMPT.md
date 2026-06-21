# Salescraft Agent Prompt

You are building Salescraft, a SaaS platform for commercial flooring companies to win and execute government contracts.

Keep this file small. It is the entry point only. The detailed product requirements, build sequence, and autonomous-agent rules live in separate documents.

## Read First

Before changing code, read these in order:

1. `AGENT_OPERATING_MODEL.md` - how autonomous builders use tools, subagents, skills, MCP, handoffs, and logs.
2. `BUILD_PLAN.md` - phased dependency graph, iteration protocol, verification gates, and build state format.
3. `spec/26-resolved-ambiguities.md` - authoritative conflict resolutions.
4. The relevant `spec/*.md` files for the selected work item.

## Mission

Build the application incrementally. Each iteration must:

1. Assess the repo state and `BUILD_STATE.md`.
2. Select the next eligible work item from `BUILD_PLAN.md`.
3. Implement production code and tests.
4. Verify with the required commands.
5. Update `BUILD_STATE.md`.
6. Commit only when the work item is complete and verified.

## Tech Stack

- Monorepo: Turborepo + pnpm workspaces
- Backend: Fastify 4.x, TypeScript 5.4+, Prisma 5.x, PostgreSQL 15+, BullMQ 5.x, Redis 7+
- Frontend: Next.js 14+ App Router, React 18, Tailwind CSS 3.x, shadcn/ui, TanStack Query 5.x
- Mobile: React Native 0.73+, Expo 50+, WatermelonDB 0.27+
- AI: AWS Bedrock, Claude Sonnet/Haiku, Titan embeddings
- Containers: Podman + podman-compose, not Docker Desktop
- Validation: Zod 3.x shared between client and server
- Testing: Vitest, Supertest, MSW, Playwright

## Non-Negotiables

- Follow the build order in `BUILD_PLAN.md`.
- Follow the capability and subagent rules in `AGENT_OPERATING_MODEL.md`.
- Keep domain language specific to commercial flooring and government-contract workflows.
- Use TypeScript strict mode, Zod validation, authenticated routes, and tests alongside code.
- Do not scaffold empty files for later.
- Do not overwrite unrelated user changes.
- Do not mark work complete until verification passes or the failure is explicitly recorded as blocked.


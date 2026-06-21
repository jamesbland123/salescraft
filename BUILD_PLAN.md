# Salescraft Build Plan

This document defines the build route for autonomous implementation. It is intentionally separate from `PROMPT.md` so the prompt can stay small.

Product requirements live in `spec/`. Agent/tool behavior lives in `AGENT_OPERATING_MODEL.md`.

## Iteration Protocol

Every iteration follows this sequence.

### Step 1: Assess State

1. Read `BUILD_STATE.md`; create it if it does not exist.
2. Run `pnpm build` if packages exist to verify the current state compiles.
3. Check git status for uncommitted changes. Investigate before editing files that already have changes.
4. Cross-reference the filesystem against `BUILD_STATE.md` completions.
5. Fix discrepancies in `BUILD_STATE.md`.

### Step 2: Select Work Items

1. Find all items in the dependency graph whose dependencies are all marked complete.
2. If any item is marked `in_progress`, `partial`, or `needs_fix`, finish or repair it first.
3. From eligible items, select up to 3 that can be parallelized in non-overlapping directories.
4. If only 1 item is eligible, build it directly.

### Step 3: Build

1. For each selected item, read the relevant spec files.
2. Use subagents only according to `AGENT_OPERATING_MODEL.md`.
3. Each work item must include production code and tests.
4. If items share files, build them sequentially.

### Step 4: Verify

1. Run `pnpm typecheck`.
2. Run `pnpm lint`.
3. Run `pnpm test`.
4. For API modules, verify the server starts and `GET /health` responds.
5. For web, verify `pnpm build --filter=@salescraft/web` succeeds.
6. For mobile, verify the Expo project health check when mobile files are touched.

### Step 5: Commit and Update State

1. Update `BUILD_STATE.md` with completed, in-progress, blocked, and next-eligible items.
2. Stage only relevant files.
3. Commit with a descriptive message: `feat(module): description`.
4. If verification cannot pass after bounded retries, record the failure in `BUILD_STATE.md` and do not claim completion.

## Parallelization Rules

- Use a maximum of 2-3 subagents per iteration.
- Subagents must work in non-overlapping directories.
- If two items both touch `packages/shared/src/schemas/` or `packages/database/prisma/schema.prisma`, they cannot run in parallel.
- After parallel work completes, run verification on the combined result before committing.
- If a conflict or type error arises from parallel work, the parent agent owns the final merge and verification.

### Directory Isolation Examples

These can be parallel:

- `packages/ai/` and `apps/api/src/modules/contacts/`
- `apps/api/src/modules/relationships/` and `apps/api/src/modules/products/`
- `apps/web/src/app/(dashboard)/contacts/` and `apps/web/src/app/(dashboard)/bids/`

These cannot be parallel:

- Two modules that both add to `packages/shared/src/schemas/index.ts`
- Two modules that both modify `packages/database/prisma/schema.prisma`
- `apps/api/src/modules/bids/` and `apps/api/src/modules/estimating/`

## Error Recovery

- Build fails on entry: fix compilation errors before starting new work. Mark the broken item as `needs_fix` in `BUILD_STATE.md`.
- Partial completion: if `BUILD_STATE.md` shows `in_progress`, inspect its directory. Finish it if viable. If broken, repair it without discarding unrelated user changes.
- Test failures: fix failing tests before proceeding to new items. After 3 focused fix attempts on the same issue, mark it `blocked` with the failing command and error summary.
- Subagent conflict: merge manually and prefer the implementation that best satisfies specs and tests.

## Dependency Graph

Items are listed as `id | depends_on | spec_file(s) | verification`. An item is eligible when all dependencies are complete.

### Phase 1: Foundation

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `foundation/root-config` | - | `spec/25-project-configuration.md` | `pnpm install` succeeds |
| `foundation/podman-compose` | `foundation/root-config` | `spec/25-project-configuration.md`, `spec/15-deployment-infrastructure.md` | `podman-compose up -d` starts postgres, redis, localstack, mailhog |
| `foundation/packages-shared` | `foundation/root-config` | `spec/01-architecture.md`, `spec/02-domain-model.md` | `pnpm build --filter=@salescraft/shared` succeeds |
| `foundation/packages-database` | `foundation/root-config`, `foundation/packages-shared` | `spec/24-complete-prisma-schema.md` | `pnpm build --filter=@salescraft/database` succeeds, `pnpm db:migrate` runs |
| `foundation/test-setup` | `foundation/root-config` | `spec/16-testing-strategy.md` | Vitest config exists, test helpers compile |

Parallelism: after `foundation/root-config`, build `foundation/podman-compose` and `foundation/packages-shared` in parallel. Then build `foundation/packages-database` and `foundation/test-setup` in parallel.

### Phase 2: Core API

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `core/api-bootstrap` | `foundation/packages-shared`, `foundation/packages-database` | `spec/01-architecture.md`, `spec/25-project-configuration.md` | API starts, `GET /health` returns 200 |
| `core/auth-module` | `core/api-bootstrap` | `spec/18-authentication.md`, `spec/13-user-roles-permissions.md` | Login, refresh, and logout endpoints work; JWT issued |
| `core/event-bus` | `core/api-bootstrap` | `spec/01-architecture.md` event system section | Events emit and are received by handlers; BullMQ queues created |

Parallelism: `core/auth-module` and `core/event-bus` in parallel after bootstrap.

### Phase 3: AI and Base Entities

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `base/packages-ai` | `foundation/packages-shared` | `spec/09-ai-engine.md` | `pnpm build --filter=@salescraft/ai` succeeds, client instantiates |
| `base/contacts-module` | `core/auth-module` | `spec/02-domain-model.md`, `spec/05-relationship-intelligence.md` | CRUD endpoints work, search works, territory filtering works |
| `base/organizations-module` | `core/auth-module` | `spec/02-domain-model.md`, `spec/03-government-procurement.md` | CRUD endpoints work, facility linking works |
| `base/territories-module` | `core/auth-module` | `spec/02-domain-model.md` | CRUD endpoints, user assignment, zip/city matching |

Parallelism: `base/packages-ai` and `base/contacts-module` in parallel. Then `base/organizations-module` and `base/territories-module` in parallel.

### Phase 4: Domain Features

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `domain/relationships` | `base/contacts-module` | `spec/05-relationship-intelligence.md` | Scoring endpoint, decay calculation, briefing generation, interest matching |
| `domain/opportunities` | `base/contacts-module`, `base/organizations-module` | `spec/04-project-intelligence.md` | CRUD, scoring, state machine transitions |
| `domain/intelligence-signals` | `domain/opportunities`, `base/packages-ai` | `spec/04-project-intelligence.md` | Signal creation, AI classification, signal-to-opportunity correlation |
| `domain/products` | `core/auth-module` | `spec/06-flooring-estimating.md` | Product CRUD, pricing, specifications |
| `domain/communications` | `base/contacts-module` | `spec/10-communication-hub.md` | Interaction logging, activity timeline, email templates |
| `domain/notifications` | `core/event-bus`, `core/auth-module` | `spec/01-architecture.md` WebSocket section | In-app notifications, WebSocket delivery |

Parallelism: `domain/relationships` and `domain/products` in parallel. Then `domain/opportunities` and `domain/communications` in parallel. Then `domain/intelligence-signals` and `domain/notifications` in parallel.

### Phase 5: Complex Modules

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `complex/bids` | `domain/opportunities`, `domain/products`, `domain/communications` | `spec/07-bid-response.md`, `spec/03-government-procurement.md` | State machine, RFP parsing, decision matrix, checklists |
| `complex/estimating` | `domain/products`, `base/organizations-module` | `spec/06-flooring-estimating.md`, `spec/21-estimate-builder-ui.md` | Area calculations, waste factors, prevailing wage, estimate totals |
| `complex/project-lifecycle` | `complex/bids` | `spec/08-project-lifecycle.md` | Project creation from bid, daily logs, punch lists, financial tracking |

Parallelism: `complex/bids` and `complex/estimating` in parallel. Then `complex/project-lifecycle`.

### Phase 6: Integrations

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `integration/file-upload` | `core/api-bootstrap` | `spec/12-integrations.md` | Presigned URL generation, upload flow, file metadata stored |
| `integration/gmail` | `domain/communications` | `spec/12-integrations.md` | OAuth flow, email sync, send-on-behalf |
| `integration/outlook` | `domain/communications` | `spec/12-integrations.md` | OAuth flow, email sync |
| `integration/bid-scrapers` | `domain/intelligence-signals` | `spec/04-project-intelligence.md`, `spec/12-integrations.md` | PlanetBids and BidNet scraping jobs produce signals |
| `integration/background-jobs` | `core/event-bus`, `domain/relationships`, `domain/intelligence-signals` | `spec/01-architecture.md` background jobs section | Scheduled jobs run: decay, email sync, scraping, enrichment |

Parallelism: `integration/file-upload` and `integration/gmail` in parallel. Then `integration/outlook` and `integration/bid-scrapers` in parallel. Then `integration/background-jobs`.

### Phase 7: Web Frontend

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `web/shell` | `foundation/packages-shared` | `spec/17-ui-ux.md`, `spec/01-architecture.md` | Next.js builds, layout renders, sidebar nav works |
| `web/auth-pages` | `web/shell`, `core/auth-module` | `spec/18-authentication.md` | Login, forgot password, accept invite pages work |
| `web/dashboard` | `web/auth-pages` | `spec/17-ui-ux.md` | Dashboard with stats, recent activity, pipeline summary |
| `web/contacts-pages` | `web/auth-pages`, `base/contacts-module` | `spec/17-ui-ux.md` | Contact list, contact detail, create/edit forms |
| `web/relationships-pages` | `web/contacts-pages`, `domain/relationships` | `spec/17-ui-ux.md`, `spec/05-relationship-intelligence.md` | Briefing cards, interest search, gesture tracking |
| `web/pipeline-pages` | `web/auth-pages`, `domain/opportunities` | `spec/17-ui-ux.md` | Kanban board, opportunity detail, scoring display |
| `web/bids-pages` | `web/auth-pages`, `complex/bids` | `spec/17-ui-ux.md`, `spec/07-bid-response.md` | Bid list, bid detail, calendar, decision matrix form |
| `web/estimating-pages` | `web/auth-pages`, `complex/estimating` | `spec/21-estimate-builder-ui.md` | Estimate builder, area markup, calculation review |
| `web/projects-pages` | `web/auth-pages`, `complex/project-lifecycle` | `spec/17-ui-ux.md`, `spec/08-project-lifecycle.md` | Project list, detail, daily logs table, punch list |
| `web/intelligence-pages` | `web/auth-pages`, `domain/intelligence-signals` | `spec/17-ui-ux.md` | Signal feed, source management |
| `web/settings-pages` | `web/auth-pages` | `spec/17-ui-ux.md`, `spec/13-user-roles-permissions.md` | User management, territories, products, integrations |

Parallelism: `web/shell` alone, then `web/auth-pages`, then `web/dashboard`, `web/contacts-pages`, and `web/pipeline-pages` 2-3 at a time. Build remaining page groups 2-3 at a time.

### Phase 8: Mobile App

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `mobile/shell` | `foundation/packages-shared` | `spec/11-mobile-field-app.md`, `spec/20-mobile-screens.md` | Expo project builds, navigation works |
| `mobile/sync-engine` | `mobile/shell`, `core/auth-module` | `spec/11-mobile-field-app.md` | WatermelonDB models, push/pull sync with API |
| `mobile/daily-log-screens` | `mobile/sync-engine`, `complex/project-lifecycle` | `spec/20-mobile-screens.md`, `spec/11-mobile-field-app.md` | Create daily log, photo capture, offline queue |
| `mobile/punch-list-screens` | `mobile/sync-engine`, `complex/project-lifecycle` | `spec/20-mobile-screens.md` | Punch list view, before/after photos, status transitions |
| `mobile/contact-screens` | `mobile/sync-engine`, `base/contacts-module` | `spec/20-mobile-screens.md` | Contact list, briefing card view |
| `mobile/schedule-screens` | `mobile/sync-engine`, `complex/project-lifecycle` | `spec/20-mobile-screens.md` | Project schedule view, crew assignments |

Parallelism: `mobile/shell` alone, then `mobile/sync-engine`, then `mobile/daily-log-screens` and `mobile/punch-list-screens` in parallel, then `mobile/contact-screens` and `mobile/schedule-screens` in parallel.

### Phase 9: Polish and E2E

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `polish/seed-data` | `complex/project-lifecycle`, `domain/relationships` | `spec/16-testing-strategy.md` | `pnpm db:seed` creates realistic sample data |
| `polish/e2e-tests` | `web/bids-pages`, `web/contacts-pages` | `spec/16-testing-strategy.md` | 5 critical Playwright flows pass |
| `polish/ci-pipeline` | `polish/e2e-tests` | `spec/15-deployment-infrastructure.md` | GitHub Actions workflow runs lint, typecheck, test, build |
| `polish/data-import` | `complex/bids`, `base/contacts-module` | `spec/22-data-import.md` | CSV import for contacts, organizations, products |

Parallelism: `polish/seed-data` and `polish/e2e-tests` in parallel. Then `polish/ci-pipeline` and `polish/data-import` in parallel.

## Module Implementation Pattern

Every backend domain module follows this structure. Read `spec/01-architecture.md` for the full pattern.

```text
apps/api/src/modules/{module-name}/
|-- {module-name}.routes.ts
|-- {module-name}.service.ts
|-- {module-name}.repository.ts
|-- {module-name}.schema.ts
|-- {module-name}.types.ts
`-- __tests__/
    |-- {module-name}.unit.test.ts
    `-- {module-name}.integration.test.ts
```

Key conventions:

- Routes use `fastify.authenticate` and `fastify.authorize('resource:action')` pre-handlers.
- Services never touch request/response objects.
- All API responses use the envelope `{ data: T, meta?: { cursor, hasMore, total } }`.
- Errors use the AppError hierarchy: `ValidationError`, `NotFoundError`, `BusinessRuleError`, `ForbiddenError`.
- Cursor-based pagination is required on list endpoints.

## Verification Commands by Phase

| Phase | Commands |
|-------|----------|
| Foundation | `pnpm install && pnpm build` |
| Core API | `podman-compose up -d && pnpm build && curl localhost:3001/health` |
| Base and Domain | `pnpm typecheck && pnpm lint && pnpm test` |
| Web Frontend | `pnpm build --filter=@salescraft/web` |
| Mobile | `cd apps/mobile && npx expo doctor` |
| E2E | `pnpm exec playwright test` |

## BUILD_STATE.md Format

Maintain this file at the project root. Create it on the first iteration.

```markdown
# Build State

## Current Phase: {1-9}
## Last Completed Item: {item-id}
## Status: {ready|in_progress|blocked}

## Completed
- [x] foundation/root-config (2026-06-20)

## In Progress
- [ ] core/auth-module - login endpoint done, refresh token WIP

## Blocked
- [ ] domain/intelligence-signals - blocked by: packages-ai type export issue

## Next Eligible
- core/event-bus (deps satisfied)
- base/contacts-module (deps satisfied)
```

## Quality Gates

Every piece of code must meet these standards:

1. TypeScript strict mode; avoid `any`, use `unknown` plus type guards when needed.
2. Zod validation on all API inputs: body, query, params.
3. Error handling on all async operations.
4. Tests alongside code: unit tests for business logic, integration tests for endpoints.
5. Consistent patterns; follow existing module structure.
6. No dead code or empty "for later" scaffolds.
7. Security by default: parameterized queries via Prisma, input validation, auth on every protected route.

## First Iteration Bootstrapping

If `BUILD_STATE.md` does not exist, this is the first iteration. Start with `foundation/root-config`:

1. Create root `package.json`, `turbo.json`, `pnpm-workspace.yaml`, `tsconfig.base.json`, `.eslintrc.js`, `.prettierrc`, `.gitignore`, `.env.example`.
2. Create `podman/` with `podman-compose.yml`, `postgres/init.sql`, `localstack/init-s3.sh`.
3. Create package shells: `packages/shared/package.json`, `packages/database/package.json`, `packages/ai/package.json`, `apps/api/package.json`, `apps/web/package.json`, `apps/mobile/package.json`.
4. Run `pnpm install`.
5. Verify the monorepo builds with empty packages.
6. Create `BUILD_STATE.md` and mark `foundation/root-config` complete.
7. Commit: `feat: initialize monorepo with turborepo, pnpm workspaces, and podman config`.

Then continue to the next eligible items in later iterations.

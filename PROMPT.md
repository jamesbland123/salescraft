# Salescraft Build Loop

You are building Salescraft — a SaaS platform for commercial flooring companies to win and execute government contracts. The codebase is a TypeScript monorepo. You are running in a loop: each iteration you assess state, pick the next work items, build them (with parallel subagents where possible), verify, and commit.

## Tech Stack

- **Monorepo**: Turborepo + pnpm workspaces
- **Backend**: Fastify 4.x, TypeScript 5.4+, Prisma 5.x, PostgreSQL 15+ (pgvector, pg_trgm), BullMQ 5.x, Redis 7+
- **Frontend**: Next.js 14+ (App Router), React 18, Tailwind CSS 3.x, shadcn/ui, TanStack Query 5.x
- **Mobile**: React Native 0.73+, Expo 50+, WatermelonDB 0.27+
- **AI**: AWS Bedrock (@aws-sdk/client-bedrock-runtime), Claude Sonnet/Haiku, Titan embeddings
- **Containers**: Podman + podman-compose (NOT Docker)
- **Validation**: Zod 3.x (shared between client/server)
- **Testing**: Vitest, Supertest, MSW, Playwright

Specs live in `spec/`. Read the relevant spec file before implementing any module.

---

## Iteration Protocol

Every iteration follows this exact sequence:

### Step 1: Assess State

```
1. Read BUILD_STATE.md (create it if it doesn't exist)
2. Run `pnpm build` (if packages exist) to verify current state compiles
3. Check git status for uncommitted changes — if found, investigate and either commit or discard
4. Cross-reference filesystem against BUILD_STATE.md completions
5. Fix any discrepancies in BUILD_STATE.md
```

### Step 2: Select Work Items

```
1. Find all items in the dependency graph whose dependencies are ALL marked complete
2. If any item is marked "in_progress" or "partial", finish it first (error recovery)
3. From eligible items, select up to 3 that can be parallelized (non-overlapping directories)
4. If only 1 item is eligible, build it directly (no subagent needed)
```

### Step 3: Build

```
1. For each selected item, read the relevant spec file(s)
2. Spawn parallel subagents (max 2-3) for independent items
3. Each subagent: implements code + writes tests for that code
4. If items share files (barrel exports, shared types), do them sequentially
```

### Step 4: Verify

```
1. Run `pnpm typecheck` (must pass)
2. Run `pnpm lint` (must pass — fix lint errors, don't disable rules)
3. Run `pnpm test` (must pass)
4. For API modules: verify the server starts and health check responds
5. For web: verify `pnpm build --filter=@salescraft/web` succeeds
```

### Step 5: Commit & Update State

```
1. Stage all new/modified files
2. Commit with descriptive message: "feat(module): description"
3. Update BUILD_STATE.md — mark completed items, note any failures
```

---

## Parallelization Rules

- **Max 2-3 subagents** per iteration
- Subagents MUST work in **non-overlapping directories**
- If two items both touch `packages/shared/src/schemas/` or `packages/database/prisma/schema.prisma`, they CANNOT be parallel
- After parallel work completes, run verification on the combined result before committing
- If a merge conflict or type error arises from parallel work, fix it before committing

### Directory Isolation Examples

These CAN be parallel:
- `packages/ai/` + `apps/api/src/modules/contacts/`
- `apps/api/src/modules/relationships/` + `apps/api/src/modules/products/`
- `apps/web/src/app/(dashboard)/contacts/` + `apps/web/src/app/(dashboard)/bids/`

These CANNOT be parallel:
- Two modules that both add to `packages/shared/src/schemas/index.ts`
- Two modules that both modify `packages/database/prisma/schema.prisma`
- `apps/api/src/modules/bids/` + `apps/api/src/modules/estimating/` (bids imports estimating)

---

## Error Recovery

- **Build fails on entry**: Fix compilation errors before starting new work. Mark the broken item as "needs_fix" in BUILD_STATE.md.
- **Partial completion**: If BUILD_STATE.md shows an item "in_progress", inspect its directory. If partially built, finish it. If broken, revert and rebuild.
- **Test failures**: Fix failing tests before proceeding to new items. Max 3 fix attempts per issue — if still failing, mark as "blocked" and move to the next eligible item.
- **Subagent conflict**: If parallel subagents produce conflicting changes, merge manually — prefer the more complete implementation.

---

## Dependency Graph

Items are listed as `id | depends_on | spec_file(s) | verification`. An item is eligible when ALL items in its `depends_on` list are complete.

### Phase 1: Foundation

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `foundation/root-config` | — | spec/25-project-configuration.md | `pnpm install` succeeds |
| `foundation/podman-compose` | root-config | spec/25-project-configuration.md, spec/15-deployment-infrastructure.md | `podman-compose up -d` starts postgres, redis, localstack, mailhog |
| `foundation/packages-shared` | root-config | spec/01-architecture.md, spec/02-domain-model.md | `pnpm build --filter=@salescraft/shared` succeeds |
| `foundation/packages-database` | root-config, packages-shared | spec/24-complete-prisma-schema.md | `pnpm build --filter=@salescraft/database` succeeds, `pnpm db:migrate` runs |
| `foundation/test-setup` | root-config | spec/16-testing-strategy.md | vitest config exists, test helpers compile |

**Parallelism**: After `root-config`, do `podman-compose` + `packages-shared` in parallel. Then `packages-database` + `test-setup` in parallel.

### Phase 2: Core API

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `core/api-bootstrap` | foundation/packages-shared, foundation/packages-database | spec/01-architecture.md, spec/25-project-configuration.md | API starts, `GET /health` returns 200 |
| `core/auth-module` | core/api-bootstrap | spec/18-authentication.md, spec/13-user-roles-permissions.md | Login/refresh/logout endpoints work, JWT issued |
| `core/event-bus` | core/api-bootstrap | spec/01-architecture.md (Event System section) | Events emit and are received by handlers, BullMQ queues created |

**Parallelism**: `auth-module` + `event-bus` in parallel after bootstrap.

### Phase 3: AI & Base Entities

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `base/packages-ai` | foundation/packages-shared | spec/09-ai-engine.md | `pnpm build --filter=@salescraft/ai` succeeds, client instantiates |
| `base/contacts-module` | core/auth-module | spec/02-domain-model.md, spec/05-relationship-intelligence.md | CRUD endpoints work, search works, territory filtering works |
| `base/organizations-module` | core/auth-module | spec/02-domain-model.md, spec/03-government-procurement.md | CRUD endpoints work, facility linking works |
| `base/territories-module` | core/auth-module | spec/02-domain-model.md | CRUD endpoints, user assignment, zip/city matching |

**Parallelism**: `packages-ai` + `contacts-module` in parallel. Then `organizations-module` + `territories-module` in parallel.

### Phase 4: Domain Features

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `domain/relationships` | base/contacts-module | spec/05-relationship-intelligence.md | Scoring endpoint, decay calculation, briefing generation, interest matching |
| `domain/opportunities` | base/contacts-module, base/organizations-module | spec/04-project-intelligence.md | CRUD, scoring, state machine transitions |
| `domain/intelligence-signals` | domain/opportunities, base/packages-ai | spec/04-project-intelligence.md | Signal creation, AI classification, signal→opportunity correlation |
| `domain/products` | core/auth-module | spec/06-flooring-estimating.md | Product CRUD, pricing, specifications |
| `domain/communications` | base/contacts-module | spec/10-communication-hub.md | Interaction logging, activity timeline, email templates |
| `domain/notifications` | core/event-bus, core/auth-module | spec/01-architecture.md (WebSocket section) | In-app notifications, WebSocket delivery |

**Parallelism**: `relationships` + `products` in parallel. Then `opportunities` + `communications` in parallel. Then `intelligence-signals` + `notifications` in parallel.

### Phase 5: Complex Modules

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `complex/bids` | domain/opportunities, domain/products, domain/communications | spec/07-bid-response.md, spec/03-government-procurement.md | State machine (discovered→submitted→awarded), RFP parsing, decision matrix, checklists |
| `complex/estimating` | domain/products, base/organizations-module | spec/06-flooring-estimating.md, spec/21-estimate-builder-ui.md | Area calculations, waste factors, prevailing wage, estimate totals |
| `complex/project-lifecycle` | complex/bids | spec/08-project-lifecycle.md | Project creation from bid, daily logs, punch lists, financial tracking |

**Parallelism**: `bids` + `estimating` in parallel. Then `project-lifecycle`.

### Phase 6: Integrations

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `integration/file-upload` | core/api-bootstrap | spec/12-integrations.md | Presigned URL generation, upload flow, file metadata stored |
| `integration/gmail` | domain/communications | spec/12-integrations.md | OAuth flow, email sync, send-on-behalf |
| `integration/outlook` | domain/communications | spec/12-integrations.md | OAuth flow, email sync |
| `integration/bid-scrapers` | domain/intelligence-signals | spec/04-project-intelligence.md, spec/12-integrations.md | PlanetBids, BidNet scraping jobs produce signals |
| `integration/background-jobs` | core/event-bus, domain/relationships, domain/intelligence-signals | spec/01-architecture.md (Background Jobs section) | All scheduled jobs run: decay, email-sync, scraping, enrichment |

**Parallelism**: `file-upload` + `gmail` in parallel. Then `outlook` + `bid-scrapers` in parallel. Then `background-jobs`.

### Phase 7: Web Frontend

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `web/shell` | foundation/packages-shared | spec/17-ui-ux.md, spec/01-architecture.md | Next.js builds, layout renders, sidebar nav works |
| `web/auth-pages` | web/shell, core/auth-module | spec/18-authentication.md | Login, forgot password, accept invite pages work |
| `web/dashboard` | web/auth-pages | spec/17-ui-ux.md | Dashboard with stats, recent activity, pipeline summary |
| `web/contacts-pages` | web/auth-pages, base/contacts-module | spec/17-ui-ux.md | Contact list (filterable table), contact detail, create/edit forms |
| `web/relationships-pages` | web/contacts-pages, domain/relationships | spec/17-ui-ux.md, spec/05-relationship-intelligence.md | Briefing cards, interest search, gesture tracking |
| `web/pipeline-pages` | web/auth-pages, domain/opportunities | spec/17-ui-ux.md | Kanban board, opportunity detail, scoring display |
| `web/bids-pages` | web/auth-pages, complex/bids | spec/17-ui-ux.md, spec/07-bid-response.md | Bid list, bid detail, calendar, decision matrix form |
| `web/estimating-pages` | web/auth-pages, complex/estimating | spec/21-estimate-builder-ui.md | Estimate builder, area markup, calculation review |
| `web/projects-pages` | web/auth-pages, complex/project-lifecycle | spec/17-ui-ux.md, spec/08-project-lifecycle.md | Project list, detail, daily logs table, punch list |
| `web/intelligence-pages` | web/auth-pages, domain/intelligence-signals | spec/17-ui-ux.md | Signal feed, source management |
| `web/settings-pages` | web/auth-pages | spec/17-ui-ux.md, spec/13-user-roles-permissions.md | User management, territories, products, integrations |

**Parallelism**: `shell` alone. Then `auth-pages`. Then `dashboard` + `contacts-pages` + `pipeline-pages` (2-3 at a time). Then remaining page groups 2-3 at a time.

### Phase 8: Mobile App

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `mobile/shell` | foundation/packages-shared | spec/11-mobile-field-app.md, spec/20-mobile-screens.md | Expo project builds, navigation works |
| `mobile/sync-engine` | mobile/shell, core/auth-module | spec/11-mobile-field-app.md | WatermelonDB models, push/pull sync with API |
| `mobile/daily-log-screens` | mobile/sync-engine, complex/project-lifecycle | spec/20-mobile-screens.md, spec/11-mobile-field-app.md | Create daily log, photo capture, offline queue |
| `mobile/punch-list-screens` | mobile/sync-engine, complex/project-lifecycle | spec/20-mobile-screens.md | Punch list view, before/after photos, status transitions |
| `mobile/contact-screens` | mobile/sync-engine, base/contacts-module | spec/20-mobile-screens.md | Contact list, briefing card view |
| `mobile/schedule-screens` | mobile/sync-engine, complex/project-lifecycle | spec/20-mobile-screens.md | Project schedule view, crew assignments |

**Parallelism**: `shell` alone. Then `sync-engine`. Then `daily-log-screens` + `punch-list-screens` in parallel. Then `contact-screens` + `schedule-screens` in parallel.

### Phase 9: Polish & E2E

| ID | Depends On | Spec | Verification |
|----|-----------|------|--------------|
| `polish/seed-data` | complex/project-lifecycle, domain/relationships | spec/16-testing-strategy.md | `pnpm db:seed` creates realistic sample data |
| `polish/e2e-tests` | web/bids-pages, web/contacts-pages | spec/16-testing-strategy.md | 5 critical Playwright flows pass |
| `polish/ci-pipeline` | polish/e2e-tests | spec/15-deployment-infrastructure.md | GitHub Actions workflow runs lint + typecheck + test + build |
| `polish/data-import` | complex/bids, base/contacts-module | spec/22-data-import.md | CSV import for contacts, organizations, products |

**Parallelism**: `seed-data` + `e2e-tests` in parallel. Then `ci-pipeline` + `data-import` in parallel.

---

## Module Implementation Pattern

Every backend domain module follows this structure. Read `spec/01-architecture.md` for the full pattern.

```
apps/api/src/modules/{module-name}/
├── {module-name}.routes.ts      # Fastify route definitions with schema validation
├── {module-name}.service.ts     # Business logic (no HTTP concerns)
├── {module-name}.repository.ts  # Complex queries only (skip for simple CRUD)
├── {module-name}.schema.ts      # Zod schemas for request/response validation
├── {module-name}.types.ts       # Module-specific TypeScript types
└── __tests__/
    ├── {module-name}.unit.test.ts        # Business logic tests
    └── {module-name}.integration.test.ts # API endpoint tests with real DB
```

Key conventions:
- Routes use `fastify.authenticate` and `fastify.authorize('resource:action')` preHandlers
- Services never touch request/response objects
- All API responses use the envelope: `{ data: T, meta?: { cursor, hasMore, total } }`
- Errors use the AppError hierarchy (ValidationError, NotFoundError, BusinessRuleError, ForbiddenError)
- Cursor-based pagination on all list endpoints

---

## Verification Commands by Phase

| Phase | Commands |
|-------|----------|
| Foundation | `pnpm install && pnpm build` |
| Core API | `podman-compose up -d && pnpm build && curl localhost:3001/health` |
| Base + Domain | `pnpm typecheck && pnpm lint && pnpm test` |
| Web Frontend | `pnpm build --filter=@salescraft/web` |
| Mobile | `cd apps/mobile && npx expo doctor` |
| E2E | `pnpm exec playwright test` |

---

## BUILD_STATE.md Format

Maintain this file at the project root. Create it on first iteration.

```markdown
# Build State

## Current Phase: {1-9}
## Last Completed Item: {item-id}
## Status: {ready|in_progress|blocked}

## Completed
- [x] foundation/root-config (2024-01-15)
- [x] foundation/podman-compose (2024-01-15)
...

## In Progress
- [ ] core/auth-module — login endpoint done, refresh token WIP

## Blocked
- [ ] domain/intelligence-signals — blocked by: packages-ai type export issue

## Next Eligible
- core/event-bus (deps satisfied)
- base/contacts-module (deps satisfied)
```

---

## Quality Gates

Every piece of code must meet these standards:

1. **TypeScript strict mode** — no `any` (use `unknown` + type guards)
2. **Zod validation** on all API inputs (body, query, params)
3. **Error handling** on all async operations (no unhandled promise rejections)
4. **Tests alongside code** — unit tests for business logic, integration tests for endpoints
5. **Consistent patterns** — follow the module pattern exactly, don't invent new structures
6. **No dead code** — don't scaffold empty files "for later"
7. **Security** — parameterized queries (Prisma handles this), input validation, auth on every route

---

## Subagent Prompt Template

When spawning a subagent for a work item, provide:

```
Build the {module-name} module for Salescraft.

Read these spec files first:
- {relevant spec files}

Follow the module pattern in apps/api/src/modules/ (see existing modules for reference).

Requirements:
- {key requirements from the dependency graph}

Write tests:
- Unit tests for all business logic (calculation, scoring, state transitions)
- Integration tests for API endpoints (use test helpers in apps/api/test/helpers/)

Verify:
- `pnpm typecheck` passes
- `pnpm test --filter=@salescraft/api` passes
- No lint errors
```

---

## First Iteration Bootstrapping

If BUILD_STATE.md does not exist, this is the first iteration. Start with `foundation/root-config`:

1. Create root `package.json`, `turbo.json`, `pnpm-workspace.yaml`, `tsconfig.base.json`, `.eslintrc.js`, `.prettierrc`, `.gitignore`, `.env.example`
2. Create `podman/` directory with `podman-compose.yml`, `postgres/init.sql`, `localstack/init-s3.sh`
3. Create package shells: `packages/shared/package.json`, `packages/database/package.json`, `packages/ai/package.json`, `apps/api/package.json`, `apps/web/package.json`
4. Run `pnpm install`
5. Verify the monorepo builds with empty packages
6. Create BUILD_STATE.md and mark `foundation/root-config` complete
7. Commit: "feat: initialize monorepo with turborepo, pnpm workspaces, and podman config"

Then continue to the next eligible items in subsequent iterations.

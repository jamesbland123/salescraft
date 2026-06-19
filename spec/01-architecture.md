# Architecture

## Vision & Purpose

Salescraft's architecture serves one goal: enable a small commercial flooring company to operate like a much larger, more sophisticated organization. The architecture must be:

- **Simple to develop** — a single developer or AI agent should be able to run the entire stack locally with one command
- **Domain-rich** — the code structure mirrors the business domains (relationships, estimating, bids, projects) not generic CRUD
- **AI-native** — every domain can leverage AI capabilities without ceremony; Bedrock calls are as easy as database queries
- **Offline-capable** — field crews work in buildings with no connectivity; the mobile app must function fully offline
- **AWS-deployable** — local podman-compose maps cleanly to AWS services; no local-only assumptions in application code

## Technology Stack

### Backend
| Layer | Choice | Version | Rationale |
|-------|--------|---------|-----------|
| Runtime | Node.js | 20 LTS | TypeScript native support, large ecosystem |
| Language | TypeScript | 5.4+ | End-to-end type safety with shared types |
| HTTP Framework | Fastify | 4.x | Performance, built-in schema validation, TypeScript-first |
| ORM | Prisma | 5.x | Type-safe queries, declarative schema, excellent migrations |
| Validation | Zod | 3.x | Runtime validation, TypeScript inference, shared client/server |
| Auth | @fastify/jwt + @fastify/cookie | latest | JWT access tokens + HTTP-only refresh cookies |
| Background Jobs | BullMQ | 5.x | Redis-backed, reliable, dashboard UI, cron scheduling |
| WebSockets | @fastify/websocket | latest | Real-time notifications, activity feeds |
| Logging | Pino | 8.x | Structured JSON, fast, Fastify-native |
| File Upload | @fastify/multipart | latest | Streaming to S3 |
| Email | Nodemailer | 6.x | SMTP/OAuth2 sending, IMAP receiving |

### Frontend (Web)
| Layer | Choice | Version | Rationale |
|-------|--------|---------|-----------|
| Framework | Next.js | 14+ (App Router) | SSR for SEO-irrelevant but fast initial loads, API routes for BFF |
| UI Library | React | 18+ | Component model, ecosystem, shared with React Native |
| Styling | Tailwind CSS | 3.x | Utility-first, consistent, fast iteration |
| Component Library | shadcn/ui | latest | Accessible, customizable, Tailwind-native components |
| State Management | TanStack Query | 5.x | Server state caching, mutations, optimistic updates |
| Forms | React Hook Form + Zod | latest | Performance, validation reuse from backend |
| Tables | TanStack Table | 8.x | Headless, sortable, filterable, virtualizable |
| Maps | Mapbox GL JS | 3.x | Territory visualization, facility locations |
| Charts | Recharts | 2.x | Pipeline charts, dashboards, forecasting |
| Date Handling | date-fns | 3.x | Tree-shakeable, immutable, timezone-aware |

### Mobile
| Layer | Choice | Version | Rationale |
|-------|--------|---------|-----------|
| Framework | React Native | 0.73+ | Code sharing with web, single team |
| Toolkit | Expo | 50+ | Managed workflow, OTA updates, camera/location |
| Navigation | React Navigation | 6.x | Standard for RN, deep linking |
| Offline Storage | WatermelonDB | 0.27+ | SQLite-backed, sync primitives, reactive queries |
| Camera | expo-camera + expo-image-picker | latest | Photo documentation |
| Notifications | expo-notifications | latest | Push via FCM/APNs |

### Data Layer
| Layer | Choice | Rationale |
|-------|--------|-----------|
| Primary DB | PostgreSQL 15+ | JSONB for flexible fields, full-text search, proven reliability |
| Cache / Queue Backend | Redis 7+ | BullMQ backing, session cache, rate limiting |
| Search | PostgreSQL pg_trgm + tsvector | Full-text and fuzzy search without separate infrastructure |
| Vector Store | pgvector extension | AI embeddings stored alongside relational data |
| File Storage | AWS S3 (LocalStack in dev) | Floor plans, photos, proposals, RFP documents |
| CDN | CloudFront (dev: direct S3 access) | Static assets and uploaded file delivery |

### AI Infrastructure
| Layer | Choice | Rationale |
|-------|--------|-----------|
| Provider | AWS Bedrock | Managed, multi-model, no GPU infrastructure |
| SDK | @aws-sdk/client-bedrock-runtime | Official AWS SDK for model invocation |
| Orchestration | Custom routing layer (not LangChain) | Simpler, fewer dependencies, full control |
| Embeddings | Amazon Titan Embed v2 | Cost-effective, good quality for semantic search |
| Generation (complex) | Claude 3.5 Sonnet via Bedrock | Proposal writing, RFP analysis, conversation intelligence |
| Generation (simple) | Claude 3 Haiku via Bedrock | Classification, extraction, scoring explanations |
| Vector Store | pgvector | Co-located with relational data, no separate service |

### Infrastructure
| Layer | Choice | Rationale |
|-------|--------|-----------|
| Container Runtime | Podman + podman-compose | Local development parity (Docker-compatible) |
| Monorepo | Turborepo | Fast builds, dependency graph, caching |
| Package Manager | pnpm | Fast, disk-efficient, workspace support |
| CI/CD | GitHub Actions | Familiar, good AWS integration |
| IaC | AWS CDK (TypeScript) | Infrastructure as code in the same language |
| Monitoring (prod) | CloudWatch + structured logs | Native AWS, no extra services |

## Project Structure

```
salescraft/
├── apps/
│   ├── api/                          # Fastify backend
│   │   ├── src/
│   │   │   ├── server.ts             # Fastify app bootstrap
│   │   │   ├── config.ts             # Environment-based configuration
│   │   │   ├── plugins/              # Fastify plugins (auth, cors, rate-limit)
│   │   │   │   ├── auth.ts
│   │   │   │   ├── cors.ts
│   │   │   │   └── error-handler.ts
│   │   │   ├── modules/              # Domain modules (one per business domain)
│   │   │   │   ├── contacts/
│   │   │   │   │   ├── contacts.routes.ts
│   │   │   │   │   ├── contacts.service.ts
│   │   │   │   │   ├── contacts.repository.ts
│   │   │   │   │   ├── contacts.schema.ts      # Zod schemas for this module
│   │   │   │   │   └── contacts.types.ts       # Module-specific types
│   │   │   │   ├── relationships/
│   │   │   │   ├── projects/
│   │   │   │   ├── bids/
│   │   │   │   ├── estimating/
│   │   │   │   ├── intelligence/
│   │   │   │   ├── communications/
│   │   │   │   ├── ai/
│   │   │   │   └── auth/
│   │   │   ├── jobs/                 # BullMQ job processors
│   │   │   │   ├── bid-scraper.job.ts
│   │   │   │   ├── email-sync.job.ts
│   │   │   │   ├── ai-enrichment.job.ts
│   │   │   │   └── relationship-decay.job.ts
│   │   │   ├── middleware/           # Cross-cutting middleware
│   │   │   │   ├── authenticate.ts
│   │   │   │   ├── authorize.ts
│   │   │   │   └── validate.ts
│   │   │   └── lib/                  # Shared utilities
│   │   │       ├── errors.ts
│   │   │       ├── pagination.ts
│   │   │       └── logger.ts
│   │   ├── test/
│   │   ├── Dockerfile
│   │   └── package.json
│   │
│   ├── web/                          # Next.js frontend
│   │   ├── src/
│   │   │   ├── app/                  # App Router pages
│   │   │   │   ├── (auth)/           # Auth group (login, register)
│   │   │   │   ├── (dashboard)/      # Main app group
│   │   │   │   │   ├── contacts/
│   │   │   │   │   ├── relationships/
│   │   │   │   │   ├── pipeline/
│   │   │   │   │   ├── bids/
│   │   │   │   │   ├── projects/
│   │   │   │   │   ├── intelligence/
│   │   │   │   │   └── settings/
│   │   │   │   └── layout.tsx
│   │   │   ├── components/           # Shared UI components
│   │   │   │   ├── ui/              # shadcn/ui base components
│   │   │   │   ├── layout/          # Shell, sidebar, header
│   │   │   │   ├── contacts/        # Contact-specific components
│   │   │   │   ├── relationships/   # Relationship briefing cards
│   │   │   │   ├── bids/            # Bid calendar, bid cards
│   │   │   │   └── common/          # DataTable, SearchInput, etc.
│   │   │   ├── hooks/               # Custom React hooks
│   │   │   ├── lib/                 # Client utilities
│   │   │   │   ├── api.ts           # API client (fetch wrapper)
│   │   │   │   ├── auth.ts          # Auth context/hooks
│   │   │   │   └── utils.ts
│   │   │   └── styles/
│   │   ├── public/
│   │   └── package.json
│   │
│   └── mobile/                       # React Native / Expo
│       ├── src/
│       │   ├── screens/
│       │   ├── components/
│       │   ├── navigation/
│       │   ├── hooks/
│       │   ├── services/
│       │   │   ├── sync.ts          # Offline sync engine
│       │   │   └── api.ts
│       │   └── db/                  # WatermelonDB models
│       ├── app.json
│       └── package.json
│
├── packages/
│   ├── shared/                       # Shared between all apps
│   │   ├── src/
│   │   │   ├── schemas/             # Zod schemas (source of truth for validation)
│   │   │   │   ├── contact.schema.ts
│   │   │   │   ├── bid.schema.ts
│   │   │   │   ├── project.schema.ts
│   │   │   │   └── index.ts
│   │   │   ├── types/               # TypeScript types derived from schemas
│   │   │   │   └── index.ts
│   │   │   ├── constants/           # Business constants (states, categories)
│   │   │   │   ├── flooring-types.ts
│   │   │   │   ├── bid-states.ts
│   │   │   │   └── government.ts
│   │   │   └── utils/               # Pure utility functions
│   │   │       ├── money.ts
│   │   │       ├── area.ts
│   │   │       └── date.ts
│   │   └── package.json
│   │
│   ├── database/                     # Prisma schema and migrations
│   │   ├── prisma/
│   │   │   ├── schema.prisma
│   │   │   ├── migrations/
│   │   │   └── seed.ts
│   │   └── package.json
│   │
│   └── ai/                           # AI/Bedrock abstraction
│       ├── src/
│       │   ├── client.ts            # Bedrock client singleton
│       │   ├── router.ts            # Model routing logic
│       │   ├── prompts/             # Prompt templates
│       │   │   ├── proposal-writer.ts
│       │   │   ├── rfp-parser.ts
│       │   │   ├── lead-scorer.ts
│       │   │   ├── interest-extractor.ts
│       │   │   └── conversation-analyzer.ts
│       │   ├── embeddings.ts        # Embedding generation
│       │   └── types.ts
│       └── package.json
│
├── podman/
│   ├── podman-compose.yml            # Full local stack
│   ├── podman-compose.test.yml       # Test environment
│   ├── postgres/
│   │   └── init.sql                 # Extensions (pgvector, pg_trgm)
│   └── redis/
│       └── redis.conf
│
├── infra/                            # AWS CDK
│   ├── lib/
│   │   ├── vpc-stack.ts
│   │   ├── database-stack.ts
│   │   ├── api-stack.ts
│   │   ├── web-stack.ts
│   │   └── ai-stack.ts
│   └── package.json
│
├── scripts/
│   ├── dev-setup.sh                 # One-command dev environment
│   ├── seed-data.ts                 # Generate realistic seed data
│   └── deploy.sh
│
├── turbo.json                        # Turborepo config
├── pnpm-workspace.yaml
├── package.json                      # Root package.json
├── .env.example                      # Environment variables template
├── .eslintrc.js
├── .prettierrc
└── tsconfig.base.json               # Shared TypeScript config
```

## Module Pattern

Every backend domain module follows this structure:

### Routes (`*.routes.ts`)
```typescript
import { FastifyPluginAsync } from 'fastify';
import { contactsService } from './contacts.service';
import { CreateContactSchema, UpdateContactSchema, ListContactsQuery } from './contacts.schema';

const contactsRoutes: FastifyPluginAsync = async (fastify) => {
  fastify.get('/', {
    schema: { querystring: ListContactsQuery },
    preHandler: [fastify.authenticate, fastify.authorize('contacts:read')],
  }, async (request, reply) => {
    const result = await contactsService.list(request.query);
    return reply.send(result);
  });

  fastify.post('/', {
    schema: { body: CreateContactSchema },
    preHandler: [fastify.authenticate, fastify.authorize('contacts:write')],
  }, async (request, reply) => {
    const contact = await contactsService.create(request.body, request.user);
    return reply.status(201).send(contact);
  });
};

export default contactsRoutes;
```

### Service (`*.service.ts`)
Business logic layer. Orchestrates repositories, AI calls, events. Never touches HTTP request/response objects.

```typescript
import { prisma } from '@salescraft/database';
import { EventBus } from '../../lib/events';

export const contactsService = {
  async create(data: CreateContactInput, actor: User) {
    const contact = await prisma.contact.create({ data: { ...data, createdBy: actor.id } });
    await EventBus.emit('contact.created', { contact, actor });
    return contact;
  },

  async list(query: ListContactsQuery) {
    // Cursor-based pagination, filtering, search
  },
};
```

### Repository (`*.repository.ts`)
Only used when queries are complex enough to warrant abstraction beyond Prisma's standard API (e.g., full-text search, complex aggregations, raw SQL for performance).

### Schema (`*.schema.ts`)
Zod schemas for request validation. These are imported from `@salescraft/shared` when they need to be shared with the frontend.

## API Design Conventions

### Base URL
```
Local:  http://localhost:3001/api/v1
Prod:   https://api.salescraft.example.com/v1
```

### Request/Response Format

All responses follow this envelope:

```typescript
// Success
interface ApiResponse<T> {
  data: T;
  meta?: {
    cursor?: string;
    hasMore?: boolean;
    total?: number;
  };
}

// Error
interface ApiError {
  error: {
    code: string;          // Machine-readable: "CONTACT_NOT_FOUND"
    message: string;       // Human-readable: "Contact with ID xyz not found"
    details?: unknown;     // Validation errors, context
    requestId: string;     // For log correlation
  };
}
```

### Pagination (Cursor-Based)
```
GET /api/v1/contacts?cursor=abc123&limit=50&sort=lastName&order=asc
```

Response includes:
```json
{
  "data": [...],
  "meta": { "cursor": "def456", "hasMore": true }
}
```

### Filtering
```
GET /api/v1/contacts?filter[organizationType]=school_district&filter[territory]=west&search=riverside
```

### Status Codes
| Code | Usage |
|------|-------|
| 200 | Success (GET, PUT, PATCH) |
| 201 | Created (POST) |
| 204 | No Content (DELETE) |
| 400 | Validation error, bad request |
| 401 | Not authenticated |
| 403 | Not authorized |
| 404 | Resource not found |
| 409 | Conflict (duplicate, state violation) |
| 422 | Business rule violation |
| 429 | Rate limited |
| 500 | Internal server error |

### Naming Conventions
- Resources are plural nouns: `/contacts`, `/bids`, `/projects`
- Actions use verbs: `POST /bids/:id/submit`, `POST /contacts/:id/enrich`
- Nested resources for strong ownership: `/projects/:id/daily-logs`
- Query params for weak associations: `/contacts?organizationId=xyz`

## Event System

### Domain Events
Every meaningful state change emits a domain event. Events are processed asynchronously via BullMQ.

```typescript
// Event types
type DomainEvent =
  | { type: 'contact.created'; payload: { contactId: string; actor: string } }
  | { type: 'contact.enriched'; payload: { contactId: string; source: string } }
  | { type: 'bid.discovered'; payload: { bidId: string; source: string } }
  | { type: 'bid.submitted'; payload: { bidId: string; submittedBy: string } }
  | { type: 'bid.awarded'; payload: { bidId: string; won: boolean } }
  | { type: 'project.status_changed'; payload: { projectId: string; from: string; to: string } }
  | { type: 'relationship.decayed'; payload: { contactId: string; daysSinceContact: number } }
  | { type: 'intelligence.opportunity_found'; payload: { type: string; details: unknown } }
  | { type: 'interaction.logged'; payload: { contactId: string; type: string } };
```

### Event Consumers
Events trigger side effects:
- `contact.created` → queue AI enrichment job
- `bid.discovered` → notify relevant sales reps
- `interaction.logged` → update relationship health score
- `relationship.decayed` → send "reconnect" notification

### Background Jobs (BullMQ Queues)
| Queue | Purpose | Schedule |
|-------|---------|----------|
| `bid-scraping` | Scan government bid portals | Every 2 hours |
| `email-sync` | Pull new emails from connected accounts | Every 5 minutes |
| `ai-enrichment` | Enrich contacts/orgs with AI | On contact creation + weekly refresh |
| `relationship-decay` | Calculate relationship health decay | Daily at 6 AM |
| `intelligence-scan` | Monitor bonds, CIPs, meeting agendas | Daily at 7 AM |
| `report-generation` | Generate scheduled reports/digests | Daily/weekly per user config |

## Authentication & Authorization

### Auth Flow
1. User logs in with email/password → receives JWT access token (15 min) + HTTP-only refresh cookie (7 days)
2. Access token sent in `Authorization: Bearer <token>` header
3. When access token expires, client calls `POST /auth/refresh` with cookie
4. Refresh token rotation: each refresh issues new refresh token, invalidating the old one

### JWT Payload
```typescript
interface JwtPayload {
  sub: string;        // User ID
  role: string;       // 'sales_rep' | 'estimator' | 'project_manager' | 'installer' | 'admin' | 'owner'
  permissions: string[]; // Computed from role
  iat: number;
  exp: number;
}
```

### Authorization Middleware
```typescript
// Usage in routes
fastify.get('/bids', {
  preHandler: [fastify.authenticate, fastify.authorize('bids:read')]
}, handler);

// Authorize checks JWT permissions array
fastify.decorate('authorize', (permission: string) => {
  return async (request: FastifyRequest) => {
    if (!request.user.permissions.includes(permission)) {
      throw new ForbiddenError(`Missing permission: ${permission}`);
    }
  };
});
```

## Error Handling

### Error Class Hierarchy
```typescript
class AppError extends Error {
  constructor(
    public statusCode: number,
    public code: string,
    message: string,
    public details?: unknown
  ) { super(message); }
}

class ValidationError extends AppError {
  constructor(details: ZodError) {
    super(400, 'VALIDATION_ERROR', 'Invalid input', details.flatten());
  }
}

class NotFoundError extends AppError {
  constructor(resource: string, id: string) {
    super(404, `${resource.toUpperCase()}_NOT_FOUND`, `${resource} with ID ${id} not found`);
  }
}

class BusinessRuleError extends AppError {
  constructor(code: string, message: string) {
    super(422, code, message);
  }
}

class ForbiddenError extends AppError {
  constructor(message: string) {
    super(403, 'FORBIDDEN', message);
  }
}
```

### Global Error Handler
```typescript
fastify.setErrorHandler((error, request, reply) => {
  const requestId = request.id;

  if (error instanceof AppError) {
    request.log.warn({ err: error, requestId }, error.message);
    return reply.status(error.statusCode).send({
      error: { code: error.code, message: error.message, details: error.details, requestId }
    });
  }

  // Unexpected errors
  request.log.error({ err: error, requestId }, 'Unhandled error');
  return reply.status(500).send({
    error: { code: 'INTERNAL_ERROR', message: 'An unexpected error occurred', requestId }
  });
});
```

## Configuration

### Environment Variables
```bash
# .env.example
NODE_ENV=development
PORT=3001
DATABASE_URL=postgresql://salescraft:salescraft@localhost:5432/salescraft
REDIS_URL=redis://localhost:6379

# Auth
JWT_SECRET=dev-secret-change-in-prod
JWT_EXPIRES_IN=15m
REFRESH_TOKEN_EXPIRES_IN=7d

# AWS (even local dev hits Bedrock)
AWS_REGION=us-west-2
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret

# S3 (LocalStack in dev)
S3_BUCKET=salescraft-files
S3_ENDPOINT=http://localhost:4566  # LocalStack; omit in prod

# Email
SMTP_HOST=localhost
SMTP_PORT=1025  # Mailhog in dev

# External APIs (for intelligence gathering)
LINKEDIN_API_KEY=
BIDNET_API_KEY=
```

### Config Module
```typescript
// apps/api/src/config.ts
import { z } from 'zod';

const configSchema = z.object({
  nodeEnv: z.enum(['development', 'test', 'production']),
  port: z.coerce.number().default(3001),
  database: z.object({
    url: z.string().url(),
  }),
  redis: z.object({
    url: z.string().url(),
  }),
  jwt: z.object({
    secret: z.string().min(32),
    expiresIn: z.string().default('15m'),
    refreshExpiresIn: z.string().default('7d'),
  }),
  aws: z.object({
    region: z.string().default('us-west-2'),
    s3Bucket: z.string(),
    s3Endpoint: z.string().optional(), // Only set for LocalStack
  }),
});

export const config = configSchema.parse({
  nodeEnv: process.env.NODE_ENV,
  port: process.env.PORT,
  database: { url: process.env.DATABASE_URL },
  redis: { url: process.env.REDIS_URL },
  jwt: {
    secret: process.env.JWT_SECRET,
    expiresIn: process.env.JWT_EXPIRES_IN,
    refreshExpiresIn: process.env.REFRESH_TOKEN_EXPIRES_IN,
  },
  aws: {
    region: process.env.AWS_REGION,
    s3Bucket: process.env.S3_BUCKET,
    s3Endpoint: process.env.S3_ENDPOINT,
  },
});
```

## Real-Time Communication

### WebSocket Events (Server → Client)
```typescript
type WsEvent =
  | { type: 'notification'; payload: Notification }
  | { type: 'bid.new'; payload: { bid: BidSummary } }
  | { type: 'relationship.alert'; payload: { contactId: string; reason: string } }
  | { type: 'project.update'; payload: { projectId: string; field: string; value: unknown } }
  | { type: 'intelligence.signal'; payload: IntelligenceSignal };
```

### Connection Management
- Authenticate WebSocket connection with same JWT
- Room-based: users auto-join rooms based on their role and assignments
- Graceful reconnection with exponential backoff on client

## Architecture Decision Records

### ADR-001: Fastify over Express
**Decision:** Use Fastify as the HTTP framework.
**Context:** Need high performance for real-time bid notifications and AI streaming responses.
**Rationale:** 2-3x faster than Express, built-in schema validation, first-class TypeScript support, plugin system for clean extension. Express ecosystem compatibility via fastify-express plugin if needed.

### ADR-002: Prisma over TypeORM/Knex
**Decision:** Use Prisma as the ORM.
**Context:** Need type-safe database access with good migration tooling.
**Rationale:** Schema-first design generates perfect TypeScript types. Migrations are declarative. Query API is intuitive. Trade-off: less control over complex queries — mitigate with `$queryRaw` for the 5% of cases that need it.

### ADR-003: BullMQ over AWS SQS for job processing
**Decision:** Use BullMQ (Redis-backed) locally and in production.
**Context:** Need reliable background job processing for bid scraping, email sync, AI enrichment.
**Rationale:** Simpler local dev (just Redis), built-in scheduling/cron, retry with backoff, job dashboard (Bull Board). Could migrate to SQS later but BullMQ on ElastiCache is production-ready. Avoids LocalStack SQS emulation issues.

### ADR-004: Monorepo with Turborepo
**Decision:** Monorepo with pnpm workspaces and Turborepo.
**Context:** Web, mobile, and API share types, validation schemas, and constants.
**Rationale:** Single source of truth for domain types. Shared Zod schemas ensure frontend and backend validation are identical. Turborepo handles build ordering and caching. pnpm for disk efficiency and strict dependency resolution.

### ADR-005: REST over GraphQL
**Decision:** REST API with consistent conventions.
**Context:** Small team, well-defined domain boundaries, mobile + web clients.
**Rationale:** Simpler to implement, debug, and cache. Our data access patterns are predictable (not deeply nested). TanStack Query handles client-side caching effectively. Mobile offline sync is easier with REST (predictable URL patterns for sync queues).

### ADR-006: PostgreSQL full-text search over Elasticsearch
**Decision:** Use PostgreSQL's built-in full-text search with pg_trgm.
**Context:** Need to search contacts, bid documents, emails, and notes.
**Rationale:** At our data volume (<100K documents), PostgreSQL FTS performs well. Eliminates operational complexity of running Elasticsearch. pg_trgm gives fuzzy/typo-tolerant matching. pgvector gives semantic search. Can add Elasticsearch later if needed (search service is abstracted).

### ADR-007: Custom AI routing over LangChain
**Decision:** Build a thin custom routing layer for Bedrock instead of using LangChain.
**Context:** Need to call multiple Bedrock models with different prompts for different tasks.
**Rationale:** LangChain adds significant complexity and abstraction layers for features we don't need (agents, complex chains). Our needs are: pick a model, send a prompt, get a response, parse it. A custom router with model configs and prompt templates is ~200 lines of code and gives full control over retry logic, cost tracking, and response parsing.

### ADR-008: WatermelonDB for mobile offline
**Decision:** Use WatermelonDB for React Native offline-first data.
**Context:** Field installers work in buildings with no connectivity.
**Rationale:** SQLite-backed (reliable), built-in sync primitives (push/pull with server), reactive queries (UI updates automatically), handles conflict resolution. Alternative (Realm) has more complex sync but less React Native integration.

## Development Workflow

### Getting Started
```bash
# Clone and install
git clone <repo>
pnpm install

# Start all services (PostgreSQL, Redis, LocalStack, Mailhog)
podman-compose up -d

# Run database migrations and seed
pnpm db:migrate
pnpm db:seed

# Start all apps in dev mode
pnpm dev
```

### Available Scripts
| Command | Action |
|---------|--------|
| `pnpm dev` | Start all apps (api + web + mobile) in watch mode |
| `pnpm dev:api` | Start API only |
| `pnpm dev:web` | Start web only |
| `pnpm build` | Build all packages and apps |
| `pnpm test` | Run all tests |
| `pnpm test:api` | Run API tests only |
| `pnpm lint` | Lint all packages |
| `pnpm db:migrate` | Run Prisma migrations |
| `pnpm db:seed` | Seed database with sample data |
| `pnpm db:studio` | Open Prisma Studio |
| `pnpm typecheck` | TypeScript type checking |

### Podman Compose Services
```yaml
services:
  postgres:
    image: pgvector/pgvector:pg16
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: salescraft
      POSTGRES_USER: salescraft
      POSTGRES_PASSWORD: salescraft
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres/init.sql:/docker-entrypoint-initdb.d/init.sql

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  localstack:
    image: localstack/localstack
    ports: ["4566:4566"]
    environment:
      SERVICES: s3
      DEFAULT_REGION: us-west-2

  mailhog:
    image: mailhog/mailhog
    ports: ["1025:1025", "8025:8025"]  # SMTP + Web UI
```

## Performance Targets

| Metric | Target |
|--------|--------|
| API response time (p95) | < 200ms |
| Search response time | < 500ms |
| AI response (streaming first token) | < 2s |
| WebSocket message delivery | < 100ms |
| Mobile offline sync (full) | < 30s for a day's data |
| Page load (web, cached) | < 1s |
| Page load (web, cold) | < 3s |

## Security Checklist

- [ ] All API endpoints require authentication (except /auth/login, /auth/refresh)
- [ ] Role-based access on every route
- [ ] Input validation via Zod on all request bodies and query params
- [ ] SQL injection prevented by Prisma parameterized queries
- [ ] XSS prevented by React's default escaping + CSP headers
- [ ] CORS restricted to known origins
- [ ] Rate limiting on auth endpoints (5 attempts/minute)
- [ ] Rate limiting on API endpoints (100 requests/minute per user)
- [ ] Secrets never in code (environment variables, AWS Secrets Manager in prod)
- [ ] HTTPS only in production (HSTS header)
- [ ] File upload validation (type, size, virus scan in prod)
- [ ] Audit log for sensitive operations (role changes, data exports, bid submissions)

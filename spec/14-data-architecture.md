# Data Architecture

## Vision & Purpose

Salescraft's data architecture must support: relational business data (contacts, bids, projects), full-text and semantic search, vector embeddings for AI, offline mobile sync, document storage, and analytics — all while remaining simple enough for a small team to operate. PostgreSQL with extensions (pgvector, pg_trgm) does the heavy lifting, avoiding the operational complexity of separate search and vector databases.

## Key Concepts

- **Single Database** — One PostgreSQL instance handles relational data, full-text search, and vector storage
- **Schema Migrations** — Prisma manages schema changes declaratively
- **Soft Delete** — Records are never physically deleted; `deletedAt` timestamp marks removal
- **Audit Trail** — All state changes and sensitive operations logged with actor and timestamp
- **Offline Sync** — Mobile devices maintain a local subset and sync bidirectionally

## Database Schema Overview

### PostgreSQL Extensions Required
```sql
-- Enable in docker/postgres/init.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";      -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgvector";        -- Vector similarity search
CREATE EXTENSION IF NOT EXISTS "pg_trgm";         -- Fuzzy text matching
CREATE EXTENSION IF NOT EXISTS "unaccent";        -- Accent-insensitive search
```

### Schema Organization
Prisma schema is organized by domain using comments as section markers:

```prisma
// packages/database/prisma/schema.prisma

generator client {
  provider        = "prisma-client-js"
  previewFeatures = ["postgresqlExtensions"]
}

datasource db {
  provider   = "postgresql"
  url        = env("DATABASE_URL")
  extensions = [pgvector, pg_trgm, uuid_ossp, unaccent]
}

// ═══════════════════════════════════════════
// AUTH & USERS
// ═══════════════════════════════════════════

model User {
  id            String    @id @default(uuid())
  email         String    @unique
  passwordHash  String
  firstName     String
  lastName      String
  role          UserRole
  phone         String?
  avatarUrl     String?
  isActive      Boolean   @default(true)
  lastLoginAt   DateTime?
  createdAt     DateTime  @default(now())
  updatedAt     DateTime  @updatedAt
  deletedAt     DateTime?

  // Relations
  territories       Territory[]      @relation("UserTerritories")
  interests         UserInterest[]
  interactions      Interaction[]
  assignedContacts  Contact[]        @relation("AssignedContacts")
  assignedBids      Bid[]            @relation("AssignedBids")
  managedProjects   Project[]        @relation("ManagedProjects")
  dailyLogs         DailyLog[]
  notifications     Notification[]
  emailAccounts     EmailAccount[]

  @@index([email])
  @@index([role])
}

enum UserRole {
  owner
  sales_manager
  sales_rep
  estimator
  project_manager
  installer
  admin
}

// ═══════════════════════════════════════════
// CONTACTS & ORGANIZATIONS
// ═══════════════════════════════════════════

model Organization {
  id                   String           @id @default(uuid())
  name                 String
  type                 OrganizationType
  subType              String?
  website              String?
  phone                String?
  street1              String?
  street2              String?
  city                 String?
  state                String?
  zip                  String?
  latitude             Float?
  longitude            Float?
  fiscalYearStart      Int              @default(7)
  annualBudget         Decimal?
  purchasingThreshold  Decimal          @default(50000)
  cooperativeContracts String[]         @default([])
  approvedVendor       Boolean          @default(false)
  approvedVendorExpiry DateTime?
  notes                String?
  tags                 String[]         @default([])
  createdAt            DateTime         @default(now())
  updatedAt            DateTime         @updatedAt
  deletedAt            DateTime?
  createdById          String

  // Relations
  facilities    Facility[]
  contacts      Contact[]
  opportunities Opportunity[]
  bids          Bid[]
  projects      Project[]

  @@index([name])
  @@index([type])
  @@index([state, city])
}

model Contact {
  id                String           @id @default(uuid())
  organizationId    String?
  firstName         String
  lastName          String
  title             String?
  role              ContactRole
  email             String?
  phone             String?
  mobile            String?
  linkedinUrl       String?
  decisionAuthority DecisionAuthority
  assignedToId      String?
  relationshipScore Int              @default(0)
  lastContactedAt   DateTime?
  lastContactedById String?
  notes             String?
  tags              String[]         @default([])
  source            String?
  isActive          Boolean          @default(true)
  createdAt         DateTime         @default(now())
  updatedAt         DateTime         @updatedAt
  deletedAt         DateTime?
  createdById       String

  // Relations
  organization  Organization?     @relation(fields: [organizationId], references: [id])
  assignedTo    User?             @relation("AssignedContacts", fields: [assignedToId], references: [id])
  interests     ContactInterest[]
  lifeEvents    ContactLifeEvent[]
  interactions  Interaction[]
  gestures      Gesture[]
  emails        EmailMessage[]

  // Full-text search
  @@index([firstName, lastName])
  @@index([organizationId])
  @@index([assignedToId])
  @@index([relationshipScore(sort: Desc)])
  @@index([lastContactedAt])
  @@index([email])
}

// ... (additional models follow the same pattern from 02-domain-model.md)
```

### Vector Storage (pgvector)

```sql
-- Embeddings table (not managed by Prisma — raw SQL migration)
CREATE TABLE embeddings (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  entity_type VARCHAR(50) NOT NULL,
  entity_id UUID NOT NULL,
  content TEXT NOT NULL,
  content_hash VARCHAR(64) NOT NULL,
  embedding vector(1024) NOT NULL,  -- Titan Embed v2 dimension
  model_id VARCHAR(100) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  UNIQUE(entity_type, entity_id, content_hash)
);

-- IVFFlat index for fast similarity search
CREATE INDEX idx_embeddings_vector ON embeddings 
  USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);  -- Tune based on data volume

-- Filter index for scoped searches
CREATE INDEX idx_embeddings_entity ON embeddings(entity_type, entity_id);
```

### Full-Text Search Configuration

```sql
-- Custom text search configuration for contacts
CREATE TEXT SEARCH CONFIGURATION salescraft (COPY = english);
ALTER TEXT SEARCH CONFIGURATION salescraft
  ALTER MAPPING FOR word WITH unaccent, english_stem;

-- Search index on contacts (triggered, maintained via Prisma middleware)
CREATE INDEX idx_contacts_fts ON contacts USING gin(
  to_tsvector('salescraft', 
    coalesce(first_name, '') || ' ' || 
    coalesce(last_name, '') || ' ' || 
    coalesce(email, '') || ' ' ||
    coalesce(title, '') || ' ' ||
    coalesce(notes, '')
  )
);

-- Trigram index for fuzzy matching
CREATE INDEX idx_contacts_trgm ON contacts USING gin(
  (first_name || ' ' || last_name) gin_trgm_ops
);

-- Bid document search
CREATE INDEX idx_bids_fts ON bids USING gin(
  to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
);
```

## Migration Strategy

### Prisma Migrations
```bash
# Create migration from schema changes
npx prisma migrate dev --name add_contact_interests

# Apply migrations in production
npx prisma migrate deploy

# Reset database (dev only)
npx prisma migrate reset
```

### Custom Migrations (for raw SQL)
For pgvector, full-text indexes, and other features Prisma doesn't natively support, use custom SQL migrations:

```
packages/database/prisma/migrations/
├── 20240101000000_init/
│   └── migration.sql              # Prisma-generated
├── 20240101000001_extensions/
│   └── migration.sql              # Custom: CREATE EXTENSION pgvector...
├── 20240102000000_add_embeddings/
│   └── migration.sql              # Custom: CREATE TABLE embeddings...
└── 20240103000000_fts_indexes/
    └── migration.sql              # Custom: full-text search indexes
```

## Document Storage (S3)

### Bucket Structure
```
salescraft-files/
├── bids/
│   └── {bidId}/
│       ├── documents/            # RFPs, addenda, plans
│       └── responses/            # Our submitted proposals
├── projects/
│   └── {projectId}/
│       ├── photos/               # Daily log + punch list photos
│       ├── documents/            # Contracts, bonds, close-out
│       └── floor-plans/          # PDF floor plans
├── estimates/
│   └── {estimateId}/
│       └── takeoffs/             # Annotated floor plans
├── contacts/
│   └── {contactId}/
│       └── avatars/
├── compliance/
│   └── {documentType}/           # Insurance, licenses, certs
└── exports/
    └── {userId}/                 # Generated reports, exports
```

### File Metadata
```typescript
interface FileMetadata {
  id: string;
  bucket: string;
  key: string;                    // S3 key (path)
  filename: string;               // Original filename
  mimeType: string;
  size: number;                   // Bytes
  entityType: string;             // "bid", "project", "contact"
  entityId: string;
  uploadedBy: string;             // FK → User
  uploadedAt: DateTime;
  isPublic: boolean;              // Whether accessible via CDN
  thumbnailKey?: string;          // For images: resized thumbnail
}
```

## Offline Sync Architecture

### Sync Protocol

```typescript
// Mobile sends sync request
interface SyncPullRequest {
  userId: string;
  lastSyncTimestamp: string;      // ISO timestamp
  tables: string[];               // Which tables to sync
}

// Server responds with changes since last sync
interface SyncPullResponse {
  changes: {
    [table: string]: {
      created: Record<string, unknown>[];
      updated: Record<string, unknown>[];
      deleted: string[];            // IDs of deleted records
    };
  };
  timestamp: string;               // New sync timestamp
}

// Mobile pushes local changes
interface SyncPushRequest {
  userId: string;
  changes: {
    [table: string]: {
      created: Record<string, unknown>[];
      updated: Record<string, unknown>[];
    };
  };
}

interface SyncPushResponse {
  accepted: { table: string; id: string }[];
  rejected: { table: string; id: string; reason: string }[];
  conflicts: {
    table: string;
    id: string;
    serverVersion: Record<string, unknown>;
    resolution: 'server_wins' | 'client_wins';
  }[];
}
```

### What Syncs to Mobile

| Table | Filter | Installer | Sales Rep |
|-------|--------|-----------|-----------|
| projects | assigned crew / assigned rep | Y | Y |
| daily_logs | own logs | Y | N |
| punch_list_items | assigned items | Y | N |
| contacts | N/A / assigned territory | N | Y |
| contact_interests | N/A / own contacts | N | Y |
| organizations | related to assigned | Y | Y |
| schedules | own schedule | Y | Y |

### Conflict Resolution Rules

| Table | Conflict Rule | Rationale |
|-------|--------------|-----------|
| daily_logs | Client wins | Field crew was physically there |
| punch_list_items.status | Server wins | PM has authority over status |
| punch_list_items.notes | Merge (append) | Both sides may add info |
| contacts | Server wins | Web user had more context |
| projects | Server wins | PM manages via web |

## Analytics & Reporting

### Materialized Views

```sql
-- Pipeline summary (refreshed every 5 minutes)
CREATE MATERIALIZED VIEW pipeline_summary AS
SELECT
  o.assigned_to_id,
  o.status,
  COUNT(*) as count,
  SUM(o.estimated_value) as total_value,
  AVG(o.score) as avg_score
FROM opportunities o
WHERE o.status NOT IN ('disqualified', 'lost')
GROUP BY o.assigned_to_id, o.status;

-- Bid analytics (refreshed hourly)
CREATE MATERIALIZED VIEW bid_analytics AS
SELECT
  DATE_TRUNC('month', b.created_at) as month,
  b.type,
  b.result,
  COUNT(*) as bid_count,
  AVG(b.submitted_amount) as avg_amount,
  SUM(CASE WHEN b.result = 'won' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) as win_rate
FROM bids b
WHERE b.result IS NOT NULL
GROUP BY DATE_TRUNC('month', b.created_at), b.type, b.result;

-- Relationship health overview (refreshed daily)
CREATE MATERIALIZED VIEW relationship_health AS
SELECT
  c.assigned_to_id,
  COUNT(*) as total_contacts,
  AVG(c.relationship_score) as avg_score,
  COUNT(*) FILTER (WHERE c.relationship_score < 40) as at_risk_count,
  COUNT(*) FILTER (WHERE c.last_contacted_at < NOW() - INTERVAL '30 days') as stale_count
FROM contacts c
WHERE c.is_active = true AND c.deleted_at IS NULL
GROUP BY c.assigned_to_id;
```

### Data Retention Policy

| Data Type | Retention | Rationale |
|-----------|-----------|-----------|
| Active entities | Indefinite | Business data |
| Soft-deleted records | 7 years | Government contract legal requirements |
| Audit logs | 7 years | Compliance |
| AI usage logs | 90 days | Cost analysis |
| Email bodies | 5 years | Reference |
| Project photos | 10 years | Warranty and legal |
| Intelligence signals | Indefinite | Pattern analysis |
| Embeddings | Regenerated when stale | Can be recomputed |
| Scraped raw HTML | 90 days | Debugging only |

## Implementation Guide

### File Locations
- `packages/database/prisma/schema.prisma` — Prisma schema
- `packages/database/prisma/migrations/` — Migration files
- `packages/database/prisma/seed.ts` — Seed data script
- `packages/database/src/` — Database utilities (search queries, sync logic)
- `docker/postgres/init.sql` — Extension initialization

### Key Dependencies
- `prisma` — ORM and migration tooling
- `@prisma/client` — Generated type-safe client
- `pg` — Raw PostgreSQL client (for custom queries, pgvector)

### Implementation Order
1. Prisma schema with core entities (users, orgs, contacts)
2. Docker Compose PostgreSQL with extensions
3. Initial migration + seed script
4. Full-text search indexes (custom migration)
5. pgvector embedding table (custom migration)
6. S3 file storage service
7. Audit log middleware (Prisma middleware)
8. Materialized views for analytics
9. Sync endpoint for mobile
10. Backup and restore scripts

### Seed Data Strategy

```typescript
// packages/database/prisma/seed.ts
import { PrismaClient } from '@prisma/client';

const prisma = new PrismaClient();

async function main() {
  // Create users (all roles represented)
  const owner = await prisma.user.create({ data: { /* ... */ } });
  const salesMgr = await prisma.user.create({ data: { /* ... */ } });
  // ... etc.

  // Create organizations (mix of school districts and cities)
  const riversideUSD = await prisma.organization.create({ data: { /* ... */ } });
  const cityOfRiverside = await prisma.organization.create({ data: { /* ... */ } });
  // 10-15 organizations with realistic names

  // Create facilities (2-5 per organization)
  // Schools, city buildings, libraries, etc.

  // Create contacts (2-4 per organization)
  // With realistic titles: facility director, purchasing agent, superintendent

  // Create contact interests (2-3 per contact)
  // Realistic hobbies: fishing, golf, sports teams, etc.

  // Create opportunities (5-10 at various stages)
  // Mix of early intelligence signals and active bids

  // Create bids (3-5 at various stages)
  // One submitted, one in preparation, one awarded (won), one lost

  // Create one active project with daily logs and punch list

  // Create products (20-30 from major manufacturers)
  // Shaw, Mohawk, Tarkett, Armstrong, Interface

  // Create jurisdiction rules
  // California state, Riverside County, specific school districts
}
```

## Testing Requirements

### Unit Tests
- Soft delete: record deleted → still in DB with deletedAt set → excluded from normal queries
- Audit log: update contact → audit entry created with before/after
- Full-text search: "riverside" → matches "Riverside USD" and contacts at Riverside orgs
- Fuzzy search: "johnsen" → matches "Johnson" via pg_trgm
- Vector search: embedding query → returns semantically similar results
- Sync: pull changes since timestamp → correct records returned
- Sync: conflict detection → resolved per rules

### Integration Tests
- Full migration from empty database
- Seed script runs without errors
- S3 upload/download cycle (via LocalStack)
- Materialized view refresh
- Backup and restore

## Performance Requirements

- Single record CRUD: < 20ms
- List queries (50 records, filtered): < 100ms
- Full-text search (5000 contacts): < 200ms
- Vector similarity search (10K embeddings): < 200ms
- Materialized view refresh: < 30 seconds
- Mobile sync (100 records): < 2 seconds

## Non-Functional Requirements

- Database encrypted at rest (RDS default encryption in prod)
- Connections via SSL in production
- Connection pooling (Prisma default: 10 connections)
- Automated daily backups with 30-day retention
- Point-in-time recovery capability (RDS PITR)
- Read replicas NOT needed at this scale (< 50 concurrent users)
- Monitoring: slow query log (> 500ms), connection pool utilization, disk usage

## Territory Schema

```prisma
model Territory {
  id          String   @id @default(uuid())
  name        String
  description String?
  counties    String[] @default([])
  cities      String[] @default([])
  zipCodes    String[] @default([])
  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt
  deletedAt   DateTime?

  // Relations
  users       User[]   @relation("UserTerritories")

  @@index([name])
}
```

## Offline Sync Protocol Details

### Endpoint

```
POST /api/v1/mobile/sync/pull
POST /api/v1/mobile/sync/push
```

### Timestamps

- All timestamps are ISO 8601 with millisecond precision: `2026-06-18T14:30:00.000Z`
- **Server clock is authoritative** for conflict resolution (not client clock)
- Client sends its `lastSyncTimestamp` (received from server on previous sync)
- Server compares record `updatedAt` against `lastSyncTimestamp` to find changes

### Batch Size

- Pull: 100 records per table per response. If `hasMore: true`, client pulls again immediately.
- Push: No limit on push size (server processes sequentially within a transaction per table)
- Photos: uploaded independently via multipart POST, max 5 concurrent uploads

### Conflict Resolution Timing

- A conflict occurs when the same record was modified on both client and server since last sync
- Detection: server checks if record's `updatedAt` > client's `lastSyncTimestamp` AND client has a pending change for that record
- Resolution applied per the rules in the Conflict Resolution table above
- For "client wins": server overwrites with client version, sets new `updatedAt`
- For "server wins": server rejects client change, returns current server version in `conflicts[]`
- For "merge (append)": server appends client notes to server notes with timestamp separator

### Partial Failure Recovery

- Each table in a push is processed independently (failure in `daily_logs` doesn't affect `punch_list_items`)
- Response contains `accepted[]` and `conflicts[]` so client knows exactly what succeeded
- Client retains unaccepted changes and retries on next sync
- If connection drops mid-push, client resends all pending changes (server is idempotent on matching `id`)

## Resolved Design Decisions

- **Partitioning:** Not needed at current scale (<50 users, <1M records). Revisit if interactions table exceeds 10M rows.
- **Row-level security:** Not implementing. Application-level authorization (middleware) is sufficient and more flexible.
- **Read replica:** Not needed. Primary is sufficient for <50 concurrent users.
- **Dedicated search index:** Start with PostgreSQL FTS + pg_trgm. Only add MeiliSearch if search p95 exceeds 500ms at scale.

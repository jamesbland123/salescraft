# Missing Entity Schemas

> This file fills entity gaps identified during spec review. These entities are referenced across other spec files but were not previously defined with full schemas.

## Vision & Purpose

Several entities are referenced throughout the Salescraft spec suite without formal schema definitions. This document provides the complete schemas for Company, Notification, and CompanySettings — filling the gaps so that implementation can proceed without ambiguity.

## Storage Decisions

- **Company** — Separate Prisma model. Single row (one company per deployment). Created during bootstrap (`POST /api/v1/auth/setup`).
- **Notification** — Separate Prisma model. One row per notification per user. High-write table; archive/purge notifications older than 90 days.
- **CompanySettings** — Single-row Prisma model with JSON columns for nested configuration objects. Avoids schema migrations for config changes. Related to Company via `companyId` FK but kept separate to allow independent versioning of settings vs. identity.

## Key Concepts

- **Single-Tenant** — There is exactly one Company row. All settings are scoped to that company.
- **Notification Routing** — Notifications are created by the system based on `NotificationConfig` rules defined in spec 13. The Notification entity is the persisted result of that routing.
- **Settings as JSON** — CompanySettings uses typed JSON columns so that adding a new scoring weight or notification default doesn't require a database migration.

---

## Entities

### Company

The single company that operates this Salescraft deployment. Created during bootstrap.

**Referenced in:** `spec/18-authentication.md` (bootstrap creates Company record)

```typescript
interface Company {
  id: string;                        // UUID
  name: string;                      // "Pacific Coast Flooring" — company display name
  emailDomain: string;               // "pacificcoastflooring.com" — used to exclude internal emails from contact timelines
  phone?: string;                    // Main company phone number
  address: Address;                  // Reuses Address interface from 02-domain-model.md
  logoUrl?: string;                  // Company logo for proposals, PDF headers, and the UI
  defaultOverheadPercentage: number; // Default overhead % applied to estimates (e.g., 12.5)
  defaultProfitPercentage: number;   // Default profit margin % for estimates (e.g., 18.0)
  minimumProfitPercentage: number;   // Alert threshold — bids below this trigger approval workflow (e.g., 15.0)
  fiscalYearStartMonth: number;      // 1-12 (e.g., 7 for July) — affects reporting periods
  bondingCapacity?: number;          // Maximum bonding capacity in dollars
  licenseNumbers: LicenseEntry[];    // State contractor license numbers (can have multiple states)
  insuranceSummary?: string;         // Free-text summary of insurance coverage (GL, workers comp, etc.)
  taxId?: string;                    // Federal Tax ID / EIN
  createdAt: DateTime;
  updatedAt: DateTime;
}

interface LicenseEntry {
  state: string;                     // 2-letter state code (e.g., "CA")
  licenseNumber: string;             // The license number itself
  classification?: string;           // License class (e.g., "C-15 Flooring")
  expiresAt?: DateTime;              // Expiration date for renewal tracking
}
```

**Prisma model:**

```prisma
model Company {
  id                       String    @id @default(uuid())
  name                     String
  emailDomain              String
  phone                    String?
  street1                  String?
  street2                  String?
  city                     String?
  state                    String?
  zip                      String?
  logoUrl                  String?
  defaultOverheadPercentage  Decimal @default(12.5)
  defaultProfitPercentage    Decimal @default(18.0)
  minimumProfitPercentage    Decimal @default(15.0)
  fiscalYearStartMonth     Int       @default(7)
  bondingCapacity          Decimal?
  licenseNumbers           Json      @default("[]")   // LicenseEntry[]
  insuranceSummary         String?
  taxId                    String?
  createdAt                DateTime  @default(now())
  updatedAt                DateTime  @updatedAt

  // Relations
  settings                 CompanySettings?
}
```

**Validation rules:**
- `name` — Required, 1-200 chars
- `emailDomain` — Required, valid domain format (no `@` prefix)
- `defaultOverheadPercentage` — 0-100
- `defaultProfitPercentage` — 0-100
- `minimumProfitPercentage` — 0-100, must be <= `defaultProfitPercentage`
- `fiscalYearStartMonth` — 1-12
- `bondingCapacity` — Positive number if provided
- `licenseNumbers` — Array of valid LicenseEntry objects

---

### Notification

A persisted notification delivered to a user. Created by the notification routing system when events fire.

**Referenced in:** `spec/14-data-architecture.md` (User.notifications[]), `spec/13-user-roles-permissions.md` (notification routing config, notification API endpoints), `spec/17-ui-ux.md` (notification bell + dropdown UI)

```typescript
interface Notification {
  id: string;                        // UUID
  userId: string;                    // FK → User — the recipient
  type: NotificationType;            // Event type that triggered this notification
  title: string;                     // Short headline (e.g., "New bid discovered in your territory")
  body: string;                      // Longer description with details
  priority: NotificationPriority;    // Urgency level
  channels: NotificationChannel[];   // Which channels were used to deliver this
  entityType?: string;               // The type of entity this relates to (e.g., "bid", "project", "contact")
  entityId?: string;                 // UUID of the related entity
  isRead: boolean;                   // Whether the user has seen/acknowledged this
  readAt?: DateTime;                 // When it was marked read (null if unread)
  link?: string;                     // In-app URL path to navigate to (e.g., "/bids/uuid-here")
  createdAt: DateTime;
}

enum NotificationType {
  BID_DISCOVERED = 'bid.discovered',
  BID_DEADLINE_APPROACHING = 'bid.deadline_approaching',
  BID_AWARDED_WON = 'bid.awarded_won',
  BID_AWARDED_LOST = 'bid.awarded_lost',
  RELATIONSHIP_DECAY_ALERT = 'relationship.decay_alert',
  DAILY_LOG_MISSING = 'daily_log.missing',
  PUNCH_LIST_NEW_ITEM = 'punch_list.new_item',
  PROJECT_MARGIN_ALERT = 'project.margin_alert',
  COMPLIANCE_EXPIRING = 'compliance.expiring',
  LIFE_EVENT_DETECTED = 'life_event.detected',
  SEASONAL_TRIGGER_FIRED = 'seasonal_trigger.fired',
  APPROVAL_REQUESTED = 'approval.requested',
  APPROVAL_GRANTED = 'approval.granted',
  APPROVAL_DENIED = 'approval.denied',
}

enum NotificationPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  URGENT = 'urgent',
}

enum NotificationChannel {
  IN_APP = 'in_app',
  PUSH = 'push',
  EMAIL = 'email',
}
```

**Prisma model:**

```prisma
model Notification {
  id          String   @id @default(uuid())
  userId      String
  type        String                          // NotificationType value
  title       String
  body        String
  priority    NotificationPriority @default(medium)
  channels    String[]  @default([])          // NotificationChannel values
  entityType  String?
  entityId    String?
  isRead      Boolean   @default(false)
  readAt      DateTime?
  link        String?
  createdAt   DateTime  @default(now())

  // Relations
  user        User      @relation(fields: [userId], references: [id])

  @@index([userId, isRead])
  @@index([userId, createdAt(sort: Desc)])
  @@index([entityType, entityId])
}

enum NotificationPriority {
  low
  medium
  high
  urgent
}
```

**Behavior notes:**
- Notifications are immutable once created (no update, only mark-read)
- `GET /api/v1/notifications` returns paginated, sorted by `createdAt` DESC
- Unread count is computed via `COUNT(*) WHERE userId = ? AND isRead = false`
- Notifications older than 90 days are archived/purged by a scheduled job
- The `link` field enables click-through navigation from the notification center to the relevant entity

---

### CompanySettings

Configuration for the company stored as typed JSON columns. Single row, related to Company.

**Referenced in:** `spec/13-user-roles-permissions.md` (settings:read/write permissions), `spec/17-ui-ux.md` (Settings screens for scoring weights, AI budget, notification preferences)

```typescript
interface CompanySettings {
  id: string;                        // UUID
  companyId: string;                 // FK → Company (unique, one settings row per company)

  // Scoring configuration
  scoringWeights: ScoringWeights;

  // AI budget limits
  aiBudget: AIBudget;

  // Notification defaults
  notificationDefaults: NotificationDefaults;

  // Working hours & locale
  timezone: string;                  // IANA timezone (e.g., "America/Los_Angeles")
  workingHours: WorkingHours;

  updatedAt: DateTime;
  updatedById: string;               // FK → User — who last changed settings
}

interface ScoringWeights {
  opportunity: OpportunityScoringWeights;
  relationship: RelationshipScoringWeights;
}

interface OpportunityScoringWeights {
  budgetAvailability: number;        // Weight 0-100, default 25
  timelineUrgency: number;           // Weight 0-100, default 20
  relationshipStrength: number;      // Weight 0-100, default 25
  competitiveLandscape: number;      // Weight 0-100, default 15
  projectFit: number;                // Weight 0-100, default 15
  // All weights should sum to 100
}

interface RelationshipScoringWeights {
  recency: number;                   // Weight 0-100 — how recently we interacted, default 30
  frequency: number;                 // Weight 0-100 — how often we interact, default 25
  depth: number;                     // Weight 0-100 — quality/depth of interactions, default 25
  breadth: number;                   // Weight 0-100 — number of contacts at the org, default 20
  // All weights should sum to 100
}

interface AIBudget {
  dailyLimitCents: number;           // Max AI spend per day in cents (e.g., 2000 = $20.00)
  monthlyLimitCents: number;         // Max AI spend per month in cents (e.g., 50000 = $500.00)
  alertThresholdPercent: number;     // Alert when usage exceeds this % of limit (e.g., 80)
  enabledModels: string[];           // Which AI models are allowed (e.g., ["claude-sonnet", "titan-embed"])
}

interface NotificationDefaults {
  quietHoursStart?: string;          // HH:MM format (e.g., "22:00") — suppress push during quiet hours
  quietHoursEnd?: string;            // HH:MM format (e.g., "07:00")
  emailDigestEnabled: boolean;       // Whether to send daily email digest instead of per-event emails
  emailDigestTime?: string;          // HH:MM when digest sends (e.g., "08:00")
  pushEnabled: boolean;              // Global push notification toggle
  minimumPriorityForPush: NotificationPriority; // Only push for this priority or higher (default: 'medium')
}

interface WorkingHours {
  monday: DayHours;
  tuesday: DayHours;
  wednesday: DayHours;
  thursday: DayHours;
  friday: DayHours;
  saturday: DayHours;
  sunday: DayHours;
}

interface DayHours {
  isWorkDay: boolean;                // Whether this is a working day
  start?: string;                    // HH:MM (e.g., "07:00")
  end?: string;                      // HH:MM (e.g., "17:00")
}
```

**Prisma model:**

```prisma
model CompanySettings {
  id                    String   @id @default(uuid())
  companyId             String   @unique
  scoringWeights        Json     @default("{}")   // ScoringWeights
  aiBudget              Json     @default("{}")   // AIBudget
  notificationDefaults  Json     @default("{}")   // NotificationDefaults
  timezone              String   @default("America/Los_Angeles")
  workingHours          Json     @default("{}")   // WorkingHours
  updatedAt             DateTime @updatedAt
  updatedById           String

  // Relations
  company               Company  @relation(fields: [companyId], references: [id])
  updatedBy             User     @relation(fields: [updatedById], references: [id])
}
```

**Validation rules:**
- `scoringWeights.opportunity.*` — Each weight 0-100; all five must sum to 100
- `scoringWeights.relationship.*` — Each weight 0-100; all four must sum to 100
- `aiBudget.dailyLimitCents` — Positive integer
- `aiBudget.monthlyLimitCents` — Positive integer, must be >= `dailyLimitCents`
- `aiBudget.alertThresholdPercent` — 1-100
- `timezone` — Must be a valid IANA timezone string
- `workingHours` — Start must be before end for each working day
- `notificationDefaults.minimumPriorityForPush` — Must be a valid NotificationPriority value

**Default values (seeded during bootstrap):**

```typescript
const DEFAULT_COMPANY_SETTINGS: Omit<CompanySettings, 'id' | 'companyId' | 'updatedAt' | 'updatedById'> = {
  scoringWeights: {
    opportunity: {
      budgetAvailability: 25,
      timelineUrgency: 20,
      relationshipStrength: 25,
      competitiveLandscape: 15,
      projectFit: 15,
    },
    relationship: {
      recency: 30,
      frequency: 25,
      depth: 25,
      breadth: 20,
    },
  },
  aiBudget: {
    dailyLimitCents: 2000,          // $20/day
    monthlyLimitCents: 50000,       // $500/month
    alertThresholdPercent: 80,
    enabledModels: ['claude-sonnet', 'titan-embed-v2'],
  },
  notificationDefaults: {
    quietHoursStart: '22:00',
    quietHoursEnd: '07:00',
    emailDigestEnabled: false,
    emailDigestTime: '08:00',
    pushEnabled: true,
    minimumPriorityForPush: 'medium',
  },
  timezone: 'America/Los_Angeles',
  workingHours: {
    monday:    { isWorkDay: true, start: '07:00', end: '17:00' },
    tuesday:   { isWorkDay: true, start: '07:00', end: '17:00' },
    wednesday: { isWorkDay: true, start: '07:00', end: '17:00' },
    thursday:  { isWorkDay: true, start: '07:00', end: '17:00' },
    friday:    { isWorkDay: true, start: '07:00', end: '17:00' },
    saturday:  { isWorkDay: false },
    sunday:    { isWorkDay: false },
  },
};
```

---

## API Endpoints

### Company

```typescript
// Company (Owner only — created during bootstrap)
GET    /api/v1/company                    // Get company profile
PUT    /api/v1/company                    // Update company profile
```

### CompanySettings

```typescript
// Settings (settings:read for GET, settings:write for PUT)
GET    /api/v1/settings                   // Get all company settings
PUT    /api/v1/settings                   // Update settings (partial update via JSON merge)
GET    /api/v1/settings/scoring           // Get scoring weights only
PUT    /api/v1/settings/scoring           // Update scoring weights
GET    /api/v1/settings/ai-budget         // Get AI budget config
PUT    /api/v1/settings/ai-budget         // Update AI budget limits
GET    /api/v1/settings/notifications     // Get notification defaults
PUT    /api/v1/settings/notifications     // Update notification defaults
```

### Notifications

Endpoints defined in `spec/13-user-roles-permissions.md`:

```typescript
GET    /api/v1/notifications              // List for current user (paginated)
PUT    /api/v1/notifications/:id/read     // Mark single notification as read
PUT    /api/v1/notifications/read-all     // Mark all notifications as read
GET    /api/v1/notifications/preferences  // Get user's notification preferences
PUT    /api/v1/notifications/preferences  // Update user's notification preferences
```

---

## Implementation Guide

### File Locations

- `packages/database/prisma/schema.prisma` — Add Company, Notification, CompanySettings models
- `apps/api/src/modules/company/` — Company CRUD
  - `company.routes.ts`
  - `company.service.ts`
  - `company.schema.ts` — Zod validation schemas
- `apps/api/src/modules/settings/` — CompanySettings management
  - `settings.routes.ts`
  - `settings.service.ts`
  - `settings.schema.ts`
- `apps/api/src/modules/notifications/` — Notification creation, querying, read-marking
  - `notifications.routes.ts`
  - `notifications.service.ts`
  - `notifications.schema.ts`
- `packages/shared/src/types/company.ts` — Shared TypeScript interfaces
- `packages/shared/src/types/notification.ts` — Shared TypeScript interfaces
- `packages/shared/src/types/settings.ts` — Shared TypeScript interfaces
- `packages/shared/src/constants/settings-defaults.ts` — DEFAULT_COMPANY_SETTINGS

### Implementation Order

1. Add Prisma models (Company, Notification, CompanySettings) to schema
2. Generate migration
3. Update bootstrap endpoint (`POST /api/v1/auth/setup`) to create Company + CompanySettings with defaults
4. Implement Company module (simple CRUD, owner-only)
5. Implement CompanySettings module (JSON merge updates, validation)
6. Implement Notification module (create via internal service, query via API)
7. Wire notification creation into event handlers from spec 13's routing config

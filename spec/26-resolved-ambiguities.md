# Spec 26: Resolved Ambiguities

This document resolves remaining ambiguities and contradictions identified during spec review. Each resolution is authoritative and supersedes conflicting statements elsewhere in the spec.

---

## 1. Frontend Architecture: Static Export SPA (NOT SSR)

**Resolution:** The web app uses Next.js with `output: 'export'` — it is a static SPA deployed to S3/CloudFront. There is NO server-side rendering and NO Next.js API routes.

- All data fetching happens client-side via TanStack Query calling the Fastify API
- Authentication is handled entirely client-side (store access token in memory, refresh via HTTP-only cookie to the API)
- The `apps/web/next.config.js` must include `output: 'export'` and `trailingSlash: true`
- Dynamic routes use client-side routing only
- This contradicts the mention of "SSR" in spec/01-architecture.md line 33 — that line should be read as "client-side rendering with fast initial loads"

---

## 2. Labor Cost Calculation (Definitive Formula)

**Resolution:** Labor cost uses hourly rate x hours. NOT per-sqft rate.

The definitive calculation:

```
estimatedDays = totalSqFt / productivityRate  (productivityRate = sqft/crew/day)
laborHours = estimatedDays * 8 * crewSize
laborCost = laborHours * hourlyRate
```

Where `hourlyRate` is either:
- Prevailing wage (journeyman rate + fringe) from wage determination lookup, OR
- Standard company rate (configurable in CompanySettings, default $45/hr)

The `laborRatePerSqFt` field on EstimateArea is a DERIVED convenience field calculated as:

```
laborRatePerSqFt = laborCost / sqFt
```

It is stored for display purposes but the authoritative calculation always uses hourly rate x hours.

---

## 3. WebSocket Implementation

**Resolution:**

**Room structure:**
- Each authenticated user joins a personal room: `user:{userId}`
- Users with role `owner` or `sales_manager` also join: `role:management`
- Users join territory rooms for their assignments: `territory:{territoryId}`

**Authentication:** JWT is validated on WebSocket upgrade (sent as query param `?token=xxx` on initial connection). If invalid, connection is rejected with 401.

**Multi-instance broadcast:** Use Redis pub/sub via `@fastify/websocket` + `ioredis` subscriber. When a notification is created, publish to Redis channel `ws:notify`. All API instances subscribe and forward to connected local clients in the target room.

**Client reconnection:** Exponential backoff: 1s, 2s, 4s, 8s, 16s, max 30s. On reconnect, client sends `lastEventId` and server replays missed notifications from the database (notifications created after that timestamp for that user).

**Connection lifecycle:**

```typescript
// Server: on upgrade
fastify.register(websocket);
fastify.get('/ws', { websocket: true }, (socket, req) => {
  const token = req.query.token;
  const user = verifyJwt(token); // throws if invalid
  joinRooms(socket, user);
  socket.on('close', () => leaveRooms(socket, user));
});
```

---

## 4. CORS Configuration

**Resolution:**

```typescript
// apps/api/src/plugins/cors.ts
import cors from '@fastify/cors';

export default fp(async (fastify) => {
  fastify.register(cors, {
    origin: (origin, cb) => {
      const allowed = [
        'http://localhost:3000',          // Web dev
        'http://localhost:19006',         // Expo web dev
        process.env.APP_URL,             // Production web URL
      ].filter(Boolean);

      if (!origin || allowed.includes(origin)) {
        cb(null, true);
      } else {
        cb(new Error('Not allowed'), false);
      }
    },
    credentials: true,                    // Required for HTTP-only cookies
    methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
    allowedHeaders: ['Content-Type', 'Authorization'],
    exposedHeaders: ['X-Request-Id'],
  });
});
```

---

## 5. Email Open/Click Tracking Implementation

**Resolution:**

**Open tracking (pixel):**
- On email send, append a 1x1 transparent GIF: `<img src="https://api.domain.com/api/v1/tracking/open/{trackingId}" width="1" height="1" />`
- The trackingId is a UUID stored alongside the EmailMessage record
- GET `/api/v1/tracking/open/:trackingId` returns a 1x1 transparent GIF and logs the open event (timestamp, IP, User-Agent)
- Disabled for .gov email addresses (per BR-COMM-006)

**Click tracking:**
- Links in outgoing emails are rewritten: original URL -> `https://api.domain.com/api/v1/tracking/click/{trackingId}/{linkIndex}`
- The endpoint logs the click (timestamp, original URL) and redirects (302) to the original URL
- Link rewrites are done at send time by the email service

**Tracking events stored as:**

```typescript
interface EmailTrackingEvent {
  id: string;
  emailMessageId: string;       // FK -> EmailMessage
  type: 'open' | 'click';
  linkUrl?: string;             // For clicks: the original URL
  ipAddress?: string;
  userAgent?: string;
  occurredAt: DateTime;
}
```

This is stored as a separate Prisma model.

---

## 6. PDF Export Implementation

**Resolution:**
- Library: `@react-pdf/renderer` (runs server-side in Node.js)
- The API endpoint `POST /estimates/:id/export-pdf` renders the PDF on the server and returns the file
- Template structure (sections in order):
  1. Cover page: company logo, project title, client org name, date, bid number
  2. Table of Contents (auto-generated from sections)
  3. Company qualifications (from AI-generated proposal narrative)
  4. Scope of work (from AI-generated proposal narrative)
  5. Pricing schedule: table of areas with product, sqft, unit cost, total per area
  6. Alternates: additional tables for each alternate
  7. Project schedule: timeline with milestones (from proposal narrative)
  8. References: past similar projects (if available)
  9. Required certifications and compliance documents (appendix)
- Company branding: logo from Company.logoUrl, company name/address in header, page numbers in footer
- Generated PDF is stored in S3 at `estimates/{estimateId}/proposal-v{version}.pdf`

---

## 7. Mapbox Territory Visualization

**Resolution:**
- Territory boundaries are drawn using zip code polygons
- Boundary data source: GeoJSON files for California zip codes (from US Census Bureau TIGER/Line shapefiles, converted to GeoJSON and stored in S3 as a static asset)
- On territory view load: fetch the static GeoJSON, filter to only the zip codes belonging to each territory, color-code by territory assignment
- Facility pins: rendered as Mapbox markers at their geocoded lat/lng
- Interaction: click a territory polygon to filter the contacts/opportunities list; hover shows territory name and assigned rep
- Mapbox access token stored in env var `MAPBOX_TOKEN`, loaded client-side via Next.js public env var `NEXT_PUBLIC_MAPBOX_TOKEN`

---

## 8. Command Palette (Cmd+K) Search

**Resolution:**
- Powered by a single API endpoint: `GET /api/v1/search?q={query}&types=contact,organization,bid,project`
- Backend implementation: queries each entity table using PostgreSQL `pg_trgm` similarity matching (trigram index) with `ILIKE` fallback
- Results ranked by: exact match first, then trigram similarity score, grouped by entity type
- Response format: `{ results: [{ type: 'contact', id, title, subtitle, url }] }` max 5 per type, 20 total
- Client-side: debounce 200ms, show results grouped by type with keyboard navigation (up/down arrows, enter to select)
- Quick actions section (static, not searched): "Log call", "Create contact", "New bid", "New note"
- Rendered using the `cmdk` library (React component for command menus)

---

## 9. AI Cost Budget Enforcement

**Resolution:**
- Budget configuration stored in CompanySettings table:
  ```
  aiConfig: {
    dailyBudgetUsd: 50,        // Default $50/day
    monthlyBudgetUsd: 1000,    // Default $1000/month
    alertThresholdPercent: 80,  // Alert at 80% of limit
    criticalTasks: ['rfp_parsing', 'signal_classification']  // Bypass daily limit
  }
  ```
- Enforcement: Before each AI call, `cost-tracker.ts` checks current day's spend against dailyBudgetUsd
- If exceeded: non-critical tasks throw `BudgetExceededError` which the caller handles by either queuing (for background jobs) or returning a user-facing message ("AI features limited today, budget reached")
- Critical tasks (tasks in the `criticalTasks` list, plus any task where the related bid has a deadline within 48 hours) always execute regardless of budget
- Monthly budget: checked similarly; when exceeded, ALL tasks are blocked (including critical) and an urgent notification is sent to the owner
- Reset: daily budget resets at midnight company timezone; monthly resets on the 1st
- Dashboard: `GET /api/v1/ai/usage` returns `{ today: { spent, limit, remaining }, month: { spent, limit, remaining } }`

---

## 10. Materialized View Refresh Mechanism

**Resolution:**
- Materialized views are refreshed by BullMQ repeatable jobs (not pg_cron, to keep all scheduling in one place)
- Job definitions:
  ```typescript
  await analyticsQueue.add('refresh-pipeline-summary', {}, { repeat: { pattern: '*/5 * * * *' } });
  await analyticsQueue.add('refresh-bid-analytics', {}, { repeat: { pattern: '0 * * * *' } });
  await analyticsQueue.add('refresh-relationship-health', {}, { repeat: { pattern: '0 6 * * *' } });
  ```
- Each job executes: `await prisma.$executeRawUnsafe('REFRESH MATERIALIZED VIEW CONCURRENTLY ${viewName}')`
- The views must have a UNIQUE INDEX for CONCURRENTLY to work:
  ```sql
  CREATE UNIQUE INDEX idx_pipeline_summary_pk ON pipeline_summary(assigned_to_id, status);
  CREATE UNIQUE INDEX idx_bid_analytics_pk ON bid_analytics(month, type, result);
  CREATE UNIQUE INDEX idx_relationship_health_pk ON relationship_health(assigned_to_id);
  ```

---

## 11. Estimate Versioning

**Resolution:**
- Separate table `EstimateVersion`:
  ```prisma
  model EstimateVersion {
    id          String   @id @default(uuid())
    estimateId  String
    version     Int
    snapshot    Json     // Full JSON snapshot of the estimate + areas at this point
    createdBy   String
    createdAt   DateTime @default(now())

    estimate    Estimate @relation(fields: [estimateId], references: [id])

    @@unique([estimateId, version])
    @@index([estimateId])
  }
  ```
- A new version is created on every explicit "Save" action (not on every field change — that would be autosave draft behavior)
- The submitted version (when estimate transitions to 'submitted') is marked immutable: the Estimate record itself becomes read-only. Further edits create a NEW estimate (clone) with version 1.
- Version snapshot contains: all EstimateArea data, all calculation results, the overhead/profit percentages used, and product pricing at time of save

---

## 12. Territory Match for Notification Routing

**Resolution:** The `territory_match` filter in notification routing works as follows:
1. Get the entity (e.g., the bid or opportunity) that triggered the notification
2. Look up the entity's associated organization
3. Get the organization's zip code from its address
4. Find all territories that include that zip code in their `zipCodes` array
5. Find all users assigned to those territories with the required role
6. Those users are the recipients

If the organization has no address/zip, fall back to ALL users with the required role.

---

## 13. Mobile Push Notifications

**Resolution:**
- Use `expo-notifications` with Expo Push Notification service (no direct FCM/APNs management needed in managed workflow)
- Registration flow:
  1. On mobile app login, call `Notifications.getExpoPushTokenAsync()` to get the Expo push token
  2. Send token to API: `POST /api/v1/users/me/push-token` with body `{ token: "ExponentPushToken[xxx]", platform: "ios" | "android" }`
  3. Server stores token on the User record (field: `expoPushTokens: String[]` — multiple devices supported)
- Sending flow:
  1. When a notification is created with channel `push`, the notification service calls the Expo Push API
  2. `POST https://exp.host/--/api/v2/push/send` with message payload
  3. Use the `expo-server-sdk` npm package for batching and error handling
- Deep linking: use the `expo-linking` URL scheme `salescraft://` with path mapping:
  - `salescraft://contacts/{id}` -> Contact detail
  - `salescraft://projects/{id}` -> Project detail
  - `salescraft://bids/{id}` -> Bid detail
  - `salescraft://daily-log/new` -> New daily log form
- The notification payload includes `data: { url: 'salescraft://...' }` for navigation on tap

---

## 14. FileMetadata as Database Entity

**Resolution:** Yes, FileMetadata is a separate Prisma model (centralized files table). All file uploads go through a common upload service that:
1. Uploads to S3
2. Creates a FileMetadata record in the database
3. Returns the file ID and URL

Entity-specific URL fields (like `BidDocument.fileUrl`, `DailyLog.photos[]`) store the S3 key which can be used to look up the FileMetadata record. The FileMetadata table enables: listing all files for an entity, tracking upload history, managing storage quotas, and cleaning up orphaned files.

---

## 15. Rate Limiting Implementation

**Resolution:**
- Plugin: `@fastify/rate-limit` with Redis store (for persistence across API instances)
- Configuration:
  ```typescript
  fastify.register(rateLimit, {
    global: true,
    max: 100,                        // 100 requests per minute per user
    timeWindow: '1 minute',
    keyGenerator: (req) => req.user?.id || req.ip,
    redis: redisClient,
  });
  ```
- Auth-specific overrides (applied per-route, more restrictive):
  - `POST /auth/login`: 5 per 15 minutes per IP
  - `POST /auth/forgot-password`: 3 per hour per IP
  - `POST /users/invite`: 20 per hour per user
- Response on limit: 429 with `Retry-After` header and body: `{ error: { code: 'RATE_LIMITED', message: 'Too many requests', retryAfter: seconds } }`

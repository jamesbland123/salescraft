# Integrations

## Vision & Purpose

Salescraft doesn't exist in a vacuum. The company already has email, calendars, accounting software, and relies on external data sources for bid intelligence. This spec defines every external system integration with enough detail that an AI agent can implement the connections without guessing at APIs, auth flows, or data mappings.

## Key Concepts

- **OAuth2 Connection** — User grants Salescraft permission to access their account (Gmail, Google Calendar, Outlook)
- **Scraper** — Automated extraction of data from websites that don't have APIs (bid boards, meeting agendas)
- **Webhook** — Real-time notification from external system when something changes
- **Sync Adapter** — Bidirectional or one-way data synchronization with an external system
- **Credential Vault** — Secure storage for API keys, tokens, and passwords used by integrations

## Integration Registry

| System | Type | Direction | Auth | Priority |
|--------|------|-----------|------|----------|
| Gmail | API | Bidirectional | OAuth2 | P0 |
| Google Calendar | API | Bidirectional | OAuth2 | P0 |
| Microsoft Outlook (email) | API | Bidirectional | OAuth2 | P1 |
| Microsoft Calendar | API | Bidirectional | OAuth2 | P1 |
| PlanetBids | Scraper | Inbound | Session cookie | P0 |
| BidNet | API/Scraper | Inbound | API key | P1 |
| PublicPurchase | Scraper | Inbound | Session cookie | P1 |
| BoardDocs | Scraper | Inbound | None (public) | P1 |
| Google Maps/Geocoding | API | Outbound | API key | P0 |
| LinkedIn (enrichment) | API/Scraper | Inbound | OAuth2/Cookie | P2 |
| QuickBooks Online | API | Bidirectional | OAuth2 | P2 |
| Twilio (SMS) | API | Outbound | API key | P2 |
| AWS S3 | API | Bidirectional | IAM role | P0 |
| AWS Bedrock | API | Outbound | IAM role | P0 |

## Integration Details

### Gmail (Google Workspace)

**Purpose:** Two-way email sync, send-on-behalf, contact enrichment.

**Auth:** OAuth2 with Google Workspace
- Scopes: `gmail.readonly`, `gmail.send`, `gmail.modify`, `userinfo.email`
- Consent screen: shows what permissions we're requesting
- Token refresh: auto-refresh before expiry

**Sync Pattern:**
```typescript
// Initial sync: fetch last 30 days
GET https://gmail.googleapis.com/gmail/v1/users/me/messages?q=newer_than:30d

// Incremental sync: use history API
GET https://gmail.googleapis.com/gmail/v1/users/me/history?startHistoryId={lastHistoryId}

// Real-time: Google Cloud Pub/Sub push notifications
// Topic: projects/{projectId}/topics/gmail-push
// Subscription: pushes to our webhook endpoint
POST /api/v1/webhooks/gmail-push
```

**Rate Limits:** 250 quota units/second per user. GET message = 5 units. Batch requests to stay under.

**Error Handling:**
- 401: Token expired → refresh → retry
- 403: Permission revoked → mark account inactive → notify user
- 429: Rate limited → exponential backoff (1s, 2s, 4s, 8s)
- 5xx: Retry up to 3 times with backoff

### Google Calendar

**Purpose:** Calendar sync, meeting detection, scheduling.

**Auth:** Same OAuth2 as Gmail (bundle scopes)
- Additional scopes: `calendar.readonly`, `calendar.events`

**Sync Pattern:**
```typescript
// List calendars
GET https://www.googleapis.com/calendar/v3/users/me/calendarList

// Sync events (incremental)
GET https://www.googleapis.com/calendar/v3/calendars/{calendarId}/events?syncToken={token}

// Watch for changes (webhooks)
POST https://www.googleapis.com/calendar/v3/calendars/{calendarId}/events/watch
// Body: { id: channelId, type: 'web_hook', address: 'https://api.ourapp.com/webhooks/gcal' }
```

**Data Mapping:**
- Google Calendar Event → Interaction (type: meeting/video_call)
- Attendee email → Contact match
- Meeting title + notes → AI extraction for personal details

### PlanetBids (Scraper)

**Purpose:** Primary bid board for California school districts and cities.

**Auth:** Session-based login (no public API)
```typescript
// Login flow
POST https://pbsystem.planetbids.com/portal/login
// Form data: username, password → returns session cookie

// Search bids
GET https://pbsystem.planetbids.com/portal/{agencyId}/bo/bo-search
// Params: category=flooring, status=open, dateRange=...
```

**Scraping Strategy:**
- Headless browser (Playwright) for JavaScript-rendered pages
- Search flooring-related categories
- Parse bid listings: title, number, deadline, description, documents
- Download bid documents (PDF) for AI parsing
- Rate limit: max 1 request per 5 seconds
- Schedule: every 2 hours during business hours (6 AM - 6 PM)

**Error Handling:**
- Login failure → alert admin, pause scraping
- CAPTCHA detected → pause, alert admin for manual intervention
- Page structure changed → log error, pause source, alert admin with last known HTML

### BidNet

**Purpose:** Secondary bid board, broader geographic coverage.

**Auth:** API key (BidNet offers a paid API for some features)
```typescript
// Search API
GET https://api.bidnet.com/v2/bids/search
Headers: { 'X-API-Key': 'our-key' }
Params: { keywords: 'flooring', state: 'CA', status: 'open' }
```

**Data Mapping:**
- BidNet bid → IntelligenceSignal (type: bid_posting)
- Dedup against PlanetBids by bidNumber + organizationName

### BoardDocs (School Board Agendas)

**Purpose:** Monitor school board meeting agendas for facility/flooring discussions.

**Auth:** None required (public content)
```typescript
// BoardDocs is widely used by school districts
// Each district has a public URL like:
GET https://go.boarddocs.com/ca/{districtCode}/Board.nsf/Public

// Agenda items are in structured JSON (varies by district)
// Parse for keywords: "flooring", "facilities", "modernization", "capital improvement"
```

**Scraping Strategy:**
- Simple HTTP + HTML parsing (Cheerio) — not JavaScript-rendered
- Schedule: daily at 6 AM (meeting agendas typically posted days in advance)
- Extract: meeting date, agenda item title, description, attachments
- Match to organizations in our database

### Google Maps / Geocoding

**Purpose:** Geocode facility addresses, calculate distances, territory mapping.

**Auth:** API key (restricted to our domain)
```typescript
// Geocoding
GET https://maps.googleapis.com/maps/api/geocode/json
Params: { address: '1234 Main St, Riverside CA', key: API_KEY }

// Distance Matrix (for territory/proximity calculations)
GET https://maps.googleapis.com/maps/api/distancematrix/json
Params: { origins: '...', destinations: '...', key: API_KEY }
```

**Usage:** Geocode on facility/organization creation. Cache results. ~100 requests/day expected.

### LinkedIn (Contact Enrichment)

**Purpose:** Enrich contact profiles with current title, employer, interests, life events.

**Auth:** OAuth2 for basic profile API. For deeper data (posts, activity), requires LinkedIn Sales Navigator API or careful public profile scraping.

**Note:** LinkedIn's terms of service restrict automated scraping. Options:
1. **LinkedIn API (legitimate, limited):** Basic profile data for connected contacts
2. **LinkedIn Sales Navigator API (paid):** Better enrichment if company has Sales Navigator
3. **Manual enrichment workflow:** System flags contacts for periodic manual review, user updates from LinkedIn

**Recommended approach for MVP:** Manual enrichment with system reminders. Add automated enrichment later if API access is secured.

### QuickBooks Online

**Purpose:** Sync invoices and payments for project financial tracking.

**Auth:** OAuth2 via Intuit Developer
```typescript
// OAuth2 token exchange
POST https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer

// Create invoice
POST https://quickbooks.api.intuit.com/v3/company/{companyId}/invoice
// Map: Project → Customer, LineItems from project financials

// Query payments
GET https://quickbooks.api.intuit.com/v3/company/{companyId}/query
// SQL-like: "SELECT * FROM Payment WHERE TxnDate > '2024-01-01'"
```

**Data Mapping:**
- Organization → QuickBooks Customer
- Project → QuickBooks Job (sub-customer)
- Invoice amount → Project.invoicedToDate
- Payment received → Project.paidToDate

### AWS S3

**Purpose:** File storage for documents, photos, floor plans, proposals.

**Auth:** IAM role (in prod) or access key (in dev with LocalStack)
```typescript
// Upload pattern
import { S3Client, PutObjectCommand } from '@aws-sdk/client-s3';

const upload = async (file: Buffer, key: string, contentType: string) => {
  const command = new PutObjectCommand({
    Bucket: config.aws.s3Bucket,
    Key: key,
    Body: file,
    ContentType: contentType,
  });
  await s3Client.send(command);
  return `https://${config.aws.s3Bucket}.s3.${config.aws.region}.amazonaws.com/${key}`;
};

// Key pattern: {entityType}/{entityId}/{filename}
// Example: "projects/proj-123/photos/daily-log-2024-06-03-001.jpg"
// Example: "bids/bid-456/documents/rfp-original.pdf"
```

**Pre-signed URLs:** Use for direct client uploads (especially mobile photos)
```typescript
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';
// Generate a 15-minute upload URL for the client
```

## Credential Management

```typescript
interface IntegrationCredential {
  id: string;
  integrationName: string;        // "gmail", "planetbids", "quickbooks"
  userId?: string;                // FK → User (for per-user OAuth tokens)
  credentialType: 'oauth2' | 'api_key' | 'session' | 'iam';
  data: {                         // Encrypted at rest
    accessToken?: string;
    refreshToken?: string;
    tokenExpiry?: string;
    apiKey?: string;
    sessionCookie?: string;
    username?: string;
    password?: string;            // For scraper logins (encrypted)
  };
  status: 'active' | 'expired' | 'revoked' | 'error';
  lastUsedAt?: DateTime;
  lastErrorAt?: DateTime;
  lastError?: string;
  createdAt: DateTime;
  updatedAt: DateTime;
}
```

## Implementation Guide

### File Locations
- `apps/api/src/modules/integrations/` — Integration management
  - `integrations.routes.ts` — OAuth flows, connection management
  - `gmail.adapter.ts` — Gmail sync implementation
  - `gcal.adapter.ts` — Google Calendar sync
  - `outlook.adapter.ts` — Microsoft Graph email
  - `mcal.adapter.ts` — Microsoft Calendar
- `apps/api/src/jobs/` — Background scraping/sync jobs
  - `scrapers/planetbids.scraper.ts`
  - `scrapers/bidnet.scraper.ts`
  - `scrapers/boarddocs.scraper.ts`
- `packages/shared/src/constants/integrations.ts` — Integration registry

### Key Dependencies
- `googleapis` — Google API client (Gmail, Calendar, Maps)
- `@microsoft/microsoft-graph-client` — Outlook, Microsoft Calendar
- `playwright` — Headless browser for JavaScript-rendered scrapers
- `cheerio` — HTML parsing for simple scrapers
- `intuit-oauth` — QuickBooks OAuth2
- `@aws-sdk/client-s3` — S3 file operations

### Implementation Order
1. S3 file storage (needed by everything else)
2. Gmail OAuth2 + email sync
3. Google Calendar sync
4. PlanetBids scraper (primary bid board)
5. Google Maps geocoding
6. BoardDocs scraper
7. Outlook/Microsoft Calendar (mirrors Google)
8. BidNet integration
9. QuickBooks sync
10. LinkedIn enrichment (manual workflow first)

## Testing Requirements

### Unit Tests
- OAuth token refresh: expired token → refreshed → retry succeeds
- Gmail message → contact matching (by email address)
- PlanetBids scraper: sample HTML → parsed bid with correct fields
- S3 key generation: project photo → correct path pattern
- Deduplication: same bid from PlanetBids and BidNet → single signal

### Integration Tests
- Full Gmail OAuth flow (use test Google account)
- Gmail sync with real API (test account with seeded emails)
- S3 upload/download (using LocalStack)
- Pre-signed URL generation and use

### Mock Strategy
- Unit tests: mock all external HTTP calls
- Integration tests: use real APIs with test accounts where possible, LocalStack for AWS
- Scraper tests: saved HTML fixtures from actual bid board pages

## Error Handling

| Integration | Failure | Handling |
|-------------|---------|----------|
| Gmail | Token revoked | Mark inactive, notify user, preserve synced data |
| Gmail | Push notification missed | Polling fallback catches it within 5 minutes |
| PlanetBids | Login failure | Retry once, then alert admin |
| PlanetBids | Page structure changed | Stop scraping, alert with error details and captured HTML |
| Google Maps | Quota exceeded | Queue geocoding requests, process next day |
| S3 | Upload failure | Retry 3x with backoff. If persists, alert user. |
| QuickBooks | Sync conflict | Log conflict, show in admin queue for manual resolution |
| Any OAuth | Refresh token rotation | Always store new refresh token immediately |

## Non-Functional Requirements

- OAuth tokens encrypted at rest (AES-256)
- Scraper credentials stored in environment variables (not database)
- All external API calls logged (for debugging and rate tracking)
- Integration health dashboard shows last successful sync per integration
- Graceful degradation: if one integration is down, others continue normally
- No PII sent to external services beyond what's necessary for the integration to function

## Resolved Design Decisions

- **Integration platform:** Build our own. Merge/Nango are $300+/month and add a dependency for just 2 OAuth providers (Gmail + Outlook). Our OAuth implementation is straightforward with `googleapis` and `@microsoft/microsoft-graph-client` packages.
- **Rotating proxies for PlanetBids:** Not initially. Use a single static IP with polite scraping (1 req/5 sec, respect robots.txt). If blocked, first try rotating User-Agent strings. Only add proxy rotation if blocks persist after 3+ consecutive failures.
- **CalDAV support:** Not for MVP. Google and Microsoft cover 95%+ of business email/calendar. CalDAV (Apple Calendar, Fastmail) can be added later if customers request it.
- **LinkedIn:** Manual enrichment only at launch. LinkedIn's API requires Marketing Developer Platform approval ($$$) and restricts data usage. Scraping violates ToS and risks legal action. Reps manually enter LinkedIn URLs and any data they learn from profiles.

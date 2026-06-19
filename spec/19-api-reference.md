# API Reference

## Vision & Purpose

This is the complete API contract for Salescraft. Every endpoint is listed with its HTTP method, path, required auth, request body, and response shape. The API follows RESTful conventions with consistent patterns for pagination, filtering, sorting, and error handling.

## Base URL

```
Development: http://localhost:3001/api/v1
Production:  https://api.{domain}/api/v1
```

## Common Patterns

### Authentication

All endpoints except auth routes require a valid JWT access token:
```
Authorization: Bearer {accessToken}
```

### Pagination (Cursor-Based)

```typescript
// Request query params
?cursor={lastId}&limit={number}   // Default limit: 50, max: 100

// Response wrapper
interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    cursor: string | null;       // null = no more pages
    hasMore: boolean;
    total?: number;              // Only on first page if requested with ?count=true
  };
}
```

### Filtering

```
?filter[field]=value              // Exact match
?filter[field][gte]=value         // Greater than or equal
?filter[field][lte]=value         // Less than or equal
?filter[field][contains]=value    // Substring match
?filter[field][in]=val1,val2      // One of several values
```

### Sorting

```
?sort=field                       // Ascending
?sort=-field                      // Descending
?sort=-score,name                 // Multiple (priority order)
```

### Error Response

```typescript
interface ErrorResponse {
  error: {
    code: string;                // Machine-readable: "VALIDATION_ERROR", "NOT_FOUND", etc.
    message: string;             // Human-readable
    details?: Record<string, string[]>; // Field-level validation errors
  };
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 204 | No Content (successful delete/update with no body) |
| 400 | Validation error (bad request body) |
| 401 | Unauthorized (missing/invalid token) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Resource not found |
| 409 | Conflict (duplicate, state violation) |
| 429 | Rate limited |
| 500 | Internal server error |

---

## Auth Endpoints

See `spec/18-authentication.md` for full details.

```
POST   /auth/setup                    // First-time bootstrap (unauthenticated)
POST   /auth/login                    // Login
POST   /auth/refresh                  // Refresh access token
POST   /auth/logout                   // Logout current session
POST   /auth/logout-all               // Logout all sessions
POST   /auth/forgot-password          // Request password reset
POST   /auth/reset-password           // Complete password reset
PUT    /auth/password                 // Change own password (authenticated)
POST   /auth/accept-invite            // Accept invitation
```

---

## Users

```
GET    /users                         // List users (owner/admin)
POST   /users/invite                  // Send invitation (owner)
GET    /users/invitations             // List pending invitations (owner)
POST   /users/invite/:id/resend       // Resend invitation (owner)
DELETE /users/invite/:id              // Revoke invitation (owner)
GET    /users/:id                     // Get user detail
PUT    /users/:id                     // Update user (owner)
PUT    /users/:id/deactivate          // Deactivate user (owner)
GET    /users/me                      // Get current user
PUT    /users/me                      // Update own profile
GET    /users/me/permissions          // Get computed permissions
```

### Request/Response Types

```typescript
// POST /users/invite
interface InviteUserRequest {
  email: string;
  firstName: string;
  lastName: string;
  role: UserRole;
  territories?: string[];
}

// GET /users/:id response
interface UserResponse {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  role: UserRole;
  phone: string | null;
  avatarUrl: string | null;
  territories: Territory[];
  isActive: boolean;
  lastLoginAt: string | null;
  createdAt: string;
}

// PUT /users/me
interface UpdateProfileRequest {
  firstName?: string;
  lastName?: string;
  phone?: string;
  avatarUrl?: string;
}
```

---

## Organizations

```
GET    /organizations                 // List organizations (paginated, filterable)
POST   /organizations                 // Create organization
GET    /organizations/:id             // Get organization detail
PUT    /organizations/:id             // Update organization
DELETE /organizations/:id             // Soft delete
GET    /organizations/:id/contacts    // List contacts at this org
GET    /organizations/:id/facilities  // List facilities
GET    /organizations/:id/bids        // List bids from this org
GET    /organizations/:id/timeline    // All interactions with anyone at this org
```

### Request/Response Types

```typescript
// POST /organizations
interface CreateOrganizationRequest {
  name: string;
  type: OrganizationType;
  subType?: string;
  website?: string;
  phone?: string;
  address?: {
    street1: string;
    street2?: string;
    city: string;
    state: string;
    zip: string;
  };
  fiscalYearStart?: number;         // 1-12
  annualBudget?: number;
  purchasingThreshold?: number;
  cooperativeContracts?: string[];
  approvedVendor?: boolean;
  approvedVendorExpiry?: string;
  notes?: string;
  tags?: string[];
}

// GET /organizations response item
interface OrganizationResponse {
  id: string;
  name: string;
  type: OrganizationType;
  subType: string | null;
  website: string | null;
  phone: string | null;
  address: Address | null;
  territory: Territory | null;
  fiscalYearStart: number;
  annualBudget: number | null;
  purchasingThreshold: number;
  cooperativeContracts: string[];
  approvedVendor: boolean;
  approvedVendorExpiry: string | null;
  notes: string | null;
  tags: string[];
  contactCount: number;
  facilityCount: number;
  activeBidCount: number;
  createdAt: string;
  updatedAt: string;
}
```

### Filters

```
?filter[type]=school_district
?filter[state]=CA
?filter[approvedVendor]=true
?filter[name][contains]=riverside
?sort=-updatedAt
```

---

## Facilities

```
GET    /facilities                    // List all facilities (filterable)
POST   /facilities                    // Create facility
GET    /facilities/:id                // Get facility detail
PUT    /facilities/:id                // Update facility
DELETE /facilities/:id                // Soft delete
```

### Request/Response Types

```typescript
// POST /facilities
interface CreateFacilityRequest {
  organizationId: string;
  name: string;
  type: FacilityType;
  address?: Address;
  yearBuilt?: number;
  totalSqFt?: number;
  flooringSqFt?: number;
  lastFlooringProject?: string;
  lastFlooringType?: string;
  conditionRating?: number;          // 1 (poor) - 5 (excellent)
  notes?: string;
}
```

---

## Contacts

```
GET    /contacts                      // List contacts (paginated, filterable)
POST   /contacts                      // Create contact
GET    /contacts/:id                  // Get contact detail (includes interests, recent interactions)
PUT    /contacts/:id                  // Update contact
DELETE /contacts/:id                  // Soft delete
GET    /contacts/:id/timeline         // Interaction timeline
GET    /contacts/:id/interests        // List interests
POST   /contacts/:id/interests        // Add interest
PUT    /contacts/:id/interests/:iid   // Update interest
DELETE /contacts/:id/interests/:iid   // Remove interest
GET    /contacts/:id/life-events      // List life events
POST   /contacts/:id/life-events      // Add life event
GET    /contacts/:id/gestures         // List gestures
POST   /contacts/:id/gestures         // Log gesture
GET    /contacts/:id/briefing         // Generate AI briefing card
POST   /contacts/:id/notes            // Quick note (creates interaction)
GET    /contacts/search               // Full-text + fuzzy search
GET    /contacts/by-interest          // Reverse interest match
```

### Request/Response Types

```typescript
// POST /contacts
interface CreateContactRequest {
  organizationId?: string;
  firstName: string;
  lastName: string;
  title?: string;
  role: ContactRole;
  email?: string;
  phone?: string;
  mobile?: string;
  linkedinUrl?: string;
  decisionAuthority: DecisionAuthority;
  assignedTo?: string;
  notes?: string;
  tags?: string[];
  source?: string;
}

// GET /contacts/:id response
interface ContactDetailResponse {
  id: string;
  organization: { id: string; name: string; type: string } | null;
  firstName: string;
  lastName: string;
  title: string | null;
  role: ContactRole;
  email: string | null;
  phone: string | null;
  mobile: string | null;
  linkedinUrl: string | null;
  decisionAuthority: DecisionAuthority;
  assignedTo: { id: string; firstName: string; lastName: string } | null;
  relationshipScore: number;
  lastContactedAt: string | null;
  daysSinceContact: number;
  notes: string | null;
  tags: string[];
  source: string | null;
  isActive: boolean;
  interests: ContactInterest[];
  recentInteractions: Interaction[];  // Last 5
  activeBids: { id: string; title: string; status: string }[];
  createdAt: string;
  updatedAt: string;
}

// GET /contacts/search
// ?q=mike+johnson&limit=20
interface SearchResponse {
  data: ContactDetailResponse[];
  pagination: { cursor: string | null; hasMore: boolean };
}

// GET /contacts/by-interest
// ?category=outdoors&name=fishing&match=related
interface InterestMatchResponse {
  data: {
    contact: ContactDetailResponse;
    matchedInterest: ContactInterest;
    matchType: 'exact' | 'related' | 'broad';
  }[];
}

// POST /contacts/:id/gestures
interface CreateGestureRequest {
  type: GestureType;
  description: string;
  value?: number;
  date: string;
  occasion?: string;
}
// Response includes ethicsCleared (validated server-side against jurisdiction limits)
```

### Filters

```
?filter[organizationId]=uuid
?filter[assignedTo]=uuid
?filter[role]=facility_director
?filter[relationshipScore][gte]=50
?filter[isActive]=true
?filter[tags][in]=vip,key_account
?sort=-relationshipScore
```

---

## Territories

```
GET    /territories                   // List all territories
POST   /territories                   // Create territory
GET    /territories/:id               // Get territory detail
PUT    /territories/:id               // Update territory
DELETE /territories/:id               // Delete territory
```

### Request/Response Types

```typescript
// POST /territories
interface CreateTerritoryRequest {
  name: string;
  description?: string;
  counties?: string[];
  cities?: string[];
  zipCodes?: string[];              // At least one of counties/cities/zipCodes required
  assignedTo?: string[];            // User IDs
}

interface TerritoryResponse {
  id: string;
  name: string;
  description: string | null;
  counties: string[];
  cities: string[];
  zipCodes: string[];
  assignedUsers: { id: string; firstName: string; lastName: string }[];
  organizationCount: number;        // Orgs within this territory
  contactCount: number;             // Contacts at those orgs
  createdAt: string;
  updatedAt: string;
}
```

---

## Opportunities

```
GET    /opportunities                 // List opportunities (filterable)
POST   /opportunities                 // Create opportunity (manual)
GET    /opportunities/:id             // Get opportunity detail
PUT    /opportunities/:id             // Update opportunity
PUT    /opportunities/:id/status      // Transition state
DELETE /opportunities/:id             // Soft delete
GET    /opportunities/:id/signals     // Related intelligence signals
```

### Request/Response Types

```typescript
// POST /opportunities
interface CreateOpportunityRequest {
  organizationId: string;
  facilityId?: string;
  title: string;
  source: OpportunitySource;
  sourceDetail?: string;
  estimatedValue?: number;
  estimatedSqFt?: number;
  estimatedTimeline?: string;
  flooringTypes?: string[];
  assignedTo?: string;
  notes?: string;
}

// PUT /opportunities/:id/status
interface TransitionStatusRequest {
  status: OpportunityStatus;
  notes?: string;                   // Required for disqualification
}

interface OpportunityResponse {
  id: string;
  organization: { id: string; name: string };
  facility: { id: string; name: string } | null;
  title: string;
  status: OpportunityStatus;
  source: OpportunitySource;
  sourceDetail: string | null;
  estimatedValue: number | null;
  estimatedSqFt: number | null;
  estimatedTimeline: string | null;
  flooringTypes: string[];
  score: number;
  scoreFactors: ScoreFactor[];
  assignedTo: { id: string; firstName: string; lastName: string } | null;
  notes: string | null;
  discoveredAt: string;
  bidExpectedBy: string | null;
  relatedBid: { id: string; title: string } | null;
  createdAt: string;
  updatedAt: string;
}
```

---

## Intelligence Signals

```
GET    /intelligence/signals          // List signals (filterable)
GET    /intelligence/signals/:id      // Get signal detail
PUT    /intelligence/signals/:id/dismiss  // Dismiss signal
POST   /intelligence/signals/:id/convert  // Convert to opportunity
GET    /intelligence/sources          // Source health status
POST   /intelligence/scrape           // Trigger manual scrape (owner)
GET    /intelligence/dashboard        // Dashboard summary data
```

### Request/Response Types

```typescript
// GET /intelligence/signals filters
// ?filter[processed]=false&filter[type]=bid_posting&sort=-createdAt

// POST /intelligence/signals/:id/convert
interface ConvertToOpportunityRequest {
  title?: string;                   // Override signal title
  assignedTo?: string;
  estimatedValue?: number;
}

interface DashboardResponse {
  highScoreOpportunities: OpportunityResponse[];  // Score > 70
  recentSignals: IntelligenceSignalResponse[];     // Last 24h, unprocessed
  sourceHealth: {
    source: string;
    lastScrapeAt: string;
    status: 'healthy' | 'warning' | 'error';
    signalsToday: number;
  }[];
  stats: {
    totalSignalsThisWeek: number;
    convertedThisWeek: number;
    conversionRate: number;
  };
}
```

---

## Bids

```
GET    /bids                          // List bids (filterable)
POST   /bids                          // Create bid (manual entry)
GET    /bids/:id                      // Get bid detail
PUT    /bids/:id                      // Update bid
PUT    /bids/:id/status               // Transition state
PUT    /bids/:id/decide               // Set bid/no-bid decision
GET    /bids/:id/documents            // List bid documents
POST   /bids/:id/documents            // Upload document
DELETE /bids/:id/documents/:docId     // Remove document
GET    /bids/:id/checklist            // Submission checklist
PUT    /bids/:id/checklist/:itemId    // Check/uncheck item
GET    /bids/:id/calendar             // Calendar entries for this bid
POST   /bids/:id/calendar             // Add calendar entry
GET    /bids/:id/score                // Get bid/no-bid decision score
POST   /bids/:id/parse-rfp            // AI parse uploaded RFP document
```

### Request/Response Types

```typescript
// POST /bids
interface CreateBidRequest {
  organizationId: string;
  facilityIds?: string[];
  title: string;
  bidNumber?: string;
  type: BidType;
  source: string;
  sourceUrl?: string;
  description?: string;
  estimatedValue?: number;
  publishedAt: string;
  preBidMeetingAt?: string;
  preBidMeetingLocation?: string;
  preBidMeetingMandatory?: boolean;
  questionsDeadline?: string;
  submissionDeadline: string;
  bondRequired?: boolean;
  bondPercentage?: number;
  prevailingWage?: boolean;
  wageCounty?: string;
  assignedTo?: string;
  estimatorId?: string;
}

// PUT /bids/:id/decide
interface BidDecisionRequest {
  decision: 'bid' | 'no_bid';
  reason?: string;                  // Required for no_bid
}

// PUT /bids/:id/status
interface BidStatusTransitionRequest {
  status: BidStatus;
  submittedAmount?: number;         // Required for submitted
  result?: BidResult;               // Required for awarded_won/lost
  winningAmount?: number;
  winningCompany?: string;
}

// POST /bids/:id/parse-rfp
// Body: { documentId: string }  (reference to uploaded bid document)
interface RfpParseResponse {
  title: string;
  bidNumber: string | null;
  issuingAgency: string;
  scope: string;
  flooringTypes: string[];
  deadlines: {
    submission: string | null;
    questions: string | null;
    preBidMeeting: string | null;
    award: string | null;
  };
  requirements: {
    bond: boolean;
    bondPercentage: number | null;
    prevailingWage: boolean;
    insurance: string | null;
    licenses: string[];
  };
  evaluationCriteria: { criterion: string; weight: number }[];
}
```

---

## Estimates

```
GET    /estimates                      // List estimates
POST   /estimates                      // Create estimate
GET    /estimates/:id                  // Get estimate detail (full breakdown)
PUT    /estimates/:id                  // Update estimate metadata
PUT    /estimates/:id/status           // Transition state
POST   /estimates/:id/areas            // Add area
PUT    /estimates/:id/areas/:areaId    // Update area
DELETE /estimates/:id/areas/:areaId    // Remove area
PUT    /estimates/:id/totals           // Recalculate and update totals
GET    /estimates/:id/history          // Version history
POST   /estimates/:id/export-pdf       // Generate PDF
POST   /estimates/:id/duplicate        // Clone estimate (for alternates)
```

### Request/Response Types

```typescript
// POST /estimates
interface CreateEstimateRequest {
  bidId?: string;
  opportunityId?: string;
  title: string;
}

// POST /estimates/:id/areas
interface CreateEstimateAreaRequest {
  name: string;
  sqFt: number;
  productId: string;
  wasteFactor?: number;             // Defaults to product's typical
  laborRatePerSqFt?: number;       // Defaults to prevailing wage calculation
  additionalMaterials?: LineItem[];
  notes?: string;
}

// PUT /estimates/:id/totals
interface UpdateTotalsRequest {
  overheadPercentage: number;
  profitPercentage: number;
  bondPercentage?: number;
  equipmentTotal?: number;
  subcontractorTotal?: number;
}

interface EstimateDetailResponse {
  id: string;
  bid: { id: string; title: string } | null;
  opportunity: { id: string; title: string } | null;
  title: string;
  status: EstimateStatus;
  estimator: { id: string; firstName: string; lastName: string };
  reviewedBy: { id: string; firstName: string; lastName: string } | null;
  version: number;
  areas: EstimateAreaResponse[];
  materialTotal: number;
  laborTotal: number;
  equipmentTotal: number;
  subcontractorTotal: number;
  subtotal: number;
  overhead: number;
  overheadPercentage: number;
  profit: number;
  profitPercentage: number;
  bondCost: number | null;
  total: number;
  notes: string | null;
  createdAt: string;
  updatedAt: string;
}
```

---

## Products

```
GET    /products                       // List products (filterable)
POST   /products                       // Create product
GET    /products/:id                   // Get product detail
PUT    /products/:id                   // Update product
DELETE /products/:id                   // Soft delete (deactivate)
GET    /products/search                // Search by name/manufacturer/type
```

### Filters

```
?filter[type]=lvt
?filter[manufacturer]=Shaw
?filter[isActive]=true
?filter[name][contains]=sustain
?sort=manufacturer,name
```

---

## Projects

```
GET    /projects                       // List projects (filterable)
POST   /projects                       // Create project (usually auto-created from bid win)
GET    /projects/:id                   // Get project detail
PUT    /projects/:id                   // Update project
PUT    /projects/:id/status            // Transition state
GET    /projects/:id/daily-logs        // List daily logs
POST   /projects/:id/daily-logs        // Create daily log
GET    /projects/:id/daily-logs/:logId // Get log detail
PUT    /projects/:id/daily-logs/:logId // Update log
GET    /projects/:id/punch-list        // List punch list items
POST   /projects/:id/punch-list        // Create punch item
GET    /projects/:id/punch-list/:itemId
PUT    /projects/:id/punch-list/:itemId
PUT    /projects/:id/punch-list/:itemId/status  // Transition item status
GET    /projects/:id/change-orders     // List change orders
POST   /projects/:id/change-orders     // Create change order
PUT    /projects/:id/change-orders/:coId       // Update/approve change order
GET    /projects/:id/documents         // List project documents
POST   /projects/:id/documents         // Upload document
GET    /projects/:id/financials        // Financial summary (margin, costs)
GET    /projects/:id/schedule          // Crew schedule
PUT    /projects/:id/schedule          // Update schedule
```

### Request/Response Types

```typescript
// POST /projects/:id/daily-logs
interface CreateDailyLogRequest {
  date: string;
  hoursWorked: number;
  sqFtInstalled: number;
  productInstalled?: string;
  areasWorked: string[];
  crewSize: number;
  weather?: string;
  issues?: string;
  materialsUsed?: string;
  photos?: string[];                // S3 URLs (uploaded separately)
  notes?: string;
}

// POST /projects/:id/punch-list
interface CreatePunchListItemRequest {
  location: string;
  description: string;
  priority: 'critical' | 'major' | 'minor' | 'cosmetic';
  assignedTo?: string;
  dueDate?: string;
  photos?: string[];
}

// PUT /projects/:id/punch-list/:itemId/status
interface PunchListStatusRequest {
  status: PunchListStatus;
  notes?: string;
  completionPhotos?: string[];      // Required for 'completed' status
}

// GET /projects/:id/financials
interface ProjectFinancialsResponse {
  contractAmount: number;
  changeOrderTotal: number;
  currentContractAmount: number;
  materialCostToDate: number;
  laborCostToDate: number;
  totalCostToDate: number;
  estimatedTotalCost: number;
  projectedFinalCost: number;
  currentMargin: number;            // Percentage
  estimatedMargin: number;          // Original estimate margin
  percentComplete: number;          // sqft installed / total sqft
  retentionHeld: number;
}
```

---

## Interactions & Communications

```
GET    /interactions                   // List interactions (filterable)
POST   /interactions                   // Log interaction
GET    /interactions/:id
PUT    /interactions/:id
GET    /interactions/feed              // Activity feed for current user

// Email
GET    /email/accounts                 // Connected email accounts
POST   /email/accounts/connect         // Start OAuth flow
DELETE /email/accounts/:id             // Disconnect
POST   /email/accounts/:id/sync        // Force sync
GET    /email/messages                 // List synced emails
GET    /email/messages/:id             // Get email detail
POST   /email/send                     // Send email via connected account

// Calls
POST   /calls                          // Log a call
GET    /calls                          // List calls
PUT    /calls/:id                      // Update call

// Templates
GET    /templates                      // List templates
POST   /templates                      // Create template
PUT    /templates/:id                  // Update template
DELETE /templates/:id                  // Delete template
POST   /templates/:id/render           // Render with contact data
```

---

## Compliance

```
GET    /compliance/documents           // List compliance documents
POST   /compliance/documents           // Add document
PUT    /compliance/documents/:id       // Update document
GET    /compliance/documents/expiring  // Documents expiring within 30 days
GET    /compliance/vendor-registrations    // Vendor registration status
POST   /compliance/vendor-registrations
PUT    /compliance/vendor-registrations/:id
GET    /compliance/jurisdiction-rules  // Jurisdiction rules
POST   /compliance/jurisdiction-rules
GET    /compliance/wage-determinations // Wage rate lookup
POST   /compliance/wage-determinations
GET    /compliance/ethics-check        // Check gift against limits
// ?contactId=uuid&value=50&jurisdictionId=uuid
```

---

## Mobile Sync

```
POST   /mobile/sync/pull               // Get changes since last sync
POST   /mobile/sync/push               // Push local changes
POST   /mobile/sync/photos             // Upload photos (multipart)
GET    /mobile/sync/status             // Sync health check
```

### Request/Response Types

```typescript
// POST /mobile/sync/pull
interface SyncPullRequest {
  lastSyncTimestamp: string;         // ISO 8601 with ms
  tables: string[];                  // Which tables to pull
}

interface SyncPullResponse {
  changes: {
    [table: string]: {
      created: Record<string, unknown>[];
      updated: Record<string, unknown>[];
      deleted: string[];
    };
  };
  timestamp: string;                 // Server timestamp (use for next pull)
  hasMore: boolean;                  // If true, pull again immediately
}

// POST /mobile/sync/push
interface SyncPushRequest {
  changes: {
    [table: string]: {
      created: Record<string, unknown>[];
      updated: Record<string, unknown>[];
    };
  };
  clientTimestamp: string;           // Client's current time (for drift detection)
}

interface SyncPushResponse {
  accepted: { table: string; id: string }[];
  conflicts: {
    table: string;
    id: string;
    serverVersion: Record<string, unknown>;
    resolution: 'server_wins' | 'client_wins';
  }[];
}

// POST /mobile/sync/photos
// Content-Type: multipart/form-data
// Fields: photos[] (files), metadata[] (JSON per photo: { id, projectId, type, caption, lat, lng })
```

---

## Notifications

```
GET    /notifications                  // List for current user
PUT    /notifications/:id/read         // Mark read
PUT    /notifications/read-all         // Mark all read
GET    /notifications/preferences      // Get preferences
PUT    /notifications/preferences      // Update preferences
GET    /notifications/unread-count     // Quick count for badge
```

---

## Search (Global)

```
GET    /search?q={query}&types=contacts,organizations,bids,projects
```

### Response

```typescript
interface GlobalSearchResponse {
  results: {
    type: 'contact' | 'organization' | 'bid' | 'project';
    id: string;
    title: string;           // Display name
    subtitle: string;        // Secondary info
    score: number;           // Relevance score
  }[];
}
```

---

## Reports & Analytics

```
GET    /reports/pipeline               // Pipeline summary by stage
GET    /reports/bid-analytics          // Win rate, avg amount, by type
GET    /reports/relationship-health    // Score distribution, at-risk contacts
GET    /reports/project-performance    // Margin analysis, schedule adherence
GET    /reports/intelligence-roi       // Signal → opportunity → win conversion
GET    /reports/team-activity          // Interactions per rep, response times
```

---

## File Upload

```
POST   /files/upload-url               // Get pre-signed S3 upload URL
// Body: { filename, mimeType, entityType, entityId }
// Response: { uploadUrl, fileKey, expiresAt }

POST   /files/confirm                  // Confirm upload complete
// Body: { fileKey }
// Response: { file: FileMetadata }

GET    /files/:key/download-url        // Get pre-signed download URL
// Response: { downloadUrl, expiresAt }
```

---

## WebSocket Events

Connection: `wss://api.{domain}/ws?token={accessToken}`

### Server → Client Events

```typescript
type WSEvent =
  | { type: 'notification'; data: Notification }
  | { type: 'bid.updated'; data: { bidId: string; status: BidStatus } }
  | { type: 'signal.new'; data: { signalId: string; title: string; score: number } }
  | { type: 'project.updated'; data: { projectId: string; status: ProjectStatus } }
  | { type: 'relationship.decay'; data: { contactId: string; score: number; daysSince: number } }
  | { type: 'sync.complete'; data: { tables: string[] } }
```

---

## Rate Limits

| Endpoint Category | Limit | Window |
|-------------------|-------|--------|
| Auth endpoints | 5-20 req | Per minute |
| Read endpoints | 100 req | Per minute |
| Write endpoints | 30 req | Per minute |
| Search | 20 req | Per minute |
| AI endpoints (briefing, parse) | 10 req | Per minute |
| File upload | 50 req | Per hour |
| Sync endpoints | 60 req | Per minute |

Rate limit headers on every response:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1687900000
```

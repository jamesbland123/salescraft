# Government Procurement

## Vision & Purpose

Government purchasing is not like commercial purchasing. There are thresholds, rules, compliance requirements, ethics limits, and formal processes that must be followed — and understanding them IS the competitive advantage. A sales rep who knows that a district's purchasing threshold just dropped from $92,600 to $85,000 (because the state updated it), or that cooperative purchasing through Sourcewell bypasses the bid process entirely, has a structural advantage over competitors who just respond to posted bids.

Salescraft encodes this knowledge so every team member — from the newest sales rep to the seasoned estimator — operates with expert-level procurement awareness.

## Key Concepts

- **Purchasing Threshold** — Dollar amount above which formal competitive bidding is legally required. Varies by jurisdiction and is periodically updated. Below threshold = can be sole-sourced or informally quoted.
- **Cooperative Purchasing** — Programs (Sourcewell, OMNIA, NASPO ValuePoint, US Communities) where a lead agency competitively bids a contract, and other agencies can "piggyback" without their own bid process.
- **Prevailing Wage** — Government-mandated minimum wage rates for construction workers, determined by county and trade classification. Adds 20-40% to labor costs vs. private work.
- **Bid Bond** — Financial guarantee (typically 10% of bid amount) that the bidder will accept the contract if awarded. Required on most formal bids.
- **Performance Bond** — Guarantee (typically 100% of contract value) that the contractor will complete the work. Issued by a surety company.
- **Payment Bond** — Guarantee that subcontractors and material suppliers will be paid.
- **Approved Vendor List (AVL)** — Pre-qualified list of vendors who can receive work without full bid process (up to a threshold or for certain categories).
- **DBE/MBE/WBE** — Disadvantaged/Minority/Women-owned Business Enterprise certifications that may provide bid preferences or participation requirements.

## User Stories

### Sales Rep
- As a sales rep, I need to know each organization's purchasing threshold so I can identify projects that can be sole-sourced vs. those requiring formal bids
- As a sales rep, I need to track which cooperative purchasing contracts we hold so I can offer clients a way to buy without bidding
- As a sales rep, I need alerts when an organization's purchasing policies change or thresholds are updated
- As a sales rep, I need to know government gift/ethics limits by jurisdiction so I can build relationships without crossing lines

### Estimator
- As an estimator, I need to know whether prevailing wage applies and pull the correct wage determinations for the project's county and trade classifications
- As an estimator, I need to factor in bonding costs when prevailing wage or bond requirements apply
- As an estimator, I need to know if the bid requires specific insurance minimums so I can include those costs

### Admin
- As an admin, I need to track our compliance documents (licenses, insurance certs, bonding capacity) and know when they expire
- As an admin, I need to maintain our vendor registrations across multiple jurisdictions and portals
- As an admin, I need to assemble compliance packages for bid responses (W-9, insurance cert, license, DBE cert, etc.)

## Technical Design

### Data Model

#### ComplianceDocument
Our company's compliance documents that get included in bid responses.

```typescript
interface ComplianceDocument {
  id: string;
  type: ComplianceDocType;
  name: string;                   // "General Liability Insurance Certificate"
  issuer?: string;                // "State Farm", "CA CSLB"
  documentNumber?: string;        // License number, policy number
  issueDate: Date;
  expirationDate?: Date;          // Null for non-expiring docs
  fileUrl: string;                // S3 URL
  isActive: boolean;
  alertDaysBefore: number;        // Days before expiry to alert (default 30)
  notes?: string;
  createdAt: DateTime;
  updatedAt: DateTime;
}

enum ComplianceDocType {
  CONTRACTOR_LICENSE = 'contractor_license',
  GENERAL_LIABILITY = 'general_liability',
  WORKERS_COMP = 'workers_comp',
  AUTO_INSURANCE = 'auto_insurance',
  UMBRELLA_POLICY = 'umbrella_policy',
  BOND_CAPACITY_LETTER = 'bond_capacity_letter',
  W9 = 'w9',
  DBE_CERTIFICATION = 'dbe_certification',
  MBE_CERTIFICATION = 'mbe_certification',
  WBE_CERTIFICATION = 'wbe_certification',
  DVBE_CERTIFICATION = 'dvbe_certification',
  DIR_REGISTRATION = 'dir_registration',  // CA Dept of Industrial Relations
  SAM_REGISTRATION = 'sam_registration',  // Federal SAM.gov
  BUSINESS_LICENSE = 'business_license',
  SAFETY_CERTIFICATION = 'safety_certification',
}
```

#### VendorRegistration
Tracking our registration status with various agencies and portals.

```typescript
interface VendorRegistration {
  id: string;
  organizationId?: string;        // FK → Organization (if agency-specific)
  portalName: string;             // "PlanetBids", "SAM.gov", "City of Riverside"
  portalUrl?: string;
  username?: string;              // Our login (not password — stored in secrets)
  registrationNumber?: string;
  status: 'active' | 'pending' | 'expired' | 'rejected';
  registeredAt?: Date;
  expirationDate?: Date;
  categories: string[];           // What categories we're registered for
  notes?: string;
  createdAt: DateTime;
  updatedAt: DateTime;
}
```

#### CooperativeContract
Cooperative purchasing vehicles available to us.

```typescript
interface CooperativeContract {
  id: string;
  programName: string;            // "Sourcewell", "OMNIA", "NASPO ValuePoint"
  contractNumber: string;         // Our contract number within the program
  manufacturer: string;           // "Shaw", "Mohawk" — which manufacturer's contract
  productCategories: string[];    // ["LVT", "Carpet Tile", "Sheet Vinyl"]
  startDate: Date;
  endDate: Date;
  discountStructure?: string;     // How pricing works under this contract
  maxOrderValue?: number;         // Per-order limit if any
  participatingStates: string[];  // Which states can use this contract
  website?: string;               // Program website
  isActive: boolean;
  notes?: string;
  createdAt: DateTime;
  updatedAt: DateTime;
}
```

#### JurisdictionRule
Procurement rules and ethics limits by jurisdiction.

```typescript
interface JurisdictionRule {
  id: string;
  jurisdictionName: string;       // "State of California", "City of Riverside", "Riverside USD"
  jurisdictionType: 'state' | 'county' | 'city' | 'school_district' | 'special_district';
  organizationId?: string;        // FK → Organization (for org-specific rules)
  rules: ProcurementRules;
  ethicsLimits: EthicsLimits;
  lastVerified: Date;             // When we last confirmed these are current
  sourceUrl?: string;             // Link to policy document
  notes?: string;
  createdAt: DateTime;
  updatedAt: DateTime;
}

interface ProcurementRules {
  informalQuoteThreshold: number;  // Below this: can get informal quotes (e.g., $15,000)
  formalBidThreshold: number;      // Above this: sealed bid required (e.g., $50,000)
  middleTierProcess?: string;      // Between informal and formal (e.g., "3 written quotes")
  prevailingWageThreshold: number; // Dollar amount triggering prevailing wage (often $1,000 for public works)
  bidBondRequired: boolean;        // Whether bid bonds are always required on formal bids
  bidBondPercentage: number;       // Typical percentage (usually 10%)
  performanceBondThreshold: number; // Above this: performance bond required
  performanceBondPercentage: number; // Usually 100%
  paymentBondThreshold: number;
  retentionPercentage: number;     // Withholding until completion (typically 5-10%)
  allowsCooperativePurchasing: boolean;
  cooperativePrograms: string[];   // Which programs they accept
  requiresLocalPreference: boolean; // Local vendor preference in evaluation
  localPreferencePercentage?: number; // e.g., 5% bid preference for local firms
  publicWorkRequirements?: string[]; // Additional requirements ("DIR registration", "project labor agreement")
}

interface EthicsLimits {
  giftLimitPerOccasion: number;   // Max gift value per event (e.g., $50)
  giftLimitAnnual: number;        // Max total gifts per year to one official (e.g., $500)
  mealLimit: number;              // Max meal value (e.g., $75)
  prohibitedGifts: string[];      // Things never allowed ("cash", "gift cards over $25")
  reportingThreshold?: number;    // Above this value, gift must be publicly reported
  cooldownPeriod?: string;        // "No gifts 30 days before/after bid submission"
  sourceReference: string;        // "CA Gov Code 89503", "City Policy 4.12"
}
```

#### WageDetermination
Prevailing wage rates by county and trade.

```typescript
interface WageDetermination {
  id: string;
  state: string;                  // "CA"
  county: string;                 // "Riverside"
  tradeClassification: string;    // "Floor Layer", "Tile Setter", "Laborer"
  journeymanRate: number;         // Hourly base wage
  fringeRate: number;             // Hourly fringe benefits
  totalRate: number;              // journeymanRate + fringeRate
  overtimeRate: number;           // Usually 1.5x base
  effectiveDate: Date;
  expirationDate?: Date;
  source: 'davis_bacon' | 'state'; // Federal or state determination
  determinationNumber?: string;   // Reference number
  lastUpdated: DateTime;
}
```

### API Endpoints

```typescript
// Compliance Documents
GET    /api/v1/compliance/documents          // List all, filter by type/status
POST   /api/v1/compliance/documents          // Upload new document
GET    /api/v1/compliance/documents/:id
PUT    /api/v1/compliance/documents/:id      // Update metadata
DELETE /api/v1/compliance/documents/:id      // Soft delete
GET    /api/v1/compliance/documents/expiring // Documents expiring within N days

// Vendor Registrations
GET    /api/v1/compliance/vendor-registrations
POST   /api/v1/compliance/vendor-registrations
PUT    /api/v1/compliance/vendor-registrations/:id
GET    /api/v1/compliance/vendor-registrations/expiring

// Cooperative Contracts
GET    /api/v1/compliance/cooperative-contracts         // List active contracts
GET    /api/v1/compliance/cooperative-contracts/match   // Given an org, which contracts can they use?
POST   /api/v1/compliance/cooperative-contracts
PUT    /api/v1/compliance/cooperative-contracts/:id

// Jurisdiction Rules
GET    /api/v1/compliance/jurisdiction-rules                 // List all
GET    /api/v1/compliance/jurisdiction-rules/:id
GET    /api/v1/compliance/jurisdiction-rules/for-org/:orgId  // Get rules applicable to a specific org
POST   /api/v1/compliance/jurisdiction-rules
PUT    /api/v1/compliance/jurisdiction-rules/:id

// Wage Determinations
GET    /api/v1/compliance/wage-determinations        // List, filter by state/county
GET    /api/v1/compliance/wage-determinations/lookup // ?state=CA&county=Riverside&trade=Floor+Layer
POST   /api/v1/compliance/wage-determinations/refresh // Trigger re-scrape of wage databases

// Ethics Check
POST   /api/v1/ethics/check                  // Check if a planned gesture is within limits
// Body: { contactId, gestureType, value, date }
// Response: { allowed: boolean, reason: string, limit: number, usedToDate: number }
```

### Business Rules

- **BR-PROC-001:** When a bid's estimated value exceeds the jurisdiction's `formalBidThreshold`, the bid type MUST be IFB or RFP (not informal quote).
- **BR-PROC-002:** When `prevailingWage = true` on a bid/project, all labor rates in the estimate must use the applicable wage determination rates (not standard rates).
- **BR-PROC-003:** When a bid requires a bid bond (`bondRequired = true`), the estimate must include bonding cost in its calculations.
- **BR-PROC-004:** Compliance documents within 30 days of expiration trigger an alert to the ADMIN role.
- **BR-PROC-005:** When assembling a bid response, the system checks all required compliance documents are current (not expired as of submission date).
- **BR-PROC-006:** Cooperative purchasing contracts have geographic limitations (`participatingStates`). Only suggest cooperative purchasing for organizations in participating states.
- **BR-PROC-007:** Ethics limits are cumulative per calendar year. When logging a gesture, check total gestures to that contact's jurisdiction in the current year.
- **BR-PROC-008:** During the bid cooldown period (if defined by jurisdiction), the system blocks gesture creation and shows a warning.
- **BR-PROC-009:** If an organization has `approvedVendor = true` and the project is below `formalBidThreshold`, flag the opportunity as "eligible for direct award" (no competitive bid needed).
- **BR-PROC-010:** Retention percentage (typically 5-10%) must be tracked and factored into cash flow projections on projects.

### Validation Rules

| Field | Rules |
|-------|-------|
| ComplianceDocument.expirationDate | Must be future date at upload time |
| ComplianceDocument.type | Must be valid ComplianceDocType enum |
| JurisdictionRule.ethicsLimits.giftLimitPerOccasion | Non-negative number |
| JurisdictionRule.procurementRules.formalBidThreshold | Positive number |
| WageDetermination.journeymanRate | Positive number |
| WageDetermination.totalRate | Must equal journeymanRate + fringeRate |
| CooperativeContract.endDate | Must be after startDate |
| Gesture.value (cross-validation) | Must not exceed jurisdiction's per-occasion limit |

## Implementation Guide

### File Locations
- `apps/api/src/modules/procurement/` — All procurement-related routes, services, repositories
- `packages/shared/src/constants/government.ts` — Procurement enums, default thresholds
- `packages/shared/src/schemas/procurement.schema.ts` — Zod schemas for procurement entities
- `packages/database/prisma/seed-data/jurisdictions.ts` — Seed data for common jurisdiction rules

### Key Dependencies
- `pdf-parse` — Extract text from government policy PDFs for AI analysis
- `@aws-sdk/client-bedrock-runtime` — AI parsing of procurement documents

### Implementation Order
1. JurisdictionRule CRUD (foundation for all procurement logic)
2. ComplianceDocument management (upload, expiry tracking)
3. WageDetermination storage and lookup
4. CooperativeContract management
5. Ethics check endpoint
6. VendorRegistration tracking
7. Expiry alert system (BullMQ cron job)

### Common Pitfalls
- Prevailing wage rates change periodically — must have a refresh mechanism, not just static data
- Ethics limits vary wildly between jurisdictions (California: $590/year to a single official; some cities: $50 per occasion)
- Cooperative purchasing eligibility depends on both the contract's state coverage AND the specific organization's policies
- Bid bonds and performance bonds have different thresholds — don't conflate them
- "Public works" definition varies by state — in California, even painting a school bathroom may trigger prevailing wage above $1,000

## Testing Requirements

### Unit Tests
- Ethics check: value within limit → allowed
- Ethics check: value exceeds per-occasion limit → blocked
- Ethics check: cumulative annual exceeds limit → blocked
- Ethics check: within cooldown period → blocked
- Wage lookup: correct rate returned for state/county/trade combo
- Compliance expiry: document expiring in 25 days (threshold 30) → should alert
- Cooperative match: org in participating state → contract available
- Cooperative match: org NOT in participating state → contract not available

### Integration Tests
- Full ethics check flow with gesture creation and jurisdiction lookup
- Bid response assembly checking all compliance documents are current
- Prevailing wage integration with estimate calculation

### Seed Data
```typescript
const sampleJurisdictions = [
  {
    jurisdictionName: 'State of California',
    jurisdictionType: 'state',
    rules: {
      informalQuoteThreshold: 15000,
      formalBidThreshold: 92600, // As of 2024, adjusted annually
      prevailingWageThreshold: 1000,
      bidBondRequired: true,
      bidBondPercentage: 10,
      performanceBondThreshold: 25000,
      performanceBondPercentage: 100,
      paymentBondThreshold: 25000,
      retentionPercentage: 5,
      allowsCooperativePurchasing: true,
      cooperativePrograms: ['Sourcewell', 'OMNIA', 'NASPO ValuePoint', 'US Communities'],
      publicWorkRequirements: ['DIR registration', 'Skilled and trained workforce'],
    },
    ethicsLimits: {
      giftLimitPerOccasion: 590,
      giftLimitAnnual: 590,
      mealLimit: 75,
      prohibitedGifts: ['cash', 'gift cards over $50'],
      reportingThreshold: 50,
      cooldownPeriod: null,
      sourceReference: 'CA Gov Code 89503, FPPC Reg 18940.2',
    },
  },
  {
    jurisdictionName: 'Riverside Unified School District',
    jurisdictionType: 'school_district',
    rules: {
      informalQuoteThreshold: 15000,
      formalBidThreshold: 92600,
      prevailingWageThreshold: 1000,
      bidBondRequired: true,
      bidBondPercentage: 10,
      performanceBondThreshold: 50000,
      performanceBondPercentage: 100,
      paymentBondThreshold: 50000,
      retentionPercentage: 10,
      allowsCooperativePurchasing: true,
      cooperativePrograms: ['Sourcewell', 'OMNIA'],
      publicWorkRequirements: ['DIR registration'],
    },
    ethicsLimits: {
      giftLimitPerOccasion: 50,
      giftLimitAnnual: 250,
      mealLimit: 50,
      prohibitedGifts: ['cash', 'alcohol', 'gift cards'],
      reportingThreshold: 25,
      cooldownPeriod: '30 days before/after board vote on contracts',
      sourceReference: 'Board Policy 3315',
    },
  },
];
```

## Error Handling

| Failure | Handling |
|---------|----------|
| Wage determination not found for county/trade | Return error with list of available trades for that county; log for data gap tracking |
| Ethics check — jurisdiction rule not configured | Block gesture with message "Ethics limits not configured for this jurisdiction — contact admin" |
| Compliance document expired at bid assembly | Block bid submission, highlight expired documents, provide renewal instructions |
| Cooperative contract expired | Remove from eligible list, notify assigned rep, suggest renewal |

## UI/UX Requirements

### Compliance Dashboard
- Grid showing all compliance documents with status (current, expiring soon, expired)
- Color coding: green (>60 days), yellow (30-60 days), red (<30 days or expired)
- Quick upload for renewals
- Link to vendor registration portals

### Bid Compliance Checker
- When preparing a bid, show a checklist of required documents
- Green checkmark if current, red X if missing/expired
- One-click to attach current documents to bid response

### Ethics Sidebar (in Contact view)
- When viewing a contact, show:
  - Applicable gift limits for their jurisdiction
  - Gifts/gestures sent YTD with running total
  - Remaining capacity before hitting limit
  - Any active cooldown periods
  - "Safe to gift" indicator

### Procurement Quick-Reference (in Bid view)
- When viewing a bid, show applicable procurement rules:
  - Threshold tier (informal/formal/sealed)
  - Bond requirements and estimated cost
  - Prevailing wage status and applicable rates
  - Cooperative purchasing eligibility

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| State wage databases (DIR, DOL) | Wage determination updates | Inbound | Monthly refresh |
| SAM.gov | Federal registration status | Bidirectional | Registration status check weekly |
| Cooperative program portals | Contract status, pricing updates | Inbound | Monthly |
| State license boards (CSLB) | License status verification | Inbound | Monthly |

## Performance Requirements

- Ethics check response: < 100ms (jurisdiction rules cached in Redis)
- Wage lookup response: < 50ms (indexed by state/county/trade)
- Compliance status check: < 200ms (for bid submission validation)

## Non-Functional Requirements

- Jurisdiction rules must be auditable (who changed what, when)
- Ethics tracking data must be retained for 7 years (legal protection)
- Wage determinations must show effective dates (for projects spanning rate changes)
- All compliance documents must be versioned (keep history of expired certs)

## Resolved Design Decisions

- **Prevailing wage updates:** Manual entry for MVP. Admin enters rates annually from DIR (California Department of Industrial Relations) website. P2: automated scraping of dir.ca.gov/OPRL/DPreWageDetermination.htm.
- **Bonding capacity:** Track as fields on a Company Settings entity (surety company name, single project limit, aggregate limit). Not a full entity — just reference data for bid/no-bid scoring.
- **Sub-tier compliance:** Not tracked for MVP. The company is responsible for their subs' compliance but this is managed via contract terms, not software. Revisit if customers request it.

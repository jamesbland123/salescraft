# Project Intelligence

## Vision & Purpose

The single greatest competitive advantage in government flooring sales is knowing about projects BEFORE they appear on bid boards. By the time a bid is posted, every competitor in the territory sees it. But bond measures pass 6-12 months before projects are bid. Capital improvement plans are published years in advance. School board meetings discuss facility needs months before funding is allocated. Building permits indicate new construction that will need flooring.

Salescraft's intelligence engine continuously monitors these public data sources, correlates signals, and surfaces scored opportunities to sales reps — often months before competitors even know a project exists. This early warning system enables relationship-building and spec influence that dramatically increases win probability.

## Key Concepts

- **Intelligence Signal** — A raw data point from a monitored source (bid posting, bond measure, meeting agenda item, etc.)
- **Opportunity Score** — AI-computed score (0-100) indicating the likelihood and value of a flooring project
- **Trigger Event** — A specific event that indicates a flooring project may be coming (bond passes, building hits 20 years old, new CIP published)
- **Signal Correlation** — Multiple signals about the same facility/organization that together indicate high probability
- **Lead Time** — How far in advance of a formal bid the opportunity was identified (the goal: maximize this)

## User Stories

### Sales Rep
- As a sales rep, I want to see a ranked list of opportunities in my territory, scored by likelihood and value, so I can prioritize my outreach
- As a sales rep, I want to be alerted immediately when a new bid is posted that matches my territory and product capabilities
- As a sales rep, I want to know when a bond measure passes in my territory so I can start building relationships before projects are defined
- As a sales rep, I want to see which architects are getting school/government projects so I can build relationships with them early

### Sales Manager
- As a sales manager, I want to see pipeline by lead time (how early are we finding opportunities) to measure our intelligence advantage
- As a sales manager, I want to see hit rate on pre-bid-identified opportunities vs. cold bid responses to justify the intelligence system

## Technical Design

### Data Model

Uses `IntelligenceSignal` and `Opportunity` entities from `02-domain-model.md`.

#### MonitoredSource
Sources the system actively monitors for intelligence signals.

```typescript
interface MonitoredSource {
  id: string;
  name: string;                   // "PlanetBids - Riverside County"
  type: SourceType;
  url: string;                    // Base URL for scraping/API
  scrapeConfig: ScrapeConfig;     // How to extract data
  schedule: string;               // Cron expression: "0 */2 * * *" (every 2 hours)
  lastScrapedAt?: DateTime;
  lastSuccessAt?: DateTime;
  errorCount: number;             // Consecutive failures
  isActive: boolean;
  territories: string[];          // Which territories this source covers
  createdAt: DateTime;
  updatedAt: DateTime;
}

enum SourceType {
  BID_BOARD = 'bid_board',           // PlanetBids, BidNet, PublicPurchase
  SCHOOL_BOARD = 'school_board',     // BoardDocs, district websites
  CITY_COUNCIL = 'city_council',     // Granicus, city websites
  BOND_TRACKER = 'bond_tracker',     // Ballotpedia, county elections
  CIP_DATABASE = 'cip_database',     // County/city CIP publications
  BUILDING_PERMITS = 'building_permits', // County building department
  NEWS = 'news',                     // Local news outlets
  ARCHITECT_REGISTRY = 'architect_registry', // DSA (Division of State Architect) for schools
}

interface ScrapeConfig {
  method: 'api' | 'html_scrape' | 'rss' | 'pdf_download';
  selectors?: Record<string, string>;  // CSS/XPath selectors for HTML scraping
  apiKey?: string;                     // Stored reference (actual key in secrets)
  searchTerms: string[];               // ["flooring", "floor covering", "carpet", "vinyl", "LVT"]
  excludeTerms: string[];              // ["wood floor refinishing", "floor cleaning"]
  pagination?: { type: 'offset' | 'cursor'; pageSize: number };
  rateLimit: { requestsPerMinute: number };
}
```

#### OpportunityScoreModel
Configuration for the AI scoring system.

```typescript
interface ScoreModelConfig {
  factors: ScoreFactorConfig[];
  minimumScoreToSurface: number;  // Below this, don't show to reps (default: 30)
  autoAssignThreshold: number;    // Above this, auto-assign to territory rep (default: 60)
}

interface ScoreFactorConfig {
  name: string;
  weight: number;                 // 0-1, all weights should sum to 1
  description: string;
  calculation: string;            // How this factor is computed
}

// Default scoring model
const DEFAULT_SCORE_MODEL: ScoreModelConfig = {
  minimumScoreToSurface: 30,
  autoAssignThreshold: 60,
  factors: [
    {
      name: 'building_age',
      weight: 0.20,
      description: 'How likely the flooring needs replacement based on building age',
      calculation: '0 if <10 years, linear 0-100 from 10-25 years, 100 if >25 years',
    },
    {
      name: 'funding_confirmed',
      weight: 0.25,
      description: 'Whether funding is secured (bond passed, budget approved)',
      calculation: '100 if bond passed or CIP-funded, 50 if budget hearing scheduled, 0 if no funding signal',
    },
    {
      name: 'relationship_strength',
      weight: 0.20,
      description: 'Our existing relationship with the decision-makers at this org',
      calculation: 'Average relationship score of our contacts at the org (0-100)',
    },
    {
      name: 'project_size',
      weight: 0.15,
      description: 'Estimated project value relative to our sweet spot',
      calculation: '100 if $100K-$500K, 80 if $50K-$100K or $500K-$1M, 40 if <$50K or >$1M',
    },
    {
      name: 'competitive_position',
      weight: 0.10,
      description: 'Our competitive advantage (approved vendor, cooperative contract, spec influence)',
      calculation: '100 if sole-source eligible, 80 if approved vendor, 60 if cooperative contract available, 40 if relationship but no structural advantage, 0 if unknown',
    },
    {
      name: 'timeline_proximity',
      weight: 0.10,
      description: 'How soon the project is expected to bid/start',
      calculation: '100 if <3 months, 70 if 3-6 months, 40 if 6-12 months, 20 if >12 months',
    },
  ],
};
```

### API Endpoints

```typescript
// Intelligence Signals
GET    /api/v1/intelligence/signals           // List signals, filter by type/territory/processed
GET    /api/v1/intelligence/signals/:id
POST   /api/v1/intelligence/signals/:id/dismiss  // Mark as not relevant
POST   /api/v1/intelligence/signals/:id/convert  // Convert to opportunity

// Opportunities
GET    /api/v1/opportunities                  // List, filter by status/territory/score
GET    /api/v1/opportunities/:id
POST   /api/v1/opportunities                  // Manual creation
PUT    /api/v1/opportunities/:id
POST   /api/v1/opportunities/:id/transition   // State transition: { to: 'qualified', notes: '...' }
GET    /api/v1/opportunities/dashboard        // Aggregated stats for sales manager

// Monitored Sources
GET    /api/v1/intelligence/sources           // List sources and their health
POST   /api/v1/intelligence/sources           // Add new source
PUT    /api/v1/intelligence/sources/:id       // Update config
POST   /api/v1/intelligence/sources/:id/trigger  // Manually trigger a scrape

// Scoring
GET    /api/v1/intelligence/score-model       // Get current scoring config
PUT    /api/v1/intelligence/score-model       // Update scoring weights
POST   /api/v1/intelligence/rescore           // Rescore all active opportunities

// Analytics
GET    /api/v1/intelligence/analytics/lead-time   // Average lead time before bid
GET    /api/v1/intelligence/analytics/conversion  // Signal → Opportunity → Bid → Win rates
GET    /api/v1/intelligence/analytics/sources     // Which sources produce best opportunities
```

### Business Rules

- **BR-INTEL-001:** New intelligence signals matching a territory with an assigned rep are auto-assigned to that rep and trigger a push notification.
- **BR-INTEL-002:** Signals from bid boards with submission deadlines <7 days away are flagged as "URGENT" and notify both the rep and sales manager.
- **BR-INTEL-003:** When multiple signals correlate to the same facility (e.g., bond measure + building age > 20 years + architect project), auto-create an Opportunity if one doesn't exist and boost its score.
- **BR-INTEL-004:** Opportunities with score > `autoAssignThreshold` (60) that have no `assignedTo` are auto-assigned to the territory rep.
- **BR-INTEL-005:** Opportunities decay by 1 point per week if no action is taken (no status change, no interactions logged with related contacts). Decay stops if status is `engaging` or later.
- **BR-INTEL-006:** When a bid is discovered (via scraping) that matches an existing Opportunity's organization + facility, auto-link them and transition the opportunity to `bid_posted`.
- **BR-INTEL-007:** Bid board scraping must include keyword matching for flooring-related terms AND exclude false positives (floor cleaning, floor stripping, floor waxing are maintenance, not installation).
- **BR-INTEL-008:** Bond measure tracking should include the full lifecycle: filed → on ballot → passed/failed → funds allocated → projects identified.
- **BR-INTEL-009:** Building age scoring uses the LATER of: year built, or last known flooring replacement date. A building built in 1980 with flooring replaced in 2015 should score based on 2015, not 1980.
- **BR-INTEL-010:** Architect project tracking focuses on architects known to spec flooring on school/government projects. Track which products they typically specify.

### Signal Correlation Logic

The system correlates signals into opportunities using these rules:

```typescript
interface CorrelationRule {
  signalTypes: SignalType[];       // What signals must be present
  minimumSignals: number;          // How many must match
  matchCriteria: 'same_org' | 'same_facility' | 'same_geography';
  scoreBoost: number;              // Additional score points when correlated
  autoCreateOpportunity: boolean;  // Whether to auto-create an opportunity
}

const CORRELATION_RULES: CorrelationRule[] = [
  {
    // Bond passed + building is old = high confidence project coming
    signalTypes: ['bond_measure', 'building_age'],
    minimumSignals: 2,
    matchCriteria: 'same_org',
    scoreBoost: 25,
    autoCreateOpportunity: true,
  },
  {
    // CIP entry + architect awarded project = project is in design
    signalTypes: ['cip_entry', 'architect_project'],
    minimumSignals: 2,
    matchCriteria: 'same_facility',
    scoreBoost: 30,
    autoCreateOpportunity: true,
  },
  {
    // Meeting agenda discusses facilities + budget approval = funding moving
    signalTypes: ['meeting_agenda_item', 'budget_approval'],
    minimumSignals: 2,
    matchCriteria: 'same_org',
    scoreBoost: 20,
    autoCreateOpportunity: false, // Still needs human review
  },
  {
    // Building permit + our territory = new construction needing flooring
    signalTypes: ['building_permit'],
    minimumSignals: 1,
    matchCriteria: 'same_geography',
    scoreBoost: 15,
    autoCreateOpportunity: false,
  },
];
```

## Implementation Guide

### File Locations
- `apps/api/src/modules/intelligence/` — Intelligence routes, services, repositories
- `apps/api/src/jobs/bid-scraper.job.ts` — BullMQ job for bid board scraping
- `apps/api/src/jobs/intelligence-scan.job.ts` — Daily intelligence scan
- `apps/api/src/jobs/opportunity-decay.job.ts` — Weekly score decay
- `packages/ai/src/prompts/signal-classifier.ts` — AI prompt for classifying signals
- `packages/ai/src/prompts/opportunity-scorer.ts` — AI prompt for scoring explanations
- `packages/shared/src/constants/intelligence.ts` — Score model config, signal types

### Key Dependencies
- `cheerio` — HTML parsing for bid board scraping
- `puppeteer` (or `playwright`) — JavaScript-rendered pages that require headless browser
- `node-cron` — Schedule expressions for BullMQ recurring jobs
- `@aws-sdk/client-bedrock-runtime` — AI signal classification and scoring

### Implementation Order
1. MonitoredSource CRUD and configuration
2. Bid board scraper (start with one portal: PlanetBids)
3. Signal storage and deduplication
4. AI signal classification (is this flooring-related? which facility?)
5. Opportunity creation from signals (manual first)
6. Scoring model implementation
7. Auto-correlation rules
8. Additional scrapers (meeting agendas, bond trackers)
9. Analytics and reporting

### Scraping Strategy

```typescript
// General scraper pattern
interface ScraperResult {
  signals: RawSignal[];
  nextCursor?: string;            // For pagination
  errors: ScraperError[];
}

interface RawSignal {
  externalId: string;             // Unique ID from source (for dedup)
  title: string;
  description?: string;
  url?: string;
  publishedAt?: Date;
  deadline?: Date;
  rawHtml?: string;               // Preserve original for re-parsing
  metadata: Record<string, string>;
}

// Deduplication: signals are deduped by (sourceId, externalId)
// If same externalId seen again, update metadata but don't create new signal
```

### AI Classification Pipeline

For each new signal:
1. **Relevance check** — Is this about flooring installation? (fast, Haiku model)
2. **Entity matching** — Which organization/facility does this relate to? (fuzzy matching + AI)
3. **Type classification** — What kind of signal is this? (bid, bond, CIP, etc.)
4. **Value estimation** — Rough estimate of project size if possible
5. **Correlation** — Does this match existing opportunities?

```typescript
// AI prompt for signal classification
const SIGNAL_CLASSIFIER_PROMPT = `
You are analyzing a government bid/project posting to determine if it involves commercial flooring installation.

Classify the following posting:
- Title: {title}
- Description: {description}
- Source: {source}

Respond with:
1. is_flooring: boolean (true if this involves floor covering installation - vinyl, carpet, tile, rubber, etc.)
2. confidence: number (0-1)
3. flooring_types: string[] (what types of flooring, if determinable)
4. estimated_sqft: number | null
5. estimated_value: number | null
6. facility_type: string | null (school, city building, etc.)
7. reasoning: string (brief explanation)

IMPORTANT: Exclude floor MAINTENANCE (cleaning, waxing, stripping). Include floor INSTALLATION and REPLACEMENT only.
`;
```

## Testing Requirements

### Unit Tests
- Score calculation with known inputs → expected score
- Score decay: opportunity inactive for 3 weeks → score reduced by 3
- Signal deduplication: same externalId from same source → no duplicate
- Keyword matching: "flooring replacement" → relevant; "floor cleaning services" → not relevant
- Building age calculation: built 2004, no replacement → 22 years (in 2026)
- Building age calculation: built 1990, replaced 2015 → 11 years (in 2026)
- Correlation: bond_measure + building_age signals for same org → opportunity auto-created
- Territory matching: signal in Riverside County → assigned to rep covering Riverside

### Integration Tests
- Full scrape → signal creation → AI classification → opportunity creation pipeline
- Score model update → all opportunities rescored
- Bid posting matches existing opportunity → auto-linked and status transitioned
- Signal notification delivery to correct reps via WebSocket

### E2E Scenarios
1. **New bid appears on PlanetBids:** Scraper finds it → signal created → AI classifies as flooring → bid entity created → rep notified
2. **Bond measure passes:** Manual signal entry (or scraper) → opportunity created for district → score calculated → rep alerted → rep begins outreach
3. **Building age triggers:** Nightly scan of facilities finds 5 buildings >20 years since last flooring → signals created → correlated with any CIP data → opportunities surfaced

### Seed Data
```typescript
const sampleSignals = [
  {
    type: 'bid_posting',
    source: 'PlanetBids',
    title: 'IFB #2024-089: Flooring Replacement - Lincoln Elementary',
    description: 'Remove existing VCT and install new LVT in 22 classrooms, hallways, and multipurpose room. Approximately 45,000 sqft.',
    url: 'https://pbsystem.planetbids.com/portal/12345/bid/67890',
    metadata: { deadline: '2024-07-15', preBid: '2024-06-28', mandatory: 'yes' },
  },
  {
    type: 'bond_measure',
    source: 'county_elections',
    title: 'Measure E - Riverside USD Facilities Bond - PASSED',
    description: '$150M facilities bond approved by voters. Includes $23M for flooring and interior improvements across 12 campuses.',
    metadata: { amount: '150000000', flooringAllocation: '23000000', passedDate: '2024-11-05' },
  },
  {
    type: 'cip_entry',
    source: 'city_of_riverside',
    title: 'City Hall Complex - Interior Renovation FY 2025-26',
    description: 'Replace flooring in public-facing areas of City Hall and adjacent annex. Budget: $800,000.',
    metadata: { fiscalYear: '2025-26', budget: '800000', category: 'facilities' },
  },
  {
    type: 'meeting_agenda_item',
    source: 'BoardDocs',
    title: 'Facilities Committee - Discussion: Campus Flooring Assessment Results',
    description: 'Staff to present results of district-wide flooring condition assessment. Prioritized replacement schedule to be discussed.',
    metadata: { meetingDate: '2024-04-15', committee: 'Facilities', district: 'Riverside USD' },
  },
];
```

## Error Handling

| Failure | Handling |
|---------|----------|
| Scraper timeout / connection error | Retry 3x with exponential backoff (1s, 5s, 15s). After 3 failures, skip and log. Increment `errorCount`. |
| Scraper blocked (403/429) | Alert admin. Pause scraping for that source for 24 hours. Suggest IP rotation or different approach. |
| AI classification fails | Queue for manual review. Don't discard the signal. |
| AI returns low confidence (<0.5) | Flag signal as "needs review" instead of auto-processing. |
| Duplicate signal detected | Update `lastSeenAt`, check for deadline changes. Don't create new signal. |
| Source structure changed (scraper broken) | Alert admin with error details. Pause source. Log last successful HTML for debugging. |

## UI/UX Requirements

### Intelligence Dashboard (Sales Rep)
- **Top section:** Urgent items (bids due <7 days, high-score opportunities needing action)
- **Opportunity pipeline:** Kanban board with columns for each OpportunityStatus
- **Signal feed:** Chronological feed of new signals with quick-action buttons (dismiss, convert, investigate)
- **Map view:** Territory map with opportunity pins, color-coded by score

### Opportunity Detail View
- Score breakdown showing each factor with its contribution
- Related signals that contributed to this opportunity
- Timeline showing when signals were detected vs. when bid is expected
- Linked contacts at the organization with relationship health
- Suggested actions ("Schedule meeting with Facility Director", "Research architect on project")

### Source Health Dashboard (Admin/Manager)
- Grid of all monitored sources
- Status indicators: green (healthy), yellow (intermittent errors), red (failing)
- Last successful scrape time
- Signal volume per source per week
- Quick toggle to enable/disable sources

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| PlanetBids | Bid postings for schools/cities | Inbound | Every 2 hours |
| BidNet | Bid postings (alternative network) | Inbound | Every 2 hours |
| PublicPurchase | Bid postings (another network) | Inbound | Every 2 hours |
| BoardDocs | School board meeting agendas | Inbound | Daily |
| Granicus/Legistar | City council agendas | Inbound | Daily |
| County election sites | Bond measure tracking | Inbound | Weekly (daily near elections) |
| DSA (CA) | School construction project approvals | Inbound | Weekly |
| County building departments | Building permits | Inbound | Daily |
| Local news (Google News API) | News articles mentioning facilities/construction | Inbound | Every 6 hours |

## Performance Requirements

- Bid board scraping: complete all sources in <10 minutes per cycle
- Signal processing (AI classification): <5 seconds per signal
- Opportunity scoring: <1 second per opportunity
- Dashboard load: <2 seconds with up to 500 active opportunities
- Signal feed: real-time via WebSocket (new signals appear within 30 seconds of scrape)

## Non-Functional Requirements

- Scraping must be rate-limited and polite (respect robots.txt, reasonable intervals)
- Raw scrape data preserved for 90 days (for debugging and re-processing)
- Signal history preserved indefinitely (for analytics and pattern learning)
- Scoring model changes must be auditable (who changed weights, when)
- System must handle graceful degradation if Bedrock is unavailable (queue signals for later processing)

## Resolved Design Decisions

- **Scraping tech:** Use Playwright for JavaScript-rendered sites (PlanetBids, BoardDocs) and Cheerio for static HTML pages (county clerk sites, simple bid boards). Start both from day one — most high-value sources are JS-rendered.
- **Paid data services:** Not for MVP. Start with free public sources. Evaluate CIPList.com after 3 months of usage data shows whether our scraping provides sufficient lead time.
- **Out-of-territory signals:** Log them but mark as `out_of_territory: true`. Don't score or notify. Surface in a monthly "expansion opportunities" report for the owner.
- **Competitor tracking:** Yes, track when public tabulation results are available. Store as fields on the Bid entity (`winningAmount`, `winningCompany`). This feeds the competitive price index over time.

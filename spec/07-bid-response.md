# Bid Response

## Vision & Purpose

A bid response is not just a price on a page — it's a carefully assembled package that must meet every requirement of the solicitation. Missing an addendum acknowledgment, forgetting a bid bond, or submitting 5 minutes late means automatic disqualification regardless of price. The bid response module ensures nothing falls through the cracks.

Beyond compliance, this module helps the team make intelligent bid/no-bid decisions. Not every bid is worth pursuing. By scoring opportunities based on relationship strength, project fit, capacity, and competitive position, the team spends their estimating bandwidth on bids they can actually win.

## Key Concepts

- **Bid/No-Bid Decision** — The strategic choice of whether to pursue a bid, based on multiple factors
- **Addendum** — Official modification to the bid documents after initial posting (must be acknowledged)
- **Responsive Bid** — A bid that meets all procedural requirements (even if not the lowest price)
- **Responsible Bidder** — A bidder who has the capacity, experience, and qualifications to perform the work
- **Pre-Bid Meeting** — A meeting (sometimes mandatory) where the agency walks bidders through the project
- **Tabulation** — Public opening/listing of all bid amounts after the deadline

## User Stories

### Sales Rep
- As a sales rep, I want a clear view of all active bids in my territory with deadlines, status, and what's needed from me (P0)
- As a sales rep, I want to record my bid/no-bid recommendation with reasoning that management can review (P0)
- As a sales rep, I want to be alerted about pre-bid meetings (especially mandatory ones) and submission deadlines (P0)
- As a sales rep, I want to log win/loss results and debrief notes so we learn from every bid (P1)

### Estimator
- As an estimator, I want to receive bid assignments with all documents (RFP, plans, addenda) already organized (P0)
- As an estimator, I want AI to parse the RFP and extract key requirements, deadlines, and scope so I don't miss anything (P1)
- As an estimator, I want to see my workload (bids assigned with deadlines) so I can manage my time (P0)

### Admin
- As an admin, I want a checklist of everything needed for each bid submission (documents, bonds, signatures) (P0)
- As an admin, I want to track addenda and ensure all are acknowledged before submission (P0)
- As an admin, I want to assemble the final submission package with all components in the correct order (P1)

### Owner/Manager
- As an owner, I want to see the bid/no-bid decision pipeline and override decisions when needed (P0)
- As an owner, I want to see bid hit rate and margins to evaluate our bidding strategy (P1)
- As an owner, I want to see competitive intelligence (who won, at what price) to adjust our pricing (P1)

## Technical Design

### Data Model

Uses `Bid` from `02-domain-model.md`. `BidDocument` and `Addendum` are embedded JSON arrays on `Bid` because they are always managed within a bid for MVP.

#### BidDecisionMatrix
Structured bid/no-bid scoring.

```typescript
interface BidDecisionMatrix {
  id: string;
  bidId: string;                  // FK → Bid
  factors: BidDecisionFactor[];
  totalScore: number;             // Weighted sum
  recommendation: 'bid' | 'no_bid' | 'discuss';
  submittedBy: string;            // FK → User (who filled out the matrix)
  approvedBy?: string;            // FK → User (manager who approved/overrode)
  approvalNotes?: string;
  createdAt: DateTime;
  updatedAt: DateTime;
}

interface BidDecisionFactor {
  name: string;
  weight: number;                 // 0-1
  score: number;                  // 0-100
  notes?: string;
}

// Default decision factors
const BID_DECISION_FACTORS = [
  { name: 'relationship_strength', weight: 0.20, description: 'Do we have relationships with decision-makers?' },
  { name: 'project_fit', weight: 0.20, description: 'Is this in our product/size sweet spot?' },
  { name: 'competitive_position', weight: 0.15, description: 'Are we approved vendor, cooperative, spec influence?' },
  { name: 'capacity', weight: 0.15, description: 'Do we have crews available for the timeline?' },
  { name: 'profitability', weight: 0.15, description: 'Can we bid this at acceptable margins?' },
  { name: 'geography', weight: 0.10, description: 'Is this in our primary territory?' },
  { name: 'competition', weight: 0.05, description: 'How many competitors are likely to bid?' },
];
```

#### SubmissionChecklist
Dynamic checklist for bid submission requirements.

```typescript
interface SubmissionChecklist {
  id: string;
  bidId: string;                  // FK → Bid
  items: ChecklistItem[];
  completionPercentage: number;   // Computed
  isComplete: boolean;            // All required items checked
  lastUpdatedAt: DateTime;
}

interface ChecklistItem {
  id: string;
  category: 'document' | 'form' | 'bond' | 'signature' | 'addendum' | 'other';
  description: string;            // "Bid Bond (10% of bid amount)"
  required: boolean;
  completed: boolean;
  completedAt?: DateTime;
  completedBy?: string;           // FK → User
  attachmentUrl?: string;         // If a document is attached
  notes?: string;
  dueDate?: Date;                 // If different from bid deadline
}
```

#### BidCalendarEntry
Unified calendar view of all bid-related dates.

```typescript
interface BidCalendarEntry {
  id: string;
  bidId: string;                  // FK → Bid
  type: 'pre_bid_meeting' | 'questions_deadline' | 'submission_deadline' | 'award_date' | 'addendum_issued' | 'internal_review' | 'custom';
  title: string;
  date: Date;
  time?: string;                  // "2:00 PM PST"
  location?: string;              // For pre-bid meetings
  mandatory: boolean;
  reminderDays: number[];         // [7, 3, 1] — days before to remind
  assignedTo: string[];           // FK → User[] (who should attend/act)
  completed: boolean;
  notes?: string;
}
```

#### WinLossRecord
Post-decision analysis.

```typescript
interface WinLossRecord {
  id: string;
  bidId: string;                  // FK → Bid
  result: 'won' | 'lost' | 'cancelled' | 'no_award';
  ourAmount?: number;             // What we bid
  winningAmount?: number;         // What won (if lost)
  winningCompany?: string;        // Who won (if lost)
  delta?: number;                 // Difference between our bid and winning bid
  deltaPercentage?: number;       // Percentage difference
  factors: string[];              // Why we won/lost: "price", "relationships", "qualifications", "schedule"
  debriefNotes?: string;          // Notes from debrief meeting with agency
  lessonsLearned?: string;        // What to do differently
  debriefedBy?: string;           // FK → User
  debriefDate?: Date;
  createdAt: DateTime;
}
```

### API Endpoints

```typescript
// Bids
GET    /api/v1/bids                            // List, filter by status/territory/deadline
GET    /api/v1/bids/:id
POST   /api/v1/bids                            // Manual bid creation (also created by intelligence scraper)
PUT    /api/v1/bids/:id
POST   /api/v1/bids/:id/transition             // State transition with guards

// Bid Decision
GET    /api/v1/bids/:id/decision               // Get decision matrix
POST   /api/v1/bids/:id/decision               // Submit bid/no-bid decision
POST   /api/v1/bids/:id/decision/approve       // Manager approval/override

// Documents & Addenda
GET    /api/v1/bids/:id/documents
POST   /api/v1/bids/:id/documents              // Upload document
DELETE /api/v1/bids/:id/documents/:docId
POST   /api/v1/bids/:id/addenda/:num/acknowledge  // Acknowledge an addendum
POST   /api/v1/bids/:id/parse                  // AI parse of RFP document

// Submission Checklist
GET    /api/v1/bids/:id/checklist
POST   /api/v1/bids/:id/checklist/generate     // Auto-generate from bid requirements
PUT    /api/v1/bids/:id/checklist/items/:itemId // Toggle/update checklist item
POST   /api/v1/bids/:id/checklist/validate     // Check if ready to submit

// Calendar
GET    /api/v1/bids/calendar                   // All bid dates for calendar view
GET    /api/v1/bids/calendar?range=7           // Next 7 days
POST   /api/v1/bids/:id/calendar-entries       // Add custom calendar entry

// Win/Loss
POST   /api/v1/bids/:id/result                 // Record bid result
GET    /api/v1/bids/:id/win-loss               // Get win/loss record
GET    /api/v1/bids/analytics/win-rate         // Win rate statistics
GET    /api/v1/bids/analytics/competition      // Competitive intelligence

// Workload
GET    /api/v1/bids/workload                   // Estimator workload view
GET    /api/v1/bids/workload/capacity          // Available estimating capacity
```

### Business Rules

- **BR-BID-001:** Bids with mandatory pre-bid meetings require a `pre_bid_meeting` calendar entry. If the meeting date passes without being marked `completed`, block transition to `preparing` status and alert the sales rep.
- **BR-BID-002:** All addenda must be acknowledged (`acknowledged = true`) before the bid can be submitted. The checklist auto-includes addendum acknowledgment items.
- **BR-BID-003:** Bid submission cannot proceed (transition to `submitted`) unless:
  - `submissionChecklist.isComplete = true`
  - Linked estimate exists with status `approved`
  - `submittedAmount` is set
  - All addenda acknowledged
- **BR-BID-004:** After submission deadline passes, if status is still `preparing` or `estimating`, auto-transition to a new status of `missed` and alert the sales manager.
- **BR-BID-005:** Bid/no-bid decision requires a score. If score > 70, recommendation is "bid". If 40-70, recommendation is "discuss". If < 40, recommendation is "no_bid". Manager can override.
- **BR-BID-006:** When a bid is marked `awarded_won`, the system automatically creates a Project entity with pre-populated data from the bid (organization, facilities, contract amount = submitted amount, sales rep, etc.).
- **BR-BID-007:** Calendar reminders fire at configured intervals (default: 7 days, 3 days, 1 day before each deadline). Urgent bids (deadline <3 days when discovered) get immediate notification.
- **BR-BID-008:** The system tracks "competitive price index" — on lost bids where we know the winning amount, calculate our delta. Over time, this shows whether we're systematically high, low, or competitive.
- **BR-BID-009:** No-bid decisions require a reason from a fixed list: "too far from territory", "below minimum project size", "don't carry specified products", "no capacity", "low win probability", "relationship concerns", "other".
- **BR-BID-010:** Bids linked to opportunities (found via intelligence) track the lead time — days between opportunity creation and bid submission deadline. This measures intelligence effectiveness.

### AI Capabilities

#### RFP Parsing
```typescript
interface RfpParseResult {
  title: string;
  bidNumber?: string;
  issuingAgency: string;
  scope: string;                  // Extracted scope of work summary
  estimatedSqFt?: number;        // If derivable from scope
  flooringTypes: string[];        // Product types mentioned
  deadlines: {
    preBidMeeting?: { date: string; location: string; mandatory: boolean };
    questionsDeadline?: string;
    submissionDeadline: string;
    awardDate?: string;
  };
  requirements: {
    bidBond: boolean;
    performanceBond: boolean;
    prevailingWage: boolean;
    insuranceMinimums?: string;
    experience?: string;          // "5 years commercial flooring" etc.
    references?: string;          // Number and type of references required
    certifications?: string[];    // Required certifications
  };
  submissionFormat: {
    copies?: number;              // "Submit 3 copies"
    format?: string;              // "Sealed envelope", "Electronic via PlanetBids"
    sections?: string[];          // Required proposal sections in order
  };
  evaluationCriteria?: Array<{ criterion: string; weight?: number }>;
  confidence: number;             // 0-1 overall confidence in parsing
}
```

## Implementation Guide

### File Locations
- `apps/api/src/modules/bids/` — Bid management module
  - `bids.routes.ts`
  - `bids.service.ts`
  - `decision.service.ts` — Bid/no-bid scoring
  - `checklist.service.ts` — Submission checklist management
  - `calendar.service.ts` — Bid calendar aggregation
  - `winloss.service.ts` — Win/loss tracking and analytics
- `packages/ai/src/prompts/rfp-parser.ts` — AI RFP parsing prompt
- `packages/shared/src/constants/bid-states.ts` — Bid status enum and transitions

### Key Dependencies
- `pdf-parse` — Extract text from RFP PDFs for AI parsing
- `ical-generator` — Export bid calendar to ICS format
- `@aws-sdk/client-bedrock-runtime` — AI RFP parsing

### Implementation Order
1. Bid CRUD with status state machine
2. Document upload and management
3. Calendar entry creation and reminders
4. Submission checklist (manual items)
5. Bid/no-bid decision matrix
6. Status transition guards
7. Auto-checklist generation from bid requirements
8. AI RFP parsing
9. Win/loss recording and analytics
10. Auto-Project creation on award
11. Competitive intelligence dashboard

### Common Pitfalls
- **Time zones on deadlines:** Government bid deadlines are LOCAL time. Store with timezone. "2:00 PM Pacific" is different from "2:00 PM UTC".
- **Addenda can change deadlines:** When an addendum is issued, check if it changes the submission deadline and update accordingly.
- **"Responsive and responsible":** Even if we're the lowest bidder, we can be rejected for being "non-responsive" (missing required documents) or "non-responsible" (insufficient experience). The checklist prevents the former.
- **Electronic vs. physical submission:** Some agencies require physical sealed envelopes, others use electronic portals. Track which format and plan accordingly.

## Testing Requirements

### Unit Tests
- Decision matrix scoring: all factors high → score > 70, recommendation = "bid"
- Decision matrix: low capacity + low relationship → score < 40, recommendation = "no_bid"
- Checklist validation: all required items complete → isComplete = true
- Checklist validation: one required item missing → isComplete = false, specific item identified
- Status transition: `preparing` → `submitted` with incomplete checklist → rejected
- Status transition: `preparing` → `submitted` with everything complete → success
- Auto-Project creation: bid marked won → project exists with correct data
- Calendar reminder calculation: deadline in 3 days → reminder fires today

### Integration Tests
- Full bid lifecycle: discover → decide → estimate → review → submit → win → project created
- AI RFP parsing: upload PDF → extract structured data → verify key fields
- Addendum workflow: new addendum → checklist updated → acknowledge → verify

### Seed Data
```typescript
const sampleBids = [
  {
    title: 'IFB #2024-089: Flooring Replacement - Lincoln Elementary',
    type: 'ifb',
    status: 'preparing',
    organization: 'Riverside USD',
    submissionDeadline: '2024-07-15T14:00:00-07:00',
    preBidMeeting: { date: '2024-06-28T10:00:00-07:00', mandatory: true, location: 'District Office' },
    bondRequired: true,
    prevailingWage: true,
    estimatedValue: 350000,
    documents: ['rfp.pdf', 'floor-plans.pdf', 'specs.pdf'],
    addenda: [{ number: 1, title: 'Revised timeline', acknowledged: true }],
  },
  {
    title: 'RFP #24-156: City Hall Flooring Modernization',
    type: 'rfp',
    status: 'reviewing',
    organization: 'City of Riverside',
    submissionDeadline: '2024-08-01T15:00:00-07:00',
    bondRequired: true,
    prevailingWage: true,
    estimatedValue: 800000,
  },
];
```

## Error Handling

| Failure | Handling |
|---------|----------|
| Submission past deadline | Block submit action. Show message: "Deadline has passed. Contact agency if extension granted." |
| PDF parsing fails (AI) | Show raw PDF text and manual entry form. Don't block workflow. |
| Checklist incomplete at submit | Detailed error listing missing items by category. Link directly to each item. |
| Duplicate bid detected (same bidNumber + org) | Warn user. Allow merge if needed (e.g., imported from scraper AND manually created). |
| Decision matrix not filled | Block transition from `reviewing` to `preparing` until decision is recorded. |

## UI/UX Requirements

### Bid Pipeline View
- Kanban board with columns: Discovered → Reviewing → Preparing → Submitted → Awarded
- Cards show: title, org, deadline, estimated value, assigned rep/estimator
- Color-coded urgency (red: deadline <3 days, yellow: <7 days)
- Filter by territory, assigned rep, bid type
- Quick-action: "No Bid" button on cards in Reviewing column

### Bid Detail View
- **Header:** Title, org, deadline countdown, status badge
- **Tabs:** Overview | Documents | Checklist | Decision | Calendar | Win/Loss
- **Overview tab:** All key dates, requirements summary, assignment, linked estimate
- **Documents tab:** All bid docs + addenda, upload button, addenda acknowledgment toggles
- **Checklist tab:** Categorized checklist with progress bar, toggle items, attach docs to items
- **Decision tab:** Factor scoring matrix, recommendation, approval status

### Bid Calendar (Full Page)
- Month/week/day views showing all bid-related dates
- Color by type: red = deadlines, blue = meetings, green = awards
- Filter by rep, territory, status
- Click date → see all bids with events that day
- Export to ICS / sync with Google Calendar

### Estimator Workload View
- List of assigned bids with deadlines, sorted by urgency
- Capacity indicator: how many hours of estimating work are queued
- "I'm blocked" button to escalate (needs decision, needs documents, etc.)

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| Intelligence module (spec/04) | Discovered bids auto-create Bid entities | Inbound | Real-time |
| Estimating module (spec/06) | Estimates linked to bids | Bidirectional | On assignment |
| Project module (spec/08) | Won bids create projects | Outbound | On award |
| Procurement module (spec/03) | Compliance docs for submission | Read | At submission |
| Calendar (Google/Outlook) | Sync bid dates to personal calendars | Outbound | On bid creation/update |
| AI (Bedrock) | RFP parsing | Outbound | On document upload |

## Performance Requirements

- Bid list load with 200 active bids: < 1 second
- Calendar view generation: < 500ms
- RFP parsing (AI): < 30 seconds for a 50-page document
- Checklist validation: < 100ms
- Win rate calculation across 500 historical bids: < 2 seconds

## Non-Functional Requirements

- All bid documents must be versioned (keep history if re-uploaded)
- Submission timestamps must be accurate to the second (for deadline compliance proof)
- Bid data must be retained for 7 years (legal requirement for government contracts)
- Audit trail on all decision matrix approvals and overrides
- Calendar reminders must be reliable — a missed deadline = disqualification

## Resolved Design Decisions

- **Electronic bid submission:** Not automated for MVP. Too many different portal formats (PlanetBids, BidSync, email, physical delivery). System prepares the complete submission package (documents, forms, checklist) and the rep submits manually. Track submission as a status transition with timestamp.
- **Competitor bid tracking:** Yes. When public bid tabulations are published, store `winningAmount` and `winningCompany` on the Bid entity. Over time this builds a competitive price index per product type and geography.
- **Joint ventures:** Not a separate workflow for MVP. If teaming, create the bid normally and note the JV partner in the description/notes. JV-specific fields (partner company, split percentage) can be added later if customers need it.
- **Bid brief:** Yes, implement as an AI-generated one-page summary. Generated on demand via `POST /bids/:id/brief`. Includes: opportunity context, decision score breakdown, estimate summary, relationship strength with key contacts, and recommended bid amount range. Presented in the Internal Review phase.

# Project Lifecycle

## Vision & Purpose

Winning a bid is only half the battle — executing the project profitably and leaving the customer satisfied is what generates repeat business and referrals. The project lifecycle module manages everything from contract award through warranty expiration, ensuring nothing falls through the cracks during the most operationally complex phase of the business.

This module bridges office and field: project managers coordinate schedules and handle paperwork while field crews log daily production, document quality, and report issues. Smooth execution builds the relationship capital that wins the NEXT project.

## Key Concepts

- **NTP (Notice to Proceed)** — Official authorization from the agency to begin work
- **Submittal** — Product samples/documentation submitted to the architect for approval before installation
- **Change Order** — Modification to the original contract scope/price (requires agency approval)
- **Retention** — Percentage of each payment withheld until project completion (typically 5-10%)
- **Substantial Completion** — When the project is usable for its intended purpose (even with minor punch list items)
- **Close-out Package** — Final documentation bundle (as-builts, warranty info, maintenance instructions)

## User Stories

### Project Manager
- As a PM, I want a single dashboard showing all my active projects with status, upcoming milestones, and issues (P0)
- As a PM, I want to assign crews to projects and see their availability across all projects (P0)
- As a PM, I want to track material orders and know when deliveries are expected (P0)
- As a PM, I want to receive daily log summaries from field crews without chasing them (P0)
- As a PM, I want to manage punch lists and track resolution through sign-off (P0)
- As a PM, I want to generate close-out documentation packages (P1)

### Field Installer
- As an installer, I want to quickly log my daily hours, production, and any issues from my phone (P0)
- As an installer, I want to see my schedule and which project I'm on each day (P0)
- As an installer, I want to take photos and attach them to daily logs or punch list items (P0)
- As an installer, I want to see floor plans and scope details for my assigned areas (P0)

### Owner/Manager
- As an owner, I want to see project margins in real-time (estimated vs. actual costs) (P0)
- As an owner, I want to be alerted when a project is going over budget or behind schedule (P0)
- As an owner, I want to see crew utilization across all active projects (P1)

## Technical Design

### Data Model

Uses `Project`, `DailyLog`, and `PunchListItem` from `02-domain-model.md`. `MaterialOrder` and `ChangeOrder` are embedded JSON arrays on `Project` because they are edited and viewed within the project context for MVP.

#### CrewAssignment
```typescript
interface CrewAssignment {
  id: string;
  projectId: string;              // FK → Project
  userId: string;                 // FK → User (installer)
  role: 'lead' | 'journeyman' | 'apprentice';
  startDate: Date;
  endDate?: Date;
  dailyRate?: number;             // For labor cost tracking
  notes?: string;
}
```

#### ProjectSchedule
```typescript
interface ProjectMilestone {
  id: string;
  projectId: string;              // FK → Project
  title: string;
  type: MilestoneType;
  plannedDate: Date;
  actualDate?: Date;
  status: 'pending' | 'completed' | 'overdue' | 'skipped';
  dependsOn?: string[];           // FK → ProjectMilestone[] (predecessors)
  assignedTo?: string;            // FK → User
  notes?: string;
}

enum MilestoneType {
  NTP = 'ntp',
  SUBMITTAL_DUE = 'submittal_due',
  SUBMITTAL_APPROVED = 'submittal_approved',
  MATERIAL_ORDER = 'material_order',
  MATERIAL_DELIVERY = 'material_delivery',
  MOBILIZATION = 'mobilization',
  PHASE_START = 'phase_start',
  PHASE_COMPLETE = 'phase_complete',
  SUBSTANTIAL_COMPLETION = 'substantial_completion',
  PUNCH_LIST_WALK = 'punch_list_walk',
  PUNCH_LIST_COMPLETE = 'punch_list_complete',
  CLOSEOUT_SUBMITTED = 'closeout_submitted',
  FINAL_PAYMENT = 'final_payment',
  WARRANTY_START = 'warranty_start',
  WARRANTY_END = 'warranty_end',
  CUSTOM = 'custom',
}
```

#### ProjectFinancials (computed view)
```typescript
interface ProjectFinancials {
  projectId: string;
  contractAmount: number;
  changeOrderTotal: number;
  currentContractAmount: number;   // contract + change orders
  
  // Costs
  laborCostToDate: number;         // From daily logs × rates
  materialCostToDate: number;      // From material orders delivered
  subcontractorCostToDate: number;
  equipmentCostToDate: number;
  totalCostToDate: number;
  
  // Budget comparison
  estimatedTotalCost: number;      // From linked estimate
  costVariance: number;            // totalCostToDate - (estimatedTotalCost * percentComplete)
  projectedFinalCost: number;      // Based on current burn rate
  
  // Margin
  currentMargin: number;           // (currentContractAmount - projectedFinalCost) / currentContractAmount
  estimatedMargin: number;         // Original estimate margin
  marginVariance: number;          // currentMargin - estimatedMargin
  
  // Progress
  percentComplete: number;         // sqFt installed / total sqFt
  daysRemaining: number;           // Based on schedule or progress rate
  onSchedule: boolean;
  
  // Billing
  invoicedToDate: number;
  retentionHeld: number;           // Held back by agency
  remainingToInvoice: number;
}
```

### API Endpoints

```typescript
// Projects
GET    /api/v1/projects                        // List, filter by status/PM/territory
GET    /api/v1/projects/:id
PUT    /api/v1/projects/:id
POST   /api/v1/projects/:id/transition         // State transition

// Crew
GET    /api/v1/projects/:id/crew               // Assigned crew members
POST   /api/v1/projects/:id/crew               // Assign crew member
DELETE /api/v1/projects/:id/crew/:userId        // Remove from project
GET    /api/v1/crew/availability               // Cross-project availability view

// Schedule
GET    /api/v1/projects/:id/milestones
POST   /api/v1/projects/:id/milestones
PUT    /api/v1/projects/:id/milestones/:mid
POST   /api/v1/projects/:id/milestones/:mid/complete
GET    /api/v1/projects/:id/schedule           // Gantt-style schedule data

// Daily Logs
GET    /api/v1/projects/:id/daily-logs         // Filter by date/user
POST   /api/v1/projects/:id/daily-logs         // Create (mobile app)
PUT    /api/v1/projects/:id/daily-logs/:logId
GET    /api/v1/projects/:id/daily-logs/summary // Aggregated production summary

// Punch Lists
GET    /api/v1/projects/:id/punch-list
POST   /api/v1/projects/:id/punch-list         // Add item
PUT    /api/v1/projects/:id/punch-list/:itemId
POST   /api/v1/projects/:id/punch-list/:itemId/complete  // Mark completed with photo
POST   /api/v1/projects/:id/punch-list/:itemId/verify    // PM verifies fix

// Change Orders
GET    /api/v1/projects/:id/change-orders
POST   /api/v1/projects/:id/change-orders
PUT    /api/v1/projects/:id/change-orders/:coId
POST   /api/v1/projects/:id/change-orders/:coId/approve

// Materials
GET    /api/v1/projects/:id/materials
POST   /api/v1/projects/:id/materials          // Log material order
PUT    /api/v1/projects/:id/materials/:matId   // Update delivery status

// Documents
GET    /api/v1/projects/:id/documents
POST   /api/v1/projects/:id/documents          // Upload project document
GET    /api/v1/projects/:id/closeout-package   // Generate close-out package

// Financials
GET    /api/v1/projects/:id/financials         // Computed financial summary
GET    /api/v1/projects/financials/overview    // All projects margin overview

// Photos
GET    /api/v1/projects/:id/photos             // All project photos
POST   /api/v1/projects/:id/photos             // Upload photo (from mobile)
```

### Business Rules

- **BR-PROJ-001:** Project cannot transition from `contracting` to `material_order` until: contract document uploaded, bond confirmation received (if required), insurance certificate current, NTP received.
- **BR-PROJ-002:** Material orders must be placed with sufficient lead time. Alert PM if `expectedDelivery` is after project `startDate`.
- **BR-PROJ-003:** Daily logs are expected from each assigned crew member for every workday on the project. Missing logs trigger a reminder notification at 6 PM.
- **BR-PROJ-004:** Punch list items must have "before" photos when created and "after" photos when marked complete. PM verification requires a separate step.
- **BR-PROJ-005:** Change orders exceeding 10% of original contract value require OWNER approval (not just PM).
- **BR-PROJ-006:** Project cannot transition to `closeout` until all punch list items are either `verified` or `disputed` (with resolution notes).
- **BR-PROJ-007:** Warranty period begins on the date of `substantial_completion` milestone, not project close-out. System auto-calculates `warrantyEndDate` based on product warranty years.
- **BR-PROJ-008:** When project `percentComplete` > 50% and `currentMargin` drops below `estimatedMargin` by more than 5 percentage points, alert the OWNER.
- **BR-PROJ-009:** Retention percentage is applied to every invoice. Track cumulative retention. Retention release is logged as a final billing event after close-out acceptance.
- **BR-PROJ-010:** Projects in `warranty` status get a check-in reminder at 6 months and 11 months (before warranty expires). This is a relationship-building touchpoint — "How's the flooring holding up?"

### State Machine Transition Guards

```typescript
const PROJECT_TRANSITIONS = {
  'awarded→contracting': {
    guard: () => true, // Always allowed after award
  },
  'contracting→material_order': {
    guard: (project) => {
      return project.documents.some(d => d.type === 'contract')
        && (!project.bid?.bondRequired || project.documents.some(d => d.type === 'bond'))
        && project.documents.some(d => d.type === 'insurance')
        && project.milestones.some(m => m.type === 'ntp' && m.status === 'completed');
    },
    errorMessage: 'Missing contract documents, bonds, insurance, or NTP',
  },
  'material_order→scheduled': {
    guard: (project) => {
      return project.materialOrders.length > 0
        && project.startDate != null
        && project.crewAssignments.length > 0;
    },
    errorMessage: 'Need material orders placed, start date set, and crew assigned',
  },
  'scheduled→in_progress': {
    guard: (project) => {
      return project.materialOrders.every(mo => mo.status === 'delivered' || mo.status === 'partial')
        && new Date() >= project.startDate;
    },
    errorMessage: 'Materials not yet delivered or start date not reached',
  },
  'in_progress→punch_list': {
    guard: (project) => project.percentComplete >= 90,
    errorMessage: 'Project must be at least 90% complete',
  },
  'punch_list→closeout': {
    guard: (project) => {
      const openItems = project.punchListItems.filter(i => !['verified', 'disputed'].includes(i.status));
      return openItems.length === 0;
    },
    errorMessage: 'Open punch list items remain',
  },
  'closeout→complete': {
    guard: (project) => {
      return project.documents.some(d => d.type === 'closeout')
        && project.documents.some(d => d.type === 'warranty');
    },
    errorMessage: 'Close-out package and warranty documentation required',
  },
};
```

## Implementation Guide

### File Locations
- `apps/api/src/modules/projects/` — Project lifecycle module
  - `projects.routes.ts`
  - `projects.service.ts` — Core project management
  - `crew.service.ts` — Crew assignment and availability
  - `daily-logs.service.ts` — Daily log management
  - `punch-list.service.ts` — Punch list workflow
  - `financials.service.ts` — Real-time financial calculations
  - `schedule.service.ts` — Milestones and scheduling
  - `closeout.service.ts` — Close-out package generation
- `packages/shared/src/constants/project-states.ts` — Status enum and transitions

### Key Dependencies
- `@react-pdf/renderer` — Generate close-out documentation PDFs
- `date-fns` — Date calculations for schedules, durations, reminders
- `decimal.js` — Financial calculations

### Implementation Order
1. Project CRUD with state machine
2. Crew assignment
3. Milestone/schedule management
4. Daily log creation (essential for mobile app)
5. Punch list management
6. Material order tracking
7. Financial calculations
8. Change order workflow
9. Close-out package generation
10. Warranty tracking and reminders
11. Historical comparison with estimates

## Testing Requirements

### Unit Tests
- Transition guard: contracting → material_order without NTP → blocked
- Transition guard: with all docs → allowed
- Financial calculation: 3 daily logs + 2 material orders → correct totals
- Margin alert: currentMargin 5+ points below estimated → alert triggered
- Percent complete: 15000 sqft installed of 20000 total → 75%
- Warranty end date: substantial completion 2024-06-15 + 10 year warranty → 2034-06-15

### Integration Tests
- Full lifecycle: award → contracting → material → schedule → in_progress → punch → closeout → complete
- Daily log from mobile (offline sync) → appears in PM dashboard
- Change order approval → contract amount updated → financials recalculated
- Punch list photo workflow: create with before photo → complete with after photo → PM verify

### Seed Data
```typescript
const sampleProject = {
  title: 'Lincoln Elementary - Flooring Replacement',
  status: 'in_progress',
  organization: 'Riverside USD',
  contractAmount: 345000,
  startDate: '2024-06-01',
  completionDate: '2024-08-15',
  prevailingWage: true,
  crew: [
    { userId: 'carlos-installer', role: 'lead' },
    { userId: 'marco-installer', role: 'journeyman' },
    { userId: 'james-installer', role: 'apprentice' },
  ],
  milestones: [
    { type: 'ntp', plannedDate: '2024-05-15', status: 'completed', actualDate: '2024-05-14' },
    { type: 'material_delivery', plannedDate: '2024-05-28', status: 'completed' },
    { type: 'mobilization', plannedDate: '2024-06-01', status: 'completed' },
    { type: 'phase_complete', title: 'Phase 1: Classrooms', plannedDate: '2024-07-01', status: 'pending' },
    { type: 'substantial_completion', plannedDate: '2024-08-10', status: 'pending' },
  ],
  dailyLogs: [
    { date: '2024-06-03', userId: 'carlos-installer', hoursWorked: 8, sqFtInstalled: 450, areasWorked: ['Room 101', 'Room 102'] },
  ],
};
```

## Error Handling

| Failure | Handling |
|---------|----------|
| Daily log sync conflict (mobile) | Last-write-wins for text fields; photos are additive (never deleted on sync). Show sync conflict indicator if detected. |
| Photo upload fails (poor connectivity) | Queue locally on device. Retry on next sync. Show "pending upload" indicator. |
| Financial calculation with missing data | Calculate with available data, flag incomplete items. Show "estimated" indicator. |
| Milestone overdue | Auto-mark as overdue. Alert PM and manager. Don't block other transitions. |

## UI/UX Requirements

### Project Dashboard (PM)
- Active projects as cards with: progress ring, margin indicator, days remaining, open issues count
- Quick filters: by status, by urgency (behind schedule, over budget)
- Click through to project detail

### Project Detail View
- **Header:** Title, org, status badge, progress bar, financial summary
- **Tabs:** Schedule | Crew | Daily Logs | Punch List | Materials | Financials | Documents
- **Schedule tab:** Timeline/Gantt showing milestones with dependencies
- **Crew tab:** Assigned members, availability conflicts, daily rate
- **Daily Logs tab:** Chronological entries with photos, filterable by crew member
- **Financials tab:** Budget vs. actual breakdown, trend chart, margin projection

### Punch List View
- Sortable/filterable table: location, description, priority, status, assigned, photos
- Bulk actions: assign all to crew member, export as PDF for field copy
- Photo modal: before/after side-by-side
- Status flow visible: Open → In Progress → Completed → Verified

### Crew Calendar (Cross-Project)
- Calendar view showing which crew member is on which project each day
- Color-coded by project
- Drag-to-reassign capability
- Availability gaps highlighted

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| Estimating module (spec/06) | Compare actuals vs. estimate | Read | Ongoing |
| Mobile app (spec/11) | Daily logs, photos, punch list updates | Bidirectional | Real-time sync |
| Relationship module (spec/05) | Warranty check-ins as relationship touchpoints | Read | At warranty milestones |
| Accounting (QuickBooks) | Invoice amounts, cost tracking | Bidirectional | On invoice/payment |
| Calendar (Google/Outlook) | Project milestones on personal calendars | Outbound | On milestone creation |

## Performance Requirements

- Project dashboard with 20 active projects: < 1 second
- Financial calculation for one project: < 500ms
- Daily log creation (mobile sync): < 200ms server-side
- Photo upload processing: < 5 seconds (resize + store)
- Gantt/schedule render with 50 milestones: < 1 second

## Non-Functional Requirements

- Daily logs must be immutable once synced (field records for labor compliance)
- All project documents must be version-controlled
- Financial data must be auditable (who changed what, when)
- Project data retained for 10 years minimum (warranty + legal)
- Mobile daily logs must work fully offline (sync when connectivity returns)

## Resolved Design Decisions

- **Project scheduling:** Build a simple built-in scheduler (milestone dates + crew assignment per day). No MS Project/Procore integration for MVP. Flooring projects are typically 1-4 weeks — a full Gantt chart is overkill. Simple calendar view with crew assignments is sufficient.
- **Cost tracking detail:** Aggregated daily via DailyLog (hours x crew size x wage rate = labor cost per day). Not per-individual. Material costs tracked via MaterialOrder entities. This gives sufficient margin visibility without burdening installers with per-person time tracking.
- **Timelapse photos:** Not for MVP. Photos are stored and viewable chronologically, but no video generation. Could be a P2 feature using project photos sorted by date.
- **Subcontractor management:** Minimal. Track as line items in the estimate (subcontractor total) and as notes on the project. Don't build a full sub-management module — that's Procore's job. If the sub needs to be onsite, the PM coordinates via phone/email outside the system.

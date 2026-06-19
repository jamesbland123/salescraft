# Flooring Estimating

## Vision & Purpose

Estimating is where money is made or lost. A miscalculated waste factor, a missed transition strip, or a wrong prevailing wage rate can turn a profitable project into a loss. Conversely, an estimator who can produce accurate proposals fast wins more work — the first accurate bid response on the table often wins informal quotes.

Salescraft's estimating module gives estimators the tools to work faster and more accurately: digital takeoffs from floor plans, automatic material calculations with correct waste factors, live prevailing wage lookups, historical data from past projects, and AI-assisted proposal writing. It integrates directly with the bid response module — an estimate flows seamlessly into a formatted bid submission.

## Key Concepts

- **Takeoff** — The process of measuring square footage from floor plans, room by room
- **Waste Factor** — Extra material beyond measured area needed for cuts, patterns, and mistakes. Varies by product type (e.g., 10% for LVT, 15% for patterned carpet)
- **Prevailing Wage** — Government-mandated minimum labor rates that apply to public works projects
- **Alternates** — Additional pricing options presented alongside the base bid (e.g., "Alternate 1: LVT instead of VCT, add $2.50/sqft")
- **Productivity Rate** — How many square feet a crew can install per day for a given product type
- **Transition** — Where two flooring types meet, requiring a metal or vinyl transition strip
- **Moisture Mitigation** — Concrete moisture testing and remediation often required before installation

## User Stories

### Estimator
- As an estimator, I want to upload a floor plan PDF and mark up rooms with their flooring types and square footage (P0)
- As an estimator, I want the system to automatically calculate materials (including waste), labor hours, and costs once I enter room measurements (P0)
- As an estimator, I want to automatically pull prevailing wage rates for the project's county and trade (P0)
- As an estimator, I want to compare products side-by-side that meet the project specifications (P0)
- As an estimator, I want to generate a formatted proposal document that meets government submission requirements (P1)
- As an estimator, I want to see actual costs from similar past projects to improve my estimate accuracy (P1)
- As an estimator, I want AI to help write the narrative sections of proposals (qualifications, approach, schedule) (P1)
- As an estimator, I want to quickly create alternate pricing scenarios (e.g., different product tiers) (P0)

### Sales Rep
- As a sales rep, I want to see estimate status (in progress, ready for review) so I know when I can submit a bid (P0)
- As a sales rep, I want to request a "quick budget number" (rough estimate without full takeoff) for relationship conversations (P1)

### Owner/Manager
- As an owner, I want to approve margins before bid submission (especially when below minimum threshold) (P0)
- As an owner, I want to compare estimated vs. actual costs on completed projects to measure estimating accuracy (P1)

## Technical Design

### Data Model

Uses `Estimate`, `EstimateArea`, `Product`, and `LineItem` entities from `02-domain-model.md`.

#### TakeoffMarkup
Annotations on floor plan PDFs.

```typescript
interface TakeoffMarkup {
  id: string;
  estimateId: string;             // FK → Estimate
  pageNumber: number;             // Which page of the PDF
  documentUrl: string;            // S3 URL of the floor plan PDF
  annotations: TakeoffAnnotation[];
  createdAt: DateTime;
  updatedAt: DateTime;
}

interface TakeoffAnnotation {
  id: string;
  areaId?: string;                // FK → EstimateArea (links to the calculated area)
  type: 'room' | 'hallway' | 'transition' | 'note' | 'measurement';
  label: string;                  // "Room 101", "Main Hallway"
  geometry: AnnotationGeometry;   // Drawing coordinates
  sqFt?: number;                  // Calculated from geometry if polygon
  flooringType?: FlooringType;    // What product goes here
  color: string;                  // Display color (for visual grouping)
}

interface AnnotationGeometry {
  type: 'polygon' | 'line' | 'point' | 'rectangle';
  coordinates: number[][];        // Array of [x, y] points
  scale?: number;                 // Pixels per foot (set from plan scale)
}
```

#### ProductSpecRequirement
Project requirements that products must match.

```typescript
interface ProductSpecRequirement {
  id: string;
  estimateId: string;             // FK → Estimate
  requirements: {
    flooringTypes: FlooringType[];   // Acceptable product types
    minWearLayerMils?: number;       // Minimum wear layer
    moistureResistant?: boolean;
    adaCompliant?: boolean;
    fireRating?: string;             // Minimum fire rating
    maxBudgetPerSqFt?: number;       // Budget ceiling
    sustainabilityCerts?: string[];  // Required certifications
    colorFamily?: string[];          // Acceptable color families
    commercialRating?: string;       // Traffic rating requirement
  };
  matchingProducts: string[];     // FK → Product[] (computed matches)
  lastMatchedAt: DateTime;
}
```

#### EstimateTemplate
Reusable estimate templates for common project types.

```typescript
interface EstimateTemplate {
  id: string;
  name: string;                   // "Standard School Classroom LVT"
  description?: string;
  projectType: string;            // "school_classroom", "corridor", "cafeteria"
  defaultProduct?: string;        // FK → Product
  defaultWasteFactor: number;
  defaultAdditionalMaterials: LineItem[]; // Standard additional items
  laborProductivitySqFtPerDay: number;   // Expected daily production
  notes?: string;
  createdAt: DateTime;
  updatedAt: DateTime;
}
```

#### HistoricalProjectData (for accuracy comparison)
```typescript
interface HistoricalComparison {
  projectId: string;              // FK → Project (completed)
  estimateId: string;             // FK → Estimate (original estimate)
  estimatedMaterialCost: number;
  actualMaterialCost: number;
  materialVariance: number;       // Percentage over/under
  estimatedLaborHours: number;
  actualLaborHours: number;
  laborVariance: number;
  estimatedTotal: number;
  actualTotal: number;
  totalVariance: number;
  factors: string[];              // What caused the variance
  createdAt: DateTime;
}
```

### API Endpoints

```typescript
// Estimates
GET    /api/v1/estimates                       // List, filter by status/bidId/estimatorId
POST   /api/v1/estimates                       // Create new estimate
GET    /api/v1/estimates/:id
PUT    /api/v1/estimates/:id                   // Update estimate details
POST   /api/v1/estimates/:id/duplicate         // Clone an estimate (for alternates/revisions)
POST   /api/v1/estimates/:id/transition        // Status transition: { to: 'ready_for_review' }
DELETE /api/v1/estimates/:id                   // Soft delete (draft only)

// Estimate Areas
GET    /api/v1/estimates/:id/areas
POST   /api/v1/estimates/:id/areas             // Add area
PUT    /api/v1/estimates/:id/areas/:areaId
DELETE /api/v1/estimates/:id/areas/:areaId
POST   /api/v1/estimates/:id/areas/calculate   // Recalculate all area totals

// Takeoff Markup
GET    /api/v1/estimates/:id/takeoff
POST   /api/v1/estimates/:id/takeoff           // Upload floor plan + initial markup
PUT    /api/v1/estimates/:id/takeoff           // Update annotations
POST   /api/v1/estimates/:id/takeoff/set-scale // Set the measurement scale

// Product Matching
POST   /api/v1/estimates/:id/match-products    // Find products meeting spec requirements
GET    /api/v1/products/compare?ids=a,b,c      // Side-by-side product comparison

// Calculations
POST   /api/v1/estimates/:id/calculate         // Full recalculation (materials + labor + totals)
GET    /api/v1/estimates/:id/wages             // Look up applicable wage rates
POST   /api/v1/estimates/:id/quick-budget      // Generate rough budget without full takeoff
// Body: { sqFt: 15000, flooringType: 'lvt', county: 'Riverside', prevailingWage: true }

// Proposals
POST   /api/v1/estimates/:id/generate-proposal // AI-generate proposal narrative
GET    /api/v1/estimates/:id/proposal          // Get generated proposal
PUT    /api/v1/estimates/:id/proposal          // Edit proposal narrative
POST   /api/v1/estimates/:id/export-pdf        // Export formatted PDF proposal

// Templates
GET    /api/v1/estimate-templates
POST   /api/v1/estimate-templates
PUT    /api/v1/estimate-templates/:id
POST   /api/v1/estimates/:id/apply-template    // Apply template to an area

// Historical Data
GET    /api/v1/estimates/:id/historical        // Similar past projects for comparison
GET    /api/v1/estimates/accuracy-report       // Overall estimating accuracy metrics
```

### Business Rules

- **BR-EST-001:** Waste factor defaults vary by product type:
  - LVT/LVP plank: 10%
  - LVT/LVP tile: 8%
  - VCT: 5%
  - Carpet tile: 5% (10% for patterns)
  - Broadloom carpet: 12-15% (depends on room widths vs. roll widths)
  - Sheet vinyl: 10-15% (similar to broadloom)
  - Rubber: 10%
  - Ceramic/porcelain tile: 12% (15% for diagonal patterns)

- **BR-EST-002:** When `prevailingWage = true`, labor rates MUST come from the WageDetermination lookup for the project's county. The relevant trade classification is "Floor Layer" (or "Tile Setter" for ceramic). If no determination exists for the specific county, use the closest available and flag for human review.

- **BR-EST-003:** Standard additional materials per area (auto-added unless removed):
  - Adhesive: 1 gallon per 150 sqft (varies by product; pulled from product specs)
  - Transition strips: 1 per doorway/material change (3 linear feet each)
  - Base cove: perimeter linear footage (pulled from room dimensions)
  - Moisture testing: $0.25/sqft for concrete slabs (waived if building <5 years old)

- **BR-EST-004:** Estimates require OWNER or SALES_MANAGER approval when profit margin is below 15%.

- **BR-EST-005:** Estimates automatically include bond cost when linked to a bid that requires bonding. Bond cost calculation: `bondPercentage/100 * estimateTotal * bondRate` where bondRate is typically 1.5-3% of bond face value.

- **BR-EST-006:** Labor productivity rates (sqft installed per crew per day):
  - LVT/LVP: 400-600 sqft/crew/day (4-person crew)
  - VCT: 800-1200 sqft/crew/day
  - Carpet tile: 600-900 sqft/crew/day
  - Broadloom: 500-800 sqft/crew/day
  - Sheet vinyl: 400-600 sqft/crew/day
  - Rubber: 300-500 sqft/crew/day
  - Ceramic tile: 100-200 sqft/crew/day
  These are used to estimate labor hours: `totalSqFt / productivityRate * 8 hours * crewSize`

- **BR-EST-007:** Quick budget estimates use rough rates:
  - LVT (prevailing wage): $8-12/sqft installed
  - LVT (no prevailing wage): $5-8/sqft installed
  - Carpet tile (prevailing wage): $6-9/sqft installed
  - VCT (prevailing wage): $5-7/sqft installed
  These are configurable and updated based on historical actuals.

- **BR-EST-008:** When creating alternates, the base bid is always Alternate 0. Additional alternates are numbered sequentially. Each alternate can change product, quantity, or both for specific areas.

- **BR-EST-009:** The system tracks "revision history" — each save creates a version. The submitted version is locked (immutable). Subsequent changes create a new version.

- **BR-EST-010:** Product pricing is only valid for the date pulled. Estimates older than 90 days without updates show a "pricing may be stale" warning.

### Calculation Engine

```typescript
// Core calculation flow
interface CalculationInput {
  areas: Array<{
    sqFt: number;
    productId: string;
    wasteFactor: number;          // Override or use default
    additionalMaterials: LineItem[];
  }>;
  prevailingWage: boolean;
  county?: string;
  overheadPercentage: number;     // Company default or bid-specific
  profitPercentage: number;       // Target margin
  includeBond: boolean;
  bondPercentage?: number;
}

interface CalculationOutput {
  areas: Array<{
    materialSqFt: number;         // sqFt * (1 + wasteFactor/100)
    materialCost: number;
    laborHours: number;
    laborCost: number;
    additionalMaterialsCost: number;
    subtotal: number;
  }>;
  totals: {
    materialTotal: number;
    laborTotal: number;
    equipmentTotal: number;
    subcontractorTotal: number;
    directCostTotal: number;
    overhead: number;
    profit: number;
    bondCost: number;
    grandTotal: number;
  };
  laborSummary: {
    totalHours: number;
    estimatedDays: number;        // Based on 8-hour days
    crewSize: number;             // Recommended crew
    rateUsed: number;             // Hourly rate applied
    rateSource: string;           // "Prevailing wage: Riverside County, Floor Layer" or "Standard rate"
  };
}
```

## Implementation Guide

### File Locations
- `apps/api/src/modules/estimating/` — Estimating module
  - `estimating.routes.ts`
  - `estimating.service.ts` — Orchestration
  - `calculation.service.ts` — Math engine (pure functions, highly testable)
  - `takeoff.service.ts` — Floor plan markup handling
  - `product-matching.service.ts` — Spec matching logic
  - `proposal.service.ts` — AI proposal generation
- `packages/shared/src/constants/flooring-types.ts` — Waste factors, productivity rates, product types
- `packages/ai/src/prompts/proposal-writer.ts` — Proposal narrative generation

### Key Dependencies
- `pdf-lib` or `pdfjs-dist` — PDF rendering for takeoff viewer
- `fabric.js` (via React wrapper) — Canvas-based markup tool for floor plan annotations
- `@react-pdf/renderer` — Generate formatted proposal PDFs
- `decimal.js` — Precise financial calculations (avoid floating point)

### Implementation Order
1. Product CRUD and catalog management
2. Estimate CRUD with area management
3. Calculation engine (pure functions, test-driven)
4. Prevailing wage lookup integration
5. Estimate state machine (transitions with guards)
6. PDF export (basic formatted proposal)
7. Takeoff markup (floor plan upload + annotation)
8. Product spec matching
9. AI proposal writing
10. Historical comparison (requires completed projects)
11. Quick budget calculator
12. Estimate templates

### Common Pitfalls
- **Floating point math:** Use `decimal.js` for all money calculations. `0.1 + 0.2 !== 0.3` in JavaScript.
- **Prevailing wage complexity:** Rates have base + fringe, and the effective date matters (a 12-month project may span a rate change).
- **Waste factor is NOT markup:** 10% waste on 1000 sqft = order 1100 sqft. Don't confuse with 10% price markup.
- **Base cove (baseboard):** Often forgotten in estimates. Calculate perimeter from room dimensions and include material + labor.
- **Mobilization/demobilization:** For multi-building projects, include crew travel and setup time.
- **Moisture testing:** Required by most manufacturers for warranty. Don't forget to include in subcontractor costs.

## Testing Requirements

### Unit Tests (Calculation Engine)
```typescript
// Test: basic material calculation
input: { sqFt: 1000, wasteFactor: 10, materialCostPerSqFt: 4.50 }
expected: { materialSqFt: 1100, materialCost: 4950.00 }

// Test: labor calculation with prevailing wage
input: { sqFt: 5000, productType: 'lvt', prevailingWage: true, county: 'Riverside' }
// Given wage rate: $55.72/hr (journeyman + fringe)
// Given productivity: 500 sqft/crew/day, crew of 4
// Expected days: 5000/500 = 10 days
// Expected hours: 10 * 8 * 4 = 320 hours
expected: { laborHours: 320, laborCost: 17830.40 }

// Test: bond cost
input: { estimateTotal: 250000, bondRequired: true, bondPercentage: 10, bondRate: 0.025 }
// Bid bond face value: 250000 * 10% = 25000
// Bond cost: 25000 * 2.5% = 625
expected: { bondCost: 625 }

// Test: full estimate calculation
input: {
  areas: [
    { sqFt: 1000, product: 'lvt', wasteFactor: 10, materialCost: 4.50 },
    { sqFt: 500, product: 'carpet_tile', wasteFactor: 5, materialCost: 3.25 },
  ],
  prevailingWage: true, county: 'Riverside',
  overhead: 15, profit: 12, bond: true, bondPercentage: 10
}
expected: { grandTotal: /* calculated */ }
```

### Integration Tests
- Create estimate → add areas → calculate → verify totals match
- Link estimate to bid → verify bond cost included
- Change wage county → recalculate → verify rates updated
- Product matching with spec requirements → correct products returned

### Seed Data
```typescript
const sampleProducts = [
  {
    manufacturer: 'Shaw',
    productLine: 'Sustain',
    name: 'Shaw Sustain 20mil LVT',
    type: 'lvt',
    specifications: { wearLayerMils: 20, totalThicknessMm: 4.5, moistureResistant: true, adaCompliant: true, fireRating: 'Class 1' },
    pricing: { listPricePerSqFt: 5.25, ourCostPerSqFt: 3.85, lastUpdated: '2024-01-15' },
    typicalWasteFactor: 10,
    typicalLaborRate: 2.50,
  },
  {
    manufacturer: 'Mohawk',
    productLine: 'EcoFlex',
    name: 'Mohawk EcoFlex Carpet Tile',
    type: 'carpet_tile',
    specifications: { moistureResistant: false, adaCompliant: true, fireRating: 'Class 1' },
    pricing: { listPricePerSqFt: 3.75, ourCostPerSqFt: 2.80, lastUpdated: '2024-02-01' },
    typicalWasteFactor: 5,
    typicalLaborRate: 1.75,
  },
];
```

## Error Handling

| Failure | Handling |
|---------|----------|
| Prevailing wage not found for county | Show warning, allow estimate to continue with manual rate entry. Log gap for admin. |
| Product pricing expired (>90 days) | Show yellow warning banner: "Pricing last updated X days ago — verify with manufacturer" |
| PDF upload fails | Return error with size/type guidance. Max 50MB, PDF only. |
| Calculation produces negative margin | Block submission if margin < 0%. Alert owner if margin < 5%. |
| Floor plan too large to render | Downsample to max 4000px width for markup. Keep original for reference. |

## UI/UX Requirements

### Estimate Builder (Desktop Primary)
- **Left panel:** Area list with quick-add, drag to reorder
- **Center:** Floor plan viewer with markup tools (polygon draw, measure, label)
- **Right panel:** Active area details (product, sqft, waste, materials, labor)
- **Bottom bar:** Running totals (materials, labor, overhead, profit, grand total)
- **Top bar:** Estimate status, version number, "Calculate" button, export actions

### Product Selector (Modal)
- Filter by type, manufacturer, specs (wear layer, fire rating, etc.)
- Side-by-side comparison (up to 3 products)
- Show: name, specs, our cost, list price, installed cost per sqft, photo/swatch
- "Select for area" button

### Proposal Preview
- Formatted document view matching government submission requirements
- Sections: cover page, company qualifications, scope of work, pricing schedule, alternates, schedule, references
- AI-generated narrative sections with edit capability
- Export as PDF

### Quick Budget Calculator (Simple Form)
- Input: total sqft, product type, county, prevailing wage yes/no
- Output: rough range ($X - $Y) based on historical rates
- "Convert to full estimate" button

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| Prevailing wage DB (spec/03) | Labor rate lookup | Read | On estimate creation/recalc |
| Product catalog | Material pricing and specs | Read | On product selection |
| Bid module (spec/07) | Estimate attached to bid | Bidirectional | On bid assignment |
| Project module (spec/08) | Actuals vs. estimate comparison | Read | Post-project |
| AI (Bedrock) | Proposal writing, spec matching | Outbound | On-demand |

## Performance Requirements

- Full estimate calculation: < 500ms (even with 50 areas)
- Product matching (filter 500 products by specs): < 200ms
- PDF export: < 10 seconds
- Floor plan render at markup resolution: < 2 seconds
- Quick budget calculation: < 100ms

## Resolved Design Decisions

- **Digital takeoff:** Build our own simplified version (see `spec/21-estimate-builder-ui.md`). PlanSwift/On-Screen are $1500+/seat and require Windows. Our polygon tool handles 90% of flooring estimating needs (rectangular rooms with simple shapes).
- **Floor plan markup detail:** Simple polygon drawing only. No auto-detect of room boundaries (unreliable with varying PDF quality). Users click to draw polygon vertices, system calculates area from the polygon. See `spec/21-estimate-builder-ui.md` for full UX.
- **Pricing storage:** Store current costs (`ourCostPerSqFt`, `listPricePerSqFt`) plus `lastUpdated` date on the Product entity. Alert when pricing is >90 days stale. Don't store full pricing agreement terms — those are PDF attachments.
- **Bid alternates:** Supported within a single estimate via the Alternate system. Base estimate + named alternates (Alt 1, Alt 2...). Each alternate can be a full clone with modifications or an additive scope addition. See `spec/21-estimate-builder-ui.md`.

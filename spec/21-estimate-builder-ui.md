# Estimate Builder UI

## Vision & Purpose

The estimate builder is the most complex UI in Salescraft — it's where estimators turn floor plans into precise cost calculations. The workflow mirrors how experienced estimators already work: look at a plan, identify rooms/areas, measure them, assign products, and calculate totals with waste, labor, and overhead factored in. The digital version adds speed (auto-calculation), accuracy (no math errors), and auditability (every number traceable).

## Workflow Overview

```
Upload Floor Plan → Set Scale → Draw Rooms → Assign Products → Review Totals → Submit
     (1)              (2)           (3)            (4)              (5)         (6)
```

Each step is a phase in the left sidebar. Users can jump between phases freely (non-linear), but the estimate cannot be submitted until all required data is present.

## Screen Layout

```
┌────────────────────────────────────────────────────────────────────────────┐
│ ← Bids  │  Lincoln Elementary Cafeteria Estimate  │  v3 Draft  │ [Save] │
├──────────┼─────────────────────────────────────────────────────────────────┤
│ PHASES   │                                                                  │
│          │  ┌────────────────────────────────────────────────────────────┐ │
│ ✓ Upload │  │                                                            │ │
│ ✓ Scale  │  │           Floor Plan Viewport                              │ │
│ ● Rooms  │  │           (zoom/pan/draw)                                  │ │
│ ○ Products│  │                                                            │ │
│ ○ Review │  │                                                            │ │
│          │  │                                                            │ │
│ ─────── │  └────────────────────────────────────────────────────────────┘ │
│ AREAS    │  ┌────────────────────────────────────────────────────────────┐ │
│          │  │ Line Items / Details Panel                                  │ │
│ Cafeteria│  │ (context-dependent on selected area)                        │ │
│ Kitchen  │  │                                                            │ │
│ Hallway A│  └────────────────────────────────────────────────────────────┘ │
│ Storage  │                                                                  │
│          │                                                                  │
│ [+ Area] │                                                                  │
├──────────┼─────────────────────────────────────────────────────────────────┤
│ TOTALS   │  Materials: $24,500  Labor: $18,200  Total: $52,340             │
└──────────┴─────────────────────────────────────────────────────────────────┘
```

## Phase 1: Upload Floor Plan

### Upload Interface

```
┌──────────────────────────────────────────────────────────────┐
│                                                               │
│    ┌─────────────────────────────────────────────────┐       │
│    │                                                  │       │
│    │     📄 Drop PDF floor plan here                 │       │
│    │        or click to browse                       │       │
│    │                                                  │       │
│    │     Supported: PDF, PNG, JPG (max 50MB)         │       │
│    └─────────────────────────────────────────────────┘       │
│                                                               │
│    Or select from bid documents:                              │
│    ┌─────────────────────────────────────────────────┐       │
│    │ Floor_Plans_Lincoln_Elem.pdf         12 pages   │       │
│    │ Architectural_Drawings.pdf           48 pages   │       │
│    └─────────────────────────────────────────────────┘       │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

**Behavior:**
- Drag-and-drop or file picker
- If bid has attached documents, show them for quick selection
- Multi-page PDFs: page selector to choose the relevant floor plan page
- Image is rendered into the viewport after upload

## Phase 2: Set Scale

### Scale Calibration

```
┌──────────────────────────────────────────────────────────────┐
│ Set Scale: Draw a line between two known points              │
│                                                               │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │                                                          │ │
│ │     [Floor plan with a drawn red line segment]           │ │
│ │     ←─────────── 45 ft ──────────────→                   │ │
│ │                                                          │ │
│ └──────────────────────────────────────────────────────────┘ │
│                                                               │
│ This line represents: [ 45 ] feet                            │
│                                                               │
│ Calculated scale: 1 pixel = 0.125 ft (1" = 8 ft)           │
│                                                               │
│ [  Confirm Scale  ]                                          │
│                                                               │
│ Tip: Use a wall dimension from the architectural drawings.   │
└──────────────────────────────────────────────────────────────┘
```

**Behavior:**
- User clicks two points on the floor plan to draw a reference line
- Enter the known real-world distance between those points
- System calculates pixels-per-foot ratio
- Ratio is saved and applied to all subsequent area calculations
- Can be re-calibrated at any time (recalculates all areas)

## Phase 3: Draw Rooms / Areas

### Polygon Drawing Tool

```
┌──────────────────────────────────────────────────────────────┐
│ Toolbar: [Select] [Draw] [Edit] [Delete]  │ Snap: [Grid ✓]  │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                                                         │ │
│  │   ┌──────────────────────────┐                         │ │
│  │   │      Cafeteria           │  ← named polygon        │ │
│  │   │      2,400 sqft          │                         │ │
│  │   └──────────────────────────┘                         │ │
│  │                        ┌──────────┐                    │ │
│  │   ┌──────┐            │ Kitchen  │                    │ │
│  │   │ Stor │            │ 800 sqft │                    │ │
│  │   │ 120  │            └──────────┘                    │ │
│  │   └──────┘                                            │ │
│  │                                                         │ │
│  │         •────•────•  ← user drawing in progress         │ │
│  │         |         (click to add point, close on first)  │ │
│  │                                                         │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  New area name: [ Hallway A ]  [  Create  ]                  │
└──────────────────────────────────────────────────────────────┘
```

**Drawing interaction:**
1. Select "Draw" tool
2. Click to place first point
3. Click to place subsequent points (straight lines between them)
4. Click on first point (or double-click) to close the polygon
5. Enter area name in prompt
6. System auto-calculates square footage from polygon + scale

**Edit interaction:**
- Select "Edit" tool
- Click a polygon to select it
- Drag vertices to adjust shape
- Square footage recalculates in real-time
- Right-click or long-press for context menu: Rename, Delete, Duplicate

**Display:**
- Each polygon shows: name + calculated sqft (semi-transparent fill)
- Colors differentiate areas (auto-assigned palette)
- Selected polygon highlighted with handles on vertices

### Manual Area Entry (No Floor Plan)

For estimates without floor plans (quick budget estimates):

```
┌──────────────────────────────────────────────────────────────┐
│ [+ Add Area Manually]                                        │
│                                                               │
│ Area Name: [ Cafeteria              ]                        │
│ Square Feet: [ 2400 ]                                        │
│ Notes: [ Main dining area, 30x80 approx ]                    │
│                                                               │
│ [  Add  ]                                                    │
└──────────────────────────────────────────────────────────────┘
```

## Phase 4: Assign Products

### Product Assignment per Area

```
┌──────────────────────────────────────────────────────────────┐
│ Selected: Cafeteria (2,400 sqft)                             │
├──────────────────────────────────────────────────────────────┤
│ PRODUCT                                                       │
│ [🔍 Search products...                               ]       │
│                                                               │
│ Results:                                                      │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ Shaw Sustain LVT - Coastal Oak (6x48 Plank)             │ │
│ │ 20 mil wear layer │ Glue-down │ $3.45/sqft              │ │
│ │ Waste factor: 10% │ FloorScore certified                │ │
│ │ [  Select  ]                                            │ │
│ ├──────────────────────────────────────────────────────────┤ │
│ │ Mohawk Group Hard Surface - River Run (12x24 Tile)      │ │
│ │ 28 mil wear layer │ Click-lock │ $4.10/sqft             │ │
│ │ Waste factor: 8% │ GreenGuard Gold                      │ │
│ │ [  Select  ]                                            │ │
│ └──────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│ CALCULATION PREVIEW (after selection)                         │
│                                                               │
│ Area: 2,400 sqft                                             │
│ Waste factor: 10%                                            │
│ Material needed: 2,640 sqft                                  │
│ Material cost: 2,640 × $3.45 = $9,108.00                    │
│                                                               │
│ Labor rate: $2.50/sqft (prevailing wage: Riverside County)   │
│ Labor cost: 2,400 × $2.50 = $6,000.00                       │
│                                                               │
│ Additional materials:                                         │
│ ┌────────────────┬─────┬──────┬────────┬──────────────────┐ │
│ │ Item           │ Qty │ Unit │ Cost   │ Total            │ │
│ ├────────────────┼─────┼──────┼────────┼──────────────────┤ │
│ │ Adhesive       │ 16  │ gal  │ $45.00 │ $720.00          │ │
│ │ Transition str │ 12  │ each │ $35.00 │ $420.00          │ │
│ │ Base cove      │ 280 │ lf   │ $2.25  │ $630.00          │ │
│ │ Moisture test  │ 2400│ sqft │ $0.25  │ $600.00          │ │
│ └────────────────┴─────┴──────┴────────┴──────────────────┘ │
│ [+ Add Line Item]                                            │
│                                                               │
│ Area subtotal: $17,478.00                                    │
└──────────────────────────────────────────────────────────────┘
```

**Behavior:**
- Product search: by name, manufacturer, type, or SKU
- On product selection, system auto-populates:
  - Waste factor (from product defaults, editable)
  - Labor rate (base + prevailing wage lookup by county)
  - Additional materials (adhesive, transitions, base cove — auto-calculated from area sqft/perimeter)
- All values are editable — auto-calculated values are starting points
- Additional line items can be added manually
- Perimeter auto-estimated as `4 * sqrt(sqft)` unless polygon provides actual perimeter

### Waste Factor Override

```
┌──────────────────────────────────────────────────┐
│ Waste Factor: [10]%  (default for LVT)           │
│                                                   │
│ Adjust for:                                       │
│ ○ Standard rooms (default)                        │
│ ● Complex layout (add 2-3%)        → suggests 12 │
│ ○ Large open area (reduce 2-3%)    → suggests 8  │
│ ○ Custom: [  ]%                                   │
└──────────────────────────────────────────────────┘
```

## Phase 5: Review Totals

### Summary View

```
┌──────────────────────────────────────────────────────────────┐
│ ESTIMATE SUMMARY - Lincoln Elementary Cafeteria               │
│ Bid: IFB #2026-FL-003  │  Riverside USD                      │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│ AREAS                                                         │
│ ┌──────────┬────────┬────────────┬──────────┬──────────────┐ │
│ │ Area     │ Sq Ft  │ Product    │ Material │ Labor        │ │
│ ├──────────┼────────┼────────────┼──────────┼──────────────┤ │
│ │Cafeteria │ 2,400  │ Shaw LVT   │ $9,108   │ $6,000       │ │
│ │Kitchen   │   800  │ Sheet Vinyl│ $2,640   │ $2,400       │ │
│ │Hallway A │ 1,200  │ Shaw LVT   │ $4,554   │ $3,000       │ │
│ │Storage   │   120  │ VCT        │   $216   │   $180       │ │
│ ├──────────┼────────┼────────────┼──────────┼──────────────┤ │
│ │ TOTAL    │ 4,520  │            │$16,518   │$11,580       │ │
│ └──────────┴────────┴────────────┴──────────┴──────────────┘ │
│                                                               │
│ COST BREAKDOWN                                                │
│ ┌───────────────────────────────────────────┬──────────────┐ │
│ │ Materials (product + additional)          │   $19,888.00 │ │
│ │ Labor (prevailing wage)                   │   $11,580.00 │ │
│ │ Equipment                                 │      $800.00 │ │
│ │ Subcontractor (moisture testing)          │    $1,130.00 │ │
│ ├───────────────────────────────────────────┼──────────────┤ │
│ │ Subtotal                                  │   $33,398.00 │ │
│ ├───────────────────────────────────────────┼──────────────┤ │
│ │ Overhead (12%)                            │    $4,007.76 │ │
│ │ Profit (18%)                              │    $6,011.64 │ │
│ │ Bond (2.5% of total)                      │    $1,085.44 │ │
│ ├───────────────────────────────────────────┼──────────────┤ │
│ │ TOTAL BID AMOUNT                          │   $44,502.84 │ │
│ └───────────────────────────────────────────┴──────────────┘ │
│                                                               │
│ Overhead %: [12]   Profit %: [18]   Bond %: [2.5]           │
│                                                               │
│ ⚠️ Profit margin 18% — within normal range                    │
│                                                               │
│ [Export PDF]  [Save Draft]  [Submit for Review]              │
└──────────────────────────────────────────────────────────────┘
```

**Behavior:**
- All percentage fields (overhead, profit, bond) are editable with real-time recalculation
- Profit below 15% shows warning and requires owner approval on submission
- "Submit for Review" changes estimate status to `ready_for_review`
- "Export PDF" generates a formatted document matching standard bid proposal format
- Running total visible in bottom bar at all times during editing

### Prevailing Wage Display

```
┌──────────────────────────────────────────────────────────────┐
│ PREVAILING WAGE: Riverside County                             │
│ Trade: Floor Layer (Group 1)                                  │
│ Base rate: $48.50/hr │ Fringe: $22.30/hr │ Total: $70.80/hr │
│ Effective: Jul 1, 2025 - Jun 30, 2026                        │
│                                                               │
│ Converted to per-sqft:                                        │
│ LVT/LVP: $2.50/sqft (based on 500 sqft/crew-day)            │
│ VCT: $1.50/sqft (based on 800 sqft/crew-day)                │
│ Sheet: $3.00/sqft (based on 400 sqft/crew-day)               │
└──────────────────────────────────────────────────────────────┘
```

## Phase 6: Alternates

### Alternate Management

```
┌──────────────────────────────────────────────────────────────┐
│ ALTERNATES                                                    │
│                                                               │
│ Base Estimate: $44,502.84                                    │
│                                                               │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ Alt 1: Premium Product Upgrade                           │ │
│ │ Replace Shaw Sustain with Mohawk Group (all areas)       │ │
│ │ Delta: +$4,200.00  │  Total: $48,702.84                 │ │
│ │ [Edit] [Delete]                                         │ │
│ └──────────────────────────────────────────────────────────┘ │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ Alt 2: Add Hallway B                                     │ │
│ │ Additional 800 sqft hallway not in base scope            │ │
│ │ Delta: +$8,100.00  │  Total: $52,602.84                 │ │
│ │ [Edit] [Delete]                                         │ │
│ └──────────────────────────────────────────────────────────┘ │
│                                                               │
│ [+ Create Alternate]                                         │
│                                                               │
│ Create alternate by:                                          │
│ ○ Clone base estimate and modify                             │
│ ○ Add new areas only (additive alternate)                    │
│ ○ Substitute product in selected areas                       │
└──────────────────────────────────────────────────────────────┘
```

**Behavior:**
- "Clone base" creates a full copy that can be independently edited
- "Add areas" adds new rooms/areas on top of base
- "Substitute product" swaps product in specified areas and recalculates
- Alternates are numbered sequentially (Alt 1, Alt 2...)
- Each alternate shows the delta from base and absolute total
- Alternates included in PDF export as separate sections

## PDF Export Format

The generated PDF matches standard bid proposal format:

```
Page 1: Cover sheet (company logo, bid info, total)
Page 2: Scope summary (areas, products, specifications)
Page 3+: Detailed line items by area
Last page: Assumptions, exclusions, validity period (90 days)
Alternates: Each on its own page
```

## Business Rules

- **BR-EST-UI-001:** Scale must be set before areas can be drawn (Phase 2 before Phase 3)
- **BR-EST-UI-002:** At least one area with an assigned product required to submit
- **BR-EST-UI-003:** All calculations use `decimal.js` for financial precision (no floating point)
- **BR-EST-UI-004:** Profit percentage below 15% triggers approval warning (yellow highlight)
- **BR-EST-UI-005:** Profit percentage below 10% blocks submission until owner approves
- **BR-EST-UI-006:** Auto-save every 30 seconds (debounced after changes)
- **BR-EST-UI-007:** Version history: each save creates a revision; "Submit for Review" locks the version
- **BR-EST-UI-008:** Prevailing wage auto-populated from bid's county; editable with audit log
- **BR-EST-UI-009:** Product pricing staleness: warn if product cost >90 days old

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Escape | Deselect / cancel drawing |
| Delete/Backspace | Delete selected polygon |
| Ctrl+Z | Undo last action |
| Ctrl+Shift+Z | Redo |
| Ctrl+S | Save draft |
| +/- | Zoom in/out |
| Space+drag | Pan viewport |
| D | Switch to Draw tool |
| V | Switch to Select tool |

## Implementation Guide

### File Locations
- `apps/web/src/components/estimates/` — Estimate builder components
  - `EstimateBuilder.tsx` — Main container with phase navigation
  - `FloorPlanViewer.tsx` — Canvas/SVG viewport with zoom/pan
  - `PolygonDrawTool.tsx` — Drawing interaction handler
  - `ProductSearch.tsx` — Product search and selection
  - `LineItemTable.tsx` — Editable line item grid
  - `EstimateSummary.tsx` — Review totals view
  - `AlternateManager.tsx` — Alternate create/edit
  - `ScaleCalibrator.tsx` — Scale setting tool
- `apps/web/src/lib/geometry.ts` — Polygon area calculation, perimeter
- `apps/web/src/lib/estimate-calc.ts` — Financial calculations (uses decimal.js)

### Key Dependencies
- `decimal.js` — Financial math without floating point errors
- `fabric.js` or `konva` — Canvas library for floor plan viewport + polygon drawing
- `react-pdf` — PDF rendering for uploaded floor plans
- `@react-pdf/renderer` — PDF generation for export
- `zustand` — Local state management for the builder (complex undo/redo state)

### Implementation Order
1. Floor plan upload and PDF page viewer
2. Scale calibration tool
3. Polygon drawing tool (create, edit, delete)
4. Area calculation from polygons
5. Product search and assignment
6. Auto-calculation engine (waste, labor, additional materials)
7. Line item table with inline editing
8. Summary view with overhead/profit/bond
9. Alternate management
10. PDF export
11. Auto-save and version history
12. Undo/redo system

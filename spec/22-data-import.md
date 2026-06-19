# Data Import

## Vision & Purpose

New Salescraft customers have existing data — contacts in spreadsheets, organizations in old CRMs, historical bid records in files. The import system provides a guided wizard to map CSV columns to Salescraft fields, validate data quality, detect duplicates, and import records in bulk. This is critical for onboarding — a system with no data provides no value.

## Supported Import Types

| Entity | Required Fields | Optional Fields |
|--------|----------------|-----------------|
| Organizations | name, type | website, phone, address, fiscal year, purchasing threshold, tags |
| Contacts | firstName, lastName | email, phone, title, role, organizationName, decisionAuthority, source, tags |
| Facilities | name, organizationName | type, address, yearBuilt, totalSqFt, conditionRating |
| Products | manufacturer, name, type | sku, pricing, specifications |
| Historical Bids | title, organizationName, submissionDeadline | bidNumber, result, submittedAmount, winningAmount |

## Import Wizard Flow

### Step 1: Upload File

```
┌──────────────────────────────────────────────────────────────┐
│ Import Data                                                   │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│ What are you importing?                                       │
│ ○ Organizations                                               │
│ ● Contacts                                                    │
│ ○ Facilities                                                  │
│ ○ Products                                                    │
│ ○ Historical Bids                                             │
│                                                               │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │     📄 Drop CSV file here or click to browse             │ │
│ │        Supported: .csv, .xlsx (max 10MB, 10,000 rows)    │ │
│ └──────────────────────────────────────────────────────────┘ │
│                                                               │
│ Download template: [contacts_template.csv]                    │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

**Behavior:**
- Templates downloadable for each entity type (pre-filled headers)
- File parsed client-side for preview; server validates on import
- Max 10,000 rows per import (larger datasets split into batches)
- XLSX support via SheetJS (converted to CSV internally)

### Step 2: Column Mapping

```
┌──────────────────────────────────────────────────────────────┐
│ Map Columns                                     Step 2 of 4   │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│ Your File Column      →    Salescraft Field                   │
│ ─────────────────────────────────────────────                 │
│ "First Name"          →    [firstName        ▾]  ✓ Required  │
│ "Last Name"           →    [lastName         ▾]  ✓ Required  │
│ "Email Address"       →    [email            ▾]              │
│ "Work Phone"          →    [phone            ▾]              │
│ "Title/Position"      →    [title            ▾]              │
│ "Company"             →    [organizationName ▾]  (lookup)    │
│ "Notes"               →    [notes            ▾]              │
│ "Category"            →    [— Skip —         ▾]              │
│ "Date Added"          →    [— Skip —         ▾]              │
│                                                               │
│ PREVIEW (first 3 rows):                                       │
│ ┌─────────┬──────────┬─────────────────┬────────────────────┐│
│ │firstName│lastName  │email            │organizationName    ││
│ ├─────────┼──────────┼─────────────────┼────────────────────┤│
│ │Mike     │Johnson   │mjohn@rusd.edu   │Riverside USD       ││
│ │Sarah    │Williams  │swilliams@rusd.edu│Riverside USD      ││
│ │Tom      │Garcia    │tgarcia@cnusd.edu│Corona-Norco USD    ││
│ └─────────┴──────────┴─────────────────┴────────────────────┘│
│                                                               │
│ [← Back]                                    [Next: Validate →]│
└──────────────────────────────────────────────────────────────┘
```

**Behavior:**
- Auto-mapping: system attempts to match column headers to fields (fuzzy match on name)
- "Skip" option for columns that don't map to any field
- Organization lookup: if `organizationName` is mapped, system will attempt to match to existing organizations
- Preview shows first 3 rows with mapped field names
- Required fields must be mapped before proceeding

### Step 3: Validate & Review

```
┌──────────────────────────────────────────────────────────────┐
│ Validation Results                              Step 3 of 4   │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│ 📊 Summary                                                    │
│ Total rows: 150                                               │
│ ✓ Valid: 142                                                  │
│ ⚠️ Warnings: 5                                                │
│ ❌ Errors: 3                                                   │
│                                                               │
│ DUPLICATES DETECTED (5)                                       │
│ ┌─────────────────────────────────────────────────────────┐  │
│ │ Row 12: "Mike Johnson" mjohn@rusd.edu                   │  │
│ │ Matches existing: Mike Johnson (Riverside USD)           │  │
│ │ Strategy: ○ Skip  ● Update existing  ○ Create new       │  │
│ ├─────────────────────────────────────────────────────────┤  │
│ │ Row 45: "Sarah Williams" swilliams@rusd.edu             │  │
│ │ Matches existing: Sarah Williams (Riverside USD)         │  │
│ │ Strategy: ○ Skip  ● Update existing  ○ Create new       │  │
│ └─────────────────────────────────────────────────────────┘  │
│                                                               │
│ ERRORS (must fix or skip)                                     │
│ ┌─────────────────────────────────────────────────────────┐  │
│ │ Row 67: Missing required field "lastName"                │  │
│ │ Row 89: Invalid email format "not-an-email"              │  │
│ │ Row 134: firstName exceeds 100 characters                │  │
│ │                                                         │  │
│ │ [Skip error rows]  [Download errors CSV for correction] │  │
│ └─────────────────────────────────────────────────────────┘  │
│                                                               │
│ WARNINGS                                                      │
│ ┌─────────────────────────────────────────────────────────┐  │
│ │ 3 rows: organizationName not found — will be left blank │  │
│ │ 2 rows: phone number format non-standard (imported as-is)│  │
│ └─────────────────────────────────────────────────────────┘  │
│                                                               │
│ Default duplicate strategy: [Update existing ▾]               │
│                                                               │
│ [← Back]                                    [Next: Import →]  │
└──────────────────────────────────────────────────────────────┘
```

**Validation rules:**
- Required fields present and non-empty
- Email format valid (if provided)
- String lengths within limits
- Enum values valid (role, type, decisionAuthority)
- Organization name lookup (warn if no match, don't error)

**Duplicate detection:**
- Match by email address (exact match)
- Match by name + organization (fuzzy: Levenshtein distance < 3)
- User chooses strategy per-duplicate or sets a default for all

### Step 4: Import & Results

```
┌──────────────────────────────────────────────────────────────┐
│ Import Complete                                 Step 4 of 4   │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│ ████████████████████████████████ 100%                        │
│                                                               │
│ Results:                                                      │
│ ✓ 140 contacts created                                       │
│ ✓ 5 contacts updated (duplicates)                            │
│ ⚠️ 2 contacts skipped (duplicates, "skip" chosen)            │
│ ❌ 3 rows skipped (validation errors)                         │
│                                                               │
│ Would you like to:                                            │
│ [View imported contacts]  [Download skipped rows]  [Done]     │
│                                                               │
│ ─────────────────────────────────────────                     │
│ Import logged: Jun 18, 2026 by Patricia Johnson               │
│ 150 rows processed in 4.2 seconds                            │
└──────────────────────────────────────────────────────────────┘
```

## Data Model

### ImportJob

```typescript
interface ImportJob {
  id: string;
  userId: string;               // FK → User (who ran the import)
  entityType: 'organization' | 'contact' | 'facility' | 'product' | 'bid';
  fileName: string;
  fileUrl: string;              // S3 location of original file
  rowCount: number;             // Total rows in file
  status: 'validating' | 'ready' | 'importing' | 'complete' | 'failed';
  mapping: Record<string, string>;  // fileColumn → salescraft field
  duplicateStrategy: 'skip' | 'update' | 'create_new';
  results: {
    created: number;
    updated: number;
    skipped: number;
    errors: number;
  };
  errors?: ImportError[];
  startedAt?: DateTime;
  completedAt?: DateTime;
  createdAt: DateTime;
}

interface ImportError {
  row: number;
  field?: string;
  value?: string;
  message: string;
  severity: 'error' | 'warning';
}
```

## API Endpoints

```typescript
// Import wizard
POST   /api/v1/import/upload              // Upload CSV, get preview
// Body: multipart/form-data (file)
// Response: { jobId, columns[], preview: first5rows[], rowCount }

POST   /api/v1/import/:jobId/map          // Set column mapping
// Body: { mapping: { fileCol: salesCraftField } }
// Response: { valid: boolean }

POST   /api/v1/import/:jobId/validate     // Run validation pass
// Response: { valid, errors[], warnings[], duplicates[] }

POST   /api/v1/import/:jobId/execute      // Execute import
// Body: { duplicateStrategy, skipRowIds? }
// Response: { status: 'importing' }

GET    /api/v1/import/:jobId/status       // Poll for completion
// Response: { status, progress, results? }

// Import history
GET    /api/v1/import/history             // List past imports
GET    /api/v1/import/:jobId/errors       // Download error rows as CSV

// Templates
GET    /api/v1/import/templates/:entity   // Download CSV template
```

## Business Rules

- **BR-IMP-001:** Only owner and admin roles can perform imports
- **BR-IMP-002:** Max 10,000 rows per import job; larger files must be split
- **BR-IMP-003:** Organization name lookup is case-insensitive and trims whitespace
- **BR-IMP-004:** Duplicate detection runs on email (exact) and name+org (fuzzy)
- **BR-IMP-005:** Import is transactional per-row (a row failure doesn't roll back others)
- **BR-IMP-006:** All imported records have `source: 'import'` set automatically
- **BR-IMP-007:** Import history retained for 1 year; original files retained for 90 days
- **BR-IMP-008:** Contact assignment: imported contacts are unassigned by default (owner assigns after)
- **BR-IMP-009:** Relationship score starts at 0 for all imported contacts

## Implementation Guide

### File Locations
- `apps/web/src/components/import/` — Import wizard UI
  - `ImportWizard.tsx` — Multi-step wizard container
  - `FileUpload.tsx` — Drag-and-drop upload
  - `ColumnMapper.tsx` — Mapping interface
  - `ValidationResults.tsx` — Error/warning display
  - `ImportProgress.tsx` — Progress bar and results
- `apps/api/src/modules/import/` — Import processing
  - `import.routes.ts` — Endpoints
  - `import.service.ts` — Orchestration
  - `import.validator.ts` — Row validation
  - `import.matcher.ts` — Duplicate detection

### Key Dependencies
- `papaparse` — CSV parsing (client and server)
- `xlsx` — Excel file parsing
- `fastest-levenshtein` — Fuzzy string matching for duplicates

### Implementation Order
1. File upload endpoint with S3 storage
2. CSV parsing and column detection
3. Column mapping UI with auto-suggestions
4. Validation engine (per-row checks)
5. Duplicate detection (email exact + name fuzzy)
6. Import execution with progress tracking
7. Import history and error export
8. Template download endpoints

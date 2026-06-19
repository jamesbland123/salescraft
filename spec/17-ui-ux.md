# UI/UX Design System

## Vision & Purpose

Salescraft's interface is built for people who spend their days in the field, on the phone, and in meetings — not staring at software. The UI must be information-dense without being overwhelming, fast to navigate, and optimized for the most common actions: log a call, check a contact's history, review a bid deadline, update a daily log.

The design system prioritizes: **clarity** (instant comprehension of status), **speed** (common actions require minimal clicks), and **context** (relevant information surfaces automatically).

## Design Principles

1. **Dense but scannable** — Show lots of data, but use hierarchy (size, weight, color) so the eye finds what matters
2. **Action-oriented** — Every view has a clear primary action. Don't make users hunt.
3. **Context-rich** — Relationship briefing cards, score badges, deadline countdowns appear where they're needed
4. **Role-adapted** — Each role sees a different home screen and navigation optimized for their workflow
5. **Mobile-first for field, desktop-first for office** — Field roles get simplified mobile views; office roles get data-rich desktop views

## Technology Choices

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Component Library | shadcn/ui | Accessible, Tailwind-native, fully customizable, no vendor lock-in |
| Styling | Tailwind CSS 3.x | Utility-first, consistent, rapid iteration |
| Icons | Lucide React | Clean, consistent, comprehensive |
| Charts | Recharts | Pipeline charts, forecasting, dashboards |
| Tables | TanStack Table | Headless, virtualizable for large datasets |
| Maps | Mapbox GL JS | Territory visualization, facility pins |
| Date Picker | react-day-picker | Lightweight, accessible |
| Toasts | sonner | Clean, stacking, auto-dismiss |
| Modals/Drawers | shadcn Dialog/Sheet | Consistent overlay patterns |
| Form Handling | React Hook Form + Zod | Performance + validation reuse |

## Layout Architecture

### Shell (Authenticated Layout)

```
┌──────────────────────────────────────────────────────────────┐
│  Logo  │ Search (⌘K)          │ Notifications │ User Avatar  │
├────────┼─────────────────────────────────────────────────────┤
│        │                                                      │
│  Nav   │  Main Content Area                                   │
│        │                                                      │
│ Home   │  ┌────────────────────────────────────────────────┐ │
│ Contacts│  │                                                │ │
│ Pipeline│  │  Page content varies by route                  │ │
│ Bids   │  │                                                │ │
│ Projects│  │                                                │ │
│ Intel  │  │                                                │ │
│ Products│  │                                                │ │
│ Reports│  │                                                │ │
│        │  └────────────────────────────────────────────────┘ │
│ ─────  │                                                      │
│ Settings│                                                     │
└────────┴─────────────────────────────────────────────────────┘
```

- Sidebar: collapsible (icon-only mode), persists selection state
- Search: global command palette (⌘K) searching contacts, bids, projects, organizations
- Notifications: bell icon with unread count, dropdown showing recent
- Content: full-width with max-width constraint for readability

### Role-Based Navigation

| Role | Nav Items |
|------|-----------|
| Owner | Home, Contacts, Pipeline, Bids, Projects, Intelligence, Products, Reports, Users, Settings |
| Sales Manager | Home, Contacts, Pipeline, Bids, Intelligence, Reports, Team |
| Sales Rep | Home, Contacts, Pipeline, Bids, Intelligence |
| Estimator | Home, Bids (assigned), Estimates, Products |
| Project Manager | Home, Projects, Crew Schedule |
| Admin | Home, Contacts, Bids, Compliance, Reports |

### Home Screens (Role-Specific)

#### Sales Rep Home
```
┌─────────────────────────────────────────────────────────────┐
│ Good morning, Alex                                           │
├─────────────────────────────┬───────────────────────────────┤
│ TODAY'S PRIORITIES           │ UPCOMING                       │
│                             │                               │
│ 🔴 Bid due tomorrow:       │ Wed: Pre-bid meeting Lincoln  │
│    Lincoln Elementary IFB   │ Thu: Call with Mike (Fac Dir)  │
│ 🟡 Follow up overdue (3)   │ Fri: Bid deadline RFP #156    │
│ 🟢 New opportunity scored   │                               │
│    85 - Riverside City Hall │                               │
├─────────────────────────────┼───────────────────────────────┤
│ RELATIONSHIPS AT RISK        │ RECENT ACTIVITY               │
│                             │                               │
│ Mike Johnson - 45 days      │ Email from Sarah (today)      │
│ Sarah Williams - 38 days    │ Called Mike (yesterday)       │
│ Tom Garcia - 52 days        │ Site visit Riverside HS (Mon) │
│                             │                               │
├─────────────────────────────┴───────────────────────────────┤
│ PIPELINE SUMMARY                                             │
│ ████████░░░░░░░░ $1.2M active │ 3 bids pending │ 5 opps    │
└─────────────────────────────────────────────────────────────┘
```

#### Project Manager Home
```
┌─────────────────────────────────────────────────────────────┐
│ Active Projects (4)                                          │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐ ┌─────────────────┐ ┌───────────────┐ │
│ │ Lincoln Elem    │ │ City Hall       │ │ Riverside HS  │ │
│ │ ████████░░ 78%  │ │ ██░░░░░░ 15%   │ │ █████░░░ 55%  │ │
│ │ On schedule     │ │ Material order  │ │ 2 punch items │ │
│ │ Margin: 18%     │ │ Margin: 22%     │ │ Margin: 16%   │ │
│ └─────────────────┘ └─────────────────┘ └───────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ TODAY'S CREW                                                 │
│ Carlos, Marco, James → Lincoln Elementary                    │
│ Roberto, Luis → Riverside HS                                 │
├─────────────────────────────────────────────────────────────┤
│ NEEDS ATTENTION                                              │
│ ⚠️ Missing daily log: Carlos (yesterday)                    │
│ ⚠️ Material delivery delayed: City Hall (2 days)            │
│ 🔴 Punch list overdue: Riverside HS (3 items, 5 days)      │
└─────────────────────────────────────────────────────────────┘
```

## Key Screen Specifications

### Contact Detail View

```
┌─────────────────────────────────────────────────────────────┐
│ ← Contacts    Mike Johnson                                   │
│               Director of Facilities, Riverside USD           │
│               📧 mjohnson@rusd.edu  📱 951-555-0123          │
│                                                              │
│ Relationship: ████████░░ 72  ↑ improving                    │
│ Last contact: 5 days ago (phone call)                        │
├──────────────────────────────────────────────────────────────┤
│ [Timeline] [Interests] [Bids] [Projects] [Notes]             │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│ BRIEFING CARD (expand ▾)                                     │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ 🎣 Shared interest: Fishing (you both fish!)             │ │
│ │ 🎓 Daughter started at ASU this fall                     │ │
│ │ 🏀 Lakers fan — season opener next week                  │ │
│ │ 💬 Last time: asked about the boat motor repair          │ │
│ │                                                          │ │
│ │ Suggested starters:                                      │ │
│ │ • "How's your daughter settling in at ASU?"              │ │
│ │ • "Lakers are looking good this year — you catch the..?" │ │
│ │ • "Did you get that boat motor sorted out?"              │ │
│ └──────────────────────────────────────────────────────────┘ │
│                                                              │
│ TIMELINE                                                     │
│ ─────────────────────────────────                            │
│ Jun 10  📞 Phone call (12 min) - discussed cafeteria project │
│         Personal: mentioned fishing trip to Lake Powell       │
│ Jun 3   📧 Email - RE: Cafeteria Flooring Samples           │
│         "Liked the Shaw Sustain samples, wants budget number" │
│ May 28  🏢 Site visit - Lincoln Elementary cafeteria         │
│ May 15  📞 Phone call - check on summer projects            │
│                                                              │
│ [+ Log Interaction]  [+ Quick Note]  [📧 Send Email]        │
└─────────────────────────────────────────────────────────────┘
```

### Bid Pipeline View (Kanban)

```
┌─────────────────────────────────────────────────────────────┐
│ Bids Pipeline    [+ New Bid]  [Calendar View]  [List View]   │
├──────────┬──────────┬──────────┬──────────┬────────────────┤
│ Discovered│ Reviewing │ Preparing │ Submitted │ Awarded       │
│ (3)      │ (2)      │ (2)      │ (1)      │ (1 won)       │
├──────────┼──────────┼──────────┼──────────┼────────────────┤
│┌────────┐│┌────────┐│┌────────┐│┌────────┐│┌────────────┐ │
││Lincoln  ││City Hall ││Park Elem││Riverside ││ ✓ MLK      │ │
││Elem IFB ││RFP      ││Carpet   ││HS Floor  ││ Middle     │ │
││         ││         ││         ││         ││ Won: $420K  │ │
││$350K est││$800K est││$120K est││$250K est││             │ │
││Due: 3d  ││Due: 14d ││Due: 7d  ││Awaiting ││             │ │
││🔴 URGENT││Score: 78││Score: 65││award    ││             │ │
│└────────┘│└────────┘│└────────┘│└────────┘│└────────────┘ │
│┌────────┐│┌────────┐│┌────────┐│          │               │
││Lib reno ││Corona   ││...      ││          │               │
││$45K    ││District ││         ││          │               │
│└────────┘│└────────┘│└────────┘│          │               │
└──────────┴──────────┴──────────┴──────────┴────────────────┘
```

### Intelligence Dashboard

```
┌─────────────────────────────────────────────────────────────┐
│ Project Intelligence                    [Map View] [List]     │
├─────────────────────────────────────────────────────────────┤
│ HIGH SCORE OPPORTUNITIES (>70)                               │
│ ┌───┬──────────────────────┬──────┬──────┬───────┬────────┐│
│ │ # │ Opportunity          │Score │Value │Lead   │Status   ││
│ ├───┼──────────────────────┼──────┼──────┼───────┼────────┤│
│ │ 85│ Riverside City Hall  │  85  │$800K │9 mo   │Engaging ││
│ │ 78│ Corona-Norco USD Bond│  78  │$2.5M │12 mo  │Research ││
│ │ 72│ Lincoln Elem Cafe    │  72  │$180K │2 mo   │Bid Exp  ││
│ └───┴──────────────────────┴──────┴──────┴───────┴────────┘│
├─────────────────────────────────────────────────────────────┤
│ RECENT SIGNALS (unprocessed)                                 │
│ 🟡 Bond Measure H - Alvord USD - $180M facilities (passed)  │
│ 🟡 BoardDocs: Facilities Committee discussing flooring needs │
│ 🟢 BidNet: IFB for flooring at Riverside Public Library     │
│                                                              │
│ [Dismiss] [Investigate] [Convert to Opportunity]             │
├─────────────────────────────────────────────────────────────┤
│ SOURCE HEALTH                                                │
│ PlanetBids: 🟢 2h ago │ BidNet: 🟢 4h ago │ BoardDocs: 🟡 │
└─────────────────────────────────────────────────────────────┘
```

### Reverse Interest Match View

```
┌─────────────────────────────────────────────────────────────┐
│ Find Contacts by Interest                                    │
├───────────────────────┬─────────────────────────────────────┤
│ YOUR INTERESTS        │ MATCHING CONTACTS                    │
│                       │                                      │
│ [🎣 Fishing] ✓       │ Mike Johnson - Riverside USD          │
│ [🏀 Lakers]          │   "Fly fishing, Kern River"          │
│ [🍖 BBQ]             │   Score: 72 │ 5 days ago             │
│ [🎸 Guitar]          │   [Suggest Outreach]                 │
│                       │                                      │
│                       │ Tom Garcia - Corona-Norco USD        │
│                       │   "Bass fishing, Lake Havasu"        │
│                       │   Score: 45 │ 38 days ago ⚠️         │
│                       │   [Suggest Outreach]                 │
│ ──────────────────── │                                      │
│ Filter:               │ James Lee - City of Moreno Valley    │
│ ○ Exact match only    │   "Trout fishing, local lakes"       │
│ ● Include related     │   Score: 28 │ 60 days ago 🔴        │
│ ○ Broad category      │   [Suggest Outreach]                 │
│                       │                                      │
│                       │ ─── 5 more contacts ───              │
│                       │                                      │
│                       │ [Plan Group Outing]                  │
│                       │ [Send All a Personal Note]           │
└───────────────────────┴─────────────────────────────────────┘
```

## Component Library (Key Components)

### RelationshipScoreBadge
- Circular progress ring showing 0-100 score
- Color: green (70-100), yellow (40-69), red (0-39)
- Size variants: sm (inline), md (card), lg (header)
- Shows trend arrow (up/down/flat)

### BriefingCard
- Expandable card shown on contact pages and before meetings
- Sections: shared interests, recent events, personal notes, conversation starters
- Auto-appears for today's scheduled meetings (dismissable)
- AI-generated content marked with subtle "AI" indicator

### BidDeadlineCounter
- Shows days/hours until deadline
- Color intensifies as deadline approaches (green → yellow → red)
- Blinks when <24 hours
- Links to bid detail

### TimelineEntry
- Icon (type), direction arrow (in/out), date, content
- Expandable for full details
- Types have distinct icons: 📧 email, 📞 call, 🏢 meeting, 📝 note, 🎁 gesture

### OpportunityScoreBar
- Horizontal bar chart showing score breakdown by factor
- Color-coded factors
- Hover for factor explanation
- Clickable to see scoring details

### DataTable (powered by TanStack Table)
- Sortable columns (click header)
- Filterable (dropdown per column or search bar)
- Selectable rows (bulk actions)
- Virtualizable for large datasets (1000+ rows)
- Column resizing and reordering

### CommandPalette (⌘K)
- Global search: contacts, organizations, bids, projects
- Quick actions: "Log call", "Create contact", "New bid"
- Recent items
- Keyboard navigable

## Responsive Breakpoints

| Breakpoint | Screen | Behavior |
|-----------|--------|----------|
| < 640px | Mobile | Stack layout, hamburger nav, simplified views |
| 640-1024px | Tablet | Condensed sidebar, two-column where possible |
| 1024-1440px | Desktop | Full sidebar, main content area |
| > 1440px | Wide | Max-width container centered, extra whitespace |

## Color System

```css
/* Semantic colors */
--color-primary: #2563eb;        /* Blue - actions, links */
--color-success: #16a34a;        /* Green - won, healthy, complete */
--color-warning: #d97706;        /* Amber - attention, approaching */
--color-danger: #dc2626;         /* Red - overdue, at-risk, urgent */
--color-info: #0891b2;           /* Cyan - informational */

/* Relationship score colors */
--score-high: #16a34a;           /* 70-100 */
--score-medium: #d97706;         /* 40-69 */
--score-low: #dc2626;            /* 0-39 */

/* Background */
--bg-primary: #ffffff;
--bg-secondary: #f8fafc;         /* Slate 50 - cards, secondary areas */
--bg-sidebar: #1e293b;           /* Slate 800 - dark sidebar */

/* Text */
--text-primary: #0f172a;         /* Slate 900 */
--text-secondary: #475569;       /* Slate 600 */
--text-muted: #94a3b8;           /* Slate 400 */
```

## Typography

```css
/* Font: Inter (clean, highly readable, good tabular numbers) */
--font-sans: 'Inter', system-ui, sans-serif;

/* Scale */
--text-xs: 0.75rem;              /* 12px - badges, meta */
--text-sm: 0.875rem;             /* 14px - secondary text, table cells */
--text-base: 1rem;               /* 16px - body text */
--text-lg: 1.125rem;             /* 18px - section headers */
--text-xl: 1.25rem;              /* 20px - page subheaders */
--text-2xl: 1.5rem;              /* 24px - page titles */
--text-3xl: 1.875rem;            /* 30px - dashboard numbers */
```

## Loading / Empty / Error States

### Loading
- Skeleton screens (not spinners) for data-heavy views
- Progressive loading: show what's available immediately
- Optimistic updates for actions (show success immediately, revert on error)

### Empty States
- Helpful illustration + message + primary action
- Example: Empty contacts → "No contacts yet. Import from CSV or add your first contact."
- Example: No bids → "No active bids. Check the intelligence feed for new opportunities."

### Error States
- Inline errors for form validation (below the field, red text)
- Toast notifications for action failures (dismissable)
- Full-page error for route failures (with retry button)
- Offline banner for connectivity loss (top of page, persistent until restored)

## Accessibility

- All interactive elements keyboard-navigable
- ARIA labels on icon-only buttons
- Focus management in modals and drawers
- Color is never the ONLY indicator (always paired with icon or text)
- Minimum 4.5:1 contrast ratio for text
- Touch targets minimum 44px on mobile

## Implementation Guide

### File Locations
- `apps/web/src/components/ui/` — shadcn/ui base components
- `apps/web/src/components/layout/` — Shell, Sidebar, Header, CommandPalette
- `apps/web/src/components/common/` — DataTable, Timeline, ScoreBadge, etc.
- `apps/web/src/components/{domain}/` — Domain-specific components (BriefingCard, BidCard, etc.)
- `apps/web/src/styles/globals.css` — Tailwind config, CSS variables
- `apps/web/tailwind.config.ts` — Theme customization

### Implementation Order
1. Tailwind + shadcn/ui setup (button, input, card, dialog, sheet, toast)
2. Layout shell (sidebar, header, content area)
3. Command palette (⌘K global search)
4. DataTable component (reusable across all list views)
5. Timeline component (reusable for contacts, projects, bids)
6. Role-based home screens
7. Contact detail with briefing card
8. Bid pipeline (kanban)
9. Intelligence dashboard
10. Estimate builder (most complex UI)
11. Project detail views
12. Relationship/interest match views
13. Reports and charts

## Settings & Admin Screens

### Company Settings (Owner)

```
┌──────────────────────────────────────────────────────────────┐
│ Settings                                                      │
├──────────────────────────────────────────────────────────────┤
│ [Company] [Integrations] [Users] [Scoring] [Import]          │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│ COMPANY INFO                                                  │
│ Company Name: [ Acme Flooring, Inc.         ]                │
│ Email Domain: [ acmeflooring.com            ]                │
│ Phone:        [ (951) 555-0100              ]                │
│ Address:      [ 123 Main St, Riverside, CA  ]                │
│                                                               │
│ DEFAULTS                                                      │
│ Default Overhead %:  [ 12 ]                                  │
│ Default Profit %:    [ 18 ]                                  │
│ Min Profit % (alert): [ 15 ]                                 │
│ Fiscal Year Start:   [ July       ▾]                         │
│                                                               │
│ [  Save Changes  ]                                           │
└──────────────────────────────────────────────────────────────┘
```

### Integrations Settings

```
┌──────────────────────────────────────────────────────────────┐
│ Integrations                                                  │
├──────────────────────────────────────────────────────────────┤
│ EMAIL ACCOUNTS                                                │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ 📧 alex@acmeflooring.com (Gmail)            Connected ✓ │ │
│ │    Last synced: 2 min ago │ [Disconnect]                 │ │
│ ├──────────────────────────────────────────────────────────┤ │
│ │ 📧 tom@acmeflooring.com (Outlook)           Connected ✓ │ │
│ │    Last synced: 5 min ago │ [Disconnect]                 │ │
│ └──────────────────────────────────────────────────────────┘ │
│ [+ Connect Email Account]                                    │
│                                                               │
│ INTELLIGENCE SOURCES                                          │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ PlanetBids    │ 🟢 Active │ Last: 2h ago │ [Configure]  │ │
│ │ BidNet        │ 🟢 Active │ Last: 4h ago │ [Configure]  │ │
│ │ BoardDocs     │ 🟡 Warning│ Last: 26h ago│ [Configure]  │ │
│ └──────────────────────────────────────────────────────────┘ │
│                                                               │
│ FILE STORAGE                                                  │
│ S3 Bucket: salescraft-acme │ Region: us-west-2              │
│ Usage: 2.4 GB of unlimited                                   │
└──────────────────────────────────────────────────────────────┘
```

### User Management

```
┌──────────────────────────────────────────────────────────────┐
│ Users                                        [+ Invite User]  │
├──────────────────────────────────────────────────────────────┤
│ ┌────────┬───────────────────┬────────────────┬────────────┐ │
│ │ Name   │ Email             │ Role           │ Status     │ │
│ ├────────┼───────────────────┼────────────────┼────────────┤ │
│ │Patricia│ patricia@acme.com │ Owner          │ 🟢 Active  │ │
│ │Tom     │ tom@acme.com      │ Sales Manager  │ 🟢 Active  │ │
│ │Alex    │ alex@acme.com     │ Sales Rep      │ 🟢 Active  │ │
│ │Maria   │ maria@acme.com    │ Estimator      │ 🟢 Active  │ │
│ │Dave    │ dave@acme.com     │ Project Manager│ 🟢 Active  │ │
│ │Carlos  │ carlos@acme.com   │ Installer      │ 🟢 Active  │ │
│ │Lisa    │ lisa@acme.com     │ Admin          │ 🟢 Active  │ │
│ └────────┴───────────────────┴────────────────┴────────────┘ │
│                                                               │
│ PENDING INVITATIONS                                           │
│ ┌────────┬───────────────────┬────────────────┬────────────┐ │
│ │ Name   │ Email             │ Role           │ Expires    │ │
│ ├────────┼───────────────────┼────────────────┼────────────┤ │
│ │Roberto │ roberto@acme.com  │ Installer      │ Jun 25     │ │
│ └────────┴───────────────────┴────────────────┴────────────┘ │
│ [Resend]  [Revoke]                                           │
└──────────────────────────────────────────────────────────────┘
```

### Scoring Model Configuration

```
┌──────────────────────────────────────────────────────────────┐
│ Scoring Weights                                               │
├──────────────────────────────────────────────────────────────┤
│ OPPORTUNITY SCORE FACTORS                                     │
│ Building Age:       [20]% ████████░░                         │
│ Funding Confirmed:  [25]% ██████████░                        │
│ Relationship:       [20]% ████████░░                         │
│ Timeline:           [15]% ██████░░░░                         │
│ Size:               [10]% ████░░░░░░                         │
│ Competition:        [10]% ████░░░░░░                         │
│                     Total: 100% ✓                            │
│                                                               │
│ RELATIONSHIP SCORE FACTORS                                    │
│ Recency:            [30]%                                    │
│ Frequency:          [25]%                                    │
│ Depth:              [20]%                                    │
│ Reciprocity:        [15]%                                    │
│ Personal Knowledge: [10]%                                    │
│                     Total: 100% ✓                            │
│                                                               │
│ [  Save Weights  ]                                           │
└──────────────────────────────────────────────────────────────┘
```

### Notification Center

Slide-out panel from the bell icon:

```
┌────────────────────────────────────────┐
│ Notifications            [Mark All Read]│
├────────────────────────────────────────┤
│ TODAY                                   │
│ ┌────────────────────────────────────┐ │
│ │ 🔴 Bid due tomorrow: Lincoln Elem  │ │
│ │    IFB #2026-FL-003               │ │
│ │    2 hours ago                     │ │
│ └────────────────────────────────────┘ │
│ ┌────────────────────────────────────┐ │
│ │ 🟡 New signal: Bond Measure H     │ │
│ │    Alvord USD - $180M             │ │
│ │    5 hours ago                     │ │
│ └────────────────────────────────────┘ │
├────────────────────────────────────────┤
│ YESTERDAY                               │
│ ┌────────────────────────────────────┐ │
│ │ ⚠️ Relationship cooling:          │ │
│ │    Tom Garcia (52 days)            │ │
│ │    Yesterday at 6:00 AM            │ │
│ └────────────────────────────────────┘ │
│ ┌────────────────────────────────────┐ │
│ │ ✓ Compliance: Insurance renewed    │ │
│ │    Yesterday at 9:15 AM            │ │
│ └────────────────────────────────────┘ │
├────────────────────────────────────────┤
│ [View All Notifications]               │
└────────────────────────────────────────┘
```

- Grouped by date
- Unread items highlighted (bold or left border)
- Click notification → navigates to relevant entity
- "View All" opens full-page notification history with filters

## Resolved Design Decisions

- **Admin template:** Build from scratch with shadcn/ui. Pre-built templates add abstraction layers that fight customization for domain-specific views.
- **Dark mode:** Deferred post-launch. Not a launch blocker; add later with CSS variables already in place.
- **Keyboard shortcuts:** Start with ⌘K only. Add navigation shortcuts (G+C, G+B, etc.) in a later release based on user demand.
- **Notification center:** Slide-out panel from bell icon (not full page). Full page accessible via "View All" link at bottom.

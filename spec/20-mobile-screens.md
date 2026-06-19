# Mobile Screens & Navigation

## Vision & Purpose

The mobile app serves two primary personas: **installers** (field crews logging daily work) and **sales reps** (relationship management on the go). The app is offline-first — all core workflows must function without connectivity. UI is optimized for one-handed use, gloved hands (larger touch targets), and bright sunlight (high contrast).

## Navigation Structure

### Bottom Tab Bar (5 tabs)

```
┌─────────────────────────────────────────────────────────────┐
│                    [Screen Content]                           │
│                                                              │
├────────┬────────┬────────┬────────┬─────────────────────────┤
│Projects│  Log   │ Punch  │Contacts│  More                    │
│  📋   │  📝   │  ✓    │  👤   │  ⋯                       │
└────────┴────────┴────────┴────────┴─────────────────────────┘
```

| Tab | Installer | Sales Rep |
|-----|-----------|-----------|
| Projects | Assigned projects (schedule, progress) | Own bid-related projects |
| Log | Daily log entry (primary workflow) | Quick note / call log |
| Punch | Punch list items assigned | Hidden |
| Contacts | Hidden | Contact list with search |
| More | Settings, sync status, profile | Settings, sync, calendar |

Tabs are role-filtered: installers don't see Contacts, sales reps don't see Punch.

## Screen Specifications

### 1. Projects List

```
┌─────────────────────────────────────────┐
│ My Projects                    [Filter ▾]│
├─────────────────────────────────────────┤
│ ┌─────────────────────────────────────┐ │
│ │ Lincoln Elementary Cafeteria        │ │
│ │ Riverside USD                       │ │
│ │ ████████░░ 78% complete             │ │
│ │ Status: In Progress                 │ │
│ │ Today: Room 101-103                 │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ Riverside High School               │ │
│ │ Riverside USD                       │ │
│ │ █████░░░░░ 55% complete             │ │
│ │ Status: In Progress                 │ │
│ │ ⚠️ 2 punch list items               │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ City Hall Annex                     │ │
│ │ City of Riverside                   │ │
│ │ ░░░░░░░░░░ Scheduled                │ │
│ │ Starts: Jun 24                      │ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

**Behavior:**
- Pull-to-refresh triggers sync
- Tap card → Project Detail
- Filter: All, In Progress, Scheduled, Punch List
- Offline: shows cached project data, syncs when connectivity returns

### 2. Project Detail (Installer View)

```
┌─────────────────────────────────────────┐
│ ← Lincoln Elementary Cafeteria          │
├─────────────────────────────────────────┤
│ Progress: ████████░░ 78%                │
│ 5,400 / 6,900 sqft installed            │
│ Days on site: 12 of 15 scheduled        │
├─────────────────────────────────────────┤
│ TODAY'S SCHEDULE                         │
│ Areas: Room 101, Room 102, Room 103     │
│ Product: Shaw Sustain LVT (Plank)       │
│ Crew: Carlos, Marco, James             │
├─────────────────────────────────────────┤
│ [    + Log Today's Work    ]            │
├─────────────────────────────────────────┤
│ RECENT LOGS                             │
│ Jun 17 - 450 sqft - Rooms 98-100       │
│ Jun 16 - 520 sqft - Rooms 95-97        │
│ Jun 15 - 380 sqft - Hallway B          │
├─────────────────────────────────────────┤
│ FLOOR PLAN                [Full Screen] │
│ ┌─────────────────────────────────────┐ │
│ │  [Minimap with green=done overlay]  │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ PUNCH LIST ITEMS (2)                    │
│ 🔴 Seam separation - Room 88           │
│ 🟡 Transition gap - Hallway A          │
└─────────────────────────────────────────┘
```

**Behavior:**
- Floor plan shows completed areas in green overlay (from daily logs linking areas)
- "Log Today's Work" button is the primary CTA
- Tap a log entry → view/edit that log
- Tap punch item → Punch List Detail

### 3. Daily Log Entry Form

```
┌─────────────────────────────────────────┐
│ ← Daily Log          Lincoln Elementary  │
├─────────────────────────────────────────┤
│ Date: [Jun 18, 2026          ▾]         │
├─────────────────────────────────────────┤
│ Hours Worked                             │
│ [  8.0  ] hours                          │
├─────────────────────────────────────────┤
│ Square Feet Installed                    │
│ [  450  ] sqft                           │
├─────────────────────────────────────────┤
│ Areas Worked                             │
│ [Room 101] [Room 102] [+ Add Area]      │
│ (tap to select from project area list)   │
├─────────────────────────────────────────┤
│ Crew Size                                │
│ [ 3 ] people                             │
├─────────────────────────────────────────┤
│ Product Installed                        │
│ [Shaw Sustain LVT - Coastal Oak    ▾]   │
├─────────────────────────────────────────┤
│ Notes                                    │
│ ┌─────────────────────────────────────┐ │
│ │ Floor prep took extra time in 101.  │ │
│ │ Moisture readings were high near    │ │
│ │ the exterior wall.                  │ │
│ └─────────────────────────────────────┘ │
│ [🎤 Voice Input]                         │
├─────────────────────────────────────────┤
│ Issues                                   │
│ ┌─────────────────────────────────────┐ │
│ │ (Any problems encountered today)    │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ Photos                                   │
│ [📷 +] [img1] [img2] [img3]             │
│ (tap + to capture, tap photo to view)    │
├─────────────────────────────────────────┤
│                                          │
│ [       Save Daily Log       ]           │
│                                          │
└─────────────────────────────────────────┘
```

**Fields:**
| Field | Type | Required | Validation |
|-------|------|----------|------------|
| date | Date picker | Yes | Cannot be future, cannot be >7 days past |
| hoursWorked | Decimal input | Yes | 0-24 |
| sqFtInstalled | Integer input | Yes | 0-10000 |
| areasWorked | Multi-select chip | Yes | At least 1 from project area list |
| crewSize | Integer stepper | Yes | 1-20 |
| productInstalled | Dropdown | No | From project's product list |
| notes | Text area | No | Max 2000 chars |
| issues | Text area | No | Max 2000 chars |
| photos | Photo array | No | Max 20 photos per log |

**Behavior:**
- Form state persists locally (not lost on app backgrounding)
- Voice input converts speech to text in the notes/issues fields
- Areas list pre-populated from project's defined areas
- Save stores locally immediately; syncs to server when online
- Success: shows confirmation toast, navigates back to project detail

### 4. Photo Capture

```
┌─────────────────────────────────────────┐
│ [Cancel]   📷 Photo    [Switch Camera]  │
├─────────────────────────────────────────┤
│                                          │
│         [Camera Viewfinder]              │
│                                          │
│                                          │
│                                          │
│                                          │
├─────────────────────────────────────────┤
│ Type: [Progress ▾]                       │
│ (progress | issue | before | after)      │
├─────────────────────────────────────────┤
│ Caption (optional):                      │
│ [Room 101 - near exterior wall       ]   │
├─────────────────────────────────────────┤
│ 📍 GPS: 33.9425, -117.3894              │
│                                          │
│ [        Capture Photo        ]          │
└─────────────────────────────────────────┘
```

**Behavior:**
- GPS coordinates captured automatically (if permission granted)
- Photo type categorizes the image for later filtering
- Caption is optional but encouraged for punch list photos
- Photos stored locally at full resolution; thumbnails generated for list views
- Upload queued for when connectivity available
- Flash/HDR options available

### 5. Punch List

#### List View

```
┌─────────────────────────────────────────┐
│ Punch List              [All Projects ▾] │
├─────────────────────────────────────────┤
│ OPEN (3)                                 │
│ ┌─────────────────────────────────────┐ │
│ │ 🔴 Seam separation at doorway       │ │
│ │    Lincoln Elem - Room 88           │ │
│ │    Due: Jun 20  │  Critical         │ │
│ └─────────────────────────────────────┘ │
│ ┌─────────────────────────────────────┐ │
│ │ 🟡 Transition strip gap             │ │
│ │    Lincoln Elem - Hallway A         │ │
│ │    Due: Jun 22  │  Major            │ │
│ └─────────────────────────────────────┘ │
│ ┌─────────────────────────────────────┐ │
│ │ 🟢 Minor adhesive bleed             │ │
│ │    Riverside HS - Room 201          │ │
│ │    Due: Jun 25  │  Cosmetic         │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ IN PROGRESS (1)                          │
│ ┌─────────────────────────────────────┐ │
│ │ 🔵 Bubble under LVT plank           │ │
│ │    Lincoln Elem - Room 92           │ │
│ │    Started: Jun 17                  │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ COMPLETED (awaiting verification) (2)    │
│ ┌─────────────────────────────────────┐ │
│ │ ✓ Grout crack - repaired            │ │
│ │   Lincoln Elem - Restroom B         │ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

#### Punch List Detail

```
┌─────────────────────────────────────────┐
│ ← Punch List Item                        │
├─────────────────────────────────────────┤
│ Seam separation at doorway transition    │
│ Priority: 🔴 Critical                    │
│ Location: Room 88, near main door       │
│ Due: Jun 20, 2026                       │
│ Reported by: Dave Chen (PM)             │
│ Reported: Jun 15                         │
├─────────────────────────────────────────┤
│ BEFORE PHOTOS                            │
│ [img1] [img2]                            │
├─────────────────────────────────────────┤
│ PM NOTES                                 │
│ "Seam is separating along the full      │
│  width of the doorway. Need to pull     │
│  back 2-3 planks and re-adhere."        │
├─────────────────────────────────────────┤
│ STATUS                                   │
│ ○ Open  → ● In Progress → ○ Completed   │
│                                          │
│ [  Mark In Progress  ]                   │
├─────────────────────────────────────────┤
│ YOUR NOTES                               │
│ ┌─────────────────────────────────────┐ │
│ │ (Add notes about the repair)        │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ AFTER PHOTOS (required for completion)   │
│ [📷 + Add Photo]                         │
│                                          │
│ (At least 1 photo required to mark      │
│  as Completed)                           │
└─────────────────────────────────────────┘
```

**Status Flow (Installer):**
- Open → In Progress (tap "Mark In Progress")
- In Progress → Completed (requires at least 1 after photo + notes)
- Only PM can verify or revert (happens on web/server)

### 6. Contact Quick View (Sales Rep)

```
┌─────────────────────────────────────────┐
│ Contacts                    [🔍 Search]  │
├─────────────────────────────────────────┤
│ ┌─────────────────────────────────────┐ │
│ │ Mike Johnson          Score: 72 🟢  │ │
│ │ Dir. of Facilities - Riverside USD  │ │
│ │ Last: 5 days ago                    │ │
│ │ [📞 Call] [📧 Email] [📝 Note]      │ │
│ └─────────────────────────────────────┘ │
│ ┌─────────────────────────────────────┐ │
│ │ Sarah Williams        Score: 45 🟡  │ │
│ │ Purchasing Agent - Riverside USD    │ │
│ │ Last: 38 days ago ⚠️                │ │
│ │ [📞 Call] [📧 Email] [📝 Note]      │ │
│ └─────────────────────────────────────┘ │
│ ┌─────────────────────────────────────┐ │
│ │ Tom Garcia            Score: 28 🔴  │ │
│ │ Facilities Mgr - Corona-Norco USD   │ │
│ │ Last: 60 days ago 🔴                │ │
│ │ [📞 Call] [📧 Email] [📝 Note]      │ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

**Tap contact card** → Contact Detail:
```
┌─────────────────────────────────────────┐
│ ← Mike Johnson                           │
│   Director of Facilities                 │
│   Riverside USD                          │
│   Relationship: ████████░░ 72           │
├─────────────────────────────────────────┤
│ QUICK ACTIONS                            │
│ [📞 Call] [📧 Email] [📝 Note] [🎁 Gift]│
├─────────────────────────────────────────┤
│ BRIEFING                                 │
│ 🎣 Shared: Fishing                      │
│ 🎓 Daughter at ASU                      │
│ 🏀 Lakers fan                           │
│ 💬 "How's the boat motor?"              │
├─────────────────────────────────────────┤
│ RECENT                                   │
│ Jun 10 📞 12 min - cafeteria project    │
│ Jun 3  📧 RE: Flooring Samples          │
│ May 28 🏢 Site visit - Lincoln          │
├─────────────────────────────────────────┤
│ INTERESTS                                │
│ Fly fishing • Lakers • BBQ              │
│ Daughter at ASU • Boats                  │
└─────────────────────────────────────────┘
```

### 7. Quick Note Entry (Sales Rep)

```
┌─────────────────────────────────────────┐
│ ← Add Note           Mike Johnson        │
├─────────────────────────────────────────┤
│ What happened?                           │
│ ┌─────────────────────────────────────┐ │
│ │ Ran into Mike at the hardware store.│ │
│ │ He mentioned the district approved  │ │
│ │ funding for Lincoln cafeteria.      │ │
│ └─────────────────────────────────────┘ │
│ [🎤 Voice]                               │
├─────────────────────────────────────────┤
│ Personal details learned (private)       │
│ ┌─────────────────────────────────────┐ │
│ │ His daughter made the Dean's list.  │ │
│ │ Boat motor finally fixed.           │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ Related to:                              │
│ [Lincoln Elem Cafeteria bid        ▾]   │
│ (optional project/bid link)              │
├─────────────────────────────────────────┤
│                                          │
│ [         Save Note         ]            │
└─────────────────────────────────────────┘
```

### 8. More / Settings

```
┌─────────────────────────────────────────┐
│ More                                     │
├─────────────────────────────────────────┤
│ SYNC STATUS                              │
│ Last synced: 2 minutes ago               │
│ Pending uploads: 3 photos                │
│ [  Sync Now  ]                           │
├─────────────────────────────────────────┤
│ ACCOUNT                                  │
│ Carlos Martinez                          │
│ carlos@company.com                       │
│ Role: Installer                          │
├─────────────────────────────────────────┤
│ SETTINGS                                 │
│ > Notifications          [On]            │
│ > Photo Quality          [High]          │
│ > Auto-Sync              [Wi-Fi Only]    │
│ > Voice Input Language   [English]       │
├─────────────────────────────────────────┤
│ STORAGE                                  │
│ Cached data: 45 MB                       │
│ Pending photos: 128 MB                   │
│ [Clear Cache]                            │
├─────────────────────────────────────────┤
│ [  Log Out  ]                            │
├─────────────────────────────────────────┤
│ App v1.0.0 • Last update: Jun 15        │
└─────────────────────────────────────────┘
```

## Offline Behavior

### Connectivity Indicator

```
┌─────────────────────────────────────────┐
│ ⚠️ You're offline. Changes will sync    │
│    when connected.              [Dismiss]│
├─────────────────────────────────────────┤
│ [Normal screen content below]            │
```

- Persistent banner at top when offline (not dismissable permanently, only hides for current session)
- All forms work normally offline
- Data saved to local WatermelonDB
- Photos stored locally, queued for upload
- Banner shows "Syncing..." with progress when reconnecting

### What Works Offline

| Feature | Offline | Notes |
|---------|---------|-------|
| View projects | Yes | Cached on assignment |
| Create daily log | Yes | Syncs later |
| Take photos | Yes | Queued for upload |
| View/update punch list | Yes | Status syncs later |
| View contacts | Yes | Cached for territory |
| Add notes | Yes | Syncs later |
| View floor plans | Yes | Pre-cached PDFs |
| Search contacts | Yes | Local DB search |
| AI briefing cards | No | Requires server |
| Email send | No | Requires connectivity |

## Interaction Patterns

### Gestures
- Pull-to-refresh on all list screens
- Swipe left on punch item → quick-mark in progress
- Swipe left on daily log → edit
- Long press photo → full screen view

### Input Optimizations
- Large touch targets (min 48px) for field use with gloves
- Numeric keyboard auto-shown for sqft/hours fields
- Voice input button prominent on text fields
- Camera launches directly (no file picker for photos)

### Notifications (Push)

| Event | Message | Action |
|-------|---------|--------|
| New punch list item | "New punch item: {description}" | Opens item |
| Schedule change | "Schedule updated for {project}" | Opens project |
| Material delivery | "Materials delivered: {project}" | Opens project |
| Missing daily log | "Don't forget to log today's work" | Opens log form |
| Sync complete | (Silent) | Badge update only |

## Business Rules

- **BR-MOB-001:** Daily log date cannot be more than 7 days in the past (prevents backlog abuse)
- **BR-MOB-002:** Punch list status can only move forward on mobile (Open → In Progress → Completed)
- **BR-MOB-003:** At least 1 "after" photo required to mark punch item completed
- **BR-MOB-004:** Photos include GPS coordinates if location permission granted; app explains why on first request
- **BR-MOB-005:** Background sync pauses below 15% battery to preserve device
- **BR-MOB-006:** App requires biometric or PIN unlock (data encrypted at rest via SQLCipher)
- **BR-MOB-007:** Pending changes indicator shows count on More tab badge

## Implementation Guide

### File Locations
- `apps/mobile/src/navigation/` — Tab navigator, stack navigators
- `apps/mobile/src/screens/` — Screen components by feature
  - `screens/projects/` — ProjectList, ProjectDetail
  - `screens/daily-log/` — DailyLogForm, DailyLogList
  - `screens/punch-list/` — PunchListScreen, PunchListDetail
  - `screens/contacts/` — ContactList, ContactDetail, QuickNote
  - `screens/more/` — Settings, SyncStatus, Profile
- `apps/mobile/src/components/` — Shared mobile components
  - `PhotoCapture.tsx` — Camera with GPS and type picker
  - `OfflineBanner.tsx` — Connectivity status
  - `VoiceInput.tsx` — Speech-to-text button
  - `ScoreBadge.tsx` — Relationship score display

### Implementation Order
1. Navigation structure (tab bar + stack navigators)
2. Project list and detail screens
3. Daily log form with local persistence
4. Photo capture with GPS
5. Punch list screens with status transitions
6. Contact list and detail (sales rep)
7. Quick note and call logging
8. Offline banner and sync status
9. Push notification handling
10. Settings and storage management

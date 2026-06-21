# Mobile Field App

## Vision & Purpose

Field installers work inside buildings — often schools and government buildings with poor or no cell service. They need to log daily production, take photos, check floor plans, and review punch lists without relying on connectivity. The mobile app is offline-first: everything works locally and syncs when connectivity returns.

The app is designed for people wearing work gloves on dusty job sites — large touch targets, minimal text entry, photo-centric workflows, and voice input where possible.

## Key Concepts

- **Offline-First** — All core features work without internet. Data is stored locally and synced when connectivity returns.
- **Sync** — Bidirectional data synchronization between the mobile device and the server. Handles conflicts gracefully.
- **Daily Log** — The primary field workflow: hours, production, photos, issues — logged at the end of each workday.
- **Punch List** — Deficiency items assigned to the installer: go to location, fix issue, photograph the fix.

## User Stories

### Field Installer
- As an installer, I want to log my daily hours and square feet installed in under 2 minutes (P0)
- As an installer, I want to take photos that automatically attach to today's daily log (P0)
- As an installer, I want to see my schedule (which project, where, what time) (P0)
- As an installer, I want to view floor plans and scope details for my current project (P0)
- As an installer, I want to see and resolve my punch list items with before/after photos (P0)
- As an installer, I want the app to work completely offline on job sites with no service (P0)
- As an installer, I want to use voice input instead of typing for notes (P1)

### Sales Rep (Mobile)
- As a sales rep, I want to quickly log a call or meeting from my phone immediately after it happens (P0)
- As a sales rep, I want to see a contact's briefing card on my phone before a meeting (P0)
- As a sales rep, I want to look up a contact's details and history while in the field (P0)
- As a sales rep, I want to scan a business card and auto-create a contact (P1)

### Project Manager (Mobile)
- As a PM, I want to receive daily log summaries as push notifications (P0)
- As a PM, I want to create punch list items on-site with photos during walk-throughs (P0)
- As a PM, I want to approve/verify punch list fixes from my phone (P1)

## Technical Design

### Architecture

```
┌─────────────────────────┐     ┌──────────────────┐
│   React Native / Expo    │     │   Salescraft API  │
│                          │     │                   │
│  ┌───────────────────┐  │     │  ┌─────────────┐ │
│  │  WatermelonDB      │◄─┼──sync──┼─│  PostgreSQL  │ │
│  │  (SQLite local)    │  │     │  └─────────────┘ │
│  └───────────────────┘  │     │                   │
│                          │     │  ┌─────────────┐ │
│  ┌───────────────────┐  │     │  │     S3       │ │
│  │  Photo Queue       │──┼──upload──┼─│  (files)    │ │
│  │  (local storage)   │  │     │  └─────────────┘ │
│  └───────────────────┘  │     │                   │
│                          │     └──────────────────┘
│  ┌───────────────────┐  │
│  │  Push Notifications│  │
│  └───────────────────┘  │
└─────────────────────────┘
```

### Data Model (Local - WatermelonDB)

The mobile app stores a SUBSET of server data locally. Only data relevant to the current user's assignments is synced.

```typescript
// Tables synced to device
type LocalTables = {
  projects: LocalProject[];       // Only assigned projects
  daily_logs: LocalDailyLog[];    // User's own logs
  punch_list_items: LocalPunchListItem[]; // Assigned items
  contacts: LocalContact[];       // Contacts at assigned orgs (sales rep)
  schedules: LocalSchedule[];     // Next 14 days of assignments
  photos: LocalPhoto[];           // Queued for upload
};

interface LocalProject {
  id: string;                     // Matches server project ID
  title: string;
  organizationName: string;
  facilityAddress: string;
  status: string;
  startDate: string;
  completionDate?: string;
  floorPlanUrls: string[];        // Pre-cached file paths
  scopeNotes: string;
  totalSqFt: number;
  installedSqFt: number;         // Running total
}

interface LocalDailyLog {
  id: string;                     // Generated locally (UUID)
  projectId: string;
  date: string;                   // YYYY-MM-DD
  hoursWorked: number;
  sqFtInstalled: number;
  areasWorked: string;            // JSON array as string
  crewSize: number;
  issues?: string;
  notes?: string;
  photoIds: string;               // JSON array of local photo IDs
  syncStatus: 'pending' | 'synced' | 'conflict';
  syncedAt?: string;
  createdAt: string;
  updatedAt: string;
}

interface LocalPhoto {
  id: string;                     // Generated locally
  localPath: string;              // File system path on device
  projectId: string;
  dailyLogId?: string;
  punchListItemId?: string;
  type: 'progress' | 'issue' | 'before' | 'after' | 'general';
  caption?: string;
  latitude?: number;
  longitude?: number;
  takenAt: string;
  uploadStatus: 'queued' | 'uploading' | 'uploaded' | 'failed';
  serverUrl?: string;             // S3 URL after upload
  retryCount: number;
}

interface LocalPunchListItem {
  id: string;
  projectId: string;
  location: string;
  description: string;
  priority: string;
  status: string;
  beforePhotoIds: string;
  afterPhotoIds: string;
  notes?: string;
  syncStatus: 'pending' | 'synced';
}
```

### Sync Strategy

```typescript
// Sync protocol
interface SyncRequest {
  lastSyncTimestamp: string;       // ISO timestamp of last successful sync
  pendingChanges: PendingChange[];
}

interface PendingChange {
  table: string;
  id: string;
  action: 'create' | 'update';
  data: Record<string, unknown>;
  localTimestamp: string;
}

interface SyncResponse {
  serverChanges: ServerChange[];   // Changes from server since lastSync
  conflicts: Conflict[];           // Items changed both locally and on server
  acceptedIds: string[];           // Local changes accepted by server
  rejectedIds: Array<{ id: string; reason: string }>;
}

interface Conflict {
  table: string;
  id: string;
  localVersion: Record<string, unknown>;
  serverVersion: Record<string, unknown>;
  resolution: 'server_wins' | 'client_wins' | 'manual';
}
```

**Sync Rules:**
- Sync triggers: on app foreground, on connectivity restored, every 15 minutes while online, manual pull-to-refresh
- Photos sync independently of data (larger payload, can be slow)
- Conflict resolution: for daily logs, last-write-wins (field data is authoritative). For punch list status changes, server wins (PM is authoritative).
- Partial sync support: if connection drops mid-sync, resume from last acknowledged batch

### API Endpoints (Mobile-Specific)

```typescript
// Sync
POST   /api/v1/mobile/sync                    // Full bidirectional sync
// Body: SyncRequest
// Response: SyncResponse

// Photo Upload
POST   /api/v1/mobile/photos                  // Upload photo
// Multipart: file + metadata (projectId, type, caption, coordinates, takenAt)
// Response: { id, serverUrl }

POST   /api/v1/mobile/photos/batch            // Upload multiple photos
// Supports up to 10 photos per request

// Offline Cache Pre-load
GET    /api/v1/mobile/cache/:userId            // Get all data needed for offline use
// Response: projects, schedules, punch lists, floor plans URLs, contacts

// Push Token
POST   /api/v1/users/me/push-token            // Register push notification token
// Body: { token, platform: 'ios' | 'android' }
```

### Business Rules

- **BR-MOB-001:** Daily logs created offline are treated as authoritative. If a PM edits the same log on web before sync, the field version wins (field crew was there).
- **BR-MOB-002:** Photos are queued locally and uploaded in order of creation. If upload fails 3 times, move to "failed" and alert user. Never discard a photo automatically.
- **BR-MOB-003:** Punch list status can only move forward on mobile: Open → In Progress → Completed. Only PM (on web) can move to Verified or revert.
- **BR-MOB-004:** The app must function with zero network for the full workday (8+ hours). All logged data persists until sync.
- **BR-MOB-005:** Floor plans are pre-downloaded when a project is assigned. Cached locally until project is complete.
- **BR-MOB-006:** Location data (GPS coordinates) is automatically attached to photos and daily logs (if permission granted). Used for project site verification.
- **BR-MOB-007:** Push notifications are delivered for: new punch list items assigned, schedule changes, material delivery updates, project status changes.
- **BR-MOB-008:** Battery-conscious: background sync only runs when device is >20% battery. Photo upload pauses below 15%.
- **BR-MOB-009:** Sales rep mobile features include: contact lookup, briefing card view, quick call log, quick note, basic interaction timeline. Full features require web.
- **BR-MOB-010:** App data is encrypted on device (SQLCipher for WatermelonDB). PIN/biometric required to open.

## Implementation Guide

### File Locations
```
apps/mobile/
├── src/
│   ├── App.tsx
│   ├── navigation/
│   │   ├── AppNavigator.tsx        # Role-based navigation
│   │   ├── InstallerStack.tsx      # Installer-specific screens
│   │   └── RepStack.tsx            # Sales rep screens
│   ├── screens/
│   │   ├── installer/
│   │   │   ├── DashboardScreen.tsx    # Today's assignment
│   │   │   ├── DailyLogScreen.tsx     # Log entry form
│   │   │   ├── PunchListScreen.tsx    # Punch list items
│   │   │   ├── FloorPlanScreen.tsx    # View floor plans
│   │   │   ├── PhotoScreen.tsx        # Camera + gallery
│   │   │   └── ScheduleScreen.tsx     # 2-week schedule
│   │   ├── rep/
│   │   │   ├── ContactsScreen.tsx     # Contact lookup
│   │   │   ├── BriefingScreen.tsx     # Pre-meeting briefing
│   │   │   ├── LogCallScreen.tsx      # Quick call logger
│   │   │   └── TimelineScreen.tsx     # Contact timeline
│   │   └── common/
│   │       ├── LoginScreen.tsx
│   │       ├── SyncStatusScreen.tsx
│   │       └── SettingsScreen.tsx
│   ├── components/
│   │   ├── PhotoCapture.tsx        # Camera with auto-tagging
│   │   ├── VoiceInput.tsx          # Speech-to-text for notes
│   │   ├── SyncIndicator.tsx       # Connection/sync status bar
│   │   ├── OfflineBanner.tsx       # "Working offline" indicator
│   │   └── LargeButton.tsx         # Glove-friendly touch targets
│   ├── services/
│   │   ├── sync.ts                 # Sync engine
│   │   ├── photos.ts              # Photo queue and upload
│   │   ├── api.ts                 # API client with offline awareness
│   │   ├── notifications.ts      # Push notification handling
│   │   └── location.ts           # GPS tracking
│   ├── db/
│   │   ├── schema.ts             # WatermelonDB schema
│   │   ├── models/               # WatermelonDB model classes
│   │   └── sync-adapter.ts       # Server sync adapter
│   └── hooks/
│       ├── useSync.ts
│       ├── usePhoto.ts
│       └── useOffline.ts
├── app.json                       # Expo config
└── package.json
```

### Key Dependencies
- `expo` — Managed workflow for camera, location, notifications, file system
- `@nozbe/watermelondb` — SQLite-based offline-first database
- `expo-camera` — Photo capture
- `expo-file-system` — Local file management for photos
- `expo-image-picker` — Photo library access
- `expo-location` — GPS coordinates
- `expo-notifications` — Push notifications (FCM/APNs)
- `expo-speech` — Voice-to-text input
- `react-native-pdf` — Floor plan PDF viewer
- `@react-navigation/native` — Navigation
- `react-native-gesture-handler` — Swipe gestures
- `expo-secure-store` — Encrypted credential storage
- `expo-local-authentication` — Biometric unlock

### Implementation Order
1. Expo project setup with navigation
2. Authentication (login, token storage, biometric)
3. WatermelonDB setup with schema
4. Basic sync engine (pull from server)
5. Installer dashboard (today's project + stats)
6. Daily log form (offline creation + local storage)
7. Photo capture with local queue
8. Photo upload service (background)
9. Push sync (upload local changes to server)
10. Punch list view and status updates
11. Floor plan viewer (PDF with zoom)
12. Push notifications
13. Schedule view
14. Sales rep screens (contact lookup, call log, briefing)
15. Business card scanner (OCR)
16. Voice input for notes

## Testing Requirements

### Unit Tests
- Sync: pending local changes → serialized correctly in SyncRequest
- Sync: server changes received → applied to local DB
- Sync: conflict detected → resolved per rules (daily log: client wins)
- Photo queue: photo taken → queued → uploaded → status updated
- Photo queue: upload fails → retry count incremented → still queued
- Offline detection: network lost → app shows offline banner → features still work

### Integration Tests
- Full sync cycle: create daily log offline → come online → sync → appears on server
- Photo flow: take photo → queue → upload → server URL returned → daily log updated
- Punch list: mark complete with photo → sync → PM sees on web

### E2E Scenarios
1. **Full workday offline:** Installer arrives at job site → logs start time → works all day (no signal) → takes 10 photos → logs production → leaves site → phone connects in parking lot → everything syncs in background
2. **Punch list resolution:** PM creates punch list on web → installer opens app → sees new items (after sync) → goes to location → takes "before" photo → fixes issue → takes "after" photo → marks complete → PM verifies on web

## Error Handling

| Failure | Handling |
|---------|----------|
| Sync fails (network error) | Queue changes locally. Show "will sync when connected" banner. Retry on next connectivity. |
| Photo upload fails | Keep in local queue. Retry with exponential backoff. Never delete local file until server confirms. |
| Conflict on punch list | Server wins for status (PM authority). Show notification: "Item was updated on server". |
| Device storage full | Alert user. Suggest clearing uploaded photos from local cache. Block new photos until space freed. |
| App crash during daily log entry | WatermelonDB persists on every field change (no "save" required). Data survives crash. |
| Token expired while offline | On next sync attempt, redirect to login. All pending data is preserved. |

## UI/UX Requirements

### Design Principles (Field-Optimized)
- **Large touch targets:** Minimum 48px, preferably 56px. Workers wear gloves.
- **High contrast:** Strong colors, large text. Job sites are bright (outdoor staging) or dim (unlit buildings).
- **Minimal typing:** Use pickers, toggles, photo buttons. Voice input for notes.
- **Fast entry:** Daily log should be completable in <2 minutes including photos.
- **Always-visible sync status:** Badge/bar showing "synced", "pending", or "offline".

### Installer Dashboard
- **Today's Project:** name, address, what area to work in today
- **Progress ring:** sq ft installed / total, with today's target
- **Quick actions:** big buttons for "Log Daily Report", "Take Photo", "View Punch List"
- **Sync status:** small indicator showing last sync time and pending items count

### Daily Log Form
- **Date:** defaults to today
- **Hours:** simple number stepper (6, 6.5, 7, 7.5, 8, 8.5, 9...)
- **Sq Ft Installed:** number input with "+100" quick buttons
- **Areas Worked:** checkboxes from project area list
- **Crew Size:** stepper (1-10)
- **Issues:** optional text field (voice input button beside it)
- **Photos:** camera button → photo grid showing today's photos
- **Save:** single big button. Confirms with checkmark animation.

### Punch List View
- List sorted by priority (critical first)
- Each item: location, description, priority badge, status badge
- Tap item → detail with before photos, instructions
- "Mark Complete" → requires after photo capture → confirmation
- Swipe left to flag as "disputed" with reason

### Floor Plan Viewer
- PDF viewer with pinch-to-zoom
- Overlay showing completed areas (green), in-progress (yellow), remaining (gray)
- Tap room → see area details, product type, notes

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| Salescraft API | Sync all data | Bidirectional | On connectivity + 15 min |
| AWS S3 | Photo upload | Outbound | Background when online |
| Expo Push Service (FCM/APNs) | Push notifications | Inbound | Real-time |
| Device Camera | Photo capture | Local | On demand |
| Device GPS | Location tagging | Local | On photo/log |

## Performance Requirements

- App launch to usable (cached data): < 2 seconds
- Daily log save (local): < 200ms
- Photo capture to saved locally: < 1 second
- Full sync (10 daily logs + 20 photos): < 60 seconds on 4G
- Floor plan PDF render: < 3 seconds
- Background photo upload: < 10 seconds per photo on 4G

## Non-Functional Requirements

- App must work fully offline for 8+ hours (full workday)
- Local data encrypted at rest (SQLCipher)
- Biometric or PIN required to open
- Photos never deleted locally until confirmed uploaded to server
- Battery usage: <5% per hour when not actively in use
- Supports iOS 15+ and Android 11+
- App size: target <50MB download, <200MB installed with cached data
- Accessibility: VoiceOver/TalkBack support for basic navigation

## Resolved Design Decisions

- **Tablet support:** Yes. The React Native/Expo app runs on tablets natively. Floor plan viewer and punch list benefit from larger screens. No separate tablet-specific layout needed — responsive design handles it (larger viewport = more content visible).
- **Photo compression:** Yes, compress before upload. Use 80% JPEG quality and max 2048px on longest edge. Original full-resolution stored locally until upload confirmed. This reduces upload time on cellular connections and storage costs. Users can toggle "High Quality" in settings for situations requiring full resolution.
- **Geofencing:** Not for MVP. Privacy concerns and battery drain outweigh the benefit. Rely on manual daily log creation (which is already required). Could add as opt-in P2 feature for automatic time tracking.
- **Multiple projects per day:** Yes, supported. The daily log is per-project-per-day, so a crew member can log work on two projects in the same day. The Projects list shows all assigned projects. This is common for small punch list returns or split crews.

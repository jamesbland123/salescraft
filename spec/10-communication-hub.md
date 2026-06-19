# Communication Hub

## Vision & Purpose

Every sale starts with a conversation — an email, a phone call, a meeting at a trade show. Salescraft captures ALL communication touchpoints in one unified timeline per contact, so nothing is ever lost and every interaction builds on the last. When a rep opens a contact record, they see every email, call, meeting, and note in chronological order with AI-extracted insights.

The communication hub doesn't replace email or phone — it integrates with them. Reps keep using Gmail/Outlook and their phones. Salescraft syncs, tracks, and enriches those communications automatically.

## Key Concepts

- **Unified Timeline** — Single chronological view of all interactions with a contact across all channels
- **Two-Way Email Sync** — Emails sent from Gmail/Outlook are automatically captured; emails can also be sent from within Salescraft
- **Call Logging** — Manual or VoIP-integrated call records with notes, duration, and outcome
- **Activity Auto-Capture** — Meetings from calendar, emails from inbox, and calls from phone system are captured without manual logging
- **Communication Templates** — Reusable email/message templates with personalization variables

## User Stories

### Sales Rep
- As a sales rep, I want all my emails with a contact to automatically appear on their timeline without me doing anything (P0)
- As a sales rep, I want to log a phone call in 30 seconds with notes and next steps (P0)
- As a sales rep, I want to send emails directly from a contact's page with my Gmail/Outlook account (P0)
- As a sales rep, I want email templates for common outreach (introduction, follow-up, bid notification) with auto-filled contact fields (P1)
- As a sales rep, I want to see email open/click tracking so I know when contacts engage (P2)

### All Users
- As any user, I want to quickly add a note to a contact's timeline (met them at a trade show, overheard something, etc.) (P0)
- As a PM, I want to log client calls about project issues on both the project and contact timelines (P1)

## Technical Design

### Data Model

Uses `Interaction` entity from `02-domain-model.md` as the core record.

#### EmailAccount (connected email accounts)
```typescript
interface EmailAccount {
  id: string;
  userId: string;                 // FK → User
  provider: 'gmail' | 'outlook';
  email: string;                  // The email address
  accessToken: string;            // Encrypted OAuth token
  refreshToken: string;           // Encrypted OAuth refresh token
  tokenExpiry: DateTime;
  syncState: {
    lastSyncedAt?: DateTime;
    historyId?: string;           // Gmail history ID / Outlook deltaLink
    syncErrors: number;
  };
  isActive: boolean;
  createdAt: DateTime;
}
```

#### EmailMessage (synced emails)
```typescript
interface EmailMessage {
  id: string;
  emailAccountId: string;         // FK → EmailAccount
  externalId: string;             // Gmail/Outlook message ID
  threadId?: string;              // For conversation threading
  interactionId?: string;         // FK → Interaction (linked to contact timeline)
  contactId?: string;             // FK → Contact (matched from email address)
  direction: 'inbound' | 'outbound';
  from: string;
  to: string[];
  cc?: string[];
  subject: string;
  bodyPreview: string;            // First 200 chars of plain text
  bodyHtml?: string;              // Full HTML body (stored in S3 for large emails)
  hasAttachments: boolean;
  attachments?: EmailAttachment[];
  sentAt: DateTime;
  readAt?: DateTime;              // When the contact opened (tracking pixel)
  labels?: string[];              // Gmail labels / Outlook categories
  aiSummary?: string;             // AI-generated one-line summary
  aiSentiment?: 'positive' | 'neutral' | 'negative';
  createdAt: DateTime;
}

interface EmailAttachment {
  id: string;
  filename: string;
  mimeType: string;
  size: number;
  fileUrl?: string;               // S3 URL if we stored it
}
```

#### CommunicationTemplate
```typescript
interface CommunicationTemplate {
  id: string;
  name: string;                   // "Introduction Email"
  category: string;               // "prospecting", "follow_up", "bid_notification", "thank_you"
  channel: 'email' | 'sms' | 'linkedin';
  subject?: string;               // For emails
  body: string;                   // With merge fields: {{firstName}}, {{organizationName}}, etc.
  mergeFields: string[];          // Available fields for this template
  isShared: boolean;              // Available to all users or personal
  createdBy: string;              // FK → User
  usageCount: number;
  createdAt: DateTime;
  updatedAt: DateTime;
}
```

#### CallLog
```typescript
interface CallLog {
  id: string;
  userId: string;                 // FK → User
  contactId: string;              // FK → Contact
  interactionId: string;          // FK → Interaction (created automatically)
  direction: 'inbound' | 'outbound';
  phoneNumber: string;
  duration: number;               // Seconds
  outcome: CallOutcome;
  notes?: string;
  nextSteps?: string;
  nextStepDate?: Date;
  recordingUrl?: string;          // If VoIP integration provides recording
  transcription?: string;         // AI transcription
  createdAt: DateTime;
}

enum CallOutcome {
  CONNECTED = 'connected',
  VOICEMAIL = 'voicemail',
  NO_ANSWER = 'no_answer',
  BUSY = 'busy',
  WRONG_NUMBER = 'wrong_number',
  LEFT_MESSAGE = 'left_message',
}
```

### API Endpoints

```typescript
// Unified Timeline
GET    /api/v1/contacts/:id/timeline           // All interactions, paginated, newest first
GET    /api/v1/contacts/:id/timeline?type=email // Filter by type
GET    /api/v1/organizations/:id/timeline      // All interactions with anyone at this org

// Email
GET    /api/v1/email/accounts                  // Connected email accounts
POST   /api/v1/email/accounts/connect          // OAuth flow initiation
DELETE /api/v1/email/accounts/:id              // Disconnect
POST   /api/v1/email/accounts/:id/sync         // Force sync
GET    /api/v1/email/messages                  // List synced emails (filter by contact/date)
GET    /api/v1/email/messages/:id              // Full email detail
POST   /api/v1/email/send                      // Send email via connected account
// Body: { accountId, to, cc?, subject, body, templateId?, contactId }

// Call Logging
POST   /api/v1/calls                           // Log a call
GET    /api/v1/calls                           // List calls (filter by user/contact/date)
PUT    /api/v1/calls/:id                       // Update call notes

// Interactions (generic)
POST   /api/v1/interactions                    // Log any interaction type
GET    /api/v1/interactions                    // List with filters
PUT    /api/v1/interactions/:id
GET    /api/v1/interactions/feed               // Activity feed for current user

// Templates
GET    /api/v1/templates                       // List templates
POST   /api/v1/templates
PUT    /api/v1/templates/:id
DELETE /api/v1/templates/:id
POST   /api/v1/templates/:id/render            // Render template with contact data
// Body: { contactId } → returns rendered subject + body

// Quick Note
POST   /api/v1/contacts/:id/notes              // Quick note (creates Interaction type=note)
// Body: { content: string, personalNotes?: string }
```

### Business Rules

- **BR-COMM-001:** Email sync runs every 5 minutes for active accounts (via BullMQ cron). Uses push notifications (Gmail Pub/Sub, Outlook webhooks) for real-time when available.
- **BR-COMM-002:** Inbound/outbound emails are auto-matched to contacts by email address. If no match, they appear in an "unmatched" queue for manual assignment.
- **BR-COMM-003:** Every email/call/meeting logged auto-updates the contact's `lastContactedAt` and `lastContactedBy` fields, which feeds relationship scoring.
- **BR-COMM-004:** Call logging with `outcome = 'connected'` and `duration > 0` counts as a substantive interaction for relationship scoring. Voicemail/no answer do not.
- **BR-COMM-005:** Email templates support these merge fields: `{{firstName}}`, `{{lastName}}`, `{{title}}`, `{{organizationName}}`, `{{repFirstName}}`, `{{repLastName}}`, `{{repPhone}}`, `{{repEmail}}`. Custom fields can be added.
- **BR-COMM-006:** Sent emails are tracked for opens (via tracking pixel) and link clicks. Open tracking is disabled for government .gov addresses (often blocked and may trigger spam filters).
- **BR-COMM-007:** When a personal note is logged via quick-add, it triggers the AI interest extraction job (background) to look for personal details.
- **BR-COMM-008:** Email body storage: first 10KB stored in database. Larger bodies stored in S3 with database reference. Attachments always stored in S3.
- **BR-COMM-009:** OAuth tokens are refreshed proactively (before expiry). If refresh fails, mark account as `inactive` and alert user.
- **BR-COMM-010:** Interactions can be linked to both a contact AND a project/bid. This provides context on both timelines.

### Email Sync Architecture

```
Gmail/Outlook → OAuth2 → Salescraft API → Database + S3
                                    ↓
                            Contact Matching
                                    ↓
                        Relationship Score Update
                                    ↓
                        AI Enrichment (background)
```

**Gmail Sync Flow:**
1. Connect via OAuth2 (scopes: `gmail.readonly`, `gmail.send`, `gmail.modify`)
2. Initial sync: fetch last 30 days of sent/received emails
3. Match emails to contacts by `from`/`to` addresses
4. Ongoing: use Gmail Push Notifications (Pub/Sub) for real-time, fall back to polling `history.list` every 5 min
5. On new email: create/update EmailMessage record, create Interaction, update contact timestamps

**Outlook Sync Flow:**
1. Connect via OAuth2 (Microsoft Graph, scopes: `Mail.ReadWrite`, `Mail.Send`)
2. Initial sync: fetch last 30 days using `/messages` endpoint
3. Ongoing: use Change Notifications (webhooks) for real-time, fall back to delta queries
4. Same processing as Gmail after message fetch

## Implementation Guide

### File Locations
- `apps/api/src/modules/communications/` — Communication hub module
  - `communications.routes.ts` — REST endpoints
  - `email.service.ts` — Email sync, send, matching
  - `calls.service.ts` — Call logging
  - `timeline.service.ts` — Unified timeline assembly
  - `templates.service.ts` — Template management and rendering
- `apps/api/src/jobs/email-sync.job.ts` — Background email sync
- `apps/api/src/jobs/email-match.job.ts` — Contact matching for unmatched emails

### Key Dependencies
- `googleapis` — Gmail API client
- `@microsoft/microsoft-graph-client` — Outlook/Graph API client
- `nodemailer` — Email sending abstraction
- `juice` — Inline CSS for HTML emails
- `handlebars` — Template rendering with merge fields

### Implementation Order
1. Interaction CRUD (log any interaction manually)
2. Unified timeline query (assemble from interactions table)
3. Quick note + call logging
4. Gmail OAuth2 connection flow
5. Gmail email sync (push + poll)
6. Contact matching on synced emails
7. Send email via Gmail
8. Outlook OAuth2 + sync (mirror Gmail logic)
9. Email templates with merge fields
10. AI enrichment pipeline (background)
11. Open/click tracking (optional, lower priority)

## Testing Requirements

### Unit Tests
- Contact matching: email "john@riversideusd.org" → matches contact with that email
- Contact matching: no match → added to unmatched queue
- Template rendering: `{{firstName}}` → "Mike"
- Call outcome: connected + duration > 0 → updates lastContactedAt
- Call outcome: voicemail → does NOT update lastContactedAt
- Timeline assembly: 3 emails + 2 calls + 1 note → sorted by date descending

### Integration Tests
- Gmail OAuth flow (mock OAuth provider)
- Email sync: new email arrives → synced → matched → appears on timeline
- Send email: compose → send via Gmail API → logged in timeline
- Full template flow: create template → render with contact → send → logged

### Seed Data
```typescript
const sampleInteractions = [
  {
    contactId: 'mike-facility-dir',
    type: 'phone_call',
    direction: 'outbound',
    subject: 'Check on flooring needs',
    summary: 'Called to discuss upcoming summer projects. Mentioned Lincoln Elementary cafeteria needs attention.',
    personalNotes: 'Asked about his fishing trip to Lake Powell - caught a 4lb bass',
    duration: 12,
    sentiment: 'positive',
    nextSteps: 'Send product samples for cafeteria',
    nextStepDueDate: '2024-04-20',
  },
  {
    contactId: 'mike-facility-dir',
    type: 'email',
    direction: 'inbound',
    subject: 'Re: Cafeteria Flooring Samples',
    summary: 'Liked the Shaw Sustain samples. Wants a budget number for the cafeteria project.',
    sentiment: 'positive',
  },
  {
    contactId: 'sarah-purchasing',
    type: 'in_person_meeting',
    direction: 'outbound',
    subject: 'CASBO Conference - purchasing roundtable',
    summary: 'Met Sarah at the CASBO conference. Discussed cooperative purchasing options.',
    personalNotes: 'Mentioned she plays golf at Indian Wells, goes to Maui every March',
    duration: 20,
  },
];
```

## Error Handling

| Failure | Handling |
|---------|----------|
| OAuth token expired, refresh fails | Mark account inactive. Notify user: "Your email connection needs re-authorization". Keep existing synced data. |
| Gmail quota exceeded | Back off for 1 hour. Resume sync. Alert user if > 4 hours. |
| Email send fails | Show error to user with details. Don't log as "sent" interaction. Retry option. |
| Contact match ambiguous (email maps to multiple contacts) | Show in "needs review" queue. Auto-select if one match has relationship score > 50. |
| Large attachment download | Skip attachments > 25MB. Store metadata only. Note in log. |

## UI/UX Requirements

### Unified Timeline (Contact Detail)
- Chronological list of ALL interactions (emails, calls, meetings, notes)
- Each entry shows: icon (type), date, subject/summary, direction arrow (in/out)
- Expandable to show full details
- Filter tabs: All | Email | Calls | Meetings | Notes
- "Log Interaction" dropdown button: Call | Meeting | Note | Site Visit | etc.

### Email Composer
- Full compose window: to (pre-filled from contact), cc, subject, rich body
- Template selector dropdown
- Merge field insert button
- Send via connected account
- Schedule send option

### Call Logger (Quick Form)
- Slide-out panel (doesn't navigate away from current page)
- Fields: Contact (pre-filled), Direction, Duration, Outcome, Notes, Personal Notes, Next Steps
- "Save & Log Another" for power-logging multiple calls
- Mobile-optimized version for logging right after a call

### Activity Feed (Dashboard)
- Cross-contact activity stream for the current user
- Shows recent interactions with quick link to contact
- "Today" section highlighting what needs attention (follow-ups due, calls to make)

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| Gmail API | Email sync and send | Bidirectional | Real-time + 5 min poll |
| Microsoft Graph | Outlook email sync and send | Bidirectional | Real-time + 5 min poll |
| Calendar (spec/06 scheduling) | Meeting detection → auto-log interaction | Inbound | On calendar sync |
| Relationship module (spec/05) | Update relationship scores on interaction | Outbound | On interaction creation |
| AI Engine (spec/09) | Sentiment analysis, interest extraction | Outbound | Background after sync |

## Performance Requirements

- Timeline load (50 interactions): < 500ms
- Email sync (incremental, 10 new messages): < 5 seconds
- Send email: < 3 seconds (including Gmail/Outlook API call)
- Template render: < 100ms
- Full initial sync (30 days, ~200 emails): < 60 seconds (background)

## Non-Functional Requirements

- Email content is stored encrypted at rest (contains confidential bid information)
- OAuth tokens stored with encryption (AES-256)
- Email sync must be resilient to API outages (retry without data loss)
- Users can disconnect email at any time (data retained but sync stops)
- No email is ever deleted from Gmail/Outlook by Salescraft (read-only except for send)
- GDPR: ability to purge all synced emails for a specific contact on request

## Notification Email Templates

The system sends transactional emails for the following events. All use a consistent branded template with company logo, action button, and unsubscribe link.

### Templates

| Event | Subject | Body Summary |
|-------|---------|-------------|
| Bid deadline approaching (7d) | "Bid due in 7 days: {bidTitle}" | Bid details, deadline, link to bid page |
| Bid deadline approaching (3d) | "⚠️ Bid due in 3 days: {bidTitle}" | Urgency, checklist status, link |
| Bid deadline approaching (1d) | "🔴 Bid due TOMORROW: {bidTitle}" | Final warning, submission status |
| Relationship decay alert | "Relationship cooling: {contactName}" | Days since contact, score, suggested action |
| New opportunity discovered | "New opportunity: {title} (Score: {score})" | Signal details, estimated value, link to investigate |
| Compliance document expiring | "Expiring in {days} days: {docName}" | Document type, expiry date, renewal action |
| Daily digest | "Your Salescraft Daily Brief" | Yesterday's signals, today's tasks, at-risk relationships, upcoming deadlines |
| Invitation | "You've been invited to Salescraft" | Welcome message, setup link, role description |
| Password reset | "Reset your Salescraft password" | Reset link (1 hour expiry), security note |
| Bid won | "🎉 Bid Won: {bidTitle}" | Celebration, contract amount, next steps |

### Template Structure

```typescript
interface EmailTemplate {
  id: string;
  event: string;
  subject: string;          // With merge fields: {bidTitle}, {contactName}, etc.
  bodyHtml: string;         // Handlebars template
  bodyText: string;         // Plain text fallback
}
```

### Daily Digest Contents

Sent weekdays at 7 AM (company timezone) to sales reps, sales managers, and owners:

1. **New signals** (last 24h): count and top 3 by score
2. **Today's tasks**: follow-ups due, meetings scheduled, bid deadlines
3. **At-risk relationships**: contacts with score dropping below 40 or >30 days since contact
4. **Upcoming deadlines**: bids due within 7 days
5. **Wins/losses**: any bid results from yesterday

## Resolved Design Decisions

- **Shared inboxes:** Not supported for MVP. Each rep connects their personal email account.
- **SMS/Twilio:** Deferred to P2. Manual call logging covers the use case for now.
- **Chrome extension:** Deferred to P2. In-app email is the primary experience.
- **Internal emails:** Emails between users with the same company email domain are excluded from contact timelines. Determined by matching the `@domain` against the company's configured email domain in settings.

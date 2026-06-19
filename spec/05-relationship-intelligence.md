# Relationship Intelligence

## Vision & Purpose

This is the soul of Salescraft. Government flooring sales is won by people who build genuine relationships — not by the lowest bidder. When a facility director trusts you, they call you for informal quotes below threshold. They put you on the approved vendor list. They tell you about projects before they're public. They specify your products.

Salescraft makes every rep a master relationship-builder by:
1. **Remembering everything** — personal details, interests, family, past conversations that humans forget
2. **Surfacing context proactively** — before every interaction, showing what matters about this person RIGHT NOW
3. **Finding authentic connections** — matching rep interests to contact interests for genuine common ground
4. **Timing outreach perfectly** — knowing when life events, seasons, and shared interests create natural reasons to reach out
5. **Preventing relationship decay** — alerting when key relationships are going cold

A rep who knows that Mike the Facility Director just got back from a fishing trip to Lake Powell, that his daughter started college at ASU, and that his building's flooring is 18 years old — that rep wins the next project.

## Key Concepts

- **Relationship Score** — Computed 0-100 score measuring the health/strength of a relationship
- **Interest Graph** — The network of personal interests connecting contacts and reps
- **Briefing Card** — Pre-interaction summary showing relevant personal context for a contact
- **Reverse Match** — Finding contacts who share a specific interest with a rep
- **Rapport Signal** — A personal detail learned during an interaction that strengthens future conversations
- **Decay Alert** — Warning that a relationship is going cold due to inactivity
- **Gesture** — A relationship-building action (gift, meal, shared article, congratulations)

## User Stories

### Sales Rep
- As a sales rep, I want to see a "briefing card" before every scheduled call or meeting showing the contact's interests, recent life events, our last personal notes, and suggested talking points (P0)
- As a sales rep, I want to search "contacts interested in fishing" to find people I can authentically connect with over shared interests (P0)
- As a sales rep, I want to quickly log personal details I learn during conversations ("mentioned his son plays varsity baseball") so I remember next time (P0)
- As a sales rep, I want alerts when seasonal events align with contact interests ("Opening day next week — 6 contacts are baseball fans") (P0)
- As a sales rep, I want to be notified when a key contact has a life event (promotion, birthday, new role) so I can reach out with congratulations (P1)
- As a sales rep, I want to track gifts and gestures, with the system warning me before I exceed ethics limits (P1)
- As a sales rep, I want to see relationship health scores for all my contacts, sorted by "most at risk of going cold" (P0)
- As a sales rep, I want AI to suggest the best time and reason to reach out to a contact I haven't spoken to in a while (P1)

### Sales Manager
- As a sales manager, I want to see which reps are maintaining strong relationships vs. letting them decay (P1)
- As a sales manager, I want to ensure we're "multi-threaded" at key accounts (not dependent on one relationship) (P1)
- As a sales manager, I want to correlate relationship strength with win rates to prove the ROI of relationship-building (P2)

## Technical Design

### Data Model

Uses `ContactInterest`, `ContactLifeEvent`, `Gesture`, and `Interaction` entities from `02-domain-model.md`.

#### RelationshipScore (computed, cached)
```typescript
interface RelationshipScore {
  contactId: string;              // FK → Contact
  overallScore: number;           // 0-100
  factors: RelationshipFactor[];
  trend: 'improving' | 'stable' | 'declining';
  trendDuration: number;          // Days the trend has persisted
  lastCalculatedAt: DateTime;
  nextDecayAt: DateTime;          // When score will next drop if no interaction
}

interface RelationshipFactor {
  name: string;
  weight: number;
  score: number;                  // 0-100 for this factor
  explanation: string;
}

// Scoring factors
const RELATIONSHIP_SCORE_FACTORS = [
  {
    name: 'recency',
    weight: 0.30,
    description: 'How recently we interacted',
    calculation: `
      100 if last interaction <7 days
      80 if 7-14 days
      60 if 14-30 days
      40 if 30-60 days
      20 if 60-90 days
      0 if >90 days
    `,
  },
  {
    name: 'frequency',
    weight: 0.20,
    description: 'How often we interact (rolling 90 days)',
    calculation: `
      100 if >8 interactions in 90 days
      80 if 5-8
      60 if 3-5
      40 if 1-3
      0 if 0
    `,
  },
  {
    name: 'depth',
    weight: 0.20,
    description: 'Quality of interactions (personal vs. transactional)',
    calculation: `
      Based on interaction types:
      - In-person meeting/lunch: +20
      - Phone call >5 min: +10
      - Personal email (not bid-related): +10
      - Site visit: +15
      - Trade show conversation: +10
      - Bid-only interaction: +5
      Capped at 100
    `,
  },
  {
    name: 'reciprocity',
    weight: 0.15,
    description: 'Whether the contact initiates contact (not just responds)',
    calculation: `
      100 if >30% of interactions are inbound
      70 if 10-30% inbound
      40 if they respond but never initiate
      0 if they don't respond
    `,
  },
  {
    name: 'personal_knowledge',
    weight: 0.15,
    description: 'How well we know them personally (interests, family, etc.)',
    calculation: `
      +20 per confirmed interest (max 3 = 60)
      +20 if we have family details
      +20 if we have personal notes from conversations
      Capped at 100
    `,
  },
];
```

#### InterestMatch (computed for reverse search)
```typescript
interface InterestMatch {
  userId: string;                 // The rep
  contactId: string;              // The matched contact
  matchedInterests: MatchedInterest[];
  matchScore: number;             // 0-100 based on quality/specificity of matches
  suggestedAction?: string;       // AI-generated suggestion
}

interface MatchedInterest {
  category: InterestCategory;
  userInterestName: string;       // "bass fishing"
  contactInterestName: string;    // "fly fishing"
  matchStrength: 'exact' | 'related' | 'broad';
  // exact: same interest ("bass fishing" ↔ "bass fishing")
  // related: same subcategory ("bass fishing" ↔ "fly fishing")
  // broad: same category ("fishing" ↔ "boating")
}
```

#### BriefingCard (generated on-demand)
```typescript
interface BriefingCard {
  contactId: string;
  generatedAt: DateTime;
  contact: {
    name: string;
    title: string;
    organization: string;
    relationshipScore: number;
    daysSinceLastContact: number;
  };
  personalContext: {
    interests: Array<{ name: string; specifics?: string; confidence: number }>;
    recentLifeEvents: ContactLifeEvent[];
    lastPersonalNotes: string[];  // From recent interactions
    familyDetails?: string;
  };
  sharedInterests: MatchedInterest[]; // What the rep and contact have in common
  suggestedTopics: string[];      // AI-generated conversation starters
  recentActivity: {
    lastInteraction?: { type: string; date: Date; summary: string };
    recentBids?: Array<{ title: string; status: string }>;
    activeProjects?: Array<{ title: string; status: string }>;
  };
  warnings: string[];             // "Ethics cooldown active", "Approaching gift limit"
}
```

#### SeasonalTrigger
Predefined triggers that surface outreach suggestions based on calendar events + interests.

```typescript
interface SeasonalTrigger {
  id: string;
  name: string;                   // "Baseball Opening Day"
  interestCategories: InterestCategory[];
  interestKeywords: string[];     // ["baseball", "MLB", specific team names]
  triggerDate: string;            // "03-28" (month-day) or "first-monday-september"
  daysBeforeToAlert: number;      // Notify rep N days before
  suggestedAction: string;        // "Send opening day text/email"
  messageTemplate?: string;       // "Hey {firstName}! Opening day is {daysAway} days away..."
  isActive: boolean;
}

// Pre-configured triggers
const DEFAULT_SEASONAL_TRIGGERS = [
  { name: 'Baseball Opening Day', interestKeywords: ['baseball', 'MLB'], triggerDate: '03-28', daysBeforeToAlert: 7 },
  { name: 'NFL Season Opener', interestKeywords: ['football', 'NFL'], triggerDate: '09-07', daysBeforeToAlert: 5 },
  { name: 'March Madness', interestKeywords: ['basketball', 'NCAA', 'college basketball'], triggerDate: '03-15', daysBeforeToAlert: 7 },
  { name: 'Fishing Season Opens', interestKeywords: ['fishing', 'bass fishing', 'fly fishing'], triggerDate: '04-15', daysBeforeToAlert: 7 },
  { name: 'Hunting Season', interestKeywords: ['hunting', 'deer hunting', 'duck hunting'], triggerDate: '10-01', daysBeforeToAlert: 14 },
  { name: 'Golf Masters', interestKeywords: ['golf'], triggerDate: '04-10', daysBeforeToAlert: 5 },
  { name: 'School Year Start', interestKeywords: ['education', 'teacher', 'principal'], triggerDate: '08-15', daysBeforeToAlert: 7 },
  { name: 'Holiday Season', interestKeywords: [], triggerDate: '12-01', daysBeforeToAlert: 14 }, // All contacts
];
```

### API Endpoints

```typescript
// Contact Interests
GET    /api/v1/contacts/:id/interests          // All interests for a contact
POST   /api/v1/contacts/:id/interests          // Add interest manually
PUT    /api/v1/contacts/:id/interests/:intId   // Update interest
DELETE /api/v1/contacts/:id/interests/:intId   // Remove interest

// User Interests (rep's own interests for matching)
GET    /api/v1/users/me/interests
POST   /api/v1/users/me/interests
PUT    /api/v1/users/me/interests/:intId
DELETE /api/v1/users/me/interests/:intId

// Reverse Interest Matching
GET    /api/v1/relationships/matches           // Contacts matching current user's interests
GET    /api/v1/relationships/matches?interest=fishing  // Contacts with specific interest
GET    /api/v1/relationships/matches?category=outdoors // Contacts in interest category

// Briefing Cards
GET    /api/v1/contacts/:id/briefing           // Generate briefing card for a contact
GET    /api/v1/relationships/upcoming-briefings // Briefings for today's scheduled meetings

// Relationship Scores
GET    /api/v1/relationships/scores            // All contacts with scores, sortable
GET    /api/v1/relationships/at-risk           // Contacts with declining scores
GET    /api/v1/relationships/leaderboard       // Rep comparison (manager view)

// Life Events
GET    /api/v1/contacts/:id/life-events
POST   /api/v1/contacts/:id/life-events
GET    /api/v1/relationships/life-events/upcoming  // Events in next 30 days
POST   /api/v1/relationships/life-events/:id/acknowledge  // Mark as acted on

// Gestures
GET    /api/v1/contacts/:id/gestures           // Gesture history for a contact
POST   /api/v1/contacts/:id/gestures           // Log a new gesture
GET    /api/v1/relationships/gestures/budget    // Ethics limits usage per contact

// Seasonal Triggers
GET    /api/v1/relationships/triggers/upcoming  // Triggers firing in next N days
GET    /api/v1/relationships/triggers           // All configured triggers
POST   /api/v1/relationships/triggers           // Create custom trigger

// Personal Notes (quick-log from conversations)
POST   /api/v1/contacts/:id/personal-notes     // Quick personal detail log
// Body: { note: "Mentioned his boat 'Gone Fishin' needs a new motor" }
// Auto-creates interaction of type NOTE and may trigger AI interest extraction

// Multi-threading Analysis
GET    /api/v1/organizations/:id/threading     // How well multi-threaded are we?
// Response: { totalContacts: 5, contactedByReps: 2, singleThreaded: true, riskLevel: 'high' }

// AI Suggestions
GET    /api/v1/relationships/suggestions       // AI outreach suggestions for current user
// Response: [{ contactId, reason, suggestedAction, suggestedTiming, confidence }]
```

### Business Rules

- **BR-REL-001:** Relationship scores are recalculated daily (cron job) and on every new interaction.
- **BR-REL-002:** When a contact's relationship score drops below 40 and was previously above 60, trigger a "relationship at risk" alert to the assigned rep.
- **BR-REL-003:** Decay rate: contacts lose 2 points per week of no interaction (applied daily as ~0.28/day). Decay stops at 0.
- **BR-REL-004:** Interest matching uses fuzzy logic: "bass fishing" matches "fishing" (broad match), "fly fishing" (related), and "bass fishing" (exact). Scoring: exact=100, related=70, broad=40.
- **BR-REL-005:** AI interest extraction from emails/calls requires confidence > 0.7 to auto-add. Below 0.7, queue for rep confirmation.
- **BR-REL-006:** Personal notes logged via quick-add are automatically scanned by AI for extractable interests (background job). Rep is notified: "It looks like Mike is interested in fly fishing — add to his profile?"
- **BR-REL-007:** Briefing cards are cached for 4 hours. If new interactions or life events occur within that window, the cache is invalidated.
- **BR-REL-008:** Seasonal triggers fire notifications to reps at 8 AM local time on `daysBeforeToAlert` days before the trigger date. Only contacts with matching interests AND relationship score > 30 are included.
- **BR-REL-009:** Gesture ethics check is MANDATORY before saving a gesture. If the gesture would exceed the jurisdiction's limit, the save is blocked with an error explaining the limit.
- **BR-REL-010:** When a contact moves to a new organization (detected via LinkedIn or manual update), preserve their interest profile and interaction history. Update the organization link but keep relationship context intact.
- **BR-REL-011:** Multi-threading alert: If an organization has estimated pipeline value > $100K and we're connected to only 1 contact there, surface a "single-threaded risk" warning.
- **BR-REL-012:** Life events with `type = 'birthday'` recur annually. The system auto-creates next year's event when current year's is acknowledged.

### Validation Rules

| Field | Rules |
|-------|-------|
| ContactInterest.name | Required, 1-100 chars |
| ContactInterest.confidence | 0-1 decimal, required |
| ContactInterest.category | Valid InterestCategory enum |
| UserInterest.name | Required, 1-100 chars |
| Gesture.value | Non-negative number |
| Gesture.date | Cannot be in the future by more than 7 days |
| ContactLifeEvent.date | Required |
| PersonalNote.note | Required, 1-2000 chars |
| SeasonalTrigger.triggerDate | Valid date format (MM-DD or cron-like) |
| SeasonalTrigger.daysBeforeToAlert | 1-90 |

## Implementation Guide

### File Locations
- `apps/api/src/modules/relationships/` — All relationship intelligence routes/services
  - `relationships.routes.ts` — REST endpoints
  - `relationships.service.ts` — Core business logic
  - `scoring.service.ts` — Relationship score calculation
  - `matching.service.ts` — Interest matching and reverse search
  - `briefing.service.ts` — Briefing card generation
  - `triggers.service.ts` — Seasonal trigger processing
- `apps/api/src/jobs/relationship-decay.job.ts` — Daily decay calculation
- `apps/api/src/jobs/interest-extraction.job.ts` — AI extraction from interactions
- `apps/api/src/jobs/life-event-scan.job.ts` — Scan LinkedIn for life events
- `apps/api/src/jobs/seasonal-triggers.job.ts` — Daily trigger check
- `packages/ai/src/prompts/interest-extractor.ts` — Extract interests from text
- `packages/ai/src/prompts/briefing-generator.ts` — Generate conversation starters
- `packages/ai/src/prompts/outreach-suggester.ts` — Suggest reasons to reach out

### Key Dependencies
- `fuse.js` — Fuzzy matching for interest search
- `@aws-sdk/client-bedrock-runtime` — AI for interest extraction, briefing generation
- `date-fns` — Date calculations for decay, triggers, event proximity

### Implementation Order
1. Contact interest CRUD (manual management first)
2. User interest CRUD (rep profiles their interests)
3. Relationship score calculation (the formula)
4. Decay job (daily cron)
5. Basic reverse matching (exact + category match)
6. Briefing card generation (compose context)
7. Gesture tracking with ethics check
8. Life event tracking (manual first)
9. AI interest extraction from interactions (background job)
10. Seasonal trigger system
11. AI-powered outreach suggestions
12. LinkedIn life event monitoring (integration)
13. Advanced fuzzy matching with AI

### AI Prompts

```typescript
// Interest extraction from conversation notes
const INTEREST_EXTRACTOR_PROMPT = `
Analyze this conversation note or email and extract any personal interests, hobbies, 
or life details mentioned about the contact.

Contact: {contactName}
Content: {content}

For each interest found, provide:
- category: one of [sports_team, sports_activity, outdoors, food_drink, music, travel, family, education, community, hobbies, pets, entertainment]
- name: the specific interest (e.g., "bass fishing", "Lakers", "woodworking")
- specifics: any additional detail (e.g., "fishes Lake Havasu on weekends")
- confidence: 0-1 how certain you are this is a genuine interest vs. passing mention

Only extract clear personal interests, not business topics.
Return empty array if no personal interests are found.
`;

// Briefing card conversation starters
const BRIEFING_GENERATOR_PROMPT = `
Generate 3 natural conversation starters for a sales rep about to meet with a contact.

Contact: {contactName}, {title} at {organization}
Their interests: {interests}
Rep's interests: {repInterests}
Shared interests: {sharedInterests}
Recent life events: {lifeEvents}
Last personal notes: {personalNotes}
Days since last contact: {daysSinceContact}

Generate starters that:
1. Feel natural and genuine (not salesy)
2. Reference shared interests or recent events
3. Show you remember past conversations
4. Are appropriate for a professional relationship

Do NOT generate anything about the business/bid/project — only personal rapport.
`;

// Outreach suggestion
const OUTREACH_SUGGESTER_PROMPT = `
Suggest a reason and timing to reach out to this contact who we haven't spoken to in a while.

Contact: {contactName}, {title} at {organization}
Days since contact: {daysSinceContact}
Their interests: {interests}
Upcoming events/seasons: {upcomingEvents}
Our relationship history: {interactionSummary}
Any active opportunities with their org: {opportunities}

Suggest:
1. reason: A natural, non-pushy reason to reach out
2. channel: Best way to reach out (call, email, text, in-person)
3. timing: When to reach out (today, this week, wait for specific event)
4. message_idea: Brief idea for what to say (1-2 sentences)

Prioritize authentic connection over business. The goal is to maintain the relationship.
`;
```

## Testing Requirements

### Unit Tests
- Relationship score calculation with mocked interaction history
- Score decay: 14 days no contact → specific score reduction
- Interest matching: "bass fishing" ↔ "fly fishing" → related match
- Interest matching: "Lakers" ↔ "Lakers" → exact match
- Interest matching: "golf" ↔ "tennis" → broad (same category: sports_activity)
- Ethics check: gesture $40 with limit $50, YTD $20 → allowed ($60 total vs $500 annual)
- Ethics check: gesture $40 with limit $50, occasion limit exceeded → blocked
- Multi-threading: 1 contact at org with $200K pipeline → risk alert
- Seasonal trigger: baseball fan + opening day in 5 days → notification generated
- Birthday recurrence: acknowledge 2025 birthday → 2026 birthday auto-created

### Integration Tests
- Full briefing card generation (DB queries + AI call)
- Interest extraction pipeline: log interaction → AI extracts interest → rep confirms → added to profile
- Reverse match with fuzzy matching across 100 contacts
- Gesture creation with ethics check against jurisdiction rules
- Score recalculation after new interaction logged

### E2E Scenarios
1. **Pre-meeting briefing:** Rep has call scheduled → opens contact → sees briefing card with interests, shared hobbies, "ask about the fishing trip he mentioned last month"
2. **Reverse interest search:** Rep loves fishing → searches "fishing" → sees 8 contacts who fish → picks one he hasn't talked to in 45 days → system suggests "Hey Mike, the trout are starting to run — been out lately?"
3. **Interest capture:** Rep has call, notes "talked about his daughter's softball team" → AI extracts interest (sports_activity: softball, family: daughter plays softball) → rep confirms → next briefing card includes this

### Seed Data
```typescript
const sampleInterests = [
  // User (sales rep) interests
  { userId: 'rep-alex', category: 'outdoors', name: 'bass fishing', specifics: 'Lakes in Southern California' },
  { userId: 'rep-alex', category: 'sports_team', name: 'Lakers', specifics: 'Season ticket holder' },
  { userId: 'rep-alex', category: 'food_drink', name: 'BBQ', specifics: 'Competition smoker' },

  // Contact interests
  { contactId: 'mike-facility-dir', category: 'outdoors', name: 'fly fishing', specifics: 'Kern River, ties his own flies', confidence: 1.0, source: 'direct_conversation' },
  { contactId: 'mike-facility-dir', category: 'sports_team', name: 'Lakers', specifics: null, confidence: 0.8, source: 'social_media' },
  { contactId: 'mike-facility-dir', category: 'family', name: 'daughter in college', specifics: 'Started at ASU Fall 2024', confidence: 1.0, source: 'direct_conversation' },

  { contactId: 'sarah-purchasing', category: 'sports_activity', name: 'golf', specifics: 'Plays at Indian Wells', confidence: 1.0, source: 'direct_conversation' },
  { contactId: 'sarah-purchasing', category: 'community', name: 'Rotary Club', specifics: 'President of local chapter', confidence: 0.9, source: 'linkedin' },
  { contactId: 'sarah-purchasing', category: 'travel', name: 'Hawaii', specifics: 'Goes to Maui every March', confidence: 0.8, source: 'direct_conversation' },
];

const sampleLifeEvents = [
  { contactId: 'mike-facility-dir', type: 'birthday', date: '1972-08-15', source: 'manual_entry' },
  { contactId: 'mike-facility-dir', type: 'work_anniversary', date: '2018-06-01', description: 'Started at Riverside USD' },
  { contactId: 'sarah-purchasing', type: 'promotion', date: '2024-11-15', description: 'Promoted to Director of Purchasing', source: 'linkedin' },
];
```

## Error Handling

| Failure | Handling |
|---------|----------|
| AI interest extraction fails | Log error, keep interaction without extracted interests. Don't retry automatically (costs). |
| AI briefing generation fails | Show briefing card with factual data only (interests, last contact, scores) — skip AI-generated starters |
| LinkedIn API rate limited | Queue life event scan for retry in 1 hour. Show "last updated" timestamp on data. |
| Fuzzy match returns too many results | Cap at 50 results, sort by match quality. Show "50 of 200 matches shown" |
| Relationship score calculation timeout | Use last cached score. Flag for re-calculation. |

## UI/UX Requirements

### Contact Detail — Relationship Tab
- **Relationship health ring** — Visual 0-100 with color (green >70, yellow 40-70, red <40)
- **Trend arrow** — Up/down/flat with duration
- **Score breakdown** — Expandable factors showing what's strong/weak
- **Interests section** — Tagged chips showing interests with source indicators
- **"Add Personal Note" quick-action** — One-click to log something personal learned
- **Life events timeline** — Upcoming and past events with acknowledge buttons
- **Gesture history** — List with ethics budget meter showing usage vs. limit

### Briefing Card (Pre-interaction popup)
- Triggered when rep opens a contact they have a meeting with today
- Shows: photo, name, title, org, relationship score
- **"What you know" section:** Top 3 interests, family details, last personal notes
- **"What's new" section:** Recent life events, interests updates
- **"In common" section:** Shared interests highlighted
- **"Conversation starters" section:** 3 AI-generated openers
- **"Last time" section:** Summary of most recent interaction + personal notes from it
- Dismissable but auto-appears for scheduled meetings

### Reverse Match View
- Left panel: user's interests as selectable chips
- Right panel: matching contacts with match quality indicators
- Each contact shows: name, org, interest match, relationship score, days since contact
- "Suggest outreach" button per contact → generates AI suggestion
- Batch action: "Create touchpoint task for all selected"

### Relationship Dashboard (Manager View)
- **Team heatmap:** Reps × Organizations, color = relationship score
- **Single-threaded risk list:** Orgs where we know only 1 person
- **Decay alert list:** Key contacts trending down
- **Interaction volume:** Interactions per rep per week (trend line)
- **Win correlation:** Chart showing relationship score at bid time vs. win/loss

### At-Risk Relationships View (Rep)
- Sorted by urgency (combination of decay speed + account importance)
- Each entry shows: contact, org, days since contact, score trend, pipeline at risk
- Quick action: "Log a call", "Send an email", "Schedule meeting"
- AI suggestion for each: "Mike mentioned a fishing trip last time — ask how it went"

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| LinkedIn API | Profile data, job changes, interests from activity | Inbound | Weekly per contact |
| Facebook (public posts) | Interest signals from public posts | Inbound | Weekly |
| Email integration | Scan for personal details in correspondence | Inbound | On new email sync |
| Calendar integration | Trigger briefing cards for upcoming meetings | Inbound | Real-time |
| Local news APIs | Community involvement, achievements, mentions | Inbound | Daily |
| AI (Bedrock) | Interest extraction, briefing generation, outreach suggestions | Outbound | On-demand + batch |

## Performance Requirements

- Briefing card generation: < 3 seconds (AI call included)
- Reverse interest match across 5,000 contacts: < 1 second
- Relationship score recalculation (single contact): < 200ms
- Daily batch score recalculation (all contacts): < 5 minutes
- Interest search autocomplete: < 100ms

## Non-Functional Requirements

- Personal interest data must be treated as sensitive (not exported without explicit permission)
- Interest confidence scores decay over time if not re-confirmed (0.01/month for AI-inferred)
- All AI-inferred interests must be flagged as such (users can see source and confidence)
- Gesture data must be retained for 7 years (government ethics record-keeping)
- Relationship score history must be preserved (for trend analysis over months/years)
- Mobile app must show briefing cards offline (cached for today's meetings)

## Resolved Design Decisions

- **Social media scanning:** Fully manual. Reps enter interests from conversations. AI extracts from email/call content (with consent via company email). No social media scraping.
- **Client-facing profiles:** No. All relationship intelligence is strictly internal. Contacts never see their profile or score.
- **Opt-out mechanism:** Add `doNotTrack: boolean` field on Contact. When true: no AI extraction on their communications, no interest tracking, no gesture logging. Relationship score still calculated from interaction frequency only.
- **Data enrichment services:** Not for MVP. Manual enrichment only. These services are expensive ($100+/month) and data quality varies for government contacts.
- **Interest matching granularity:** Same category = "related" match (e.g., Lakers ↔ Celtics both under `sports_team`). The system suggests conversation starters that acknowledge the match without assuming alignment ("I see you're into basketball too — are you following the playoffs?"). Antagonistic matchups are fine — friendly rivalry is still a conversation starter.

# Complete Prisma Schema

This document provides the complete Prisma schema for Salescraft, expanding on the partial schema shown in `spec/14-data-architecture.md`. This is the authoritative persistence contract for implementation. If an earlier spec shows a simplified TypeScript interface or partial Prisma model that conflicts with this file, use this file.

Design decisions encoded in this schema:

- **Normalized models** for all first-class entities with their own lifecycle and query patterns
- **Company profile vs. settings split** — Company stores legal/identity data; CompanySettings stores operational defaults and JSON configuration
- **JSON columns** for embedded arrays/objects that are always read/written with their parent (e.g., Bid.documents, Product.specifications)
- **String arrays** (native PostgreSQL) for simple tag/ID lists
- **Decimal type** for all monetary amounts to avoid floating-point precision issues
- **Soft delete** (`deletedAt`) on all main business entities
- **UUID primary keys** across all tables

```prisma
// packages/database/prisma/schema.prisma

generator client {
  provider        = "prisma-client-js"
  previewFeatures = ["postgresqlExtensions"]
}

datasource db {
  provider   = "postgresql"
  url        = env("DATABASE_URL")
  extensions = [pgvector, pg_trgm, uuid_ossp, unaccent]
}

// ═══════════════════════════════════════════
// ENUMS
// ═══════════════════════════════════════════

enum UserRole {
  owner
  sales_manager
  sales_rep
  estimator
  project_manager
  installer
  admin
}

enum OrganizationType {
  school_district
  charter_school
  community_college
  university
  city
  county
  state_agency
  federal
  special_district
}

enum FacilityType {
  elementary_school
  middle_school
  high_school
  admin_building
  library
  community_center
  city_hall
  fire_station
  police_station
  recreation_center
  courthouse
  public_works
  other
}

enum ContactRole {
  facility_director
  maintenance_supervisor
  purchasing_agent
  cfo
  superintendent
  board_member
  city_manager
  public_works_director
  architect
  general_contractor
  principal
  other
}

enum DecisionAuthority {
  decision_maker
  influencer
  budget_holder
  gatekeeper
  end_user
  champion
}

enum InterestCategory {
  sports_team
  sports_activity
  outdoors
  food_drink
  music
  travel
  family
  education
  community
  hobbies
  pets
  entertainment
}

enum InterestSource {
  direct_conversation
  social_media
  ai_inferred
  news_mention
  manual_entry
  referral
}

enum LifeEventType {
  birthday
  work_anniversary
  promotion
  new_role
  retirement
  award
  child_milestone
  personal_achievement
  elected
  term_ended
}

enum InteractionType {
  email
  phone_call
  video_call
  in_person_meeting
  site_visit
  pre_bid_meeting
  trade_show
  school_board_meeting
  city_council_meeting
  lunch_dinner
  text_sms
  linkedin_message
  note
}

enum GestureType {
  gift
  meal
  event_tickets
  handwritten_note
  article_shared
  referral_given
  congratulations
  donation
}

enum FlooringType {
  lvt
  lvp
  vct
  sheet_vinyl
  carpet_tile
  broadloom_carpet
  rubber
  linoleum
  epoxy
  polished_concrete
  hardwood
  laminate
  ceramic_tile
  porcelain_tile
}

enum OpportunityStatus {
  signal
  researching
  qualified
  engaging
  bid_expected
  bid_posted
  disqualified
  lost
}

enum OpportunitySource {
  bond_measure
  capital_improvement_plan
  meeting_agenda
  building_age
  bid_board
  relationship
  architect_project
  news_article
  manual
}

enum BidType {
  ifb
  rfp
  rfq
  cooperative
  sole_source
}

enum BidStatus {
  discovered
  reviewing
  decided_no_bid
  preparing
  estimating
  internal_review
  submitted
  under_evaluation
  awarded_won
  awarded_lost
  cancelled
  protested
}

enum BidDecision {
  pending
  bid
  no_bid
}

enum BidResult {
  won
  lost
  cancelled
  no_award
}

enum EstimateStatus {
  draft
  in_progress
  ready_for_review
  approved
  rejected
  submitted
}

enum ProjectStatus {
  awarded
  contracting
  material_order
  scheduled
  in_progress
  punch_list
  closeout
  complete
  warranty
  closed
  on_hold
}

enum PunchListStatus {
  open
  in_progress
  completed
  verified
  disputed
}

enum SignalType {
  bid_posting
  bond_measure
  cip_entry
  meeting_agenda_item
  news_article
  job_posting
  architect_project
  building_permit
  budget_approval
}

enum CallOutcome {
  connected
  voicemail
  no_answer
  busy
  wrong_number
  left_message
}

enum PushPlatform {
  ios
  android
}

enum ComplianceDocType {
  contractor_license
  general_liability
  workers_comp
  auto_insurance
  umbrella_policy
  bond_capacity_letter
  w9
  dbe_certification
  mbe_certification
  wbe_certification
  dvbe_certification
  dir_registration
  sam_registration
  business_license
  safety_certification
}

enum VendorRegistrationStatus {
  active
  pending
  expired
  rejected
}

enum SourceType {
  bid_board
  school_board
  city_council
  bond_tracker
  cip_database
  building_permits
  news
  architect_registry
}

enum BidRecommendation {
  bid
  no_bid
  discuss
}

enum CalendarEntryType {
  pre_bid_meeting
  questions_deadline
  submission_deadline
  award_date
  addendum_issued
  internal_review
  custom
}

enum CrewRole {
  lead
  journeyman
  apprentice
}

enum MilestoneType {
  ntp
  submittal_due
  submittal_approved
  material_order
  material_delivery
  mobilization
  phase_start
  phase_complete
  substantial_completion
  punch_list_walk
  punch_list_complete
  closeout_submitted
  final_payment
  warranty_start
  warranty_end
  custom
}

enum MilestoneStatus {
  pending
  completed
  overdue
  skipped
}

// ═══════════════════════════════════════════
// AUTH & USERS
// ═══════════════════════════════════════════

model User {
  id            String    @id @default(uuid())
  email         String    @unique
  passwordHash  String
  firstName     String
  lastName      String
  role          UserRole
  phone         String?
  avatarUrl     String?
  isActive      Boolean   @default(true)
  lastLoginAt   DateTime?
  createdAt     DateTime  @default(now())
  updatedAt     DateTime  @updatedAt
  deletedAt     DateTime?

  // Relations
  territories          Territory[]          @relation("UserTerritories")
  interests            UserInterest[]
  interactions         Interaction[]
  assignedContacts     Contact[]            @relation("AssignedContacts")
  lastContactedFor     Contact[]            @relation("LastContactedBy")
  assignedBids         Bid[]                @relation("AssignedBids")
  estimatedBids        Bid[]                @relation("EstimatedBids")
  decidedBids          Bid[]                @relation("DecidedBids")
  managedProjects      Project[]            @relation("ManagedProjects")
  salesProjects        Project[]            @relation("SalesProjects")
  dailyLogs            DailyLog[]
  notifications        Notification[]
  emailAccounts        EmailAccount[]
  callLogs             CallLog[]
  gestures             Gesture[]
  sessions             Session[]
  pushTokens           UserPushToken[]
  invitationsSent      Invitation[]         @relation("InvitedByUser")
  passwordResetTokens  PasswordResetToken[]
  assignedOpportunities Opportunity[]       @relation("AssignedOpportunities")
  dismissedSignals     IntelligenceSignal[] @relation("DismissedSignals")
  assignedPunchList    PunchListItem[]      @relation("AssignedPunchList")
  reportedPunchList    PunchListItem[]      @relation("ReportedPunchList")
  estimates            Estimate[]           @relation("EstimatorEstimates")
  reviewedEstimates    Estimate[]           @relation("ReviewedEstimates")
  createdTemplates     CommunicationTemplate[]
  acknowledgedEvents   ContactLifeEvent[]   @relation("AcknowledgedByUser")
  auditEntries         AuditEntry[]
  aiUsageLogs          AIUsageLog[]
  createdOrganizations Organization[]       @relation("CreatedByUser")
  createdContacts      Contact[]            @relation("CreatedByUser")
  updatedCompanySettings CompanySettings[]   @relation("CompanySettingsUpdatedBy")
  crewAssignments      CrewAssignment[]
  assignedMilestones   ProjectMilestone[]    @relation("AssignedMilestones")
  submittedBidMatrices BidDecisionMatrix[]   @relation("SubmittedBidMatrices")
  approvedBidMatrices  BidDecisionMatrix[]   @relation("ApprovedBidMatrices")
  completedChecklistItems ChecklistItem[]     @relation("CompletedChecklistItems")
  uploadedComplianceDocuments ComplianceDocument[] @relation("UploadedComplianceDocuments")

  @@index([email])
  @@index([role])
}

model Session {
  id         String   @id @default(uuid())
  userId     String
  tokenHash  String
  userAgent  String?
  ipAddress  String?
  expiresAt  DateTime
  createdAt  DateTime @default(now())
  lastUsedAt DateTime @default(now())

  // Relations
  user User @relation(fields: [userId], references: [id])

  @@index([userId])
  @@index([expiresAt])
}

model UserPushToken {
  id        String       @id @default(uuid())
  userId    String
  token     String       @unique
  platform  PushPlatform
  createdAt DateTime     @default(now())
  lastUsedAt DateTime?
  revokedAt DateTime?

  // Relations
  user User @relation(fields: [userId], references: [id])

  @@index([userId])
  @@index([revokedAt])
}

model Invitation {
  id          String   @id @default(uuid())
  email       String
  firstName   String
  lastName    String
  role        UserRole
  territories String[] @default([])
  tokenHash   String
  status      String   @default("pending") // pending, accepted, expired, revoked
  expiresAt   DateTime
  invitedById String
  acceptedAt  DateTime?
  createdAt   DateTime @default(now())

  // Relations
  invitedBy User @relation("InvitedByUser", fields: [invitedById], references: [id])

  @@index([email])
  @@index([tokenHash])
}

model PasswordResetToken {
  id        String    @id @default(uuid())
  userId    String
  tokenHash String
  expiresAt DateTime
  usedAt    DateTime?
  createdAt DateTime  @default(now())

  // Relations
  user User @relation(fields: [userId], references: [id])

  @@index([tokenHash])
  @@index([userId])
}

// ═══════════════════════════════════════════
// COMPANY
// ═══════════════════════════════════════════

model Company {
  id              String   @id @default(uuid())
  name            String
  emailDomain     String
  phone           String?
  street1         String?
  street2         String?
  city            String?
  state           String?
  zip             String?
  logoUrl         String?
  bondingCapacity Decimal?
  licenseNumbers  Json     @default("[]") // Array of LicenseEntry objects
  insuranceSummary String?
  taxId           String?
  createdAt       DateTime @default(now())
  updatedAt       DateTime @updatedAt

  // Relations
  settings CompanySettings?
}

model CompanySettings {
  id                        String   @id @default(uuid())
  companyId                 String   @unique
  defaultOverheadPct        Decimal  @default(15)
  defaultProfitPct          Decimal  @default(10)
  minimumProfitPct          Decimal  @default(5)
  defaultWasteFactor        Decimal  @default(10)
  standardLaborHourlyRate   Decimal  @default(45)
  giftLimitDefault          Decimal  @default(50)
  fiscalYearStartMonth      Int      @default(7)
  scoringWeights            Json     @default("{}") // Opportunity and relationship scoring weights
  aiConfig                  Json     @default("{}") // Daily/monthly budgets, critical tasks, enabled models
  notificationDefaults      Json     @default("{}")
  timezone                  String   @default("America/Los_Angeles")
  workingHours              Json     @default("{}")
  updatedAt                 DateTime @updatedAt
  updatedById               String?

  // Relations
  company   Company @relation(fields: [companyId], references: [id])
  updatedBy User?   @relation("CompanySettingsUpdatedBy", fields: [updatedById], references: [id])
}

// ═══════════════════════════════════════════
// GOVERNMENT PROCUREMENT & COMPLIANCE
// ═══════════════════════════════════════════

model ComplianceDocument {
  id              String            @id @default(uuid())
  type            ComplianceDocType
  name            String
  issuer          String?
  documentNumber  String?
  issueDate       DateTime
  expirationDate  DateTime?
  fileUrl         String
  isActive        Boolean           @default(true)
  alertDaysBefore Int               @default(30)
  notes           String?
  uploadedById    String?
  createdAt       DateTime          @default(now())
  updatedAt       DateTime          @updatedAt
  deletedAt       DateTime?

  // Relations
  uploadedBy User? @relation("UploadedComplianceDocuments", fields: [uploadedById], references: [id])

  @@index([type])
  @@index([expirationDate])
  @@index([isActive])
}

model VendorRegistration {
  id                 String                   @id @default(uuid())
  organizationId     String?
  portalName         String
  portalUrl          String?
  username           String?
  registrationNumber String?
  status             VendorRegistrationStatus @default(pending)
  registeredAt       DateTime?
  expirationDate     DateTime?
  categories         String[]                 @default([])
  notes              String?
  createdAt          DateTime                 @default(now())
  updatedAt          DateTime                 @updatedAt
  deletedAt          DateTime?

  // Relations
  organization Organization? @relation(fields: [organizationId], references: [id])

  @@index([organizationId])
  @@index([portalName])
  @@index([status])
  @@index([expirationDate])
}

model CooperativeContract {
  id                  String   @id @default(uuid())
  programName         String
  contractNumber      String
  manufacturer        String
  productCategories   String[] @default([])
  startDate           DateTime
  endDate             DateTime
  discountStructure   String?
  maxOrderValue       Decimal?
  participatingStates String[] @default([])
  website             String?
  isActive            Boolean  @default(true)
  notes               String?
  createdAt           DateTime @default(now())
  updatedAt           DateTime @updatedAt
  deletedAt           DateTime?

  @@index([programName])
  @@index([contractNumber])
  @@index([manufacturer])
  @@index([isActive])
}

model JurisdictionRule {
  id               String   @id @default(uuid())
  jurisdictionName String
  jurisdictionType String   // 'state' | 'county' | 'city' | 'school_district' | 'special_district'
  organizationId   String?
  rules            Json     // ProcurementRules object
  ethicsLimits     Json     // EthicsLimits object
  lastVerified     DateTime
  sourceUrl        String?
  notes            String?
  createdAt        DateTime @default(now())
  updatedAt        DateTime @updatedAt
  deletedAt        DateTime?

  // Relations
  organization Organization? @relation(fields: [organizationId], references: [id])

  @@index([organizationId])
  @@index([jurisdictionType])
  @@index([jurisdictionName])
}

model WageDetermination {
  id                  String   @id @default(uuid())
  state               String
  county              String
  tradeClassification String
  journeymanRate      Decimal
  fringeRate          Decimal
  totalRate           Decimal
  overtimeRate        Decimal
  effectiveDate       DateTime
  expirationDate      DateTime?
  source              String   // 'davis_bacon' | 'state'
  determinationNumber String?
  lastUpdated         DateTime @updatedAt

  @@index([state, county, tradeClassification])
  @@index([expirationDate])
}

// ═══════════════════════════════════════════
// CONTACTS & ORGANIZATIONS
// ═══════════════════════════════════════════

model Organization {
  id                   String           @id @default(uuid())
  name                 String
  type                 OrganizationType
  subType              String?
  website              String?
  phone                String?
  street1              String?
  street2              String?
  city                 String?
  state                String?
  zip                  String?
  latitude             Float?
  longitude            Float?
  fiscalYearStart      Int              @default(7)
  annualBudget         Decimal?
  purchasingThreshold  Decimal          @default(50000)
  cooperativeContracts String[]         @default([])
  approvedVendor       Boolean          @default(false)
  approvedVendorExpiry DateTime?
  notes                String?
  tags                 String[]         @default([])
  createdAt            DateTime         @default(now())
  updatedAt            DateTime         @updatedAt
  deletedAt            DateTime?
  createdById          String

  // Relations
  createdBy     User          @relation("CreatedByUser", fields: [createdById], references: [id])
  facilities    Facility[]
  contacts      Contact[]
  opportunities Opportunity[]
  bids          Bid[]
  projects      Project[]
  vendorRegistrations VendorRegistration[]
  jurisdictionRules   JurisdictionRule[]
  signals       IntelligenceSignal[]

  @@index([name])
  @@index([type])
  @@index([state, city])
}

model Facility {
  id                  String       @id @default(uuid())
  organizationId      String
  name                String
  type                FacilityType
  street1             String?
  street2             String?
  city                String?
  state               String?
  zip                 String?
  latitude            Float?
  longitude           Float?
  yearBuilt           Int?
  totalSqFt           Int?
  flooringSqFt        Int?
  lastFlooringProject DateTime?
  lastFlooringType    String?
  conditionRating     Int?
  notes               String?
  createdAt           DateTime     @default(now())
  updatedAt           DateTime     @updatedAt
  deletedAt           DateTime?

  // Relations
  organization Organization         @relation(fields: [organizationId], references: [id])
  opportunities Opportunity[]
  signals       IntelligenceSignal[]

  @@index([organizationId])
  @@index([type])
  @@index([state, city])
}

model Contact {
  id                    String           @id @default(uuid())
  organizationId        String?
  firstName             String
  lastName              String
  title                 String?
  role                  ContactRole
  email                 String?
  phone                 String?
  mobile                String?
  linkedinUrl           String?
  decisionAuthority     DecisionAuthority
  assignedToId          String?
  relationshipScore     Int              @default(0)
  lastContactedAt       DateTime?
  lastContactedById     String?
  previousOrganizations Json?            // Array of PreviousOrg objects
  notes                 String?
  tags                  String[]         @default([])
  source                String?
  isActive              Boolean          @default(true)
  createdAt             DateTime         @default(now())
  updatedAt             DateTime         @updatedAt
  deletedAt             DateTime?
  createdById           String

  // Relations
  organization  Organization?     @relation(fields: [organizationId], references: [id])
  assignedTo    User?             @relation("AssignedContacts", fields: [assignedToId], references: [id])
  lastContactedBy User?           @relation("LastContactedBy", fields: [lastContactedById], references: [id])
  createdBy     User              @relation("CreatedByUser", fields: [createdById], references: [id])
  interests     ContactInterest[]
  lifeEvents    ContactLifeEvent[]
  interactions  Interaction[]
  gestures      Gesture[]
  emails        EmailMessage[]
  callLogs      CallLog[]

  // Full-text search
  @@index([firstName, lastName])
  @@index([organizationId])
  @@index([assignedToId])
  @@index([relationshipScore(sort: Desc)])
  @@index([lastContactedAt])
  @@index([email])
}

model ContactInterest {
  id              String           @id @default(uuid())
  contactId       String
  category        InterestCategory
  name            String
  specifics       String?
  confidence      Float            @default(1.0)
  source          InterestSource
  sourceDetail    String?
  lastConfirmedAt DateTime?
  createdAt       DateTime         @default(now())
  updatedAt       DateTime         @updatedAt

  // Relations
  contact Contact @relation(fields: [contactId], references: [id])

  @@index([contactId])
  @@index([category, name])
}

model ContactLifeEvent {
  id              String        @id @default(uuid())
  contactId       String
  type            LifeEventType
  description     String
  date            DateTime
  source          String
  acknowledged    Boolean       @default(false)
  acknowledgedById String?
  acknowledgedAt  DateTime?
  createdAt       DateTime      @default(now())

  // Relations
  contact        Contact @relation(fields: [contactId], references: [id])
  acknowledgedBy User?   @relation("AcknowledgedByUser", fields: [acknowledgedById], references: [id])

  @@index([contactId])
  @@index([date])
}

model UserInterest {
  id        String           @id @default(uuid())
  userId    String
  category  InterestCategory
  name      String
  specifics String?
  isPublic  Boolean          @default(true)
  createdAt DateTime         @default(now())

  // Relations
  user User @relation(fields: [userId], references: [id])

  @@index([userId])
  @@index([category, name])
}

model Interaction {
  id               String          @id @default(uuid())
  contactId        String
  userId           String
  type             InteractionType
  direction        String          // 'inbound' | 'outbound'
  subject          String?
  summary          String?
  details          String?
  personalNotes    String?
  duration         Int?            // Minutes
  sentiment        String?         // 'positive' | 'neutral' | 'negative'
  nextSteps        String?
  nextStepDueDate  DateTime?
  relatedBidId     String?
  relatedProjectId String?
  createdAt        DateTime        @default(now())

  // Relations
  contact        Contact  @relation(fields: [contactId], references: [id])
  user           User     @relation(fields: [userId], references: [id])
  relatedBid     Bid?     @relation(fields: [relatedBidId], references: [id])
  relatedProject Project? @relation(fields: [relatedProjectId], references: [id])
  callLog        CallLog?
  emailMessages  EmailMessage[]

  @@index([contactId, createdAt(sort: Desc)])
  @@index([userId, createdAt(sort: Desc)])
}

model Gesture {
  id                String     @id @default(uuid())
  contactId         String
  userId            String
  type              GestureType
  description       String
  value             Decimal?
  date              DateTime
  occasion          String?
  reaction          String?
  ethicsCleared     Boolean    @default(false)
  jurisdictionLimit Decimal?
  createdAt         DateTime   @default(now())

  // Relations
  contact Contact @relation(fields: [contactId], references: [id])
  user    User    @relation(fields: [userId], references: [id])

  @@index([contactId])
  @@index([userId])
}

// ═══════════════════════════════════════════
// PIPELINE & OPPORTUNITIES
// ═══════════════════════════════════════════

model Opportunity {
  id                String            @id @default(uuid())
  facilityId        String?
  organizationId    String
  title             String
  status            OpportunityStatus
  source            OpportunitySource
  sourceDetail      String?
  estimatedValue    Decimal?
  estimatedSqFt     Int?
  estimatedTimeline String?
  flooringTypes     String[]          @default([])
  score             Int               @default(0)
  scoreFactors      Json?             // Array of ScoreFactor objects
  assignedToId      String?
  notes             String?
  discoveredAt      DateTime          @default(now())
  bidExpectedBy     DateTime?
  createdAt         DateTime          @default(now())
  updatedAt         DateTime          @updatedAt
  deletedAt         DateTime?

  // Relations
  facility     Facility?    @relation(fields: [facilityId], references: [id])
  organization Organization @relation(fields: [organizationId], references: [id])
  assignedTo   User?        @relation("AssignedOpportunities", fields: [assignedToId], references: [id])
  relatedBid   Bid?         @relation("OpportunityBid")
  estimates    Estimate[]   @relation("OpportunityEstimates")
  signals      IntelligenceSignal[]

  @@index([organizationId])
  @@index([status])
  @@index([score(sort: Desc)])
}

// ═══════════════════════════════════════════
// BIDS
// ═══════════════════════════════════════════

model Bid {
  id                      String      @id @default(uuid())
  opportunityId           String?     @unique
  organizationId          String
  facilityIds             String[]    @default([])
  title                   String
  bidNumber               String?
  type                    BidType
  status                  BidStatus
  source                  String
  sourceUrl               String?
  description             String?
  estimatedValue          Decimal?
  publishedAt             DateTime
  preBidMeetingAt         DateTime?
  preBidMeetingLocation   String?
  preBidMeetingMandatory  Boolean     @default(false)
  questionsDeadline       DateTime?
  submissionDeadline      DateTime
  awardDate               DateTime?
  bondRequired            Boolean     @default(false)
  bondPercentage          Decimal?
  prevailingWage          Boolean     @default(false)
  wageCounty              String?
  insuranceRequirements   String?
  documents               Json?       // Array of BidDocument objects
  addenda                 Json?       // Array of Addendum objects
  decision                BidDecision @default(pending)
  decisionReason          String?
  decisionById            String?
  assignedToId            String?
  estimatorId             String?
  submittedAt             DateTime?
  submittedAmount         Decimal?
  result                  BidResult?
  resultReason            String?
  winningAmount           Decimal?
  winningCompany          String?
  createdAt               DateTime    @default(now())
  updatedAt               DateTime    @updatedAt
  deletedAt               DateTime?

  // Relations
  opportunity  Opportunity?  @relation("OpportunityBid", fields: [opportunityId], references: [id])
  organization Organization  @relation(fields: [organizationId], references: [id])
  decisionBy   User?         @relation("DecidedBids", fields: [decisionById], references: [id])
  assignedTo   User?         @relation("AssignedBids", fields: [assignedToId], references: [id])
  estimator    User?         @relation("EstimatedBids", fields: [estimatorId], references: [id])
  estimate     Estimate?     @relation("BidEstimate")
  project      Project?      @relation("BidProject")
  interactions Interaction[]
  decisionMatrices BidDecisionMatrix[]
  submissionChecklist SubmissionChecklist?
  calendarEntries BidCalendarEntry[]
  winLossRecord WinLossRecord?

  @@index([submissionDeadline])
  @@index([status])
  @@index([organizationId])
  @@index([opportunityId])
  @@index([bidNumber])
}

model BidDecisionMatrix {
  id            String            @id @default(uuid())
  bidId         String
  factors       Json              // Array of BidDecisionFactor objects
  totalScore    Int
  recommendation BidRecommendation
  submittedById String
  approvedById  String?
  approvalNotes String?
  createdAt     DateTime          @default(now())
  updatedAt     DateTime          @updatedAt

  // Relations
  bid         Bid   @relation(fields: [bidId], references: [id])
  submittedBy User  @relation("SubmittedBidMatrices", fields: [submittedById], references: [id])
  approvedBy  User? @relation("ApprovedBidMatrices", fields: [approvedById], references: [id])

  @@index([bidId])
  @@index([submittedById])
}

model SubmissionChecklist {
  id                   String   @id @default(uuid())
  bidId                String   @unique
  completionPercentage Int      @default(0)
  isComplete           Boolean  @default(false)
  lastUpdatedAt        DateTime @updatedAt

  // Relations
  bid   Bid             @relation(fields: [bidId], references: [id])
  items ChecklistItem[]
}

model ChecklistItem {
  id          String   @id @default(uuid())
  checklistId String
  category    String   // 'document' | 'form' | 'bond' | 'signature' | 'addendum' | 'other'
  description String
  required    Boolean  @default(true)
  completed   Boolean  @default(false)
  completedAt DateTime?
  completedById String?
  attachmentUrl String?
  notes       String?
  dueDate     DateTime?

  // Relations
  checklist   SubmissionChecklist @relation(fields: [checklistId], references: [id])
  completedBy User?               @relation("CompletedChecklistItems", fields: [completedById], references: [id])

  @@index([checklistId])
  @@index([completedById])
}

model BidCalendarEntry {
  id           String            @id @default(uuid())
  bidId        String
  type         CalendarEntryType
  title        String
  date         DateTime
  time         String?
  location     String?
  mandatory    Boolean           @default(false)
  reminderDays Int[]             @default([])
  assignedToIds String[]         @default([])
  completed    Boolean           @default(false)
  notes        String?

  // Relations
  bid Bid @relation(fields: [bidId], references: [id])

  @@index([bidId])
  @@index([date])
}

model WinLossRecord {
  id               String    @id @default(uuid())
  bidId            String    @unique
  result           BidResult
  ourAmount        Decimal?
  winningAmount    Decimal?
  winningCompany   String?
  delta            Decimal?
  deltaPercentage  Decimal?
  factors          String[]  @default([])
  debriefNotes     String?
  lessonsLearned   String?
  debriefedById    String?
  debriefDate      DateTime?
  createdAt        DateTime  @default(now())

  // Relations
  bid Bid @relation(fields: [bidId], references: [id])

  @@index([result])
}

// ═══════════════════════════════════════════
// ESTIMATING
// ═══════════════════════════════════════════

model Estimate {
  id                   String         @id @default(uuid())
  bidId                String?        @unique
  opportunityId        String?
  title                String
  status               EstimateStatus @default(draft)
  estimatorId          String
  reviewedById         String?
  version              Int            @default(1)
  materialTotal        Decimal        @default(0)
  laborTotal           Decimal        @default(0)
  equipmentTotal       Decimal        @default(0)
  subcontractorTotal   Decimal        @default(0)
  subtotal             Decimal        @default(0)
  overhead             Decimal        @default(0)
  overheadPercentage   Decimal        @default(15)
  profit               Decimal        @default(0)
  profitPercentage     Decimal        @default(10)
  bondCost             Decimal?
  total                Decimal        @default(0)
  notes                String?
  createdAt            DateTime       @default(now())
  updatedAt            DateTime       @updatedAt
  deletedAt            DateTime?

  // Relations
  bid           Bid?            @relation("BidEstimate", fields: [bidId], references: [id])
  opportunity   Opportunity?    @relation("OpportunityEstimates", fields: [opportunityId], references: [id])
  estimator     User            @relation("EstimatorEstimates", fields: [estimatorId], references: [id])
  reviewedBy    User?           @relation("ReviewedEstimates", fields: [reviewedById], references: [id])
  areas         EstimateArea[]
  takeoffMarkups TakeoffMarkup[]
  versions      EstimateVersion[]

  @@index([status])
  @@index([estimatorId])
  @@index([bidId])
}

model EstimateArea {
  id                  String  @id @default(uuid())
  estimateId          String
  name                String
  sqFt                Decimal
  productId           String?
  productName         String?
  wasteFactor         Decimal @default(10)
  materialSqFt        Decimal @default(0)
  materialCostPerSqFt Decimal @default(0)
  materialTotal       Decimal @default(0)
  laborRatePerSqFt    Decimal @default(0)
  laborTotal          Decimal @default(0)
  additionalMaterials Json?   // Array of LineItem objects
  notes               String?
  createdAt           DateTime @default(now())
  updatedAt           DateTime @updatedAt

  // Relations
  estimate Estimate @relation(fields: [estimateId], references: [id])
  product  Product? @relation(fields: [productId], references: [id])

  @@index([estimateId])
}

model TakeoffMarkup {
  id          String   @id @default(uuid())
  estimateId  String
  pageNumber  Int
  documentUrl String
  annotations Json?    // Array of TakeoffAnnotation objects
  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt

  // Relations
  estimate Estimate @relation(fields: [estimateId], references: [id])

  @@index([estimateId])
}

model EstimateTemplate {
  id                         String   @id @default(uuid())
  name                       String
  description                String?
  projectType                String
  defaultProductId           String?
  defaultWasteFactor         Decimal  @default(10)
  defaultAdditionalMaterials Json?    // Array of LineItem objects
  laborProductivitySqFtPerDay Decimal @default(0)
  notes                      String?
  createdAt                  DateTime @default(now())
  updatedAt                  DateTime @updatedAt

  // Relations
  defaultProduct Product? @relation(fields: [defaultProductId], references: [id])

  @@index([projectType])
}

model Product {
  id                 String      @id @default(uuid())
  manufacturer       String
  productLine        String
  name               String
  type               FlooringType
  subType            String?
  sku                String?
  specifications     Json?       // ProductSpecs object (includes colorOptions)
  pricing            Json?       // ProductPricing object
  warrantyYears      Int?
  adaCompliant       Boolean     @default(false)
  fireRating         String?
  sustainabilityCerts String[]   @default([])
  installationMethod String?
  typicalWasteFactor Decimal     @default(10)
  typicalLaborRate   Decimal     @default(0)
  isActive           Boolean     @default(true)
  createdAt          DateTime    @default(now())
  updatedAt          DateTime    @updatedAt
  deletedAt          DateTime?

  // Relations
  estimateAreas     EstimateArea[]
  estimateTemplates EstimateTemplate[]

  @@index([manufacturer])
  @@index([type])
  @@index([isActive])
}

// ═══════════════════════════════════════════
// PROJECTS
// ═══════════════════════════════════════════

model Project {
  id                    String        @id @default(uuid())
  bidId                 String?       @unique
  organizationId        String
  facilityIds           String[]      @default([])
  title                 String
  contractNumber        String?
  status                ProjectStatus @default(awarded)
  projectManagerId      String
  salesRepId            String
  contractAmount        Decimal       @default(0)
  changeOrderTotal      Decimal       @default(0)
  currentContractAmount Decimal       @default(0)
  startDate             DateTime?
  completionDate        DateTime?
  warrantyEndDate       DateTime?
  ntpDate               DateTime?
  scheduleConstraints   String?
  prevailingWage        Boolean       @default(false)
  wageCounty            String?
  materialOrders        Json?         // Array of MaterialOrder objects
  changeOrders          Json?         // Array of ChangeOrder objects
  documents             Json?         // Array of ProjectDocument objects
  createdAt             DateTime      @default(now())
  updatedAt             DateTime      @updatedAt
  deletedAt             DateTime?

  // Relations
  bid            Bid?            @relation("BidProject", fields: [bidId], references: [id])
  organization   Organization    @relation(fields: [organizationId], references: [id])
  projectManager User            @relation("ManagedProjects", fields: [projectManagerId], references: [id])
  salesRep       User            @relation("SalesProjects", fields: [salesRepId], references: [id])
  dailyLogs      DailyLog[]
  punchListItems PunchListItem[]
  interactions   Interaction[]
  crewAssignments CrewAssignment[]
  milestones      ProjectMilestone[]

  @@index([status])
  @@index([projectManagerId])
  @@index([organizationId])
  @@index([completionDate])
}

model CrewAssignment {
  id        String    @id @default(uuid())
  projectId String
  userId    String
  role      CrewRole
  startDate DateTime
  endDate   DateTime?
  dailyRate Decimal?
  notes     String?

  // Relations
  project Project @relation(fields: [projectId], references: [id])
  user    User    @relation(fields: [userId], references: [id])

  @@index([projectId])
  @@index([userId])
  @@index([startDate, endDate])
}

model ProjectMilestone {
  id          String          @id @default(uuid())
  projectId   String
  title       String
  type        MilestoneType
  plannedDate DateTime
  actualDate  DateTime?
  status      MilestoneStatus @default(pending)
  dependsOnIds String[]       @default([])
  assignedToId String?
  notes       String?

  // Relations
  project    Project @relation(fields: [projectId], references: [id])
  assignedTo User?   @relation("AssignedMilestones", fields: [assignedToId], references: [id])

  @@index([projectId])
  @@index([plannedDate])
  @@index([status])
}

model DailyLog {
  id               String    @id @default(uuid())
  projectId        String
  userId           String
  date             DateTime
  hoursWorked      Decimal
  sqFtInstalled    Decimal   @default(0)
  productInstalled String?
  areasWorked      String[]  @default([])
  crewSize         Int       @default(1)
  weather          String?
  issues           String?
  materialsUsed    String?
  photos           String[]  @default([])
  notes            String?
  syncedAt         DateTime?
  createdAt        DateTime  @default(now())
  deletedAt        DateTime?

  // Relations
  project Project @relation(fields: [projectId], references: [id])
  user    User    @relation(fields: [userId], references: [id])

  @@index([projectId])
  @@index([userId])
  @@index([date])
}

model PunchListItem {
  id               String          @id @default(uuid())
  projectId        String
  location         String
  description      String
  priority         String          // 'critical' | 'major' | 'minor' | 'cosmetic'
  status           PunchListStatus @default(open)
  assignedToId     String?
  reportedById     String
  reportedAt       DateTime
  dueDate          DateTime?
  completedAt      DateTime?
  photos           String[]        @default([])
  completionPhotos String[]        @default([])
  notes            String?
  createdAt        DateTime        @default(now())
  updatedAt        DateTime        @updatedAt
  deletedAt        DateTime?

  // Relations
  project    Project @relation(fields: [projectId], references: [id])
  assignedTo User?   @relation("AssignedPunchList", fields: [assignedToId], references: [id])
  reportedBy User    @relation("ReportedPunchList", fields: [reportedById], references: [id])

  @@index([projectId])
  @@index([status])
  @@index([assignedToId])
}

// ═══════════════════════════════════════════
// INTELLIGENCE
// ═══════════════════════════════════════════

model IntelligenceSignal {
  id              String     @id @default(uuid())
  type            SignalType
  source          String
  title           String
  description     String?
  url             String?
  rawData         Json?
  organizationId  String?
  facilityId      String?
  opportunityId   String?
  processed       Boolean    @default(false)
  dismissed       Boolean    @default(false)
  dismissedById   String?
  createdAt       DateTime   @default(now())
  deletedAt       DateTime?

  // Relations
  organization Organization? @relation(fields: [organizationId], references: [id])
  facility     Facility?     @relation(fields: [facilityId], references: [id])
  opportunity  Opportunity?  @relation(fields: [opportunityId], references: [id])
  dismissedBy  User?         @relation("DismissedSignals", fields: [dismissedById], references: [id])

  @@index([type])
  @@index([processed, dismissed, createdAt(sort: Desc)])
  @@index([organizationId])
}

model MonitoredSource {
  id            String     @id @default(uuid())
  name          String
  type          SourceType
  url           String
  scrapeConfig  Json       // ScrapeConfig object
  schedule      String     // Cron expression
  lastScrapedAt DateTime?
  lastSuccessAt DateTime?
  errorCount    Int        @default(0)
  lastError     String?
  isActive      Boolean    @default(true)
  territoryIds  String[]   @default([])
  createdAt     DateTime   @default(now())
  updatedAt     DateTime   @updatedAt

  @@index([type])
  @@index([isActive])
}

model OpportunityScoreModel {
  id                    String   @id @default(uuid())
  name                  String
  config                Json     // ScoreModelConfig object
  isActive              Boolean  @default(false)
  minimumScoreToSurface Int      @default(30)
  autoAssignThreshold   Int      @default(60)
  createdAt             DateTime @default(now())
  updatedAt             DateTime @updatedAt

  @@index([isActive])
}

// ═══════════════════════════════════════════
// TERRITORIES
// ═══════════════════════════════════════════

model Territory {
  id          String   @id @default(uuid())
  name        String
  description String?
  counties    String[] @default([])
  cities      String[] @default([])
  zipCodes    String[] @default([])
  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt
  deletedAt   DateTime?

  // Relations
  users User[] @relation("UserTerritories")

  @@index([name])
}

// ═══════════════════════════════════════════
// COMMUNICATION
// ═══════════════════════════════════════════

model EmailAccount {
  id           String   @id @default(uuid())
  userId       String
  provider     String   // 'gmail' | 'outlook'
  email        String
  accessToken  String
  refreshToken String
  tokenExpiry  DateTime
  syncState    Json?    // SyncState object (lastSyncedAt, historyId, syncErrors)
  isActive     Boolean  @default(true)
  createdAt    DateTime @default(now())
  updatedAt    DateTime @updatedAt

  // Relations
  user     User           @relation(fields: [userId], references: [id])
  messages EmailMessage[]

  @@index([userId])
  @@index([email])
}

model EmailMessage {
  id              String   @id @default(uuid())
  emailAccountId  String
  externalId      String
  threadId        String?
  interactionId   String?
  contactId       String?
  direction       String   // 'inbound' | 'outbound'
  fromAddress     String
  toAddresses     String[]
  ccAddresses     String[] @default([])
  subject         String
  bodyPreview     String
  bodyHtml        String?
  hasAttachments  Boolean  @default(false)
  attachments     Json?    // Array of EmailAttachment objects
  sentAt          DateTime
  readAt          DateTime?
  labels          String[] @default([])
  aiSummary       String?
  aiSentiment     String?  // 'positive' | 'neutral' | 'negative'
  createdAt       DateTime @default(now())

  // Relations
  emailAccount   EmailAccount       @relation(fields: [emailAccountId], references: [id])
  interaction    Interaction?       @relation(fields: [interactionId], references: [id])
  contact        Contact?           @relation(fields: [contactId], references: [id])
  trackingEvents EmailTrackingEvent[]

  @@unique([emailAccountId, externalId])
  @@index([contactId])
  @@index([threadId])
  @@index([sentAt(sort: Desc)])
}

model CallLog {
  id            String      @id @default(uuid())
  userId        String
  contactId     String
  interactionId String      @unique
  direction     String      // 'inbound' | 'outbound'
  phoneNumber   String
  duration      Int         // Seconds
  outcome       CallOutcome
  notes         String?
  nextSteps     String?
  nextStepDate  DateTime?
  recordingUrl  String?
  transcription String?
  createdAt     DateTime    @default(now())

  // Relations
  user        User        @relation(fields: [userId], references: [id])
  contact     Contact     @relation(fields: [contactId], references: [id])
  interaction Interaction @relation(fields: [interactionId], references: [id])

  @@index([userId])
  @@index([contactId])
  @@index([createdAt(sort: Desc)])
}

model CommunicationTemplate {
  id          String   @id @default(uuid())
  name        String
  category    String
  channel     String   // 'email' | 'sms' | 'linkedin'
  subject     String?
  body        String
  mergeFields String[] @default([])
  isShared    Boolean  @default(false)
  createdById String
  usageCount  Int      @default(0)
  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt

  // Relations
  createdBy User @relation(fields: [createdById], references: [id])

  @@index([category])
  @@index([channel])
}

// ═══════════════════════════════════════════
// INTEGRATIONS & FILES
// ═══════════════════════════════════════════

model IntegrationCredential {
  id              String    @id @default(uuid())
  integrationName String
  userId          String?
  credentialType  String    // 'oauth2' | 'api_key' | 'session' | 'iam'
  data            Json      // Encrypted at rest: accessToken, refreshToken, apiKey, etc.
  status          String    @default("active") // 'active' | 'expired' | 'revoked' | 'error'
  lastUsedAt      DateTime?
  lastErrorAt     DateTime?
  lastError       String?
  createdAt       DateTime  @default(now())
  updatedAt       DateTime  @updatedAt

  @@index([integrationName])
  @@index([userId])
  @@index([status])
}

model FileMetadata {
  id           String   @id @default(uuid())
  bucket       String
  key          String   // S3 key (path)
  filename     String
  mimeType     String
  size         Int      // Bytes
  entityType   String   // 'bid' | 'project' | 'contact' | 'estimate'
  entityId     String
  uploadedById String
  uploadedAt   DateTime @default(now())
  isPublic     Boolean  @default(false)
  thumbnailKey String?

  @@index([entityType, entityId])
  @@index([uploadedById])
}

// ═══════════════════════════════════════════
// NOTIFICATIONS
// ═══════════════════════════════════════════

model Notification {
  id         String   @id @default(uuid())
  userId     String
  type       String   // Event type: 'bid.discovered', 'bid.deadline_approaching', etc.
  title      String
  body       String?
  priority   String   @default("medium") // 'low' | 'medium' | 'high' | 'urgent'
  channels   String[] @default([])       // 'in_app' | 'push' | 'email'
  readAt     DateTime?
  entityType String?  // Related entity type
  entityId   String?  // Related entity ID
  link       String?  // In-app URL path
  createdAt  DateTime @default(now())

  // Relations
  user User @relation(fields: [userId], references: [id])

  @@index([userId, readAt])
  @@index([userId, createdAt(sort: Desc)])
  @@index([entityType, entityId])
}

// ═══════════════════════════════════════════
// AUDIT & AI
// ═══════════════════════════════════════════

model AuditEntry {
  id         String   @id @default(uuid())
  entityType String
  entityId   String
  action     String   // 'create' | 'update' | 'delete' | 'state_change' | 'access'
  actorId    String
  changes    Json?    // { field: { from, to } }
  metadata   Json?
  createdAt  DateTime @default(now())

  // Relations
  actor User @relation(fields: [actorId], references: [id])

  @@index([entityType, entityId])
  @@index([actorId])
  @@index([createdAt(sort: Desc)])
}

model AIUsageLog {
  id           String   @id @default(uuid())
  taskType     String
  modelId      String
  inputTokens  Int
  outputTokens Int
  cost         Decimal
  latencyMs    Int
  success      Boolean
  errorMessage String?
  userId       String?
  moduleSource String
  createdAt    DateTime @default(now())

  // Relations
  user User? @relation(fields: [userId], references: [id])

  @@index([taskType])
  @@index([createdAt(sort: Desc)])
  @@index([userId])
  @@index([moduleSource])
}

// ═══════════════════════════════════════════
// EMBEDDINGS (Vector Search)
// ═══════════════════════════════════════════

model Embedding {
  id          String                    @id @default(uuid())
  entityType  String                    // "contact", "interaction", "bid_document", "email"
  entityId    String
  content     String                    // Original text that was embedded
  contentHash String                    // For dedup/invalidation
  embedding   Unsupported("vector(1024)") // pgvector column (Titan v2 = 1024 dims)
  modelId     String                    // Embedding model used
  createdAt   DateTime                  @default(now())

  @@index([entityType, entityId])
  @@index([contentHash])
}

// ═══════════════════════════════════════════
// ESTIMATE VERSIONING
// ═══════════════════════════════════════════

model EstimateVersion {
  id         String   @id @default(uuid())
  estimateId String
  version    Int
  snapshot   Json     // Full JSON snapshot of the estimate + areas at this point
  createdById String
  createdAt  DateTime @default(now())

  // Relations
  estimate Estimate @relation(fields: [estimateId], references: [id])

  @@unique([estimateId, version])
  @@index([estimateId])
}

// ═══════════════════════════════════════════
// EMAIL TRACKING
// ═══════════════════════════════════════════

model EmailTrackingEvent {
  id             String   @id @default(uuid())
  emailMessageId String
  type           String   // 'open' | 'click'
  linkUrl        String?  // For clicks: the original URL
  ipAddress      String?
  userAgent      String?
  occurredAt     DateTime

  // Relations
  emailMessage EmailMessage @relation(fields: [emailMessageId], references: [id])

  @@index([emailMessageId])
  @@index([occurredAt(sort: Desc)])
}
```

# User Roles & Permissions

## Vision & Purpose

Salescraft serves users from field installers to company owners — each needing different levels of access and different workflows. The permission system ensures that financial data stays with management, field crews only see their assigned projects, and sales reps can't accidentally modify each other's contact relationships.

Since this is a single-tenant system (one company), we don't need complex multi-org hierarchies. But we still need role-based access control that maps to how a flooring company actually operates.

## Key Concepts

- **Role** — A predefined set of permissions assigned to a user (e.g., "sales_rep", "estimator")
- **Permission** — A specific action allowed on a resource (e.g., "bids:write", "financials:read")
- **Resource Ownership** — Some resources have owners (contacts assigned to reps, projects assigned to PMs)
- **Approval Authority** — Who can approve specific actions (bid margins, change orders, etc.)

## Roles and Permissions Matrix

### Role Definitions

| Role | Description | Primary Platform | Users (typical) |
|------|-------------|-----------------|-----------------|
| `owner` | Company owner/president. Full access to everything including financials. | Web | 1-2 |
| `sales_manager` | Manages sales team. Sees all pipeline, approves bids, mentors reps. | Web | 1-2 |
| `sales_rep` | Owns contacts and relationships. Identifies opportunities, responds to bids. | Web + Mobile | 3-8 |
| `estimator` | Builds estimates and proposals. Manages product catalog and pricing. | Web | 2-4 |
| `project_manager` | Manages awarded projects through completion. Coordinates field crews. | Web + Mobile | 2-4 |
| `installer` | Field crew member. Logs daily work, takes photos, handles punch lists. | Mobile only | 10-30 |
| `admin` | Office admin. Handles compliance docs, vendor registrations, data entry. | Web | 1-2 |

### Permission Matrix

| Permission | Owner | Sales Mgr | Sales Rep | Estimator | PM | Installer | Admin |
|-----------|-------|-----------|-----------|-----------|----|-----------| ------|
| **Contacts** | | | | | | | |
| contacts:read (all) | Y | Y | Own territory | Y (bid-related) | Y (project-related) | N | Y |
| contacts:write | Y | Y | Own territory | N | N | N | Y |
| contacts:delete | Y | Y | N | N | N | N | N |
| **Relationships** | | | | | | | |
| relationships:read | Y | Y | Own contacts | N | N | N | N |
| relationships:write | Y | Y | Own contacts | N | N | N | N |
| interests:read | Y | Y | Own contacts | N | N | N | N |
| interests:write | Y | Y | Own contacts | N | N | N | N |
| gestures:read | Y | Y | Own | N | N | N | Y |
| gestures:write | Y | Y | Own | N | N | N | N |
| **Intelligence** | | | | | | | |
| intelligence:read | Y | Y | Own territory | N | N | N | N |
| intelligence:write | Y | Y | Own territory | N | N | N | N |
| opportunities:read | Y | Y | Own territory | Y (assigned) | N | N | N |
| opportunities:write | Y | Y | Own territory | N | N | N | N |
| **Bids** | | | | | | | |
| bids:read | Y | Y | Own territory | Assigned | N | N | Y |
| bids:write | Y | Y | Own territory | Assigned | N | N | Y |
| bids:decide | Y | Y | N | N | N | N | N |
| bids:approve_margin | Y | N (unless delegated) | N | N | N | N | N |
| bids:submit | Y | Y | Y (own) | N | N | N | Y |
| **Estimates** | | | | | | | |
| estimates:read | Y | Y | Y (own bids) | Y (all) | N | N | N |
| estimates:write | Y | N | N | Y | N | N | N |
| estimates:approve | Y | Y | N | N | N | N | N |
| **Projects** | | | | | | | |
| projects:read (all) | Y | Y | Y (own) | Y (related) | Assigned | Assigned | Y |
| projects:write | Y | Y | N | N | Assigned | N | Y |
| projects:financials | Y | Y | N | N | Read only | N | N |
| daily_logs:read | Y | Y | N | N | Assigned | Own | N |
| daily_logs:write | Y | N | N | N | N | Own | N |
| punch_list:read | Y | Y | N | N | Assigned | Assigned | N |
| punch_list:write | Y | N | N | N | Assigned | N | N |
| punch_list:complete | Y | N | N | N | N | Assigned | N |
| punch_list:verify | Y | N | N | N | Assigned | N | N |
| **Products** | | | | | | | |
| products:read | Y | Y | Y | Y | N | N | Y |
| products:write | Y | N | N | Y | N | N | N |
| **Compliance** | | | | | | | |
| compliance:read | Y | Y | Y | Y | N | N | Y |
| compliance:write | Y | N | N | N | N | N | Y |
| **Financials** | | | | | | | |
| financials:read | Y | Y | N | N | Read (own projects) | N | N |
| financials:write | Y | N | N | N | N | N | N |
| **Users** | | | | | | | |
| users:read | Y | Y | N | N | N | N | Y |
| users:write | Y | N | N | N | N | N | N |
| users:create | Y | N | N | N | N | N | N |
| **Settings** | | | | | | | |
| settings:read | Y | Y | N | N | N | N | Y |
| settings:write | Y | N | N | N | N | N | N |
| **AI/Admin** | | | | | | | |
| ai:usage | Y | N | N | N | N | N | N |
| ai:config | Y | N | N | N | N | N | N |
| integrations:manage | Y | N | N | N | N | N | Y |

### Resource Ownership Rules

Some permissions are scoped to "owned" resources:

```typescript
interface OwnershipRule {
  resource: string;
  ownerField: string;             // Which field determines ownership
  rule: string;                   // When ownership grants access
}

const OWNERSHIP_RULES: OwnershipRule[] = [
  { resource: 'contacts', ownerField: 'assignedTo', rule: 'Sales reps can read/write contacts assigned to them' },
  { resource: 'opportunities', ownerField: 'assignedTo', rule: 'Sales reps can manage opportunities in their territory' },
  { resource: 'bids', ownerField: 'assignedTo', rule: 'Sales reps see bids in their territory' },
  { resource: 'estimates', ownerField: 'estimatorId', rule: 'Estimators can edit their assigned estimates' },
  { resource: 'projects', ownerField: 'projectManagerId', rule: 'PMs manage their assigned projects' },
  { resource: 'daily_logs', ownerField: 'userId', rule: 'Installers create/edit their own logs' },
  { resource: 'punch_list_items', ownerField: 'assignedTo', rule: 'Installers see/complete their assigned items' },
];
```

### Approval Workflows

| Action | Required Approver | Condition |
|--------|-------------------|-----------|
| Bid margin below 15% | Owner | profitPercentage < 15 |
| Bid submission | Sales Manager or Owner | Always for formal bids |
| Estimate approval | Sales Manager or Owner | Always |
| Change order > 10% | Owner | changeOrderAmount > contractAmount * 0.10 |
| Contact deletion | Sales Manager or Owner | Always (soft delete) |
| User creation | Owner | Always |
| Gift/gesture over $100 | Sales Manager | value > 100 |

## Technical Design

### API Endpoints

```typescript
// Users (Owner only)
GET    /api/v1/users                           // List all users
POST   /api/v1/users/invite                    // Invite user
GET    /api/v1/users/invitations               // List pending invitations
POST   /api/v1/users/invite/:id/resend         // Resend invitation
DELETE /api/v1/users/invite/:id                // Revoke invitation
GET    /api/v1/users/:id
PUT    /api/v1/users/:id                       // Update user (role, status, territories)
PUT    /api/v1/users/:id/deactivate            // Deactivate (don't delete)
PUT    /api/v1/users/:id/password              // Reset password (owner or self)

// Current User
GET    /api/v1/users/me                        // Get current user profile
PUT    /api/v1/users/me                        // Update own profile (limited fields)
PUT    /api/v1/users/me/password               // Change own password
GET    /api/v1/users/me/permissions            // Get computed permission list
POST   /api/v1/users/me/push-token             // Register mobile Expo push token
DELETE /api/v1/users/me/push-token/:id         // Revoke mobile push token

// Notifications
GET    /api/v1/notifications                   // List for current user
PUT    /api/v1/notifications/:id/read          // Mark read
PUT    /api/v1/notifications/read-all          // Mark all read
GET    /api/v1/notifications/preferences       // Notification settings
PUT    /api/v1/notifications/preferences       // Update preferences
```

### Notification Routing

```typescript
interface NotificationConfig {
  event: string;
  recipients: RecipientRule[];
  channels: ('in_app' | 'push' | 'email')[];
  priority: 'low' | 'medium' | 'high' | 'urgent';
}

const NOTIFICATION_CONFIGS: NotificationConfig[] = [
  {
    event: 'bid.discovered',
    recipients: [{ role: 'sales_rep', filter: 'territory_match' }],
    channels: ['in_app', 'push'],
    priority: 'medium',
  },
  {
    event: 'bid.deadline_approaching',
    recipients: [
      { role: 'sales_rep', filter: 'assigned' },
      { role: 'estimator', filter: 'assigned' },
      { role: 'sales_manager', filter: 'always' },
    ],
    channels: ['in_app', 'push', 'email'],
    priority: 'high',
  },
  {
    event: 'bid.awarded_won',
    recipients: [
      { role: 'sales_rep', filter: 'assigned' },
      { role: 'owner', filter: 'always' },
      { role: 'project_manager', filter: 'assigned' },
    ],
    channels: ['in_app', 'push', 'email'],
    priority: 'high',
  },
  {
    event: 'relationship.decay_alert',
    recipients: [{ role: 'sales_rep', filter: 'assigned' }],
    channels: ['in_app'],
    priority: 'low',
  },
  {
    event: 'daily_log.missing',
    recipients: [{ role: 'installer', filter: 'assigned' }],
    channels: ['push'],
    priority: 'medium',
  },
  {
    event: 'punch_list.new_item',
    recipients: [{ role: 'installer', filter: 'assigned' }],
    channels: ['push'],
    priority: 'medium',
  },
  {
    event: 'project.margin_alert',
    recipients: [{ role: 'owner', filter: 'always' }],
    channels: ['in_app', 'email'],
    priority: 'high',
  },
  {
    event: 'compliance.expiring',
    recipients: [{ role: 'admin', filter: 'always' }],
    channels: ['in_app', 'email'],
    priority: 'medium',
  },
  {
    event: 'life_event.detected',
    recipients: [{ role: 'sales_rep', filter: 'assigned' }],
    channels: ['in_app'],
    priority: 'low',
  },
  {
    event: 'seasonal_trigger.fired',
    recipients: [{ role: 'sales_rep', filter: 'territory_match' }],
    channels: ['in_app'],
    priority: 'low',
  },
];
```

### Middleware Implementation

```typescript
// Authorization middleware factory
function authorize(permission: string) {
  return async (request: FastifyRequest, reply: FastifyReply) => {
    const user = request.user; // Set by authenticate middleware
    
    // Check role-based permission
    if (!hasPermission(user.role, permission)) {
      throw new ForbiddenError(`Missing permission: ${permission}`);
    }
    
    // Check resource ownership if applicable
    const resourceId = request.params.id;
    if (resourceId && isOwnershipScoped(permission)) {
      const isOwner = await checkOwnership(user, permission, resourceId);
      if (!isOwner && !hasPermission(user.role, permission.replace(':read', ':read_all'))) {
        throw new ForbiddenError('You can only access your own resources');
      }
    }
  };
}

function hasPermission(role: UserRole, permission: string): boolean {
  return ROLE_PERMISSIONS[role].includes(permission) || ROLE_PERMISSIONS[role].includes('*');
}
```

## Implementation Guide

### File Locations
- `apps/api/src/modules/auth/` — Authentication module
  - `auth.routes.ts` — Login, refresh, password change
  - `auth.service.ts` — Token generation, validation
- `apps/api/src/middleware/authenticate.ts` — JWT verification
- `apps/api/src/middleware/authorize.ts` — Permission checking
- `apps/api/src/modules/users/` — User management
- `apps/api/src/modules/notifications/` — Notification routing and delivery
- `packages/shared/src/constants/permissions.ts` — Role-permission matrix

### Implementation Order
1. Auth module (login, JWT, refresh tokens)
2. Role-permission matrix as constants
3. Authorize middleware
4. User CRUD (owner only)
5. Ownership check functions per resource
6. Notification storage and delivery (in-app)
7. Push notification integration
8. Email notification delivery
9. Notification preferences

## Testing Requirements

### Unit Tests
- Permission check: sales_rep + "bids:read" → allowed
- Permission check: installer + "financials:read" → denied
- Ownership: rep accessing own contact → allowed
- Ownership: rep accessing another rep's contact → denied
- Owner accessing anything → allowed
- Approval workflow: margin 12% → requires owner approval

### Seed Data
```typescript
const sampleUsers = [
  { email: 'patricia@company.com', role: 'owner', firstName: 'Patricia', lastName: 'Johnson' },
  { email: 'tom@company.com', role: 'sales_manager', firstName: 'Tom', lastName: 'Williams' },
  { email: 'alex@company.com', role: 'sales_rep', firstName: 'Alex', lastName: 'Garcia', territories: ['west'] },
  { email: 'maria@company.com', role: 'estimator', firstName: 'Maria', lastName: 'Rodriguez' },
  { email: 'dave@company.com', role: 'project_manager', firstName: 'Dave', lastName: 'Chen' },
  { email: 'carlos@company.com', role: 'installer', firstName: 'Carlos', lastName: 'Martinez' },
  { email: 'lisa@company.com', role: 'admin', firstName: 'Lisa', lastName: 'Thompson' },
];
```

## Resolved Design Decisions

- **Custom roles:** Not supported for MVP. The 7 predefined roles cover all typical flooring company positions. If a user needs mixed permissions, assign the higher role.
- **Cross-territory contact visibility:** Sales reps can only see contacts in their own territory. Sales managers see all. This prevents confusion about ownership.
- **Installer financial visibility:** Installers cannot see contract values or margins. They see only project scope, schedule, and their tasks.
- **Deployment model:** Single-tenant (one company per deployment). No multi-org support needed.
- **User invitation:** Only the owner can invite new users. See `spec/18-authentication.md` for the full invitation flow.
- **Territory-based access:** Contacts and opportunities are filtered by the user's assigned territories. A territory is defined by zip codes/cities/counties (see `spec/02-domain-model.md` Territory entity).

# Authentication & User Management

## Vision & Purpose

Salescraft uses an invitation-only model — there's no public signup. The company owner bootstraps the system, then invites team members via email. Authentication uses JWT access tokens (short-lived) paired with HTTP-only refresh tokens (longer-lived) for security. The system is single-tenant: one company per deployment.

## Key Concepts

- **Bootstrap** — First-time setup creates the owner account and company record
- **Invitation** — Owner/admin sends email invitations with a one-time setup link
- **Access Token** — Short-lived JWT (15 min) sent in Authorization header
- **Refresh Token** — Longer-lived (7 days) stored in HTTP-only secure cookie
- **Session** — A refresh token represents an active session; max 5 per user

## Authentication Flow

### First-Time Bootstrap

Only works once — when no users exist in the database.

```
POST /api/v1/auth/setup
Body: {
  companyName: string,
  firstName: string,
  lastName: string,
  email: string,
  password: string
}
Response: {
  user: User,
  accessToken: string
}
Set-Cookie: refreshToken=...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

**Guards:**
- Returns 409 Conflict if any user already exists
- Password must meet complexity requirements (min 8 chars, 1 uppercase, 1 number)
- Creates the user with role `owner`
- Creates a Company record with the provided name

### Login

```
POST /api/v1/auth/login
Body: {
  email: string,
  password: string
}
Response: {
  user: User (without passwordHash),
  accessToken: string
}
Set-Cookie: refreshToken=...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

**Guards:**
- Rate limited: 5 failed attempts per email → 15-minute lockout
- Returns generic 401 for both "user not found" and "wrong password" (no user enumeration)
- Inactive users (`isActive = false`) cannot login (401)
- Updates `lastLoginAt` on success

### Token Refresh

```
POST /api/v1/auth/refresh
Cookie: refreshToken=...
Response: {
  accessToken: string
}
Set-Cookie: refreshToken=...; (new token, rotated)
```

**Behavior:**
- Validates the refresh token (not expired, not revoked, matches stored hash)
- Issues a new access token AND rotates the refresh token (old one invalidated)
- If the refresh token is expired or invalid → 401 (user must re-login)

### Logout

```
POST /api/v1/auth/logout
Cookie: refreshToken=...
Response: 204 No Content
Set-Cookie: refreshToken=; Max-Age=0
```

**Behavior:**
- Revokes the refresh token (deletes from sessions table)
- Clears the cookie

### Logout All Sessions

```
POST /api/v1/auth/logout-all
Authorization: Bearer {accessToken}
Response: 204 No Content
```

**Behavior:**
- Revokes ALL refresh tokens for the current user
- Useful when password is changed or account is compromised

## User Invitation Flow

### Send Invitation

```
POST /api/v1/users/invite
Authorization: Bearer {accessToken} (owner only)
Body: {
  email: string,
  firstName: string,
  lastName: string,
  role: UserRole,
  territories?: string[]
}
Response: {
  invitation: {
    id: string,
    email: string,
    role: UserRole,
    expiresAt: DateTime,
    status: 'pending'
  }
}
```

**Behavior:**
- Creates an invitation record with a unique token (UUID)
- Sends an email with a setup link: `{APP_URL}/accept-invite?token={inviteToken}`
- Invitation expires in 7 days
- Only `owner` role can invite new users

### Accept Invitation

```
POST /api/v1/auth/accept-invite
Body: {
  token: string,        // From the invitation link
  password: string,
  phone?: string
}
Response: {
  user: User,
  accessToken: string
}
Set-Cookie: refreshToken=...
```

**Behavior:**
- Validates the invitation token (exists, not expired, not already used)
- Creates the user with the pre-assigned role and territories
- Marks invitation as `accepted`
- Logs the user in immediately

### Resend Invitation

```
POST /api/v1/users/invite/:id/resend
Authorization: Bearer {accessToken} (owner only)
Response: 204 No Content
```

**Behavior:**
- Generates a new token (old one invalidated)
- Resets expiry to 7 days from now
- Sends a new email

## Password Management

### Change Own Password

```
PUT /api/v1/auth/password
Authorization: Bearer {accessToken}
Body: {
  currentPassword: string,
  newPassword: string
}
Response: 204 No Content
```

**Behavior:**
- Validates current password
- Updates password hash
- Revokes all other sessions (forces re-login on other devices)

### Request Password Reset

```
POST /api/v1/auth/forgot-password
Body: {
  email: string
}
Response: 204 No Content (always, even if email not found — no enumeration)
```

**Behavior:**
- If email exists: generates a reset token (valid 1 hour), sends email with link
- If email doesn't exist: still returns 204 (no information leakage)
- Link format: `{APP_URL}/reset-password?token={resetToken}`
- Rate limited: 3 requests per email per hour

### Complete Password Reset

```
POST /api/v1/auth/reset-password
Body: {
  token: string,
  newPassword: string
}
Response: 204 No Content
```

**Behavior:**
- Validates token (exists, not expired, not used)
- Updates password hash
- Marks token as used
- Revokes all existing sessions

## Data Model

### Session (Refresh Token Storage)

```typescript
interface Session {
  id: string;
  userId: string;             // FK → User
  tokenHash: string;          // bcrypt hash of the refresh token
  userAgent?: string;         // Browser/device info
  ipAddress?: string;         // For audit
  expiresAt: DateTime;
  createdAt: DateTime;
  lastUsedAt: DateTime;
}
```

### Invitation

```typescript
interface Invitation {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  role: UserRole;
  territories: string[];
  tokenHash: string;          // Hash of the invite token
  status: 'pending' | 'accepted' | 'expired' | 'revoked';
  expiresAt: DateTime;
  invitedBy: string;          // FK → User
  acceptedAt?: DateTime;
  createdAt: DateTime;
}
```

### PasswordResetToken

```typescript
interface PasswordResetToken {
  id: string;
  userId: string;             // FK → User
  tokenHash: string;          // Hash of the reset token
  expiresAt: DateTime;
  usedAt?: DateTime;
  createdAt: DateTime;
}
```

## JWT Token Structure

### Access Token Payload

```typescript
interface AccessTokenPayload {
  sub: string;               // User ID
  email: string;
  role: UserRole;
  iat: number;               // Issued at (Unix timestamp)
  exp: number;               // Expires at (15 min from iat)
}
```

### Token Configuration

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Access token TTL | 15 minutes | Short-lived, limits exposure if intercepted |
| Refresh token TTL | 7 days | Convenient for daily users without constant re-login |
| Max sessions per user | 5 | Prevents unlimited device accumulation |
| Refresh token rotation | Yes | Each refresh invalidates old token (detect reuse = breach) |
| Algorithm | HS256 | Sufficient for single-service; symmetric key from env var |

## Security Measures

### Rate Limiting

| Endpoint | Limit | Window | Lockout |
|----------|-------|--------|---------|
| POST /auth/login | 5 attempts | 15 min | Per email |
| POST /auth/forgot-password | 3 requests | 1 hour | Per email |
| POST /auth/setup | 1 attempt | Forever | Global (only works once) |
| POST /users/invite | 20 invitations | 1 hour | Per user |

### Refresh Token Reuse Detection

If a refresh token that was already rotated is presented again, it indicates a potential token theft. Response:
- Revoke ALL sessions for that user
- Return 401
- Log a security event

### Password Requirements

- Minimum 8 characters
- At least 1 uppercase letter
- At least 1 number
- No maximum length (up to 128 chars)
- Hashed with bcrypt (cost factor 12)

## Business Rules

- **BR-AUTH-001:** Only owner can create users (via invitation)
- **BR-AUTH-002:** Users cannot delete their own account
- **BR-AUTH-003:** Deactivated users cannot login but their data is preserved
- **BR-AUTH-004:** Password change revokes all other sessions
- **BR-AUTH-005:** Refresh token rotation on every use (sliding window)
- **BR-AUTH-006:** Session limit enforced: creating a 6th session revokes the oldest
- **BR-AUTH-007:** All auth events logged in audit trail (login, logout, password change, failed attempts)

## API Endpoints Summary

```typescript
// Bootstrap (unauthenticated, one-time only)
POST   /api/v1/auth/setup

// Login/Logout (unauthenticated)
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
POST   /api/v1/auth/logout-all

// Password (mixed auth)
POST   /api/v1/auth/forgot-password         // Unauthenticated
POST   /api/v1/auth/reset-password          // Unauthenticated (token-based)
PUT    /api/v1/auth/password                // Authenticated (change own)

// Invitation (owner only)
POST   /api/v1/users/invite
GET    /api/v1/users/invitations            // List pending invitations
POST   /api/v1/users/invite/:id/resend
DELETE /api/v1/users/invite/:id             // Revoke invitation
POST   /api/v1/auth/accept-invite           // Unauthenticated (token-based)
```

## Implementation Guide

### File Locations
- `apps/api/src/modules/auth/auth.routes.ts` — Route definitions
- `apps/api/src/modules/auth/auth.service.ts` — Login, token generation, refresh
- `apps/api/src/modules/auth/auth.schemas.ts` — Zod validation schemas
- `apps/api/src/modules/auth/invitation.service.ts` — Invitation CRUD and email
- `apps/api/src/modules/auth/password.service.ts` — Password reset flow
- `apps/api/src/middleware/authenticate.ts` — JWT verification middleware
- `packages/shared/src/constants/auth.ts` — Token TTLs, rate limits

### Key Dependencies
- `jsonwebtoken` — JWT signing/verification
- `bcrypt` — Password hashing
- `nodemailer` — Invitation and reset emails

### Implementation Order
1. JWT signing/verification utilities
2. Authenticate middleware (validates access token on protected routes)
3. Login endpoint + session creation
4. Refresh endpoint with token rotation
5. Bootstrap/setup endpoint
6. Invitation flow (create, send email, accept)
7. Password reset flow
8. Rate limiting middleware
9. Logout and session management

## Testing Requirements

### Unit Tests
- JWT generation and validation
- Password hashing and verification
- Rate limit counter logic
- Token rotation detection (reuse of old token)

### Integration Tests
- Full bootstrap flow: setup → login → get user
- Invitation flow: invite → accept → login
- Token refresh: login → wait → refresh → access protected route
- Password reset: forgot → email sent → reset → login with new password
- Rate limiting: 6 failed logins → lockout → wait → unlock
- Session limit: login on 6 devices → oldest session revoked

### Seed Data
The seed script should use the bootstrap endpoint internally to create the owner, then create remaining users via direct DB insert (with pre-hashed passwords) for convenience.

```typescript
const SEED_PASSWORD = 'Password123'; // All seed users use this
const SEED_PASSWORD_HASH = await bcrypt.hash(SEED_PASSWORD, 12);
```

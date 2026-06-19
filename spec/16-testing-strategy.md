# Testing Strategy

## Vision & Purpose

Salescraft's test strategy ensures that an AI agent (or developer) can build features with confidence that they work correctly without manual verification of every code path. Tests serve as executable specifications — they define correct behavior precisely.

The testing approach is pragmatic: heavy coverage on business logic (calculations, state machines, scoring) and critical paths (bid submission, financial calculations), lighter coverage on CRUD and framework plumbing.

## Test Pyramid

```
        ┌─────────┐
        │   E2E   │  5-10 critical flows (bid lifecycle, daily log offline sync)
        ├─────────┤
      ┌─┤  Integ  ├─┐  API endpoint tests with real database
      │ ├─────────┤ │
    ┌─┤ │  Unit   │ ├─┐  Business logic, calculations, state machines
    │ │ └─────────┘ │ │
    └───────────────────┘
     70% unit / 25% integration / 5% E2E
```

## Frameworks and Tools

| Tool | Purpose |
|------|---------|
| Vitest | Unit and integration test runner (fast, TypeScript-native) |
| Supertest | HTTP endpoint testing |
| Prisma (test client) | Database testing with real PostgreSQL |
| Testcontainers | Spin up PostgreSQL/Redis for integration tests |
| MSW (Mock Service Worker) | Mock external APIs (Gmail, Bedrock, bid boards) |
| Playwright | E2E browser tests (web app) |
| Detox | E2E mobile tests (React Native) — optional |
| Faker.js | Generate realistic test data |

## Test Organization

```
apps/api/
├── src/
│   └── modules/
│       └── estimating/
│           ├── calculation.service.ts
│           └── __tests__/
│               ├── calculation.unit.test.ts    # Pure logic tests
│               └── estimating.integration.test.ts  # API + DB tests
├── test/
│   ├── setup.ts                 # Global test setup
│   ├── helpers/
│   │   ├── db.ts               # Database test utilities (seed, cleanup)
│   │   ├── auth.ts             # Generate test JWTs
│   │   ├── factories.ts        # Entity factories (Faker-based)
│   │   └── mocks.ts            # External service mocks
│   └── fixtures/
│       ├── rfp-sample.pdf       # Real RFP for parsing tests
│       ├── bid-board-html.html  # Saved bid board page for scraper tests
│       └── ai-responses.json   # Deterministic AI response fixtures

packages/shared/
└── src/
    └── __tests__/               # Schema validation tests
```

## Unit Tests

### What to Unit Test
- **Calculation engines** (estimating math, scoring algorithms, cost calculations)
- **State machine transitions** (guards, side effects)
- **Business rule validation** (ethics checks, threshold checks, deadline logic)
- **Data transformations** (API response shaping, sync conflict resolution)
- **Utility functions** (date calculations, money formatting, area conversions)

### What NOT to Unit Test
- Prisma queries (test via integration tests with real DB)
- Route handler wiring (covered by integration tests)
- Third-party library behavior
- Simple getters/setters

### Example: Calculation Engine Tests

```typescript
// apps/api/src/modules/estimating/__tests__/calculation.unit.test.ts
import { describe, it, expect } from 'vitest';
import { calculateArea, calculateEstimateTotals } from '../calculation.service';

describe('calculateArea', () => {
  it('applies waste factor to square footage', () => {
    const result = calculateArea({
      sqFt: 1000,
      wasteFactor: 10,
      materialCostPerSqFt: 4.50,
      laborRatePerSqFt: 2.50,
    });
    
    expect(result.materialSqFt).toBe(1100); // 1000 * 1.10
    expect(result.materialCost).toBe(4950); // 1100 * 4.50
    expect(result.laborCost).toBe(2500);    // 1000 * 2.50 (labor on actual sqft, not waste)
  });

  it('handles zero waste factor', () => {
    const result = calculateArea({ sqFt: 500, wasteFactor: 0, materialCostPerSqFt: 3.00, laborRatePerSqFt: 2.00 });
    expect(result.materialSqFt).toBe(500);
    expect(result.materialCost).toBe(1500);
  });

  it('uses decimal precision for financial calculations', () => {
    const result = calculateArea({ sqFt: 333, wasteFactor: 7, materialCostPerSqFt: 4.33, laborRatePerSqFt: 2.17 });
    // No floating point issues
    expect(Number.isFinite(result.materialCost)).toBe(true);
    expect(result.materialCost.toString().split('.')[1]?.length || 0).toBeLessThanOrEqual(2);
  });
});

describe('calculateEstimateTotals', () => {
  it('sums areas and applies overhead and profit', () => {
    const result = calculateEstimateTotals({
      areas: [
        { materialCost: 5000, laborCost: 2500, additionalMaterialsCost: 300 },
        { materialCost: 3000, laborCost: 1500, additionalMaterialsCost: 200 },
      ],
      overheadPercentage: 15,
      profitPercentage: 12,
      bondCost: 0,
    });
    
    const directCosts = 5000 + 2500 + 300 + 3000 + 1500 + 200; // 12500
    const overhead = directCosts * 0.15; // 1875
    const subtotal = directCosts + overhead; // 14375
    const profit = subtotal * 0.12; // 1725
    
    expect(result.directCostTotal).toBe(12500);
    expect(result.overhead).toBe(1875);
    expect(result.profit).toBe(1725);
    expect(result.grandTotal).toBe(14375 + 1725); // 16100
  });

  it('includes bond cost in grand total', () => {
    const result = calculateEstimateTotals({
      areas: [{ materialCost: 100000, laborCost: 50000, additionalMaterialsCost: 0 }],
      overheadPercentage: 10,
      profitPercentage: 10,
      bondCost: 625,
    });
    
    expect(result.grandTotal).toBeGreaterThan(result.profit + result.overhead + 150000);
    expect(result.bondCost).toBe(625);
  });
});
```

### Example: Relationship Scoring Tests

```typescript
// apps/api/src/modules/relationships/__tests__/scoring.unit.test.ts
describe('calculateRelationshipScore', () => {
  it('gives full recency points for contact within 7 days', () => {
    const score = calculateRelationshipScore({
      lastContactedAt: subDays(new Date(), 3),
      interactions90Days: [],
      personalKnowledge: { interests: 0, hasFamily: false, hasPersonalNotes: false },
    });
    expect(score.factors.find(f => f.name === 'recency')?.score).toBe(100);
  });

  it('decays recency score after 30 days', () => {
    const score = calculateRelationshipScore({
      lastContactedAt: subDays(new Date(), 45),
      interactions90Days: [],
      personalKnowledge: { interests: 0, hasFamily: false, hasPersonalNotes: false },
    });
    expect(score.factors.find(f => f.name === 'recency')?.score).toBe(40);
  });

  it('weights personal knowledge for contacts we know well', () => {
    const score = calculateRelationshipScore({
      lastContactedAt: subDays(new Date(), 5),
      interactions90Days: [{ type: 'phone_call' }, { type: 'lunch_dinner' }],
      personalKnowledge: { interests: 3, hasFamily: true, hasPersonalNotes: true },
    });
    expect(score.factors.find(f => f.name === 'personal_knowledge')?.score).toBe(100);
  });
});
```

### Example: Ethics Check Tests

```typescript
describe('checkEthicsCompliance', () => {
  it('allows gesture within per-occasion limit', () => {
    const result = checkEthicsCompliance({
      contactId: 'contact-1',
      gestureValue: 40,
      jurisdiction: { giftLimitPerOccasion: 50, giftLimitAnnual: 500 },
      existingGesturesYTD: [{ value: 100 }],
    });
    expect(result.allowed).toBe(true);
  });

  it('blocks gesture exceeding per-occasion limit', () => {
    const result = checkEthicsCompliance({
      contactId: 'contact-1',
      gestureValue: 75,
      jurisdiction: { giftLimitPerOccasion: 50, giftLimitAnnual: 500 },
      existingGesturesYTD: [],
    });
    expect(result.allowed).toBe(false);
    expect(result.reason).toContain('per-occasion limit');
  });

  it('blocks gesture when annual total would exceed limit', () => {
    const result = checkEthicsCompliance({
      contactId: 'contact-1',
      gestureValue: 40,
      jurisdiction: { giftLimitPerOccasion: 50, giftLimitAnnual: 250 },
      existingGesturesYTD: [{ value: 100 }, { value: 80 }, { value: 50 }], // 230 YTD
    });
    expect(result.allowed).toBe(false);
    expect(result.reason).toContain('annual limit');
  });
});
```

## Integration Tests

### Setup
```typescript
// apps/api/test/setup.ts
import { beforeAll, afterAll, beforeEach } from 'vitest';
import { PrismaClient } from '@prisma/client';
import { execSync } from 'child_process';

const prisma = new PrismaClient();

beforeAll(async () => {
  // Run migrations on test database
  execSync('pnpm prisma migrate deploy', { env: { ...process.env, DATABASE_URL: TEST_DB_URL } });
});

beforeEach(async () => {
  // Truncate all tables between tests
  const tables = await prisma.$queryRaw`
    SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename != '_prisma_migrations'
  `;
  for (const { tablename } of tables) {
    await prisma.$executeRawUnsafe(`TRUNCATE TABLE "${tablename}" CASCADE`);
  }
});

afterAll(async () => {
  await prisma.$disconnect();
});
```

### Example: API Integration Test

```typescript
// apps/api/src/modules/bids/__tests__/bids.integration.test.ts
import { describe, it, expect, beforeEach } from 'vitest';
import { app } from '../../../server';
import { createTestUser, createTestOrganization, createTestBid } from '../../../../test/helpers/factories';
import { getAuthToken } from '../../../../test/helpers/auth';

describe('POST /api/v1/bids/:id/transition', () => {
  let token: string;
  let bid: any;

  beforeEach(async () => {
    const user = await createTestUser({ role: 'sales_rep' });
    token = getAuthToken(user);
    const org = await createTestOrganization();
    bid = await createTestBid({ organizationId: org.id, assignedToId: user.id, status: 'reviewing' });
  });

  it('transitions bid from reviewing to preparing when decision is made', async () => {
    // First, submit a bid decision
    await app.inject({
      method: 'POST',
      url: `/api/v1/bids/${bid.id}/decision`,
      headers: { Authorization: `Bearer ${token}` },
      payload: { decision: 'bid', factors: [{ name: 'relationship_strength', score: 80 }] },
    });

    const response = await app.inject({
      method: 'POST',
      url: `/api/v1/bids/${bid.id}/transition`,
      headers: { Authorization: `Bearer ${token}` },
      payload: { to: 'preparing' },
    });

    expect(response.statusCode).toBe(200);
    expect(response.json().data.status).toBe('preparing');
  });

  it('rejects transition without decision', async () => {
    const response = await app.inject({
      method: 'POST',
      url: `/api/v1/bids/${bid.id}/transition`,
      headers: { Authorization: `Bearer ${token}` },
      payload: { to: 'preparing' },
    });

    expect(response.statusCode).toBe(422);
    expect(response.json().error.code).toBe('TRANSITION_GUARD_FAILED');
  });
});
```

## E2E Tests

### Critical User Flows

```typescript
// e2e/bid-lifecycle.spec.ts (Playwright)
import { test, expect } from '@playwright/test';

test('complete bid lifecycle: discover → decide → estimate → submit → win', async ({ page }) => {
  await page.goto('/login');
  await page.fill('[name=email]', 'alex@company.com');
  await page.fill('[name=password]', 'testpassword');
  await page.click('button[type=submit]');
  
  // Navigate to bids
  await page.click('[data-nav=bids]');
  
  // Open a bid in "reviewing" status
  await page.click('[data-testid=bid-card-reviewing]');
  
  // Complete bid decision
  await page.click('[data-tab=decision]');
  // ... fill out decision matrix
  await page.click('[data-action=decide-bid]');
  
  // Verify status changed to "preparing"
  await expect(page.locator('[data-status]')).toHaveText('Preparing');
  
  // ... continue through estimate, review, submit, win
});
```

### Flows to Cover
1. **Bid lifecycle** (discover → decide → estimate → submit → win → project created)
2. **Daily log offline** (go offline → create log → come online → verify synced)
3. **Relationship briefing** (open contact → see briefing card → log interaction → score updates)
4. **Contact creation with interest** (create contact → add interest → reverse match works)
5. **Estimate calculation** (add areas → select products → calculate → verify totals)

## Test Data Factories

```typescript
// apps/api/test/helpers/factories.ts
import { faker } from '@faker-js/faker';
import { prisma } from './db';

export async function createTestUser(overrides: Partial<User> = {}) {
  return prisma.user.create({
    data: {
      email: faker.internet.email(),
      passwordHash: await hash('testpassword'),
      firstName: faker.person.firstName(),
      lastName: faker.person.lastName(),
      role: 'sales_rep',
      isActive: true,
      ...overrides,
    },
  });
}

export async function createTestOrganization(overrides: Partial<Organization> = {}) {
  return prisma.organization.create({
    data: {
      name: `${faker.location.city()} Unified School District`,
      type: 'school_district',
      state: 'CA',
      city: faker.location.city(),
      purchasingThreshold: 92600,
      fiscalYearStart: 7,
      createdById: 'system',
      ...overrides,
    },
  });
}

export async function createTestContact(overrides: Partial<Contact> = {}) {
  return prisma.contact.create({
    data: {
      firstName: faker.person.firstName(),
      lastName: faker.person.lastName(),
      email: faker.internet.email(),
      role: 'facility_director',
      decisionAuthority: 'decision_maker',
      relationshipScore: faker.number.int({ min: 0, max: 100 }),
      createdById: 'system',
      ...overrides,
    },
  });
}

export async function createTestBid(overrides: Partial<Bid> = {}) {
  return prisma.bid.create({
    data: {
      title: `IFB #${faker.number.int({ min: 1000, max: 9999 })}: Flooring Replacement`,
      type: 'ifb',
      status: 'discovered',
      submissionDeadline: faker.date.future(),
      bondRequired: true,
      prevailingWage: true,
      decision: 'pending',
      ...overrides,
    },
  });
}
```

## Mocking External Services

```typescript
// apps/api/test/helpers/mocks.ts
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';

export const mockServer = setupServer(
  // Mock Bedrock
  http.post('https://bedrock-runtime.us-west-2.amazonaws.com/model/*/invoke', () => {
    return HttpResponse.json({
      body: JSON.stringify({ content: [{ text: '{"is_flooring": true, "confidence": 0.95}' }] }),
    });
  }),

  // Mock Gmail API
  http.get('https://gmail.googleapis.com/gmail/v1/users/me/messages', () => {
    return HttpResponse.json({ messages: [], resultSizeEstimate: 0 });
  }),

  // Mock Google Maps Geocoding
  http.get('https://maps.googleapis.com/maps/api/geocode/json', () => {
    return HttpResponse.json({
      results: [{ geometry: { location: { lat: 33.9533, lng: -117.3961 } } }],
      status: 'OK',
    });
  }),
);
```

## CI Configuration

```typescript
// vitest.config.ts (root)
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    include: ['**/*.{test,spec}.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      include: ['apps/api/src/**', 'packages/*/src/**'],
      exclude: ['**/__tests__/**', '**/node_modules/**'],
      thresholds: {
        branches: 70,
        functions: 70,
        lines: 70,
        statements: 70,
      },
    },
    setupFiles: ['./apps/api/test/setup.ts'],
  },
});
```

## Implementation Guide

### Implementation Order
1. Vitest setup with test database configuration
2. Test helpers (factories, auth token generation)
3. MSW mock server for external APIs
4. Unit tests for calculation engine (TDD — write tests first)
5. Unit tests for state machine transitions
6. Unit tests for scoring algorithms
7. Integration tests for critical API endpoints
8. Integration test for full sync cycle
9. Playwright setup for E2E
10. E2E tests for 5 critical flows

### Testing Principles
- **Business logic first:** Prioritize testing calculations, scoring, and state machines over CRUD
- **Real database for integration:** Don't mock Prisma; use test PostgreSQL instance
- **Deterministic AI tests:** Use MSW to mock Bedrock responses; test prompt rendering separately
- **Fast feedback:** Unit tests run in <5 seconds, integration in <30 seconds
- **No test pollution:** Each test gets a clean database state (truncate between tests)

## Resolved Design Decisions

- **Integration test database:** Use Testcontainers (per-test-suite PostgreSQL container). Slightly slower startup (~2-3s) but fully isolated — no test pollution, no shared state issues. Truncate tables between tests within a suite for speed; fresh container per suite file.
- **Visual regression tests:** Not for MVP. The UI will change rapidly during initial development, causing constant false positives. Add visual regression (Playwright screenshot comparison) after the UI stabilizes post-launch.
- **Performance tests:** Not in CI for MVP. Run k6 load tests manually before launch and after major changes. Adding to CI increases build time significantly and performance issues are unlikely at <50 users. Add to CI when approaching 100+ users.

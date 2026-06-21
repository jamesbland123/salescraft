# Deployment & Infrastructure

## Vision & Purpose

Salescraft is built and tested locally using Podman + podman-compose, then packaged for AWS deployment. The infrastructure is designed for a single company with 20-50 users — not hyper-scale. Simplicity and cost-efficiency are priorities. The deployment should be automated enough that a single developer can manage it.

Local development uses Podman, not Docker Desktop. The API still includes an OCI-compatible `Dockerfile` for CI/ECS image builds; GitHub Actions may use Docker commands to build and push that image.

## Key Concepts

- **Local Dev** — Podman-compose runs all services locally; application code runs natively (hot reload)
- **AWS Target** — ECS Fargate (containers), RDS PostgreSQL, ElastiCache Redis, S3
- **IaC** — AWS CDK (TypeScript) defines all infrastructure as code
- **CI/CD** — GitHub Actions for test → build → deploy pipeline

## Local Development Environment

### Podman Compose Stack

```yaml
# podman/podman-compose.yml

services:
  postgres:
    image: pgvector/pgvector:pg16
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: salescraft
      POSTGRES_USER: salescraft
      POSTGRES_PASSWORD: salescraft
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U salescraft"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s

  localstack:
    image: localstack/localstack:3.0
    ports:
      - "4566:4566"
    environment:
      SERVICES: s3
      DEFAULT_REGION: us-west-2
      EAGER_SERVICE_LOADING: 1
    volumes:
      - localstack_data:/var/lib/localstack
      - ./localstack/init-s3.sh:/etc/localstack/init/ready.d/init-s3.sh

  mailhog:
    image: mailhog/mailhog
    ports:
      - "1025:1025"    # SMTP
      - "8025:8025"    # Web UI
    
  bull-board:
    image: node:20-slim
    ports:
      - "3002:3002"
    command: npx bull-board --redis redis://redis:6379 --port 3002
    depends_on:
      - redis

volumes:
  postgres_data:
  redis_data:
  localstack_data:
```

### Dev Setup Script
```bash
#!/bin/bash
# scripts/dev-setup.sh

echo "Setting up Salescraft development environment..."

# Check prerequisites
command -v node >/dev/null 2>&1 || { echo "Node.js 20+ required"; exit 1; }
command -v pnpm >/dev/null 2>&1 || { echo "pnpm required: npm install -g pnpm"; exit 1; }
command -v podman >/dev/null 2>&1 || { echo "Podman required"; exit 1; }
command -v podman-compose >/dev/null 2>&1 || { echo "podman-compose required: pip install podman-compose"; exit 1; }

# Install dependencies
pnpm install

# Start infrastructure services
podman-compose -f podman/podman-compose.yml up -d

# Wait for PostgreSQL
echo "Waiting for PostgreSQL..."
until podman exec salescraft_postgres_1 pg_isready -U salescraft; do
  sleep 1
done

# Run migrations
pnpm db:migrate

# Seed database
pnpm db:seed

# Copy .env.example if .env doesn't exist
if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env from .env.example — update AWS credentials for Bedrock access"
fi

echo "Development environment ready!"
echo "Run 'pnpm dev' to start all applications"
```

## AWS Architecture

### Service Map

```
┌─────────────────────────────────────────────────────────────┐
│                        AWS Account                            │
│                                                               │
│  ┌─────────────┐     ┌───────────────────────────────────┐  │
│  │ CloudFront  │────▶│  S3 (static assets + files)       │  │
│  │   (CDN)     │     └───────────────────────────────────┘  │
│  └──────┬──────┘                                             │
│         │                                                     │
│  ┌──────▼──────┐     ┌───────────────────────────────────┐  │
│  │     ALB     │────▶│  ECS Fargate (API containers)     │  │
│  │  (HTTPS)    │     │  - api service (2 tasks)          │  │
│  └─────────────┘     │  - worker service (1 task)        │  │
│                       └──────────┬────────────────────────┘  │
│                                  │                            │
│               ┌──────────────────┼──────────────────┐        │
│               │                  │                  │        │
│  ┌────────────▼───┐ ┌───────────▼──┐ ┌────────────▼────┐   │
│  │  RDS PostgreSQL │ │ ElastiCache  │ │   AWS Bedrock   │   │
│  │  (db.t3.medium) │ │   (Redis)    │ │  (AI models)    │   │
│  │  + pgvector     │ │ (cache.t3.)  │ │                 │   │
│  └─────────────────┘ └──────────────┘ └─────────────────┘   │
│                                                               │
│  ┌─────────────────┐  ┌──────────────────────────────────┐  │
│  │ Secrets Manager │  │  CloudWatch (logs + metrics)     │  │
│  └─────────────────┘  └──────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### AWS Services Used

| Service | Purpose | Sizing | Estimated Monthly Cost |
|---------|---------|--------|----------------------|
| ECS Fargate (API) | API server containers | 2 tasks × 0.5 vCPU, 1GB RAM | ~$30 |
| ECS Fargate (Worker) | Background jobs | 1 task × 0.25 vCPU, 0.5GB RAM | ~$10 |
| RDS PostgreSQL | Primary database | db.t3.medium, 100GB storage | ~$50 |
| ElastiCache Redis | Cache + job queues | cache.t3.micro | ~$15 |
| S3 | File storage | ~50GB | ~$2 |
| CloudFront | CDN for web + files | Moderate traffic | ~$10 |
| ALB | Load balancer + HTTPS | Single ALB | ~$20 |
| Secrets Manager | API keys, tokens | ~20 secrets | ~$10 |
| CloudWatch | Logs + monitoring | Standard tier | ~$15 |
| Bedrock | AI inference | Based on usage | ~$15-50 |
| Route 53 | DNS | 1 hosted zone | ~$1 |
| ACM | SSL certificates | Free | $0 |
| **Total** | | | **~$180-230/month** |

### CDK Stack Structure

```typescript
// infra/lib/salescraft-stack.ts

// VPC Stack
export class VpcStack extends cdk.Stack {
  public readonly vpc: ec2.Vpc;
  // 2 AZs, public + private subnets
  // NAT Gateway for outbound from private subnets
}

// Database Stack
export class DatabaseStack extends cdk.Stack {
  public readonly cluster: rds.DatabaseCluster;
  public readonly redis: elasticache.CfnCacheCluster;
  // RDS in private subnet
  // Redis in private subnet
  // Security groups allowing access only from ECS
}

// API Stack
export class ApiStack extends cdk.Stack {
  // ECS Cluster
  // Fargate Service (API) with ALB
  // Fargate Service (Worker) - no ALB, just processes jobs
  // Task definitions with environment variables from Secrets Manager
  // Auto-scaling: 2-4 tasks based on CPU
}

// Web Stack
export class WebStack extends cdk.Stack {
  // S3 bucket for Next.js static export
  // CloudFront distribution
  // Custom domain + ACM certificate
}

// Storage Stack
export class StorageStack extends cdk.Stack {
  // S3 bucket for file uploads
  // Lifecycle rules: move to IA after 90 days, Glacier after 365 days
  // CORS configuration for direct uploads from web/mobile
}
```

## CI/CD Pipeline

### GitHub Actions Workflow

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: pgvector/pgvector:pg16
        env:
          POSTGRES_DB: salescraft_test
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
        ports: ["5432:5432"]
      redis:
        image: redis:7-alpine
        ports: ["6379:6379"]
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v2
      - uses: actions/setup-node@v4
        with: { node-version: 20, cache: 'pnpm' }
      - run: pnpm install
      - run: pnpm db:migrate
        env:
          DATABASE_URL: postgresql://test:test@localhost:5432/salescraft_test
      - run: pnpm typecheck
      - run: pnpm lint
      - run: pnpm test
        env:
          DATABASE_URL: postgresql://test:test@localhost:5432/salescraft_test
          REDIS_URL: redis://localhost:6379

  build:
    needs: test
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      - uses: aws-actions/amazon-ecr-login@v2
      - run: |
          docker build -t salescraft-api -f apps/api/Dockerfile .
          docker tag salescraft-api:latest $ECR_REGISTRY/salescraft-api:${{ github.sha }}
          docker push $ECR_REGISTRY/salescraft-api:${{ github.sha }}

  deploy:
    needs: build
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: |
          # Update ECS service with new image
          aws ecs update-service --cluster salescraft --service api --force-new-deployment
          aws ecs update-service --cluster salescraft --service worker --force-new-deployment
```

### Dockerfile (API)

```dockerfile
# apps/api/Dockerfile
FROM node:20-slim AS base
RUN corepack enable && corepack prepare pnpm@latest --activate

FROM base AS deps
WORKDIR /app
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY apps/api/package.json ./apps/api/
COPY packages/shared/package.json ./packages/shared/
COPY packages/database/package.json ./packages/database/
COPY packages/ai/package.json ./packages/ai/
RUN pnpm install --frozen-lockfile --prod

FROM base AS build
WORKDIR /app
COPY . .
RUN pnpm install --frozen-lockfile
RUN pnpm build

FROM base AS runner
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY --from=build /app/apps/api/dist ./apps/api/dist
COPY --from=build /app/packages/shared/dist ./packages/shared/dist
COPY --from=build /app/packages/database/dist ./packages/database/dist
COPY --from=build /app/packages/ai/dist ./packages/ai/dist
COPY --from=build /app/packages/database/prisma ./packages/database/prisma

EXPOSE 3001
CMD ["node", "apps/api/dist/server.js"]
```

## Background Jobs

### Job Scheduling (BullMQ)

```typescript
// apps/api/src/jobs/scheduler.ts
import { Queue, Worker } from 'bullmq';

const connection = { host: config.redis.host, port: config.redis.port };

// Queues
const bidScrapingQueue = new Queue('bid-scraping', { connection });
const emailSyncQueue = new Queue('email-sync', { connection });
const intelligenceQueue = new Queue('intelligence', { connection });
const relationshipQueue = new Queue('relationship', { connection });
const reportQueue = new Queue('reports', { connection });

// Scheduled jobs (repeatable)
await bidScrapingQueue.add('scrape-all', {}, { repeat: { pattern: '0 */2 6-18 * * *' } }); // Every 2h, 6AM-6PM
await emailSyncQueue.add('sync-all', {}, { repeat: { pattern: '*/5 * * * *' } }); // Every 5 min
await intelligenceQueue.add('daily-scan', {}, { repeat: { pattern: '0 7 * * *' } }); // Daily 7 AM
await relationshipQueue.add('decay-calc', {}, { repeat: { pattern: '0 6 * * *' } }); // Daily 6 AM
await relationshipQueue.add('seasonal-triggers', {}, { repeat: { pattern: '0 8 * * *' } }); // Daily 8 AM
await reportQueue.add('daily-digest', {}, { repeat: { pattern: '0 7 * * 1-5' } }); // Weekdays 7 AM
```

## Monitoring & Alerting

### Structured Logging

```typescript
// All logs are JSON, shipped to CloudWatch
{
  "level": "info",
  "timestamp": "2024-06-03T14:30:00.000Z",
  "requestId": "req_abc123",
  "userId": "user_xyz",
  "module": "bids",
  "action": "bid.submitted",
  "bidId": "bid_456",
  "duration": 234,
  "message": "Bid submitted successfully"
}
```

### Health Checks

```typescript
// GET /health — load balancer check
{
  "status": "healthy",
  "version": "1.2.3",
  "uptime": 3600,
  "checks": {
    "database": "ok",
    "redis": "ok",
    "s3": "ok",
    "bedrock": "ok"
  }
}
```

### Alerts (CloudWatch Alarms)

| Alert | Condition | Action |
|-------|-----------|--------|
| API 5xx rate | >1% of requests in 5 min | Email + PagerDuty |
| API latency | p95 > 2 seconds for 5 min | Email |
| Database CPU | >80% for 10 min | Email |
| Database connections | >80% of max | Email |
| Redis memory | >80% of max | Email |
| Scraper failures | >3 consecutive for any source | Email |
| Bid deadline approaching (no response) | Bid deadline <24h, status still "preparing" | Push + Email to rep |
| Compliance doc expiring | Within 7 days | Email to admin |
| AI cost spike | Daily cost > 2x average | Email to owner |

## Security

### Network Security
- All services in private subnets (no public IPs)
- ALB in public subnet (only port 443)
- Security groups: ALB → ECS → RDS/Redis (minimal ports)
- NAT Gateway for outbound (API calls to Gmail, Bedrock, etc.)

### Secret Management
```typescript
// Secrets stored in AWS Secrets Manager
const SECRETS = {
  'salescraft/database': { url: 'postgresql://...' },
  'salescraft/jwt': { secret: '...' },
  'salescraft/gmail-oauth': { clientId: '...', clientSecret: '...' },
  'salescraft/outlook-oauth': { clientId: '...', clientSecret: '...' },
  'salescraft/planetbids': { username: '...', password: '...' },
  'salescraft/bidnet': { apiKey: '...' },
};
```

### Backup Strategy
- RDS: automated daily snapshots, 30-day retention, PITR enabled
- S3: versioning enabled, lifecycle to Glacier after 1 year
- Redis: not backed up (ephemeral cache/queue data)

## Implementation Guide

### File Locations
- `podman/` — Podman Compose and service configs
- `infra/` — AWS CDK stacks
- `scripts/dev-setup.sh` — Developer onboarding
- `scripts/deploy.sh` — Manual deployment helper
- `.github/workflows/` — CI/CD pipelines
- `apps/api/Dockerfile` — API container build

### Implementation Order
1. Podman Compose with PostgreSQL, Redis, LocalStack, Mailhog
2. `dev-setup.sh` script (one-command dev environment)
3. API Dockerfile (buildable and runnable)
4. GitHub Actions: test job (lint, typecheck, test)
5. GitHub Actions: build job (Docker image → ECR)
6. AWS CDK: VPC + Database stacks
7. AWS CDK: API stack (ECS Fargate)
8. AWS CDK: Web + Storage stacks
9. GitHub Actions: deploy job
10. Monitoring and alerting setup

## Resolved Design Decisions

- **ECS Fargate vs. App Runner:** Stay with ECS Fargate. App Runner is simpler but lacks BullMQ worker support (no long-running background processes). Fargate gives us separate API and Worker services with independent scaling.
- **Frontend deployment:** Static export (S3 + CloudFront). Next.js `output: 'export'` generates static HTML/JS/CSS. The app is a SPA that calls the API — no server-side rendering needed. This is cheaper, faster (CDN-served), and simpler to deploy.
- **Staging environment:** Yes, implement as the same CDK stack with a `stage` parameter (`staging` vs `production`). Staging uses smaller instance sizes (db.t3.micro, cache.t3.micro) and shares the same AWS account. Deploy to staging on PR merge, promote to production manually.
- **Connection pooling:** Prisma's built-in pooling (10 connections default) is sufficient for <50 concurrent users. RDS Proxy adds $15/month and latency for no benefit at this scale. Revisit if connection errors appear under load.

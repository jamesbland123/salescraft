# Salescraft

A SaaS platform for commercial flooring companies to win and execute government contracts. Salescraft combines relationship intelligence, opportunity discovery, bid management, estimating, and project lifecycle tracking in a single purpose-built system.

## What It Does

- **Finds opportunities early** — monitors bond measures, capital improvement plans, meeting agendas, and building age data to identify flooring projects months before they hit bid boards
- **Builds relationships** — surfaces personal interests, life events, and commonalities so every interaction strengthens trust with facility directors and procurement officers
- **Responds faster** — AI-assisted proposal writing, automated compliance assembly, and intelligent bid/no-bid decisions
- **Executes flawlessly** — coordinates from contract award through installation with real-time field communication and offline-capable mobile app

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Monorepo | Turborepo + pnpm workspaces |
| Backend | Fastify, TypeScript, Prisma, PostgreSQL (pgvector, pg_trgm), BullMQ, Redis |
| Frontend | Next.js 14 (App Router), React, Tailwind CSS, shadcn/ui, TanStack Query |
| Mobile | React Native, Expo, WatermelonDB (offline-first) |
| AI | AWS Bedrock (Claude Sonnet/Haiku, Titan embeddings) |
| Containers | Podman + podman-compose |
| Testing | Vitest, Supertest, MSW, Playwright |

## Project Structure

```
salescraft/
├── apps/
│   ├── api/          # Fastify backend API
│   ├── web/          # Next.js frontend (static SPA)
│   └── mobile/       # React Native / Expo field app
├── packages/
│   ├── shared/       # Zod schemas, types, constants, utilities
│   ├── database/     # Prisma schema and migrations
│   └── ai/           # AWS Bedrock abstraction layer
├── podman/           # podman-compose + service init scripts
├── infra/            # AWS CDK (TypeScript)
└── spec/             # Complete domain specifications (27 files)
```

## Getting Started

### Prerequisites

- Node.js 20+
- pnpm 8+
- Podman + podman-compose
- AWS credentials (for Bedrock AI features)

### Setup

```bash
pnpm install
podman-compose -f podman/podman-compose.yml up -d
pnpm db:migrate
pnpm db:seed
pnpm dev
```

### Services (local)

| Service | Port |
|---------|------|
| API | http://localhost:3001 |
| Web | http://localhost:3000 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| LocalStack (S3) | 4566 |
| Mailhog UI | http://localhost:8025 |
| Bull Board | http://localhost:3002 |

## Documentation

All specifications live in the `spec/` directory:

- `spec/00-vision.md` — Product vision, personas, competitive positioning
- `spec/01-architecture.md` — Tech stack, project structure, module patterns, API conventions
- `spec/02-domain-model.md` — Entity definitions, state machines, validation rules
- `spec/09-ai-engine.md` — AI model routing, prompt templates, cost management
- `spec/16-testing-strategy.md` — Test pyramid, frameworks, example tests
- `spec/24-complete-prisma-schema.md` — Full database schema
- `spec/25-project-configuration.md` — All config files verbatim

The `PROMPT.md` file contains the AI agent build loop protocol with a phased dependency graph for implementation.

## Target Users

- **Sales Reps** — prospecting, relationship building, bid pursuit
- **Estimators** — takeoffs, material pricing, proposal generation
- **Project Managers** — post-award execution, crew coordination
- **Field Installers** — daily logs, punch lists, photo documentation (mobile)
- **Owners/Managers** — pipeline visibility, approvals, reporting

## License

This project is licensed under the MIT-0 License. See [LICENSE](LICENSE) for details.

# Spec 25: Project Configuration Files

This specification provides the complete contents of all configuration files for the Salescraft monorepo. These files are referenced throughout the other specifications but their contents are defined here.

---

## 1. `turbo.json`

```json
{
  "$schema": "https://turbo.build/schema.json",
  "globalDependencies": ["**/.env.*local"],
  "pipeline": {
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["dist/**", ".next/**", "!.next/cache/**"]
    },
    "dev": {
      "cache": false,
      "persistent": true
    },
    "lint": {
      "dependsOn": ["^build"],
      "outputs": []
    },
    "typecheck": {
      "dependsOn": ["^build"],
      "outputs": []
    },
    "test": {
      "dependsOn": ["^build"],
      "outputs": ["coverage/**"]
    },
    "db:migrate": {
      "cache": false
    },
    "db:seed": {
      "cache": false
    },
    "db:studio": {
      "cache": false,
      "persistent": true
    }
  }
}
```

---

## 2. `pnpm-workspace.yaml`

```yaml
packages:
  - "apps/*"
  - "packages/*"
```

---

## 3. Root `package.json`

```json
{
  "name": "salescraft",
  "private": true,
  "scripts": {
    "dev": "turbo dev",
    "build": "turbo build",
    "test": "turbo test",
    "lint": "turbo lint",
    "typecheck": "turbo typecheck",
    "db:migrate": "turbo db:migrate --filter=@salescraft/database",
    "db:seed": "turbo db:seed --filter=@salescraft/database",
    "db:studio": "turbo db:studio --filter=@salescraft/database",
    "format": "prettier --write \"**/*.{ts,tsx,js,jsx,json,md}\"",
    "format:check": "prettier --check \"**/*.{ts,tsx,js,jsx,json,md}\""
  },
  "devDependencies": {
    "@types/node": "^20.11.16",
    "eslint": "^8.56.0",
    "prettier": "^3.2.5",
    "turbo": "^1.12.4",
    "typescript": "^5.4.2"
  },
  "engines": {
    "node": ">=20"
  },
  "packageManager": "pnpm@8.15.4"
}
```

---

## 4. `tsconfig.base.json`

```json
{
  "compilerOptions": {
    "strict": true,
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "esModuleInterop": true,
    "resolveJsonModule": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "isolatedModules": true,
    "verbatimModuleSyntax": true,
    "lib": ["ES2022"],
    "baseUrl": ".",
    "paths": {
      "@salescraft/shared": ["packages/shared/src/index.ts"],
      "@salescraft/shared/*": ["packages/shared/src/*"],
      "@salescraft/database": ["packages/database/src/index.ts"],
      "@salescraft/database/*": ["packages/database/src/*"],
      "@salescraft/ai": ["packages/ai/src/index.ts"],
      "@salescraft/ai/*": ["packages/ai/src/*"]
    }
  },
  "exclude": ["node_modules", "dist"]
}
```

---

## 5. `apps/api/package.json`

```json
{
  "name": "@salescraft/api",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "tsx watch src/server.ts",
    "build": "tsc",
    "start": "node dist/server.js",
    "test": "vitest",
    "lint": "eslint src/ --ext .ts",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@aws-sdk/client-bedrock-runtime": "^3.515.0",
    "@aws-sdk/client-s3": "^3.515.0",
    "@aws-sdk/s3-request-presigner": "^3.515.0",
    "@fastify/cookie": "^9.3.1",
    "@fastify/cors": "^9.0.1",
    "@fastify/jwt": "^8.0.0",
    "@fastify/multipart": "^8.1.0",
    "@fastify/rate-limit": "^9.1.0",
    "@fastify/websocket": "^10.0.1",
    "@microsoft/microsoft-graph-client": "^3.0.7",
    "@salescraft/ai": "workspace:*",
    "@salescraft/database": "workspace:*",
    "@salescraft/shared": "workspace:*",
    "bcrypt": "^5.1.1",
    "bullmq": "^5.4.2",
    "decimal.js": "^10.4.3",
    "fastify": "^4.26.1",
    "googleapis": "^133.0.0",
    "handlebars": "^4.7.8",
    "jsonwebtoken": "^9.0.2",
    "juice": "^10.0.0",
    "nodemailer": "^6.9.9",
    "pino": "^8.19.0",
    "zod": "^3.22.4"
  },
  "devDependencies": {
    "@types/bcrypt": "^5.0.2",
    "@types/jsonwebtoken": "^9.0.5",
    "@types/nodemailer": "^6.4.14",
    "tsx": "^4.7.1",
    "typescript": "^5.4.2",
    "vitest": "^1.3.1"
  }
}
```

---

## 6. `apps/web/package.json`

```json
{
  "name": "@salescraft/web",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev --port 3000",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@hookform/resolvers": "^3.3.4",
    "@salescraft/shared": "workspace:*",
    "@tanstack/react-query": "^5.24.1",
    "@tanstack/react-table": "^8.13.2",
    "autoprefixer": "^10.4.17",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.0",
    "cmdk": "^0.2.1",
    "date-fns": "^3.3.1",
    "lucide-react": "^0.344.0",
    "mapbox-gl": "^3.2.0",
    "next": "^14.1.1",
    "postcss": "^8.4.35",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-hook-form": "^7.50.1",
    "recharts": "^2.12.2",
    "sonner": "^1.4.3",
    "tailwind-merge": "^2.2.1",
    "tailwindcss": "^3.4.1",
    "zod": "^3.22.4"
  },
  "devDependencies": {
    "@types/mapbox-gl": "^3.1.0",
    "@types/react": "^18.2.61",
    "@types/react-dom": "^18.2.19",
    "eslint-config-next": "^14.1.1",
    "typescript": "^5.4.2"
  }
}
```

---

## 7. `packages/shared/package.json`

```json
{
  "name": "@salescraft/shared",
  "version": "0.1.0",
  "private": true,
  "main": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "import": "./dist/index.js"
    },
    "./*": {
      "types": "./dist/*.d.ts",
      "import": "./dist/*.js"
    }
  },
  "scripts": {
    "build": "tsc",
    "dev": "tsc --watch",
    "lint": "eslint src/ --ext .ts",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "zod": "^3.22.4"
  },
  "devDependencies": {
    "typescript": "^5.4.2"
  }
}
```

---

## 8. `packages/database/package.json`

```json
{
  "name": "@salescraft/database",
  "version": "0.1.0",
  "private": true,
  "main": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "import": "./dist/index.js"
    },
    "./*": {
      "types": "./dist/*.d.ts",
      "import": "./dist/*.js"
    }
  },
  "scripts": {
    "build": "prisma generate && tsc",
    "dev": "tsc --watch",
    "db:migrate": "prisma migrate dev",
    "db:seed": "tsx prisma/seed.ts",
    "db:studio": "prisma studio",
    "db:generate": "prisma generate",
    "lint": "eslint src/ --ext .ts",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@prisma/client": "^5.10.2",
    "bcrypt": "^5.1.1",
    "decimal.js": "^10.4.3"
  },
  "devDependencies": {
    "@types/bcrypt": "^5.0.2",
    "prisma": "^5.10.2",
    "tsx": "^4.7.1",
    "typescript": "^5.4.2"
  }
}
```

---

## 9. `packages/ai/package.json`

```json
{
  "name": "@salescraft/ai",
  "version": "0.1.0",
  "private": true,
  "main": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "import": "./dist/index.js"
    },
    "./*": {
      "types": "./dist/*.d.ts",
      "import": "./dist/*.js"
    }
  },
  "scripts": {
    "build": "tsc",
    "dev": "tsc --watch",
    "lint": "eslint src/ --ext .ts",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "@aws-sdk/client-bedrock-runtime": "^3.515.0",
    "zod": "^3.22.4"
  },
  "devDependencies": {
    "typescript": "^5.4.2"
  }
}
```

---

## 10. `.env.example`

```bash
# =============================================================================
# Salescraft Environment Variables
# =============================================================================
# Copy this file to .env and fill in your values.
# NEVER commit the .env file to version control.

# -----------------------------------------------------------------------------
# Application
# -----------------------------------------------------------------------------
NODE_ENV=development
APP_URL=http://localhost:3000
API_URL=http://localhost:3001
API_PORT=3001

# -----------------------------------------------------------------------------
# Database (PostgreSQL)
# -----------------------------------------------------------------------------
DATABASE_URL=postgresql://salescraft:salescraft@localhost:5432/salescraft?schema=public

# -----------------------------------------------------------------------------
# Redis
# -----------------------------------------------------------------------------
REDIS_URL=redis://localhost:6379

# -----------------------------------------------------------------------------
# Authentication / JWT
# -----------------------------------------------------------------------------
JWT_SECRET=your-jwt-secret-min-32-chars-long
JWT_REFRESH_SECRET=your-refresh-jwt-secret-min-32-chars
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# -----------------------------------------------------------------------------
# Google OAuth (Gmail & Calendar integration)
# -----------------------------------------------------------------------------
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URI=http://localhost:3001/api/auth/google/callback

# -----------------------------------------------------------------------------
# Microsoft OAuth (Outlook & Calendar integration)
# -----------------------------------------------------------------------------
MICROSOFT_CLIENT_ID=your-microsoft-client-id
MICROSOFT_CLIENT_SECRET=your-microsoft-client-secret
MICROSOFT_REDIRECT_URI=http://localhost:3001/api/auth/microsoft/callback
MICROSOFT_TENANT_ID=common

# -----------------------------------------------------------------------------
# AWS (S3 file storage + Bedrock AI)
# -----------------------------------------------------------------------------
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key

# S3 / LocalStack
S3_BUCKET=salescraft-files
S3_ENDPOINT=http://localhost:4566
S3_FORCE_PATH_STYLE=true

# Bedrock (AI model inference)
BEDROCK_MODEL_ID=anthropic.claude-3-sonnet-20240229-v1:0
BEDROCK_EMBEDDING_MODEL_ID=amazon.titan-embed-text-v2:0

# -----------------------------------------------------------------------------
# Email (SMTP)
# -----------------------------------------------------------------------------
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@salescraft.local

# -----------------------------------------------------------------------------
# Mapbox (map visualizations)
# -----------------------------------------------------------------------------
MAPBOX_TOKEN=pk.your-mapbox-public-token

# -----------------------------------------------------------------------------
# Bull Board (job queue dashboard)
# -----------------------------------------------------------------------------
BULL_BOARD_PORT=3002
BULL_BOARD_USERNAME=admin
BULL_BOARD_PASSWORD=admin

# -----------------------------------------------------------------------------
# File Upload
# -----------------------------------------------------------------------------
MAX_FILE_SIZE_MB=50
ALLOWED_FILE_TYPES=pdf,doc,docx,xls,xlsx,csv,png,jpg,jpeg,dwg,dxf

# -----------------------------------------------------------------------------
# Feature Flags
# -----------------------------------------------------------------------------
ENABLE_AI_FEATURES=true
ENABLE_EMAIL_SYNC=true
ENABLE_CALENDAR_SYNC=true

# -----------------------------------------------------------------------------
# Logging
# -----------------------------------------------------------------------------
LOG_LEVEL=debug
```

---

## 11. `podman/postgres/init.sql`

```sql
-- =============================================================================
-- PostgreSQL Initialization Script for Salescraft
-- =============================================================================
-- This script runs on first database creation inside the Podman container.

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgvector";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "unaccent";

-- Create custom text search configuration for improved full-text search
-- This configuration uses unaccent to normalize accented characters
CREATE TEXT SEARCH CONFIGURATION salescraft (COPY = english);
ALTER TEXT SEARCH CONFIGURATION salescraft
  ALTER MAPPING FOR hword, hword_part, word
  WITH unaccent, english_stem;
```

---

## 12. `podman/localstack/init-s3.sh`

```bash
#!/bin/bash
# =============================================================================
# LocalStack S3 Initialization Script
# =============================================================================
# Creates the required S3 bucket and configures CORS for local development.

set -euo pipefail

echo "Creating salescraft-files S3 bucket..."

awslocal s3 mb s3://salescraft-files

echo "Setting CORS configuration on salescraft-files bucket..."

awslocal s3api put-bucket-cors --bucket salescraft-files --cors-configuration '{
  "CORSRules": [
    {
      "AllowedOrigins": ["http://localhost:3000"],
      "AllowedMethods": ["PUT", "GET", "HEAD"],
      "AllowedHeaders": ["*"],
      "ExposeHeaders": ["ETag", "Content-Length", "Content-Type"],
      "MaxAgeSeconds": 3600
    }
  ]
}'

echo "S3 bucket salescraft-files created and CORS configured successfully."
```

---

## 13. `.eslintrc.js`

```js
/** @type {import('eslint').Linter.Config} */
module.exports = {
  root: true,
  env: {
    node: true,
    es2022: true,
  },
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaVersion: 2022,
    sourceType: 'module',
  },
  plugins: ['@typescript-eslint'],
  extends: ['eslint:recommended', 'plugin:@typescript-eslint/recommended'],
  rules: {
    '@typescript-eslint/no-unused-vars': [
      'warn',
      { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
    ],
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/consistent-type-imports': [
      'error',
      { prefer: 'type-imports' },
    ],
    'no-console': ['warn', { allow: ['warn', 'error'] }],
  },
  ignorePatterns: ['dist/', 'node_modules/', '.next/', 'coverage/'],
};
```

---

## 14. `.prettierrc`

```json
{
  "semi": true,
  "singleQuote": true,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2,
  "bracketSpacing": true,
  "arrowParens": "always",
  "endOfLine": "lf"
}
```

---

## Notes

- All version numbers are pinned to stable releases available as of early 2025.
- The `workspace:*` protocol is pnpm's mechanism for linking internal packages.
- The `tsconfig.base.json` is extended by each app/package's own `tsconfig.json`.
- The init scripts are mounted into their respective containers via `podman-compose.yml` (see spec/15-deployment-infrastructure.md).
- The `.env.example` uses LocalStack endpoints for S3 in development; production values are configured via environment-specific secrets.

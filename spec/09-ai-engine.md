# AI Engine

## Vision & Purpose

AI is not a feature — it's the nervous system of Salescraft. Every module relies on AI capabilities: intelligence uses AI to classify signals and score opportunities, relationships use AI to extract interests and generate briefing cards, estimating uses AI to write proposals, and bid response uses AI to parse RFPs. Rather than scattering AI code across modules, the AI Engine provides a unified abstraction layer over AWS Bedrock with model routing, cost management, prompt versioning, and safety guardrails.

The key design principle: **AI should be as easy to use as a database query.** Any module can call `ai.generate(promptTemplate, context)` without worrying about which model to use, how to handle retries, or how to parse structured output. The routing layer makes those decisions.

## Key Concepts

- **Model Router** — Selects the appropriate Bedrock model based on task complexity, cost, and latency requirements
- **Prompt Template** — Versioned prompt with variable interpolation, stored as code
- **Structured Output** — AI responses parsed into typed TypeScript objects (not raw strings)
- **Embedding** — Vector representation of text for semantic search and similarity
- **RAG (Retrieval-Augmented Generation)** — Querying relevant context from the vector store before generating responses
- **Token Budget** — Per-request and daily cost limits to prevent runaway spending

## User Stories

### System (Internal)
- As the intelligence module, I need to classify signals as flooring-related with high accuracy (P0)
- As the relationship module, I need to extract personal interests from conversation text (P0)
- As the estimating module, I need to generate professional proposal narratives (P1)
- As the bid module, I need to parse RFP documents into structured data (P1)
- As the relationship module, I need to generate natural conversation starters (P1)
- As the communication module, I need to draft contextual email responses (P2)

### Admin/Owner
- As an admin, I want to see AI usage costs and which features consume the most tokens (P1)
- As an admin, I want to set daily/monthly spending limits to prevent runaway costs (P1)

## Technical Design

### Architecture

```
┌──────────────────────────────────────────────────────────┐
│                      AI Engine Package                      │
│  packages/ai/src/                                          │
│                                                            │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐ │
│  │ Model Router │  │ Prompt Store │  │ Embedding Engine │ │
│  │             │  │              │  │                 │ │
│  │ - Task→Model│  │ - Templates  │  │ - Generate      │ │
│  │ - Fallback  │  │ - Versioning │  │ - Store/Query   │ │
│  │ - Cost track│  │ - Variables  │  │ - Similarity    │ │
│  └──────┬──────┘  └──────┬───────┘  └────────┬────────┘ │
│         │                 │                    │           │
│  ┌──────┴─────────────────┴────────────────────┴────────┐ │
│  │                  Bedrock Client                        │ │
│  │  @aws-sdk/client-bedrock-runtime                      │ │
│  │  - InvokeModel / InvokeModelWithResponseStream        │ │
│  │  - Retry with exponential backoff                     │ │
│  │  - Token counting and cost tracking                   │ │
│  └───────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### Data Model

#### AIUsageLog
```typescript
interface AIUsageLog {
  id: string;
  taskType: string;               // "signal_classification", "interest_extraction", etc.
  modelId: string;                // "anthropic.claude-3-5-sonnet-20241022-v2:0"
  inputTokens: number;
  outputTokens: number;
  cost: number;                   // Calculated USD cost
  latencyMs: number;
  success: boolean;
  errorMessage?: string;
  userId?: string;                // FK → User (if user-initiated)
  moduleSource: string;           // "intelligence", "relationships", "estimating", etc.
  createdAt: DateTime;
}
```

#### EmbeddingRecord
```typescript
interface EmbeddingRecord {
  id: string;
  entityType: string;             // "contact", "interaction", "bid_document", "email"
  entityId: string;               // FK → source entity
  content: string;                // Original text that was embedded
  contentHash: string;            // For dedup/invalidation
  embedding: number[];            // Vector (stored in pgvector)
  modelId: string;                // Embedding model used
  createdAt: DateTime;
}
```

### Model Router

```typescript
// packages/ai/src/router.ts

interface ModelConfig {
  modelId: string;                // Bedrock model ID
  maxInputTokens: number;
  maxOutputTokens: number;
  costPerInputToken: number;      // USD
  costPerOutputToken: number;     // USD
  avgLatencyMs: number;           // Expected latency
  capabilities: string[];         // What it's good at
}

const MODELS: Record<string, ModelConfig> = {
  'claude-sonnet': {
    modelId: 'anthropic.claude-3-5-sonnet-20241022-v2:0',
    maxInputTokens: 200000,
    maxOutputTokens: 8192,
    costPerInputToken: 0.000003,
    costPerOutputToken: 0.000015,
    avgLatencyMs: 3000,
    capabilities: ['complex_reasoning', 'long_documents', 'creative_writing', 'code_generation'],
  },
  'claude-haiku': {
    modelId: 'anthropic.claude-3-5-haiku-20241022-v1:0',
    maxInputTokens: 200000,
    maxOutputTokens: 8192,
    costPerInputToken: 0.0000008,
    costPerOutputToken: 0.000004,
    avgLatencyMs: 1000,
    capabilities: ['classification', 'extraction', 'simple_generation', 'scoring'],
  },
  'titan-embed': {
    modelId: 'amazon.titan-embed-text-v2:0',
    maxInputTokens: 8192,
    maxOutputTokens: 0,           // Embeddings only
    costPerInputToken: 0.0000002,
    costPerOutputToken: 0,
    avgLatencyMs: 200,
    capabilities: ['embedding'],
  },
};

// Task → Model mapping
const TASK_MODEL_MAP: Record<string, string> = {
  // Fast, cheap tasks → Haiku
  'signal_classification': 'claude-haiku',
  'interest_extraction': 'claude-haiku',
  'sentiment_analysis': 'claude-haiku',
  'entity_matching': 'claude-haiku',
  'lead_scoring_explanation': 'claude-haiku',
  'email_categorization': 'claude-haiku',

  // Complex tasks → Sonnet
  'rfp_parsing': 'claude-sonnet',
  'proposal_writing': 'claude-sonnet',
  'conversation_starters': 'claude-sonnet',
  'outreach_suggestions': 'claude-sonnet',
  'competitive_analysis': 'claude-sonnet',
  'meeting_summary': 'claude-sonnet',
  'email_drafting': 'claude-sonnet',

  // Embeddings → Titan
  'embedding': 'titan-embed',
};

interface RouterOptions {
  taskType: string;
  maxLatencyMs?: number;          // Override if latency-sensitive
  forceModel?: string;            // Override routing
  maxCostUsd?: number;            // Per-request budget
}

function selectModel(options: RouterOptions): ModelConfig {
  if (options.forceModel) return MODELS[options.forceModel];
  const preferredModel = TASK_MODEL_MAP[options.taskType] || 'claude-haiku';
  return MODELS[preferredModel];
}
```

### Core API

```typescript
// packages/ai/src/client.ts

interface GenerateOptions {
  taskType: string;               // Routes to correct model
  prompt: string;                 // Final rendered prompt
  systemPrompt?: string;
  temperature?: number;           // Default 0.3 for determinism
  maxTokens?: number;
  stream?: boolean;               // Streaming response
  responseFormat?: 'text' | 'json'; // If json, validates response
  schema?: ZodSchema;             // If provided, parse and validate response
  retries?: number;               // Default 2
  timeout?: number;               // Default 30000ms
}

interface GenerateResult<T = string> {
  content: T;                     // Parsed response (string or typed object)
  model: string;                  // Which model was used
  inputTokens: number;
  outputTokens: number;
  cost: number;                   // USD
  latencyMs: number;
}

// Usage in other modules:
import { ai } from '@salescraft/ai';

// Simple text generation
const result = await ai.generate({
  taskType: 'conversation_starters',
  prompt: renderPrompt('briefing-generator', { contactName, interests, sharedInterests }),
});

// Structured output with validation
const parsed = await ai.generate({
  taskType: 'rfp_parsing',
  prompt: renderPrompt('rfp-parser', { documentText }),
  responseFormat: 'json',
  schema: RfpParseResultSchema,   // Zod schema — retries if validation fails
});

// Streaming
const stream = await ai.generate({
  taskType: 'proposal_writing',
  prompt: renderPrompt('proposal-writer', { projectDetails, qualifications }),
  stream: true,
});
for await (const chunk of stream) {
  // Send to client via WebSocket
}
```

### Embedding Engine

```typescript
// packages/ai/src/embeddings.ts

interface EmbedOptions {
  text: string;
  entityType: string;
  entityId: string;
}

interface SearchOptions {
  query: string;
  entityTypes?: string[];         // Filter by type
  limit?: number;                 // Max results (default 10)
  threshold?: number;             // Minimum similarity (default 0.7)
}

interface SearchResult {
  entityType: string;
  entityId: string;
  content: string;
  similarity: number;             // 0-1
}

// Usage:
import { embeddings } from '@salescraft/ai';

// Store embedding
await embeddings.embed({
  text: 'Meeting notes: Discussed the flooring project timeline...',
  entityType: 'interaction',
  entityId: 'interaction-123',
});

// Semantic search
const results = await embeddings.search({
  query: 'which contacts are interested in sustainability?',
  entityTypes: ['contact_interest', 'interaction'],
  limit: 20,
});
```

### RAG Pipeline

```typescript
// packages/ai/src/rag.ts

interface RAGOptions {
  query: string;                  // The user's question or task
  contextSources: string[];       // Entity types to pull context from
  maxContextTokens?: number;      // How much context to include (default 4000)
  taskType: string;               // For model routing
  prompt: string;                 // Prompt template with {context} placeholder
}

// Usage: Generate an outreach suggestion with relevant context
const suggestion = await ai.rag({
  query: 'Generate an outreach suggestion for Mike Smith',
  contextSources: ['interaction', 'contact_interest', 'opportunity'],
  taskType: 'outreach_suggestions',
  prompt: `Given the following context about the contact:
{context}

Generate a natural, personalized reason to reach out to {contactName}.`,
});
```

### Prompt Templates

```typescript
// packages/ai/src/prompts/signal-classifier.ts
export const SIGNAL_CLASSIFIER = {
  name: 'signal-classifier',
  version: '1.0.0',
  model: 'claude-haiku',
  temperature: 0.1,
  systemPrompt: 'You are a classifier for government bid and project postings related to commercial flooring.',
  template: `Classify this posting:
Title: {{title}}
Description: {{description}}
Source: {{source}}

Respond in JSON:
{
  "is_flooring": boolean,
  "confidence": number (0-1),
  "flooring_types": string[],
  "estimated_sqft": number | null,
  "estimated_value": number | null,
  "reasoning": string
}`,
  schema: z.object({
    is_flooring: z.boolean(),
    confidence: z.number().min(0).max(1),
    flooring_types: z.array(z.string()),
    estimated_sqft: z.number().nullable(),
    estimated_value: z.number().nullable(),
    reasoning: z.string(),
  }),
};

// packages/ai/src/prompts/interest-extractor.ts
export const INTEREST_EXTRACTOR = {
  name: 'interest-extractor',
  version: '1.0.0',
  model: 'claude-haiku',
  temperature: 0.2,
  systemPrompt: 'You extract personal interests and hobbies from conversation notes. Only extract clear personal interests, not business topics.',
  template: `Analyze this text for personal interests of {{contactName}}:

"{{content}}"

For each interest found, provide:
- category: one of [sports_team, sports_activity, outdoors, food_drink, music, travel, family, education, community, hobbies, pets, entertainment]
- name: specific interest
- specifics: additional details if available
- confidence: 0-1

Return JSON array. Empty array if no interests found.`,
  schema: z.array(z.object({
    category: z.enum([/* InterestCategory values */]),
    name: z.string(),
    specifics: z.string().nullable(),
    confidence: z.number().min(0).max(1),
  })),
};

// packages/ai/src/prompts/proposal-writer.ts
export const PROPOSAL_WRITER = {
  name: 'proposal-writer',
  version: '1.0.0',
  model: 'claude-sonnet',
  temperature: 0.5,
  systemPrompt: `You write professional proposal narratives for commercial flooring installation bids. 
The tone should be confident, professional, and specific to the project. 
Highlight relevant experience, qualifications, and approach.
Do NOT include pricing information — that comes from the estimate.`,
  template: `Write proposal sections for this project:

Project: {{projectTitle}}
Client: {{organizationName}}
Scope: {{scope}}
Product(s): {{products}}
Timeline: {{timeline}}
Our Qualifications: {{qualifications}}
Similar Past Projects: {{pastProjects}}

Generate the following sections:
1. Executive Summary (2-3 paragraphs)
2. Understanding of Scope
3. Technical Approach
4. Project Schedule
5. Quality Assurance
6. Company Qualifications
7. Relevant Experience

Format as markdown with section headers.`,
};

// packages/ai/src/prompts/briefing-generator.ts
export const BRIEFING_GENERATOR = {
  name: 'briefing-generator',
  version: '1.0.0',
  model: 'claude-sonnet',
  temperature: 0.7,
  systemPrompt: `Generate natural, genuine conversation starters for a sales rep meeting a contact. 
Focus on personal rapport, NOT business. Be warm but professional. 
Reference shared interests and recent events naturally.`,
  template: `Generate 3 conversation starters for this upcoming interaction:

Contact: {{contactName}}, {{title}} at {{organization}}
Their interests: {{interests}}
Your shared interests: {{sharedInterests}}
Recent life events: {{lifeEvents}}
Last personal notes from previous conversations: {{personalNotes}}
Days since last contact: {{daysSinceContact}}
Season/time of year context: {{seasonalContext}}

Requirements:
- Feel natural, not scripted
- Reference specific shared interests or recent events
- Show memory of past conversations
- Appropriate for a professional sales relationship
- Short (1-2 sentences each)`,
};

// packages/ai/src/prompts/rfp-parser.ts
export const RFP_PARSER = {
  name: 'rfp-parser',
  version: '1.0.0',
  model: 'claude-sonnet',
  temperature: 0.1,
  maxTokens: 4096,
  systemPrompt: `You parse government RFP/IFB documents and extract structured information. 
Be precise with dates and requirements. If information is not found, use null.`,
  template: `Parse this government bid document and extract key information:

{{documentText}}

Extract as JSON:
{
  "title": "official bid title",
  "bidNumber": "bid/RFP number or null",
  "issuingAgency": "agency name",
  "scope": "brief scope description",
  "estimatedSqFt": number or null,
  "flooringTypes": ["types mentioned"],
  "deadlines": {
    "preBidMeeting": {"date": "ISO", "location": "string", "mandatory": boolean} or null,
    "questionsDeadline": "ISO date" or null,
    "submissionDeadline": "ISO date with timezone",
    "awardDate": "ISO date" or null
  },
  "requirements": {
    "bidBond": boolean,
    "performanceBond": boolean,
    "prevailingWage": boolean,
    "insuranceMinimums": "description" or null,
    "experience": "requirement" or null,
    "certifications": ["required certs"] or []
  },
  "submissionFormat": {
    "copies": number or null,
    "format": "description",
    "sections": ["required sections"]
  },
  "evaluationCriteria": [{"criterion": "name", "weight": number or null}] or null
}`,
};
```

### API Endpoints

```typescript
// AI Administration
GET    /api/v1/ai/usage                        // Usage stats (tokens, cost, by module)
GET    /api/v1/ai/usage/daily                  // Daily cost breakdown
GET    /api/v1/ai/models                       // Available models and routing config
PUT    /api/v1/ai/config                       // Update routing or budget config
GET    /api/v1/ai/prompts                      // List prompt templates and versions

// These are NOT direct AI endpoints (modules call AI internally)
// But for admin visibility and debugging:
GET    /api/v1/ai/logs                         // Recent AI calls with inputs/outputs
GET    /api/v1/ai/logs/:id                     // Specific call detail
```

### Business Rules

- **BR-AI-001:** Daily cost limit default: $50/day. If exceeded, non-critical AI tasks queue until next day. Critical tasks (bid deadline approaching) bypass the limit.
- **BR-AI-002:** Monthly cost limit: $1000/month. Alert owner at 80% threshold.
- **BR-AI-003:** Retry logic: on Bedrock throttling (429) or timeout, retry with exponential backoff (1s, 3s, 9s). Max 3 retries. On persistent failure, fall back to simpler model if available.
- **BR-AI-004:** Structured output validation: if AI response doesn't match the Zod schema, retry with a correction prompt (once). If still invalid, return error to caller.
- **BR-AI-005:** Embeddings are regenerated when source content changes (content hash comparison). Stale embeddings (>90 days without content change) are still valid.
- **BR-AI-006:** All AI inputs and outputs are logged for debugging and improvement. PII in logs is retained for 30 days only.
- **BR-AI-007:** AI-generated content (proposals, email drafts, suggestions) is ALWAYS marked as "AI-generated" and requires human review before external use.
- **BR-AI-008:** Temperature settings: 0.1 for classification/extraction (deterministic), 0.3-0.5 for structured generation, 0.7 for creative writing (conversation starters, outreach ideas).
- **BR-AI-009:** Token budget per request: input limited to 50% of model's max context. If document exceeds this, truncate intelligently (keep first and last pages, section headers).
- **BR-AI-010:** Cost calculation formula: `(inputTokens * costPerInputToken) + (outputTokens * costPerOutputToken)`. Logged per request and aggregated per module per day.

### Cost Estimation

| Task | Model | Avg Tokens | Estimated Cost | Daily Volume | Daily Cost |
|------|-------|-----------|---------------|-------------|-----------|
| Signal classification | Haiku | 500 in / 200 out | $0.001 | 50 signals | $0.05 |
| Interest extraction | Haiku | 1000 in / 300 out | $0.002 | 20 interactions | $0.04 |
| RFP parsing | Sonnet | 10000 in / 2000 out | $0.06 | 2 RFPs | $0.12 |
| Proposal writing | Sonnet | 5000 in / 3000 out | $0.06 | 1 proposal | $0.06 |
| Briefing generation | Sonnet | 2000 in / 500 out | $0.014 | 10 briefings | $0.14 |
| Outreach suggestions | Sonnet | 3000 in / 800 out | $0.021 | 5 suggestions | $0.10 |
| Embeddings | Titan | 500 in | $0.0001 | 100 embeds | $0.01 |
| **Total estimated daily cost** | | | | | **~$0.52** |

## Implementation Guide

### File Locations
```
packages/ai/
├── src/
│   ├── index.ts                 # Public API exports
│   ├── client.ts                # Bedrock client wrapper
│   ├── router.ts                # Model selection logic
│   ├── embeddings.ts            # Embedding generation and search
│   ├── rag.ts                   # RAG pipeline
│   ├── cost-tracker.ts          # Usage tracking and budgets
│   ├── types.ts                 # TypeScript interfaces
│   └── prompts/
│       ├── index.ts             # Prompt registry
│       ├── signal-classifier.ts
│       ├── interest-extractor.ts
│       ├── proposal-writer.ts
│       ├── briefing-generator.ts
│       ├── rfp-parser.ts
│       ├── outreach-suggester.ts
│       ├── conversation-analyzer.ts
│       └── email-drafter.ts
├── test/
│   ├── router.test.ts
│   ├── embeddings.test.ts
│   └── prompts/
│       └── *.test.ts            # Per-prompt output validation tests
└── package.json
```

### Key Dependencies
- `@aws-sdk/client-bedrock-runtime` — Bedrock API calls
- `zod` — Response schema validation
- `tiktoken` (or custom approximation) — Token counting before sending
- `pgvector` — Vector similarity search (via Prisma raw query)

### Implementation Order
1. Bedrock client wrapper (invoke, stream, retry)
2. Model router (task → model mapping)
3. Cost tracker (log usage, enforce budgets)
4. Prompt template system (render with variables)
5. Structured output (JSON parsing + Zod validation + retry)
6. Embedding generation and storage (pgvector)
7. Embedding search (similarity query)
8. RAG pipeline (search + augment + generate)
9. Streaming support (for proposal writing)
10. Admin API (usage stats, config)

### Common Pitfalls
- **Token counting:** Estimate tokens BEFORE sending to avoid "context too long" errors. Claude uses ~1.3 tokens per word as a rough heuristic.
- **JSON parsing from AI:** Models sometimes wrap JSON in markdown code fences. Strip these before parsing.
- **Embedding dimensions:** Titan Embed v2 outputs 1024-dimensional vectors. Ensure pgvector column matches.
- **Rate limiting:** Bedrock has per-model rate limits (requests/second and tokens/minute). Implement a queue with concurrency control.
- **Cost drift:** Without budgets, AI costs can escalate quickly if a bug causes infinite retries or repeated calls. Always have circuit breakers.

## Testing Requirements

### Unit Tests
- Router: `taskType = 'signal_classification'` → selects Haiku
- Router: `taskType = 'proposal_writing'` → selects Sonnet
- Router: with `forceModel = 'claude-sonnet'` → overrides default
- Cost calculation: 1000 input tokens on Haiku → $0.0008
- Token estimation: 500-word text → ~650 tokens (within 20% tolerance)
- Prompt rendering: template with variables → correct output
- Structured output: valid JSON matching schema → parsed successfully
- Structured output: invalid JSON → retry triggered

### Integration Tests
- Full generate flow: prompt → Bedrock call → response parsed → usage logged
- Embedding: generate → store in pgvector → query by similarity → results returned
- RAG: query → relevant embeddings found → augmented prompt → generated response
- Budget enforcement: daily limit reached → non-critical tasks queued
- Streaming: proposal writing → chunks received in order → complete response assembled

### Mock Strategy
- Unit tests mock the Bedrock SDK (don't hit AWS)
- Integration tests use Bedrock (require AWS credentials)
- Test prompts with deterministic temperature (0.0) for reproducibility
- Keep a fixtures file with sample AI responses for snapshot testing

## Error Handling

| Failure | Handling |
|---------|----------|
| Bedrock throttled (429) | Exponential backoff: 1s, 3s, 9s. After 3 retries, queue for later. |
| Bedrock timeout | Retry once. If second attempt timeouts, return error to caller. |
| Bedrock model unavailable | Fall back: Sonnet → Haiku for degraded service. Log degradation. |
| JSON parse failure | Retry with prompt: "Your previous response was not valid JSON. Please try again." (once) |
| Schema validation failure | Retry with prompt showing validation errors. If 2nd attempt fails, return raw text. |
| Daily budget exceeded | Queue non-critical tasks. Allow critical tasks (bid-deadline proximity). Alert admin. |
| Invalid credentials | Fatal error. Alert immediately. No retry. |

## UI/UX Requirements

### AI Usage Dashboard (Admin)
- Daily/monthly cost chart with breakdown by module
- Top consumers table: which prompts/tasks cost the most
- Budget status: current spend vs. limits with projection
- Recent calls log: searchable, filterable, expandable to see full input/output
- Model performance: latency p50/p95/p99 per model

### AI-Generated Content Indicator
- Everywhere AI content is shown (proposals, suggestions, briefings), display a small "AI" badge
- Content is editable before use
- "Regenerate" button to get a different response
- Feedback mechanism: thumbs up/down on AI suggestions (for future improvement)

## Integration Points

| System | Purpose | Direction | Frequency |
|--------|---------|-----------|-----------|
| AWS Bedrock | Model invocation | Outbound | On-demand |
| PostgreSQL (pgvector) | Vector storage and search | Bidirectional | On embed/search |
| All Salescraft modules | AI capabilities provider | Consumed | On-demand |
| BullMQ | Batch processing queue | Internal | Background jobs |

## Performance Requirements

- Haiku classification response: < 2 seconds
- Sonnet generation response: < 5 seconds (first token streaming < 2s)
- Embedding generation: < 1 second
- Vector similarity search (10K embeddings): < 200ms
- Cost tracking overhead: < 10ms per request

## Non-Functional Requirements

- All prompts versioned in code (not in database) for reproducibility
- AI responses never shown to external parties without human review
- Usage logs retained for 90 days (for cost analysis and prompt improvement)
- PII in AI logs automatically redacted after 30 days
- System must function (degraded) if Bedrock is temporarily unavailable

## Resolved Design Decisions

- **Prompt A/B testing:** Not for MVP. Use versioned prompt templates and evaluate quality manually. A/B testing requires significant traffic volume to be statistically meaningful — we won't have enough at launch.
- **Response caching:** Yes, cache briefing cards (4 hours), interest extraction results (until new interaction), and signal classifications (permanent for same input hash). Invalidate briefing cache on new interaction with that contact. Don't cache proposal generation (always fresh).
- **Bedrock Knowledge Bases vs. pgvector:** Use pgvector for RAG. Simpler architecture (no additional service), sufficient for our scale (<100K embeddings), and we control the chunking/retrieval logic directly. Bedrock Knowledge Bases adds operational complexity with minimal benefit at our size.
- **Fine-tuning:** Not needed. Claude models perform well on flooring/construction classification with good prompts. Fine-tuning requires training data we don't have yet and costs ongoing maintenance. Revisit after 6 months of production usage data.

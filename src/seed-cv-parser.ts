import { createContextManager } from "./client";
import { AgentContextNode } from "./types";

const cm = createContextManager();
const P = "cv-parser";

async function seed() {
  console.log("Seeding CV Parser technical execution plan...\n");

  // ═══════════════════════════════════════════════════════════════════════════
  // SKILLS
  // ═══════════════════════════════════════════════════════════════════════════
  console.log("--- Adding Skills ---");

  await cm.addSkill(
    "CV Parsing with Gemini Multimodal",
    "Extract structured candidate profile data from PDF CVs using Google Gemini Flash 3.1 Lite multimodal capabilities. Uses ADK for orchestration and Tool Calling for structured output with entity resolution.",
    P
  );

  await cm.addSkill(
    "ADK Agent Orchestration",
    "Google ADK framework for managing LLM extraction flows with function calling tools. Tools include resolveOrganisations, resolveCertifications, resolveSkills, and resolveCity for entity resolution during CV parsing.",
    P
  );

  await cm.addSkill(
    "Gemini Context Caching",
    "Lazy-initialized Gemini context cache bundling system prompt, tool declarations, and Fields of Study taxonomy (~1200 rows). 72h TTL with automatic refresh on expiration. Reduces token costs on repeated extractions.",
    P
  );

  await cm.addSkill(
    "Replace-by-Deletion Transactions",
    "Atomic transactional pattern for overwriting candidate profile sections. Deletes existing data for selected sections then inserts new parsed data within a single DB transaction. Prevents partial state on failure.",
    P
  );

  await cm.addSkill(
    "SSE Realtime Updates via Pub/Sub",
    "Server-Sent Events integration through realtime-gateway for live CV parsing status updates. Pub/Sub command to realtime-gateway-cmd-send-event topic, routed to candidate:{candidateId} channel.",
    P
  );

  await cm.addSkill(
    "Golden Dataset LLM Evaluation",
    "Evaluation framework for non-deterministic LLM outputs using 80+ anonymized CVs with ground_truth.json. Measures schema match rate, field-level accuracy (exact match, F1, semantic similarity), and hallucination rate.",
    P
  );

  // ═══════════════════════════════════════════════════════════════════════════
  // MEMORIES — Key decisions, constraints, and architectural rules
  // ═══════════════════════════════════════════════════════════════════════════
  console.log("\n--- Adding Memories ---");

  await cm.addMemory(
    "The cv-parser module lives inside data service (not a separate service) to reduce complexity, inter-service comms, and network round-trips. All entity resolution happens on data service except city resolution which calls placesSvc.",
    "decision",
    "agent",
    9,
    P,
    {},
    false
  );

  await cm.addMemory(
    "candidate_cv_parsing_tracking table acts as the state machine and single source of truth for CV extraction. States: pending, processing, completed, failed. Only one active job (PENDING/PROCESSING) allowed per candidate at a time to prevent race conditions.",
    "architecture",
    "agent",
    10,
    P,
    {},
    false
  );

  await cm.addMemory(
    "SSE events are fire-and-forget for the UI. The candidate_parsing_jobs DB table is the canonical source of truth. On SSE disconnect/reconnect, frontend re-syncs via currentCvParseJob polling query. Low-frequency polling is an optional fallback.",
    "architecture",
    "agent",
    8,
    P,
    {},
    false
  );

  await cm.addMemory(
    "The LLM extraction uses Google Gemini 3.1 Flash Lite with ADK tool calling. Instead of putting massive ID lists in the prompt, the LLM calls tools (resolveOrganisations, resolveCertifications, resolveSkills, resolveCity) to query internal taxonomies. All tool calls must be batched where possible.",
    "architecture",
    "agent",
    9,
    P,
    {},
    false
  );

  await cm.addMemory(
    "Entity resolution for skills uses 'discard' logic: skills not found in the taxonomy are silently dropped. For organisations and certificates, a 'get-or-create' pattern is used (INSERT ... ON CONFLICT DO NOTHING). Organisations must be resolved before certificates because certificates need provider_id.",
    "constraint",
    "agent",
    9,
    P,
    {},
    false
  );

  await cm.addMemory(
    "Replace-by-deletion is atomic and transactional. If parse fails, no deletion occurs. If parse succeeds but DB insertion fails, job status is set to FAILED. The mutation startCVParse only returns the job ID; deletion + insertion happens after successful parse, before setting status to COMPLETED.",
    "constraint",
    "agent",
    10,
    P,
    {},
    false
  );

  await cm.addMemory(
    "Rate limiting: startCvParsing is limited to 2 requests per 60 seconds per candidate ID. Storage upload URLs are restricted to 10MB max size and signed for 15 minutes.",
    "constraint",
    "agent",
    7,
    P,
    {},
    false
  );

  await cm.addMemory(
    "All date strings in CV parsing must be YYYY-MM-DD format. Dates are normalized to day 02 of month as standard for TP dates which have only month accuracy.",
    "constraint",
    "agent",
    6,
    P,
    {},
    false
  );

  await cm.addMemory(
    "CV parsing accuracy benchmarks (86 CVs tested): 92.4% avg score, ~15s avg latency, 84.5% cache hit rate, $0.00223/CV cost. 84 passed, 0 failed, 2 expected errors.",
    "observation",
    "agent",
    8,
    P,
    {},
    false
  );

  await cm.addMemory(
    "Overwrite flow for existing candidates: frontend shows overwrite selection modal with unchecked-by-default checkboxes. User selects sections to overwrite, frontend calls startFullCvParsing with selectedSections. Only selected sections are re-parsed and replaced via replace-by-deletion.",
    "decision",
    "agent",
    8,
    P,
    {},
    false
  );

  await cm.addMemory(
    "Security mitigations: CV content treated as untrusted input (prompt injection risk), strict tool allowlist, output schema validation, server-side post-validation. Ownership checks enforce candidateId from session never client-trusted (IDOR prevention). Single active job prevents race conditions.",
    "constraint",
    "agent",
    9,
    P,
    {},
    false
  );

  await cm.addMemory(
    "Observability: extraction telemetry sent to BigQuery on terminal states (EXTRACTED/COMPLETED/FAILED/ABORTED). Tracks taskId, selectedSections, status, model, latency, tokens (input/output/cached), and estimated cost in EUR.",
    "observation",
    "agent",
    6,
    P,
    {},
    false
  );

  // ═══════════════════════════════════════════════════════════════════════════
  // CONTEXT NODES — Hierarchical architecture tree
  // ═══════════════════════════════════════════════════════════════════════════
  console.log("\n--- Adding Context Nodes ---");

  // Root
  const root = await cm.addContextNode(
    "contextfs://cv-parser",
    "CV Parser System",
    "A new module inside data service that extracts structured candidate profile data from PDF CVs using Google Gemini Flash 3.1 Lite multimodal capabilities with ADK orchestration and tool calling for entity resolution.",
    "The cv-parser module handles full CV extraction for new candidates during application submission and selective section overwrite for existing candidates. It uses a state machine (candidate_cv_parsing_tracking) to manage job lifecycle, SSE for realtime updates, and atomic replace-by-deletion for data persistence.",
    undefined,
    null,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Data Model ──
  const dataModel = await cm.addContextNode(
    "contextfs://cv-parser/data-model",
    "Data Model & Schema",
    "Core database schema additions: candidate_cv_parsing_tracking table as state machine for extraction process with status enum (pending/processing/completed/failed).",
    `CREATE TABLE candidate_cv_parsing_tracking (
    candidate_id        BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    cv_file_id          TEXT NOT NULL,
    status              TEXT NOT NULL,  -- pending, processing, completed, failed
    error_message       TEXT,
    result              TEXT,
    started_at          TIMESTAMPTZ DEFAULT NOW(),
    completed_at        TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_parsing_jobs_candidate_status ON candidate_parsing_jobs(candidate_id, status);

Only one active job (PENDING/PROCESSING) per candidate at a time. This table is the single source of truth for job state.`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── GraphQL Schema ──
  const graphql = await cm.addContextNode(
    "contextfs://cv-parser/graphql",
    "GraphQL Schema",
    "New GraphQL types, queries, and mutations for CV parsing: CvParseTask type, ParseJobStatus enum, ParseSection enum, CvParseInput input, and startFullCvParsing mutation.",
    `Types: CvParseTask (id, status, selectedSections, createdAt, completedAt, errorMessage), ParseJobStatus enum (PENDING/PROCESSING/COMPLETED/FAILED), ParseSection enum (IDENTITY/PROFESSIONAL_EXPERIENCE/EDUCATION/PROJECTS/CERTIFICATIONS/SKILLS/LANGUAGES).

Query: currentCvParseJob returns current parse job for authenticated candidate (polling).

Mutation: startFullCvParsing(input: CvParseInput!) where CvParseInput has fileId (from generateCvUploadInfo) and optional selectedSections (null = full parse). Returns existing job if one is active.

Validation: fileId must exist (FILE_NOT_FOUND), selectedSections must have at least one if provided (INVALID_SECTION_LIST), dates normalized to YYYY-MM-DD day 02.`,
    `enum ParseSection {
  IDENTITY
  PROFESSIONAL_EXPERIENCE
  EDUCATION
  PROJECTS
  CERTIFICATIONS
  SKILLS
  LANGUAGES
}

enum ParseJobStatus {
  PENDING
  PROCESSING
  COMPLETED
  FAILED
}

type CvParseTask {
  id: ID!
  status: ParseJobStatus!
  selectedSections: [ParseSection!]!
  createdAt: DateTime!
  completedAt: DateTime
  errorMessage: String
}

input CvParseInput {
  fileId: String!
  selectedSections: [ParseSection!]
}

type Query {
  currentCvParseJob: CvParseTask
}

type Mutation {
  startFullCvParsing(input: CvParseInput!): CvParseTask!
}`,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── AI Infrastructure ──
  const ai = await cm.addContextNode(
    "contextfs://cv-parser/ai-infrastructure",
    "AI Infrastructure",
    "Google Gemini 3.1 Flash Lite as primary engine with ADK for agent orchestration and tool calling. Gemini context caching for system prompt, tools, and taxonomy data.",
    "Primary engine: Gemini 3.1 Flash Lite. ADK manages extraction flow with function calling tools. Context cache bundles system prompt + tool declarations + Fields of Study taxonomy (~1200 rows). Lazy initialization with 72h TTL, automatic refresh on expiration. Cache resource name persisted in memory.",
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Context Cache ──
  await cm.addContextNode(
    "contextfs://cv-parser/ai-infrastructure/context-cache",
    "Gemini Context Cache",
    "Lazy-initialized context cache bundling system prompt, tool declarations, and Fields of Study taxonomy. 72h TTL with automatic refresh on cache expiration errors.",
    `Cache content: system prompt + tool declarations + 1200-row Fields of Study taxonomy.
Initialization: First extraction request checks for valid cacheResourceName in memory. If missing/expired, generates cache with 72h TTL.
Runtime: Injects cache resource name and strips system prompt/tool configs from payload (Gemini API requirement).
Refresh: On cache expiration error, recreates ad-hoc and retries. Handles pod scaling gracefully.`,
    undefined,
    ai.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── ADK Tools ──
  await cm.addContextNode(
    "contextfs://cv-parser/ai-infrastructure/adk-tools",
    "ADK Tool Calling",
    "Four ADK tools for entity resolution during extraction: resolveOrganisations, resolveCertifications, resolveSkills, resolveCity. All batch where possible.",
    `Tools:
- resolveOrganisations(names: string[]): Batch get-or-create for company/school names. Returns Map<string, Number> of rawName → internalId. Uses INSERT ... ON CONFLICT DO NOTHING.
- resolveCertifications(inputs: {providerName, certName}[]): Composite key batching. Requires organisations resolved first for provider_id.
- resolveSkills(names: string[]): Filter-only — returns Map of skills found in taxonomy. Skills not found are silently discarded.
- resolveCity(query): Calls placesSvc to check for existing city. Returns cityId or null.

Order matters: Organisations → Certificates (needs provider_id) → Skills/City (independent).`,
    undefined,
    ai.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Realtime Integration ──
  const realtime = await cm.addContextNode(
    "contextfs://cv-parser/realtime",
    "Realtime SSE Integration",
    "Live CV parsing updates via SSE through realtime-gateway. Pub/Sub command to realtime-gateway-cmd-send-event topic, routed to candidate:{candidateId} channel.",
    `Architecture:
- Frontend opens SSE connection to GET /realtime/events (cookie-authenticated).
- cv-parser owns parsing workflow and candidate_cv_parsing_tracking state.
- On status change, data publishes Pub/Sub command to realtime-gateway-cmd-send-event.
- realtime-gateway routes to channel candidate:{candidateId}, pushes SSE event.

Pub/Sub command contract (topic: realtime-gateway-cmd-send-event):
{
  "targetUserIds": [candidateId],
  "targetUserType": "candidate",
  "eventType": "CV_PARSE_FINISHED",
  "payload": { "status": "...", "selectedSections": [...], "errorMessage": null }
}

Emit on: job insert (PENDING), parser start (PROCESSING), completion (COMPLETED), failure (FAILED).
Frontend subscribes to CV_PARSE_COMPLETED, re-syncs on disconnect via currentCvParseJob query.`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Business Logic Flows ──
  const flows = await cm.addContextNode(
    "contextfs://cv-parser/business-flows",
    "Business Logic Flows",
    "Two main flows: full parse during application submission (new candidates) and selective overwrite for existing candidates with section selection modal.",
    undefined,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  await cm.addContextNode(
    "contextfs://cv-parser/business-flows/full-parse",
    "Full Parse Flow (New Candidates)",
    "Triggered during application submission when user has a new default CV. Creates candidate record, validates no active job exists, inserts PENDING job, triggers async PubSub extraction. User verifies email then sees extracted data on talent profile.",
    `Sequence:
1. Frontend finalizes application, candidate record created.
2. cv-parser validates no active job for candidate (PENDING/PROCESSING). If exists, returns existing taskId.
3. Inserts new job with status = PENDING (full parse).
4. Asynchronously triggers PubSub extraction call to cv-parser module.
5. User verifies email, then presented with extracted data on talent profile.`,
    undefined,
    flows.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  await cm.addContextNode(
    "contextfs://cv-parser/business-flows/overwrite",
    "Overwrite Flow (Existing Candidates)",
    "For existing candidates with profile data. Upload new CV, frontend shows overwrite selection modal (unchecked by default), user selects sections, calls startFullCvParsing with selectedSections. Only selected sections re-parsed and replaced.",
    `Sequence:
1. User uploads new CV (generateCVUploadInfo flow).
2. Frontend checks profile state; if non-empty, shows overwrite selection modal (not startCVParsing).
3. User selects checkboxes (unchecked by default), clicks "Confirm Overwrite".
4. Frontend calls startFullCvParsing(input: { fileId, selectedSections }).
5. Data inserts parsing job with selectedSections, triggers PubSub only for those sections.
6. After successful parse, replace-by-deletion only for listed sections.
7. Frontend notified via SSE of completion.`,
    undefined,
    flows.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Replace-by-Deletion ──
  await cm.addContextNode(
    "contextfs://cv-parser/replace-by-deletion",
    "Replace-by-Deletion Implementation",
    "Atomic transactional pattern: delete existing data for selected sections then insert new parsed data. Skills require special handling due to cross-entity assignments.",
    `Pseudo-code:
async function replaceSections(candidateId, sections, newData) {
  return db.transaction(async (trx) => {
    for (const section of sections) {
      switch (section) {
        case 'EXPERIENCE':
          await trx('candidate_professional_experiences').where({ candidate_id: candidateId }).delete();
          await trx('candidate_professional_experiences').insert(newData.experience.map(/* mapping */));
          break;
        case 'EDUCATION': // similar
        // ... etc.
      }
    }
    // Skills: delete all candidate_skills entries and reassign from parsed list
    // because skills may be attached to other entities
  });
}

Atomic: if parse fails → no deletion. If parse succeeds but insert fails → FAILED status.
Deletion + insertion happens after successful parse, before COMPLETED status.`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Entity Resolution ──
  const entityRes = await cm.addContextNode(
    "contextfs://cv-parser/entity-resolution",
    "Entity Resolution in Data Service",
    "Resolves LLM-extracted strings (organisations, certificates, skills) to canonical internal IDs. Get-or-create for orgs/certs, filter-only for skills. Dedicated gRPC endpoints wrapped as ADK tools.",
    `TalentProfileEntityResolutionService:
- resolveOrganisations(names): Get-or-create via VALUES clause + LEFT JOIN + INSERT ON CONFLICT DO NOTHING + final SELECT. Returns Map<string, Number>.
- resolveCertificates(inputs: {providerName, certName}[]): Composite key batching. Requires provider_id (orgs must be resolved first).
- resolveSkills(names): Filter only — returns Map of existing skills in taxonomy. Non-matching skills discarded.

Processing order: Organisations → Certificates → Skills (independent).
All entity types resolved in batch, once per type, iterating over experiences/education/certificates.`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Avro Commands ──
  await cm.addContextNode(
    "contextfs://cv-parser/messaging",
    "Messaging & Avro Commands",
    "Pub/Sub command records for triggering CV extraction. CvExtractFullProfileAttrs record on data-cmd-cv-extract-full-profile topic with taskId, fileId, fileName, candidateId, selectedSections.",
    `Avro command (commands.avdl in data, consumed by data itself):

@Event("data-cmd-cv-extract-full-profile")
record CvExtractFullProfileAttrs {
  string taskId;
  string fileId;
  string fileName;
  int candidateId;
  array<string> selectedSections;
}

Idempotency: SSE events are fire-and-forget. candidate_parsing_jobs table is source of truth.
If candidate refreshes, UI re-syncs by polling job status.`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Extraction Prompt ──
  await cm.addContextNode(
    "contextfs://cv-parser/extraction-prompt",
    "LLM Extraction Prompt",
    "Conditional prompt template for Gemini extraction. Returns JSON array of extraction objects with extractionClass, extractionText, and attributes. Supports 7 entity types: identity, experience, education, project, certification, skill, language.",
    `Entity types extracted:
- identity: firstName, lastName, phone, professionalLinks, city, city_id, country, confidence_score
- experience: jobTitle, companyName, company_id, startDate, endDate, employmentType_raw/id, city, city_id, country, description, skills, confidence_score
- education: schoolName, degree_raw/id, fieldOfStudy_raw/id, startYear, endYear, city, city_id, country, description, skills, confidence_score
- project: projectName, description, link, startDate, endDate, skills, confidence_score
- certification: credentialName, providerName, certification_entity_id, issueDate, expiryDate, score, credentialUrl, skills, confidence_score
- skill: skillName, proficiency (0-4: NO_EXPERIENCE to EXPERT), confidence_score
- language: language, isoCode, cefrLevel_raw, cefrLevel (A1=1..C2=6), confidence_score, language_id

Employment types: 1-14 (Dual studies, Seasonal, Apprenticeship, Interim, Worker, Contract, Side job, Working student, Traineeship, Volunteer, Internship, Employee, Freelance, Not specified)
Degree types: 1-5 (Bachelor's, Master's, Vocational, PhD, Secondary Education)

Tool calls must happen before final JSON output. Batch all tool calls where possible.`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Rate Limiting & Security ──
  await cm.addContextNode(
    "contextfs://cv-parser/security",
    "Rate Limiting & Security",
    "Rate limits: startCvParsing 2/60s per candidate. Upload: 10MB max, 15min signed URL. Security mitigations for IDOR, prompt injection, race conditions, and cost amplification.",
    `Rate limits:
- startCvParsing: 2 per 60 seconds per candidate ID
- Upload URL: max 10MB, signed for 15 minutes

Security risks & mitigations:
- IDOR: Ownership checks on every mutation/query, candidateId from session (never client-trusted), row-level guards.
- Prompt injection: CV treated as untrusted input, strict tool allowlist, output schema validation, server-side post-validation.
- Race conditions: Single active job per candidate (PENDING/PROCESSING lock), transactional replace-by-deletion, terminal-state validation.
- Cost amplification: Candidate-scoped rate limits, payload size/type validation, retry caps with backoff, anomaly alerting.`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Observability ──
  await cm.addContextNode(
    "contextfs://cv-parser/observability",
    "Observability & Telemetry",
    "Extraction telemetry sent to BigQuery on terminal states. Tracks task metadata, tokens, latency, and cost. Integration with Latitude.so or LangFuse under consideration.",
    `CvParseObservedAttrs (emitted on EXTRACTED/COMPLETED/FAILED/ABORTED):
- taskId, selectedSections, status, model (e.g. gemini-2.0-flash-lite)
- startedAtMs, endedAtMs, elapsedMs
- errorMessage, extractedData
- inputTokens, outputTokens, cachedTokens
- estimatedCostEur

Benchmarks (86 CVs): 92.4% avg score, ~15s latency, 84.5% cache rate, $0.00223/CV.`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Testing & Evaluation ──
  await cm.addContextNode(
    "contextfs://cv-parser/evaluation",
    "LLM Quality Testing & Evaluation",
    "Golden dataset evaluation framework with 80+ anonymized CVs. Measures schema match rate (100% goal), field-level accuracy, and hallucination rate with weighted scoring.",
    `Golden Set: 80+ anonymized CVs with ground_truth.json per CV.

Metrics:
- Schema Match Rate: % of outputs passing JSON schema validation (goal: 100%)
- Field-Level Accuracy: exact match (categorical), F1 (lists), semantic similarity (text)
- Hallucination Rate: frequency of invented data not in source

Scoring (weighted Quality Score Q):
- 1.0: Exact match with ground truth
- 0.10-0.84: Partial credit via normalized Levenshtein distance
- 0.0: Syntax error, execution failure, or empty results

Section weights:
- Identity: 20% (name 35%, phone 20%, links 20%, city 15%, country 10%)
- Experience Detail: 15%, Skills: 12% (Jaccard IoU), Education Detail: 12%
- Experience Count: 10%, Languages: 10%, Education Count: 8%
- Certificates: 7%, Projects: 6%`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  // ── Error Codes ──
  await cm.addContextNode(
    "contextfs://cv-parser/error-codes",
    "Exception & Warning Types",
    "Standardized error codes for CV parsing: FILE_NOT_FOUND, INVALID_SECTION_LIST, INVALID_DATE_FORMAT, RATE_LIMIT_EXCEEDED, LLM_EXTRACTION_FAILED, SCHEMA_MISMATCH, TRANSACTION_ABORTED.",
    `Error codes:
- FILE_NOT_FOUND (Validation): fileId doesn't exist in GCS or isn't associated with candidate.
- INVALID_SECTION_LIST (Validation): selectedSections provided but empty or invalid enum values.
- INVALID_DATE_FORMAT (Validation): Dates fail YYYY-MM-DD regex check.
- RATE_LIMIT_EXCEEDED (Security): Exceeded 2 req/60s limit.
- LLM_EXTRACTION_FAILED (AI Engine): Gemini failed to return valid response or timed out.
- SCHEMA_MISMATCH (AI Engine): LLM output failed JSON/Avro schema validation.
- TRANSACTION_ABORTED (Persistence): Replace-by-deletion atomic transaction failed (e.g. DB deadlock).`,
    undefined,
    root.uri,
    P,
    {},
    false
  ) as AgentContextNode;

  console.log("\nSeeding complete! CV Parser execution plan persisted to Elasticsearch.");
}

seed().catch(console.error);

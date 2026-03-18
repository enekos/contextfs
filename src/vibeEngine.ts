/**
 * Vibe Engine — LLM-powered free-text query and mutation interface.
 *
 * vibe-query:    prompt → LLM generates searches → runs them → presents results
 * vibe-mutation: prompt → LLM searches context → plans mutations → diff → user approves → execute
 */
import { GoogleGenAI } from "@google/genai";
import { extractJsonObject } from "./jsonUtils";
import { config } from "./config";
import { ContextManager } from "./ContextManager";
import { MemoryCategory, MemoryOwner } from "./types";

const LLM_MODEL = config.llmModel;

const ai = config.geminiApiKey
  ? new GoogleGenAI({ apiKey: config.geminiApiKey })
  : null;

const MAX_RETRIES = 3;
const RETRY_DELAY_MS = 1000;

async function generateWithRetry(model: string, contents: string, attempt = 1): Promise<any> {
  if (!ai) throw new Error("GoogleGenAI not initialized");
  try {
    return await ai.models.generateContent({ model, contents });
  } catch (error: unknown) {
    if (attempt < MAX_RETRIES && ((error as { status?: number })?.status === 429 || ((error as { status?: number })?.status ?? 0) >= 500 || (error as { message?: string })?.message?.includes("fetch failed"))) {
      const delay = RETRY_DELAY_MS * Math.pow(2, attempt - 1);
      console.warn(`[vibeEngine] API error (${error instanceof Error ? error.message : String(error)}), retrying in ${delay}ms (attempt ${attempt + 1}/${MAX_RETRIES})`);
      await new Promise((resolve) => setTimeout(resolve, delay));
      return generateWithRetry(model, contents, attempt + 1);
    }
    throw error;
  }
}

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export interface VibeSearchPlan {
  queries: Array<{
    store: "memory" | "skill" | "node";
    query: string;
    filters?: Record<string, string>;
  }>;
  reasoning: string;
}

export interface VibeMutationOp {
  op: "create_memory" | "update_memory" | "delete_memory"
    | "create_skill" | "update_skill" | "delete_skill"
    | "create_node" | "update_node" | "delete_node";
  /** For updates/deletes: the target ID or URI */
  target?: string;
  /** Human-readable description of this change */
  description: string;
  /** The data for creates/updates */
  data: Record<string, any>;
}

export interface VibeMutationPlan {
  reasoning: string;
  operations: VibeMutationOp[];
}

export interface VibeQueryResult {
  reasoning: string;
  results: Array<{
    store: string;
    query: string;
    items: Record<string, any>[];
  }>;
}

// ─────────────────────────────────────────────────────────────────────────────
// Vibe Query
// ─────────────────────────────────────────────────────────────────────────────

export async function planVibeSearch(prompt: string, project?: string): Promise<VibeSearchPlan> {
  if (!ai) throw new Error("GEMINI_API_KEY is not set");

  const systemPrompt = `You are a search planner for a context/memory database with three stores:
- "memory": agent memories (facts, observations, decisions). Fields: content, category, owner, importance.
- "skill": capability descriptions. Fields: name, description.
- "node": hierarchical context nodes (documentation, architecture). Fields: uri, name, abstract, overview, content.

Given a user's free-text prompt, generate search queries to find relevant information.
${project ? `Project namespace: "${project}"` : "No project filter."}

Respond with ONLY a JSON object:
{
  "reasoning": "brief explanation of your search strategy",
  "queries": [
    { "store": "memory"|"skill"|"node", "query": "semantic search text" }
  ]
}

Generate 1-4 queries. Use different angles/phrasings to maximize recall.`;

  const response = await generateWithRetry(LLM_MODEL, `${systemPrompt}\n\nUSER PROMPT: ${prompt}`);
  const parsed = extractJsonObject(response.text?.trim() || "");

  if (!parsed || !Array.isArray(parsed.queries)) {
    return { reasoning: "Falling back to direct search", queries: [{ store: "memory", query: prompt }, { store: "node", query: prompt }] };
  }

  return {
    reasoning: parsed.reasoning || "",
    queries: parsed.queries.filter((q: any) =>
      typeof q === "object" && typeof q.store === "string" && typeof q.query === "string" &&
      ["memory", "skill", "node"].includes(q.store)
    ),
  };
}

export async function executeVibeQuery(
  cm: ContextManager,
  prompt: string,
  project?: string,
  topK = 5
): Promise<VibeQueryResult> {
  const plan = await planVibeSearch(prompt, project);
  const results: VibeQueryResult["results"] = [];

  await Promise.all(
    plan.queries.map(async (q) => {
      const opts = { topK, project };
      let items: Record<string, any>[];
      switch (q.store) {
        case "memory":
          items = await cm.searchMemories(q.query, opts);
          break;
        case "skill":
          items = await cm.searchSkills(q.query, opts);
          break;
        case "node":
          items = await cm.searchContext(q.query, opts);
          break;
        default:
          items = [];
      }
      results.push({ store: q.store, query: q.query, items });
    })
  );

  return { reasoning: plan.reasoning, results };
}

// ─────────────────────────────────────────────────────────────────────────────
// Vibe Mutation
// ─────────────────────────────────────────────────────────────────────────────

export async function planVibeMutation(
  cm: ContextManager,
  prompt: string,
  project?: string,
  topK = 10
): Promise<VibeMutationPlan> {
  if (!ai) throw new Error("GEMINI_API_KEY is not set");

  // Step 1: search existing data for context
  const queryResult = await executeVibeQuery(cm, prompt, project, topK);

  // Flatten all results for the LLM to see
  const existingContext: Array<Record<string, any>> = queryResult.results
    .flatMap((r) =>
      r.items.map((item) => ({
        store: r.store,
        ...item,
      }))
    );

  // Deduplicate by id/uri
  const seen = new Set<string>();
  const deduped = existingContext.filter((item) => {
    const key = (item.id || item.uri) as string | undefined;
    if (!key || seen.has(key)) return false;
    seen.add(key);
    return true;
  });

  const contextStr = deduped.length > 0
    ? JSON.stringify(deduped.map((item) => {
        // Trim verbose fields for the LLM prompt
        const { _vector_score, _keyword_score, _recency_score, _importance_score, _hybrid_score, embedding: _embedding, distance: _distance, ...rest } = item;
        return rest;
      }), null, 2)
    : "(no existing entries found)";

  const systemPrompt = `You are a mutation planner for a context/memory database. Based on the user's intent, plan what entries to create, update, or delete.

DATABASE STORES:
- memory: { id, content, category (one of: profile, preferences, entities, events, cases, patterns, observation, reflection, decision, constraint, architecture), owner (user|agent|system), importance (1-10), project }
- skill: { id, name, description, project }
- node: { uri, name, abstract, overview?, content?, parent_uri?, project }

EXISTING ENTRIES (from semantic search):
${contextStr}

RULES:
- For "create" ops: provide all required fields in "data"
- For "update" ops: set "target" to the existing ID/URI, and "data" to ONLY the changed fields
- For "delete" ops: set "target" to the ID/URI, "data" can be empty
- Each operation must have a clear "description" explaining the change
- For memory categories, use one of: profile, preferences, entities, events, cases, patterns, observation, reflection, decision, constraint, architecture
- For memory owner, use one of: user, agent, system
${project ? `- Use project: "${project}" for new entries` : ""}
- Only plan mutations that directly address the user's prompt
- If an existing entry already covers the intent, prefer "update" over "create"

Respond with ONLY a JSON object:
{
  "reasoning": "brief explanation of your mutation plan",
  "operations": [
    {
      "op": "create_memory"|"update_memory"|"delete_memory"|"create_skill"|"update_skill"|"delete_skill"|"create_node"|"update_node"|"delete_node",
      "target": "id or uri (for update/delete)",
      "description": "human-readable description of this change",
      "data": { ... }
    }
  ]
}`;

  const response = await generateWithRetry(LLM_MODEL, `${systemPrompt}\n\nUSER PROMPT: ${prompt}`);
  const parsed = extractJsonObject(response.text?.trim() || "");

  if (!parsed || !Array.isArray(parsed.operations)) {
    throw new Error(`LLM returned unparseable mutation plan:\n${(response.text || "").slice(0, 500)}`);
  }

  const validOps = [
    "create_memory", "update_memory", "delete_memory",
    "create_skill", "update_skill", "delete_skill",
    "create_node", "update_node", "delete_node",
  ];

  return {
    reasoning: parsed.reasoning || "",
    operations: parsed.operations.filter((op: any) =>
      typeof op === "object" &&
      typeof op.op === "string" &&
      validOps.includes(op.op) &&
      typeof op.description === "string"
    ).map((op: any) => ({
      op: op.op,
      target: op.target,
      description: op.description,
      data: op.data || {},
    })),
  };
}

export async function executeMutationOp(
  cm: ContextManager,
  op: VibeMutationOp,
  project?: string
): Promise<string> {
  const d = op.data;
  switch (op.op) {
    case "create_memory": {
      const result = await cm.addMemory(
        d.content,
        (d.category || "observation") as MemoryCategory,
        (d.owner || "agent") as MemoryOwner,
        d.importance ?? 5,
        d.project || project,
        d.metadata || {},
        false
      );
      if ("id" in result) return `Created memory: ${result.id}`;
      return `Memory write result: ${JSON.stringify(result)}`;
    }
    case "update_memory": {
      await cm.updateMemory(op.target!, d);
      return `Updated memory: ${op.target}`;
    }
    case "delete_memory": {
      await cm.deleteMemory(op.target!);
      return `Deleted memory: ${op.target}`;
    }
    case "create_skill": {
      const result = await cm.addSkill(d.name, d.description, d.project || project, d.metadata || {});
      return `Created skill: ${result.id}`;
    }
    case "update_skill": {
      await cm.updateSkill(op.target!, d);
      return `Updated skill: ${op.target}`;
    }
    case "delete_skill": {
      await cm.deleteSkill(op.target!);
      return `Deleted skill: ${op.target}`;
    }
    case "create_node": {
      const result = await cm.addContextNode(
        d.uri, d.name, d.abstract, d.overview, d.content,
        d.parent_uri || null, d.project || project, d.metadata || {}, false
      );
      if ("uri" in result) return `Created node: ${result.uri}`;
      return `Node write result: ${JSON.stringify(result)}`;
    }
    case "update_node": {
      await cm.updateContextNode(op.target!, d);
      return `Updated node: ${op.target}`;
    }
    case "delete_node": {
      await cm.deleteContextNode(op.target!);
      return `Deleted node: ${op.target}`;
    }
    default:
      return `Unknown operation: ${op.op}`;
  }
}

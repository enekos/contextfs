/**
 * Dreaming: background memory consolidation.
 *
 * Three passes run sequentially:
 * 1. Deduction — merge duplicates, resolve contradictions
 * 2. Induction — extract patterns from memory clusters
 * 3. Context node consolidation — refresh abstracts, merge siblings, regroup orphans
 */
import { MeilisearchDB } from "./storage/meilisearchDB";
import { Embedder } from "./storage/embedder";
import { config } from "./core/config";
import { extractJsonObject } from "./core/jsonUtils";
import { llmGenerate } from "./llm/llmUtils";
import type { AgentMemory, AgentContextNode } from "./core/types";

const DEDUP_SIMILARITY_THRESHOLD = 0.85;
const PATTERN_SIMILARITY_THRESHOLD = 0.75;
const SIBLING_SIMILARITY_THRESHOLD = 0.9;
const ORPHAN_SIMILARITY_THRESHOLD = 0.8;
const LIST_PAGE_SIZE = 100;

function getDB(): MeilisearchDB {
  return new MeilisearchDB(config.meili.url, config.meili.apiKey || undefined);
}


async function fetchAllMemories(db: MeilisearchDB, project: string): Promise<AgentMemory[]> {
  const all: AgentMemory[] = [];
  let offset = 0;
  while (true) {
    const page = await db.listMemories({ project }, LIST_PAGE_SIZE, offset);
    if (page.length === 0) break;
    all.push(...page);
    offset += page.length;
  }
  return all;
}

interface DeductionDecision {
  candidateIndex: number;
  action: "MERGE" | "CONTRADICTION" | "KEEP_BOTH";
  mergedContent?: string;
}

export async function deductionPass(project: string): Promise<void> {
  const db = getDB();
  const memories = await fetchAllMemories(db, project);
  const processed = new Set<string>();

  for (const memory of memories) {
    if (processed.has(memory.id)) continue;
    processed.add(memory.id);

    const embedding = await Embedder.getEmbedding(memory.content);
    const candidates = await db.searchMemoriesByVector(embedding, { topK: 10, project });

    // Filter: similar enough, not self, not already processed
    const similar = candidates.filter(
      (c) => c.id !== memory.id && !processed.has(c.id) && c._score >= DEDUP_SIMILARITY_THRESHOLD
    );

    if (similar.length === 0) continue;

    const candidateList = similar
      .map((c, i) => `[${i}] ID: ${c.id} | Created: ${c.created_at} | Content: ${c.content}`)
      .join("\n");

    const prompt = `You are a memory consolidation agent. Given a source memory and candidates, decide for each candidate:
- MERGE: They say essentially the same thing. Provide mergedContent combining both.
- CONTRADICTION: They conflict. The newer one is current truth; the older should be removed.
- KEEP_BOTH: They are related but distinct facts.

Source memory (ID: ${memory.id}, Created: ${memory.created_at}):
${memory.content}

Candidates:
${candidateList}

Respond with ONLY a JSON object:
{"decisions":[{"candidateIndex":0,"action":"MERGE"|"CONTRADICTION"|"KEEP_BOTH","mergedContent":"...if MERGE"}]}`;

    try {
      const text = await llmGenerate(prompt);
      const parsed = extractJsonObject(text) as { decisions?: DeductionDecision[] } | null;
      if (!parsed?.decisions) continue;

      for (const decision of parsed.decisions) {
        const candidate = similar[decision.candidateIndex];
        if (!candidate) continue;

        const sourceDate = new Date(memory.created_at).getTime();
        const candidateDate = new Date(candidate.created_at).getTime();
        const [older, newer] = sourceDate <= candidateDate
          ? [memory, candidate]
          : [candidate, memory];

        if (decision.action === "MERGE" && decision.mergedContent) {
          const mergedEmbedding = await Embedder.getEmbedding(decision.mergedContent);
          await db.updateMemory(newer.id, { content: decision.mergedContent }, mergedEmbedding);
          await db.deleteMemory(older.id);
        } else if (decision.action === "CONTRADICTION") {
          await db.deleteMemory(older.id);
        }
        // Always mark candidate as processed regardless of action
        processed.add(candidate.id);
      }
    } catch (e) {
      console.warn(`[dreamer] Deduction failed for memory ${memory.id}:`, e);
    }
  }
}

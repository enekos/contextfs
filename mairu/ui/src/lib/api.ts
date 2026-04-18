async function fetchJSON(url: string, init?: RequestInit): Promise<any> {
  const resp = await fetch(url, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  });
  if (!resp.ok) throw new Error(`API error: ${resp.status}`);
  return resp.json();
}

// ── Memories ────────────────────────────────────────────────────

export async function listMemories(project: string, limit = 100) {
  return fetchJSON(`/api/memories?project=${encodeURIComponent(project)}&limit=${limit}`);
}

export async function createMemory(input: any) {
  return fetchJSON('/api/memories', { method: 'POST', body: JSON.stringify(input) });
}

export async function updateMemory(input: any) {
  return fetchJSON('/api/memories', { method: 'PUT', body: JSON.stringify(input) });
}

export async function deleteMemory(id: string) {
  return fetchJSON(`/api/memories?id=${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function applyMemoryFeedback(id: string, reward: number) {
  return fetchJSON('/api/memories/feedback', { method: 'POST', body: JSON.stringify({ id, reward }) });
}

// ── Skills ──────────────────────────────────────────────────────

export async function listSkills(project: string, limit = 100) {
  return fetchJSON(`/api/skills?project=${encodeURIComponent(project)}&limit=${limit}`);
}

export async function createSkill(input: any) {
  return fetchJSON('/api/skills', { method: 'POST', body: JSON.stringify(input) });
}

export async function updateSkill(input: any) {
  return fetchJSON('/api/skills', { method: 'PUT', body: JSON.stringify(input) });
}

export async function deleteSkill(id: string) {
  return fetchJSON(`/api/skills?id=${encodeURIComponent(id)}`, { method: 'DELETE' });
}

// ── Context Nodes ───────────────────────────────────────────────

export async function listContextNodes(project: string, parentURI?: string, limit = 100) {
  let url = `/api/context?project=${encodeURIComponent(project)}&limit=${limit}`;
  if (parentURI) url += `&parentUri=${encodeURIComponent(parentURI)}`;
  return fetchJSON(url);
}

export async function createContextNode(input: any) {
  return fetchJSON('/api/context', { method: 'POST', body: JSON.stringify(input) });
}

export async function updateContextNode(input: any) {
  return fetchJSON('/api/context', { method: 'PUT', body: JSON.stringify(input) });
}

export async function deleteContextNode(uri: string) {
  return fetchJSON(`/api/context?uri=${encodeURIComponent(uri)}`, { method: 'DELETE' });
}

// ── Search ──────────────────────────────────────────────────────

export async function search(opts: any) {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(opts)) {
    if (v !== undefined && v !== null && v !== '') params.set(k, String(v));
  }
  return fetchJSON(`/api/search?${params}`);
}

// ── Dashboard ───────────────────────────────────────────────────

export async function dashboard(limit = 1000, project = '') {
  return fetchJSON(`/api/dashboard?limit=${limit}&project=${encodeURIComponent(project)}`);
}

export async function health() {
  return fetchJSON('/api/health');
}

export async function clusterStats() {
  return fetchJSON('/api/cluster');
}

// ── Vibe ────────────────────────────────────────────────────────

export async function vibeMutationPlan(prompt: string, project: string, topK: number) {
  return fetchJSON('/api/vibe/mutation/plan', { method: 'POST', body: JSON.stringify({ prompt, project, topK }) });
}

export async function vibeMutationExecute(operations: any[], project: string) {
  return fetchJSON('/api/vibe/mutation/execute', { method: 'POST', body: JSON.stringify({ operations, project }) });
}

// ── Moderation ──────────────────────────────────────────────────

export async function listModerationQueue(limit = 100) {
  return fetchJSON(`/api/moderation/queue?limit=${limit}`);
}

export async function reviewModeration(input: any) {
  return fetchJSON('/api/moderation/review', { method: 'POST', body: JSON.stringify(input) });
}

// ── Sessions (chat) ─────────────────────────────────────────────

export async function listSessions() {
  return fetchJSON('/api/sessions').then((r: any) => r.sessions);
}

export async function createSession(name: string) {
  return fetchJSON('/api/sessions', { method: 'POST', body: JSON.stringify({ name }) });
}

export async function loadSessionHistory(session: string) {
  return fetchJSON(`/api/sessions/${encodeURIComponent(session)}/messages`).then((r: any) => r.messages);
}

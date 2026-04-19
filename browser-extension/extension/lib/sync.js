export async function syncOnce(queue, apiUrl, fetchImpl = fetch) {
  const now = Date.now();
  const due = await queue.takeDue(now);
  const BATCH = 5;
  const batch = due.slice(0, BATCH);
  const results = { ok: 0, fail: 0 };
  for (const entry of batch) {
    try {
      const { id: _id, ...body } = entry.payload;
      void _id;
      const r = await fetchImpl(`${apiUrl}/api/context`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (r && r.ok) {
        await queue.ack(entry.payload.id);
        results.ok++;
      } else {
        await queue.markFailed(entry.payload.id, Date.now());
        results.fail++;
      }
    } catch (err) {
      void err;
      await queue.markFailed(entry.payload.id, Date.now());
      results.fail++;
    }
  }
  return results;
}

export function buildPayload(page) {
  const extras = [];
  if (page.selection) extras.push(`\n\n### Current Selection\n${page.selection}`);
  if (page.active_element) extras.push(`\n\n### Active Element (Focus)\n${page.active_element}`);
  if (page.console_errors?.length) {
    extras.push(`\n\n### Console Errors\n${page.console_errors.join('\n')}`);
  }
  if (page.network_errors?.length) {
    extras.push(`\n\n### Network Errors\n${page.network_errors.join('\n')}`);
  }
  if (page.storage_state && Object.keys(page.storage_state).length) {
    extras.push(
      `\n\n### Storage State\n` +
        Object.entries(page.storage_state).map(([k, v]) => `- **${k}**: ${v}`).join('\n'),
    );
  }
  if (page.visual_rects && Object.keys(page.visual_rects).length) {
    extras.push(
      `\n\n### Visual Layout (Bounding Rects)\n` +
        Object.entries(page.visual_rects).map(([k, v]) => `- \`${k}\`: ${v}`).join('\n'),
    );
  }
  const id = stableId(page);
  return {
    id,
    uri: `contextfs://browser/${encodeB64(page.url)}`,
    project: 'browser',
    name: page.title,
    abstract: (page.sections?.[0]?.text || '').slice(0, 200),
    overview: (page.sections || [])
      .filter((s) => s.kind === 'heading' || s.kind === 'Heading')
      .map((s) => s.text)
      .join('\n'),
    content: (page.sections || []).map((s) => s.text).join('\n\n') + extras.join(''),
  };
}

function stableId(page) {
  const bucket = Math.floor((page.timestamp || Date.now()) / 60_000);
  const s = `${page.url}|${page.content_hash}|${bucket}`;
  let h = 2166136261 >>> 0;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return `p_${(h >>> 0).toString(16)}`;
}

function encodeB64(s) {
  try {
    if (typeof btoa === 'function') return btoa(unescape(encodeURIComponent(s)));
  } catch (err) {
    void err;
  }
  // Node fallback (tests)
  if (typeof Buffer !== 'undefined') return Buffer.from(s, 'utf8').toString('base64');
  return s;
}

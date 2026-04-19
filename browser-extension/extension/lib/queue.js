const BACKOFFS_MS = [1_000, 5_000, 30_000, 120_000, 600_000, 900_000];

export function createQueue({ storageKey, maxEntries = 500 }) {
  let entries = [];
  let dropped = 0;
  let loaded = false;

  async function persist() {
    try {
      await chrome.storage.local.set({ [storageKey]: { entries, dropped } });
    } catch (err) {
      void err;
    }
  }

  async function load() {
    try {
      const s = await chrome.storage.local.get(storageKey);
      const v = s[storageKey];
      if (v && Array.isArray(v.entries)) {
        entries = v.entries;
        dropped = v.dropped || 0;
      }
    } catch (err) {
      void err;
    }
    loaded = true;
  }

  async function enqueue(payload) {
    if (!loaded) await load();
    if (!payload || typeof payload.id !== 'string') {
      throw new Error('queue.enqueue: payload.id required');
    }
    const existing = entries.findIndex((e) => e.payload.id === payload.id);
    const now = Date.now();
    if (existing >= 0) {
      entries[existing] = { ...entries[existing], payload };
    } else {
      entries.push({ payload, attempts: 0, nextAttemptAt: now, createdAt: now });
    }
    while (entries.length > maxEntries) {
      entries.shift();
      dropped++;
    }
    await persist();
  }

  async function takeDue(now) {
    if (!loaded) await load();
    return entries.filter((e) => e.nextAttemptAt <= now);
  }

  async function markFailed(id, now) {
    const e = entries.find((x) => x.payload.id === id);
    if (!e) return;
    e.attempts += 1;
    const idx = Math.min(e.attempts - 1, BACKOFFS_MS.length - 1);
    e.nextAttemptAt = now + BACKOFFS_MS[idx];
    await persist();
  }

  async function ack(id) {
    entries = entries.filter((e) => e.payload.id !== id);
    await persist();
  }

  async function clear() {
    entries = [];
    dropped = 0;
    await persist();
  }

  return {
    load,
    enqueue,
    takeDue,
    markFailed,
    ack,
    clear,
    size: () => entries.length,
    droppedCount: () => dropped,
    all: () => entries.slice(),
  };
}

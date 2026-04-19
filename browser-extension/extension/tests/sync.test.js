import { describe, it, expect, vi } from 'vitest';
import { syncOnce, buildPayload } from '../lib/sync.js';
import { createQueue } from '../lib/queue.js';

describe('syncOnce', () => {
  it('acks on 200', async () => {
    const q = createQueue({ storageKey: 'sync-ok', maxEntries: 10 });
    await q.load();
    await q.enqueue({ id: 'p1', uri: 'u', project: 'p', name: 'n' });
    const fetchImpl = vi.fn(async () => ({ ok: true }));
    const r = await syncOnce(q, 'http://x', fetchImpl);
    expect(q.size()).toBe(0);
    expect(r).toEqual({ ok: 1, fail: 0 });
  });

  it('retries on 500', async () => {
    const q = createQueue({ storageKey: 'sync-500', maxEntries: 10 });
    await q.load();
    await q.enqueue({ id: 'p2' });
    const fetchImpl = vi.fn(async () => ({ ok: false, status: 500 }));
    const r = await syncOnce(q, 'http://x', fetchImpl);
    expect(q.size()).toBe(1);
    expect(q.all()[0].attempts).toBe(1);
    expect(r).toEqual({ ok: 0, fail: 1 });
  });

  it('network error increments attempts', async () => {
    const q = createQueue({ storageKey: 'sync-net', maxEntries: 10 });
    await q.load();
    await q.enqueue({ id: 'p3' });
    const fetchImpl = vi.fn(async () => {
      throw new Error('off');
    });
    await syncOnce(q, 'http://x', fetchImpl);
    expect(q.all()[0].attempts).toBe(1);
  });

  it('respects batch size of 5', async () => {
    const q = createQueue({ storageKey: 'sync-batch', maxEntries: 20 });
    await q.load();
    for (let i = 0; i < 7; i++) await q.enqueue({ id: `p${i}` });
    const fetchImpl = vi.fn(async () => ({ ok: true }));
    await syncOnce(q, 'http://x', fetchImpl);
    expect(fetchImpl).toHaveBeenCalledTimes(5);
    expect(q.size()).toBe(2);
  });
});

describe('buildPayload', () => {
  it('produces a stable id per (url, hash, 1min bucket)', () => {
    const p = { url: 'https://a.test', content_hash: 12345, timestamp: 1_700_000_000_000, sections: [{ text: 'hi', kind: 'body' }] };
    const a = buildPayload(p);
    const b = buildPayload(p);
    expect(a.id).toBe(b.id);
  });

  it('includes extras when present', () => {
    const p = {
      url: 'https://a.test',
      content_hash: 1,
      timestamp: 1,
      sections: [{ text: 'main', kind: 'body' }],
      selection: 'selected!',
      console_errors: ['err1'],
      storage_state: { 'localStorage[k]': 'v' },
    };
    const out = buildPayload(p);
    expect(out.content).toContain('Current Selection');
    expect(out.content).toContain('Console Errors');
    expect(out.content).toContain('Storage State');
  });
});

import { describe, it, expect } from 'vitest';
import { createQueue } from '../lib/queue.js';

describe('queue', () => {
  it('enqueues and returns due items', async () => {
    const q = createQueue({ storageKey: 'k', maxEntries: 10 });
    await q.load();
    await q.enqueue({ id: 'a', url: 'u1' });
    const due = await q.takeDue(Date.now());
    expect(due).toHaveLength(1);
    expect(due[0].payload.id).toBe('a');
  });

  it('dedupes by id', async () => {
    const q = createQueue({ storageKey: 'k2', maxEntries: 10 });
    await q.load();
    await q.enqueue({ id: 'same', x: 1 });
    await q.enqueue({ id: 'same', x: 2 });
    expect(q.size()).toBe(1);
    expect(q.all()[0].payload.x).toBe(2);
  });

  it('schedules backoff after failure', async () => {
    const q = createQueue({ storageKey: 'k3', maxEntries: 10 });
    await q.load();
    await q.enqueue({ id: 'x' });
    await q.markFailed('x', Date.now());
    const due = await q.takeDue(Date.now());
    expect(due).toHaveLength(0);
    const later = await q.takeDue(Date.now() + 120_000);
    expect(later).toHaveLength(1);
    expect(later[0].attempts).toBe(1);
  });

  it('drops oldest when full', async () => {
    const q = createQueue({ storageKey: 'k4', maxEntries: 3 });
    await q.load();
    for (const id of ['a', 'b', 'c', 'd']) await q.enqueue({ id });
    expect(q.size()).toBe(3);
    const ids = (await q.takeDue(Date.now())).map((e) => e.payload.id);
    expect(ids).toEqual(['b', 'c', 'd']);
    expect(q.droppedCount()).toBe(1);
  });

  it('persists to chrome.storage.local', async () => {
    const q = createQueue({ storageKey: 'persist', maxEntries: 10 });
    await q.load();
    await q.enqueue({ id: 'p1' });
    const stored = await chrome.storage.local.get('persist');
    expect(stored.persist.entries.length).toBe(1);
  });

  it('reloads persisted entries', async () => {
    const q1 = createQueue({ storageKey: 'reload', maxEntries: 10 });
    await q1.load();
    await q1.enqueue({ id: 'p1' });
    const q2 = createQueue({ storageKey: 'reload', maxEntries: 10 });
    await q2.load();
    expect(q2.size()).toBe(1);
  });

  it('removes on ack', async () => {
    const q = createQueue({ storageKey: 'ack', maxEntries: 10 });
    await q.load();
    await q.enqueue({ id: 'x' });
    await q.ack('x');
    expect(q.size()).toBe(0);
  });

  it('clear empties queue and dropped counter', async () => {
    const q = createQueue({ storageKey: 'clr', maxEntries: 5 });
    await q.load();
    await q.enqueue({ id: '1' });
    await q.clear();
    expect(q.size()).toBe(0);
    expect(q.droppedCount()).toBe(0);
  });

  it('throws on payload without id', async () => {
    const q = createQueue({ storageKey: 'throw', maxEntries: 10 });
    await q.load();
    await expect(q.enqueue({ url: 'x' })).rejects.toThrow();
  });
});

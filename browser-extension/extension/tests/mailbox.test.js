import { describe, it, expect, vi } from 'vitest';
import { createMailbox } from '../lib/mailbox.js';

describe('mailbox', () => {
  it('caps at N, drops oldest', () => {
    const logger = { warn: vi.fn() };
    const m = createMailbox({ cap: 2, logger });
    m.push({ message: { type: 'a' } });
    m.push({ message: { type: 'b' } });
    m.push({ message: { type: 'c' } });
    expect(m.size()).toBe(2);
    expect(logger.warn).toHaveBeenCalledWith('mailbox.drop', { type: 'a' });
  });

  it('flush delivers in order and empties', () => {
    const m = createMailbox({ cap: 5 });
    m.push({ x: 1 });
    m.push({ x: 2 });
    const seen = [];
    m.flush((e) => seen.push(e.x));
    expect(seen).toEqual([1, 2]);
    expect(m.size()).toBe(0);
  });

  it('handler errors do not break flush', () => {
    const m = createMailbox({ cap: 5 });
    m.push({ x: 1 });
    m.push({ x: 2 });
    const seen = [];
    m.flush((e) => {
      if (e.x === 1) throw new Error('fail');
      seen.push(e.x);
    });
    expect(seen).toEqual([2]);
  });
});

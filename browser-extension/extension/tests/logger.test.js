import { describe, it, expect, vi } from 'vitest';
import { createLogger, LEVELS } from '../lib/logger.js';

describe('logger', () => {
  it('logs events with fields to ring buffer', () => {
    const log = createLogger({ capacity: 3 });
    log.info('sync.ok', { pages: 2 });
    const buf = log.snapshot();
    expect(buf.length).toBe(1);
    expect(buf[0].event).toBe('sync.ok');
    expect(buf[0].fields.pages).toBe(2);
    expect(buf[0].level).toBe('info');
  });

  it('overwrites oldest when over capacity', () => {
    const log = createLogger({ capacity: 2 });
    log.info('a');
    log.info('b');
    log.info('c');
    const buf = log.snapshot();
    expect(buf.map((e) => e.event)).toEqual(['b', 'c']);
  });

  it('notifies subscribers', () => {
    const log = createLogger({ capacity: 5 });
    const fn = vi.fn();
    log.subscribe(fn);
    log.warn('hello');
    expect(fn).toHaveBeenCalledOnce();
    expect(fn.mock.calls[0][0].event).toBe('hello');
  });

  it('subscribe returns unsubscribe function', () => {
    const log = createLogger({ capacity: 5 });
    const fn = vi.fn();
    const unsub = log.subscribe(fn);
    log.info('a');
    unsub();
    log.info('b');
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('exposes LEVELS', () => {
    expect(LEVELS).toEqual(['debug', 'info', 'warn', 'error']);
  });

  it('never throws on a subscriber error', () => {
    const log = createLogger({ capacity: 5 });
    log.subscribe(() => { throw new Error('nope'); });
    expect(() => log.error('boom')).not.toThrow();
  });
});

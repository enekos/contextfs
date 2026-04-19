import { describe, it, expect } from 'vitest';
import { validate, MESSAGE_TYPES } from '../lib/messages.js';

describe('messages.validate', () => {
  it('accepts a well-formed page_content', () => {
    const r = validate({
      type: 'page_content',
      payload: {
        url: 'https://a.test/',
        html: '<p>',
        timestamp: 1,
        selection: null,
        active_element: null,
        console_errors: [],
        network_errors: [],
        visual_rects: {},
        storage_state: {},
        dwell_ms: 0,
        interaction_count: 0,
        iframes: [],
      },
    });
    expect(r.ok).toBe(true);
  });

  it('rejects page_content with missing url', () => {
    const r = validate({ type: 'page_content', payload: { html: '', timestamp: 1 } });
    expect(r.ok).toBe(false);
    expect(r.error.field).toBe('payload.url');
  });

  it('rejects unknown message type', () => {
    const r = validate({ type: 'nope' });
    expect(r.ok).toBe(false);
    expect(r.error.code).toBe('unknown_type');
  });

  it('enumerates execute commands', () => {
    expect(MESSAGE_TYPES.execute.commands).toContain('click');
    expect(MESSAGE_TYPES.execute.commands).toContain('fill');
    expect(MESSAGE_TYPES.execute.commands).toContain('show_thought');
  });

  it('rejects execute with missing selector for click', () => {
    const r = validate({ type: 'execute', command: 'click' });
    expect(r.ok).toBe(false);
    expect(r.error.field).toBe('selector');
  });

  it('accepts execute with all required fields', () => {
    expect(validate({ type: 'execute', command: 'click', selector: '#x' }).ok).toBe(true);
    expect(validate({ type: 'execute', command: 'scroll' }).ok).toBe(true);
    expect(validate({ type: 'execute', command: 'fill', selector: '#x', value: 'y' }).ok).toBe(true);
  });

  it('rejects non-object messages', () => {
    expect(validate(null).ok).toBe(false);
    expect(validate('string').ok).toBe(false);
  });
});

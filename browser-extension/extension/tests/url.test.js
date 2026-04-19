import { describe, it, expect } from 'vitest';
import { validateApiUrl } from '../lib/url.js';

describe('validateApiUrl', () => {
  it('accepts http/https', () => {
    expect(validateApiUrl('http://127.0.0.1:7080/').ok).toBe(true);
    expect(validateApiUrl('https://api.example.com').ok).toBe(true);
  });

  it('rejects non-http', () => {
    expect(validateApiUrl('ftp://x').ok).toBe(false);
    expect(validateApiUrl('javascript:alert(1)').ok).toBe(false);
  });

  it('strips trailing slash', () => {
    expect(validateApiUrl('http://x/').url).toBe('http://x');
  });

  it('rejects empty/garbage', () => {
    expect(validateApiUrl('').ok).toBe(false);
    expect(validateApiUrl('not a url').ok).toBe(false);
    expect(validateApiUrl(null).ok).toBe(false);
  });
});

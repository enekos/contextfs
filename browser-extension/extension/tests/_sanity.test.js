import { describe, it, expect } from 'vitest';

describe('sanity', () => {
  it('chrome mock present', () => {
    expect(globalThis.chrome.runtime.id).toBe('mock-extension-id');
  });
});

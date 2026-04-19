import { describe, it, expect } from 'vitest';
import { waitForSpaHydration } from '../lib/hydration.js';

describe('hydration', () => {
  it('resolves quickly when DOM is stable', async () => {
    const r = await waitForSpaHydration(document, { quietMs: 20, hardCapMs: 200 });
    expect(['quiet', 'hardcap']).toContain(r);
  });

  it('resolves via hardcap when mutations never stop', async () => {
    const interval = setInterval(() => {
      document.body.appendChild(document.createElement('div'));
    }, 5);
    const r = await waitForSpaHydration(document, { quietMs: 200, hardCapMs: 100 });
    clearInterval(interval);
    expect(r).toBe('hardcap');
  });

  it('resolves via quiet when mutations stop', async () => {
    setTimeout(() => {
      document.body.appendChild(document.createElement('div'));
    }, 10);
    const r = await waitForSpaHydration(document, { quietMs: 50, hardCapMs: 500 });
    expect(r).toBe('quiet');
  });
});

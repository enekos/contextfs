import { describe, it, expect } from 'vitest';
import { installOverlay } from '../lib/overlay.js';

describe('overlay', () => {
  it('creates a single mairu-overlay element', () => {
    const o = installOverlay(document);
    expect(document.querySelectorAll('mairu-overlay').length).toBe(1);
    o.destroy();
    expect(document.querySelectorAll('mairu-overlay').length).toBe(0);
  });

  it('shows and hides thought text via closed shadow root', () => {
    const o = installOverlay(document);
    o.showThought('hello');
    const shadow = o._element._testShadow();
    expect(shadow.textContent).toContain('hello');
    o.hideThought();
    o.destroy();
  });

  it('is idempotent on repeated install', () => {
    const a = installOverlay(document);
    installOverlay(document);
    expect(document.querySelectorAll('mairu-overlay').length).toBe(1);
    a.destroy();
  });
});

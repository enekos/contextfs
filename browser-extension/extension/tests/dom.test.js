import { describe, it, expect } from 'vitest';
import { getCssSelector, querySelectorDeep, walkShadowRoots, rafBatch } from '../lib/dom.js';

describe('dom helpers', () => {
  it('getCssSelector returns #id when present', () => {
    document.body.innerHTML = '<div><p></p><p id="x"></p></div>';
    expect(getCssSelector(document.getElementById('x'))).toBe('#x');
  });

  it('getCssSelector uses nth-of-type when no id', () => {
    document.body.innerHTML = '<div><p></p><p class="target"></p></div>';
    const sel = getCssSelector(document.querySelector('.target'));
    expect(sel).toContain('p:nth-of-type(2)');
  });

  it('querySelectorDeep finds elements inside open shadow roots', () => {
    const host = document.createElement('div');
    document.body.appendChild(host);
    const r = host.attachShadow({ mode: 'open' });
    r.innerHTML = '<button class="deep">x</button>';
    const el = querySelectorDeep(document, '.deep');
    expect(el?.textContent).toBe('x');
  });

  it('walkShadowRoots yields each open root', () => {
    const host = document.createElement('div');
    document.body.appendChild(host);
    host.attachShadow({ mode: 'open' });
    const roots = [...walkShadowRoots(document.body)];
    expect(roots.length).toBeGreaterThan(0);
  });

  it('rafBatch coalesces multiple calls into one', async () => {
    let count = 0;
    const b = rafBatch(() => { count++; });
    b();
    b();
    b();
    await new Promise((r) => setTimeout(r, 40));
    expect(count).toBe(1);
  });
});

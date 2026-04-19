import { describe, it, expect } from 'vitest';
import { serializeWithBudget } from '../lib/serializer.js';

describe('serializer budget', () => {
  it('marks truncated when over size limit', () => {
    document.body.innerHTML = '<div>' + 'x'.repeat(20) + '</div>';
    const r = serializeWithBudget(document.documentElement, { sizeLimit: 10 });
    expect(r.truncated).toBe(true);
  });

  it('returns full html under budget', () => {
    document.body.innerHTML = '<p>hi</p>';
    const r = serializeWithBudget(document.documentElement, { sizeLimit: 10000, timeBudgetMs: 1000 });
    expect(r.truncated).toBe(false);
    expect(r.html).toContain('<p>hi</p>');
  });

  it('escapes text content', () => {
    document.body.innerHTML = '<p></p>';
    document.querySelector('p').textContent = '<script>alert(1)</script>';
    const r = serializeWithBudget(document.documentElement, { sizeLimit: 10000, timeBudgetMs: 1000 });
    expect(r.html).not.toContain('<script>alert');
    expect(r.html).toContain('&lt;script&gt;');
  });

  it('stubs out script/style/svg', () => {
    document.body.innerHTML = '<script>var a=1</script><style>.x{}</style><p>ok</p>';
    const r = serializeWithBudget(document.documentElement, { sizeLimit: 10000, timeBudgetMs: 1000 });
    expect(r.html).toContain('<script></script>');
    expect(r.html).toContain('<style></style>');
    expect(r.html).toContain('<p>ok</p>');
  });
});

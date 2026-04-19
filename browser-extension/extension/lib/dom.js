export function getCssSelector(el) {
  if (!el || el.nodeType !== 1) return '';
  const path = [];
  while (el && el.nodeType === 1) {
    if (el.id) {
      path.unshift('#' + cssEscape(el.id));
      break;
    }
    let selector = el.localName;
    let sib = el;
    let nth = 1;
    while ((sib = sib.previousElementSibling)) {
      if (sib.localName === el.localName) nth++;
    }
    if (nth !== 1) selector += `:nth-of-type(${nth})`;
    path.unshift(selector);
    const parent = el.parentNode;
    if (!parent) break;
    // Walk past shadow-root boundaries
    if (parent.host) {
      el = parent.host;
      continue;
    }
    if (parent === parent.ownerDocument) break;
    el = parent;
  }
  return path.join(' > ');
}

function cssEscape(s) {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') return CSS.escape(s);
  return String(s).replace(/[^a-zA-Z0-9_-]/g, (c) => `\\${c}`);
}

function getShadowRoot(el) {
  if (typeof chrome !== 'undefined' && chrome.dom && typeof chrome.dom.openOrClosedShadowRoot === 'function') {
    try { return chrome.dom.openOrClosedShadowRoot(el); } catch (err) { void err; }
  }
  return el.shadowRoot || null;
}

export function* walkShadowRoots(root) {
  const stack = [root];
  while (stack.length) {
    const node = stack.pop();
    if (!node || !node.querySelectorAll) continue;
    for (const el of node.querySelectorAll('*')) {
      const sr = getShadowRoot(el);
      if (sr) {
        yield sr;
        stack.push(sr);
      }
    }
  }
}

export function querySelectorDeep(root, selector) {
  if (!root) return null;
  if (root.querySelector) {
    const found = root.querySelector(selector);
    if (found) return found;
  }
  for (const r of walkShadowRoots(root)) {
    const hit = r.querySelector(selector);
    if (hit) return hit;
  }
  return null;
}

export function querySelectorAllDeep(root, selector, limit = 1000) {
  const out = [];
  const visit = (n) => {
    if (out.length >= limit || !n.querySelectorAll) return;
    for (const el of n.querySelectorAll(selector)) {
      if (out.length >= limit) return;
      out.push(el);
    }
  };
  visit(root);
  for (const r of walkShadowRoots(root)) visit(r);
  return out;
}

export function rafBatch(fn) {
  let scheduled = false;
  return function () {
    if (scheduled) return;
    scheduled = true;
    const raf = typeof requestAnimationFrame === 'function' ? requestAnimationFrame : (cb) => setTimeout(cb, 16);
    raf(() => {
      scheduled = false;
      try { fn(); } catch (err) { void err; }
    });
  };
}

export function isVisible(el) {
  try {
    const s = getComputedStyle(el);
    return s.display !== 'none' && s.visibility !== 'hidden' && s.opacity !== '0';
  } catch (err) {
    void err;
    return true;
  }
}

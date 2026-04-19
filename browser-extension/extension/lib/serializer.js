export function serializeWithBudget(root, { timeBudgetMs = 120, sizeLimit = 2 * 1024 * 1024 } = {}) {
  const start = (typeof performance !== 'undefined' ? performance.now() : Date.now());
  const parts = [];
  let size = 0;
  let truncated = false;

  function now() {
    return typeof performance !== 'undefined' ? performance.now() : Date.now();
  }
  function push(s) {
    if (truncated) return;
    if (size + s.length > sizeLimit) {
      truncated = true;
      return;
    }
    parts.push(s);
    size += s.length;
  }
  function escapeText(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }
  function serNode(node) {
    if (truncated) return;
    if (now() - start > timeBudgetMs) {
      truncated = true;
      return;
    }
    if (node.nodeType === 3) {
      push(escapeText(node.textContent));
      return;
    }
    if (node.nodeType !== 1) return;
    const tag = node.localName;
    if (tag === 'script' || tag === 'style' || tag === 'svg' || tag === 'noscript') {
      push(`<${tag}></${tag}>`);
      return;
    }
    push('<' + tag);
    for (const a of node.attributes) {
      push(` ${a.name}="${a.value.replace(/"/g, '&quot;')}"`);
    }
    push('>');
    if (node.shadowRoot) {
      push('<template shadowrootmode="open">');
      for (const c of node.shadowRoot.childNodes) serNode(c);
      push('</template>');
    }
    for (const c of node.childNodes) serNode(c);
    push(`</${tag}>`);
  }

  serNode(root);
  return { html: parts.join(''), truncated, size };
}

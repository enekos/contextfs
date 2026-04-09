// mairu-ext/extension/content.js

// Capture page content and forward to service worker.
// Runs at document_idle — DOM is ready.

(function () {
  const MUTATION_DEBOUNCE_MS = 2000;
  let debounceTimer = null;
  let consoleErrors = [];

  // 1. Inject script to trap console errors
  const script = document.createElement('script');
  script.textContent = `
    (function() {
      const originalError = console.error;
      console.error = function(...args) {
        window.postMessage({ type: '__MAIRU_ERROR', error: args.map(a => String(a)).join(' ') }, '*');
        return originalError.apply(this, args);
      };
      window.addEventListener('error', function(e) {
        window.postMessage({ type: '__MAIRU_ERROR', error: e.message + ' at ' + e.filename + ':' + e.lineno }, '*');
      });
      window.addEventListener('unhandledrejection', function(e) {
        window.postMessage({ type: '__MAIRU_ERROR', error: 'Unhandled Rejection: ' + e.reason }, '*');
      });
    })();
  `;
  document.documentElement.appendChild(script);
  script.remove();

  window.addEventListener('message', (event) => {
    if (event.data && event.data.type === '__MAIRU_ERROR') {
      consoleErrors.push(event.data.error);
      if (consoleErrors.length > 50) consoleErrors.shift(); // keep last 50
    }
  });

  // 2. Helper to get CSS selector for an element
  function getCssSelector(el) {
    if (!el || el.nodeType !== 1) return '';
    let path = [];
    while (el && el.nodeType === 1) {
      let selector = el.localName;
      if (el.id) {
        selector += '#' + el.id;
        path.unshift(selector);
        break;
      } else {
        let sib = el, nth = 1;
        while (sib = sib.previousElementSibling) {
          if (sib.localName === el.localName) nth++;
        }
        if (nth !== 1) selector += ":nth-of-type(" + nth + ")";
      }
      path.unshift(selector);
      el = el.parentNode;
    }
    return path.join(' > ');
  }

  // 3. Shadow DOM Serializer + Form value sync
  function getSerializedHtml() {
    function serializeNode(node) {
      if (node.nodeType === Node.TEXT_NODE) {
        // Escape HTML for text nodes by using a dummy div
        const div = document.createElement('div');
        div.textContent = node.textContent;
        return div.innerHTML;
      }
      if (node.nodeType !== Node.ELEMENT_NODE) return '';

      const tag = node.localName;
      
      // Inline sync for inputs, textareas, selects (handles shadow DOM automatically)
      if (tag === 'input' || tag === 'textarea' || tag === 'select') {
        if (node.type === 'checkbox' || node.type === 'radio') {
          if (node.checked) node.setAttribute('checked', '');
          else node.removeAttribute('checked');
        } else if (node.value !== undefined) {
          node.setAttribute('value', node.value);
          if (tag === 'textarea') {
              node.textContent = node.value;
          }
        }
      }

      let html = '<' + tag;
      for (const attr of node.attributes) {
        html += ` ${attr.name}="${attr.value.replace(/"/g, '&quot;')}"`;
      }
      html += '>';

      // Inject declarative shadow DOM if exists
      if (node.shadowRoot) {
        html += '<template shadowrootmode="open">';
        html += Array.from(node.shadowRoot.childNodes).map(serializeNode).join('');
        html += '</template>';
      }

      html += Array.from(node.childNodes).map(serializeNode).join('');
      html += `</${tag}>`;
      return html;
    }

    return serializeNode(document.documentElement);
  }

  function captureAndSend() {
    const html = getSerializedHtml();
    const selection = window.getSelection().toString();
    
    // Find real active element (piercing shadow dom)
    let activeEl = document.activeElement;
    while (activeEl && activeEl.shadowRoot && activeEl.shadowRoot.activeElement) {
        activeEl = activeEl.shadowRoot.activeElement;
    }
    const active_element = getCssSelector(activeEl);

    chrome.runtime.sendMessage({
      type: "page_content",
      payload: {
        url: location.href,
        html: html,
        timestamp: Date.now(),
        selection: selection || null,
        active_element: active_element || null,
        console_errors: consoleErrors,
      },
    });
  }

  // Initial capture
  captureAndSend();

  // Watch for significant DOM mutations (SPA navigation, dynamic content)
  const observer = new MutationObserver(() => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(captureAndSend, MUTATION_DEBOUNCE_MS);
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });

  // Also listen for selection changes and focus changes
  document.addEventListener('selectionchange', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(captureAndSend, 1000);
  });
  
  document.addEventListener('focusin', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(captureAndSend, 1000);
  });

  // Detect SPA route changes
  let lastUrl = location.href;
  const urlCheck = setInterval(() => {
    if (location.href !== lastUrl) {
      lastUrl = location.href;
      // Small delay for new content to render
      setTimeout(captureAndSend, 500);
    }
  }, 1000);

  // Cleanup on unload
  window.addEventListener("unload", () => {
    observer.disconnect();
    clearInterval(urlCheck);
    clearTimeout(debounceTimer);
  });
})();

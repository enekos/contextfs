document.addEventListener('DOMContentLoaded', () => {
  const sessionIdEl = document.getElementById('session-id');
  const pageCountEl = document.getElementById('page-count');
  const pendingCountEl = document.getElementById('pending-count');
  const nativeLabel = document.getElementById('native-label');
  const statusDot = document.getElementById('status-dot');
  const syncBtn = document.getElementById('sync-btn');
  const searchInput = document.getElementById('search-input');
  const resultsList = document.getElementById('results-list');
  const apiUrlInput = document.getElementById('api-url');
  const saveBtn = document.getElementById('save-btn');
  const toast = document.getElementById('toast');

  let toastTimer = null;

  function showToast(msg) {
    toast.textContent = msg;
    toast.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toast.classList.remove('show'), 2000);
  }

  // Load saved API URL
  chrome.storage.local.get('mairu_api_url', ({ mairu_api_url }) => {
    if (mairu_api_url) apiUrlInput.value = mairu_api_url;
  });

  saveBtn.addEventListener('click', () => {
    const url = apiUrlInput.value.trim();
    if (!url) return;
    chrome.storage.local.set({ mairu_api_url: url }, () => showToast('Saved'));
    // Notify service worker of updated URL
    chrome.runtime.sendMessage({ type: 'set_api_url', url });
  });

  function updateStats() {
    chrome.runtime.sendMessage({ type: 'get_status' }, (response) => {
      if (chrome.runtime.lastError || !response) {
        statusDot.className = 'dot err';
        nativeLabel.textContent = 'error';
        return;
      }

      const connected = response.nativeHostConnected;
      statusDot.className = connected ? 'dot ok' : 'dot err';
      nativeLabel.textContent = connected ? 'connected' : 'disconnected';

      if (response.sessionId) {
        // Show last segment only to save space
        const parts = response.sessionId.split('-');
        sessionIdEl.textContent = parts[parts.length - 1] || response.sessionId;
        sessionIdEl.title = response.sessionId;
      }

      const summary = response.summary;
      pageCountEl.textContent = summary?.page_count ?? 0;
      pendingCountEl.textContent = response.pendingCount ?? 0;
    });
  }

  updateStats();
  setInterval(updateStats, 3000);

  // Search
  let searchTimer = null;
  searchInput.addEventListener('input', () => {
    clearTimeout(searchTimer);
    const q = searchInput.value.trim();
    if (!q) {
      resultsList.innerHTML = '';
      return;
    }
    searchTimer = setTimeout(() => {
      chrome.runtime.sendMessage({ type: 'search', query: q, limit: 5 }, (res) => {
        if (!res || !res.results) {
          resultsList.innerHTML = '<div class="empty">No results</div>';
          return;
        }
        if (res.results.length === 0) {
          resultsList.innerHTML = '<div class="empty">No results</div>';
          return;
        }
        resultsList.innerHTML = res.results.map(r => `
          <div class="result-item" data-url="${escapeAttr(r.url)}">
            <div class="title">${escapeHtml(r.title || r.url)}</div>
            <div class="url">${escapeHtml(r.url)}</div>
            ${r.snippet ? `<div class="snippet">${escapeHtml(r.snippet)}</div>` : ''}
          </div>
        `).join('');
        resultsList.querySelectorAll('.result-item').forEach(el => {
          el.addEventListener('click', () => {
            chrome.tabs.create({ url: el.dataset.url });
          });
        });
      });
    }, 300);
  });

  // Force sync
  syncBtn.addEventListener('click', () => {
    syncBtn.textContent = 'Syncing…';
    syncBtn.disabled = true;
    chrome.runtime.sendMessage({ type: 'force_sync' }, () => {
      setTimeout(() => {
        syncBtn.textContent = 'Force Sync';
        syncBtn.disabled = false;
        updateStats();
        showToast('Sync complete');
      }, 1200);
    });
  });

  function escapeHtml(str) {
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function escapeAttr(str) {
    return String(str).replace(/"/g, '&quot;');
  }
});

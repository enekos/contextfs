import { vi, beforeEach } from 'vitest';

function makeStorageArea() {
  const store = new Map();
  return {
    get: vi.fn(async (keys) => {
      if (keys == null) return Object.fromEntries(store);
      if (typeof keys === 'string') return store.has(keys) ? { [keys]: store.get(keys) } : {};
      if (Array.isArray(keys)) {
        const out = {};
        for (const k of keys) if (store.has(k)) out[k] = store.get(k);
        return out;
      }
      const out = {};
      for (const [k, def] of Object.entries(keys)) out[k] = store.has(k) ? store.get(k) : def;
      return out;
    }),
    set: vi.fn(async (obj) => {
      for (const [k, v] of Object.entries(obj)) store.set(k, v);
    }),
    remove: vi.fn(async (keys) => {
      const arr = Array.isArray(keys) ? keys : [keys];
      for (const k of arr) store.delete(k);
    }),
    clear: vi.fn(async () => store.clear()),
    _store: store,
  };
}

function makeChromeMock() {
  const listeners = new Map();
  return {
    runtime: {
      onMessage: {
        addListener: vi.fn((fn) => {
          listeners.set('onMessage', fn);
        }),
        removeListener: vi.fn(),
      },
      sendMessage: vi.fn(async (msg) => {
        const fn = listeners.get('onMessage');
        return fn ? await new Promise((r) => fn(msg, {}, r)) : undefined;
      }),
      lastError: null,
      id: 'mock-extension-id',
      getURL: vi.fn((p) => `chrome-extension://mock-extension-id/${p}`),
    },
    storage: {
      local: makeStorageArea(),
      session: makeStorageArea(),
    },
    tabs: {
      query: vi.fn(async () => [{ id: 1, url: 'https://example.com/', active: true }]),
      sendMessage: vi.fn(async () => ({ ok: true })),
      captureVisibleTab: vi.fn((_w, _o, cb) => cb('data:image/jpeg;base64,AAA')),
      create: vi.fn(),
    },
    cookies: { set: vi.fn((d, cb) => cb({ ...d })) },
    scripting: {
      executeScript: vi.fn(async () => [{ result: null }]),
    },
    contextMenus: {
      create: vi.fn(),
      onClicked: { addListener: vi.fn() },
    },
    webNavigation: {
      onCommitted: { addListener: vi.fn() },
    },
    alarms: {
      create: vi.fn(),
      onAlarm: { addListener: vi.fn() },
    },
    _listeners: listeners,
  };
}

beforeEach(() => {
  globalThis.chrome = makeChromeMock();
});

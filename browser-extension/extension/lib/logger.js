export const LEVELS = ['debug', 'info', 'warn', 'error'];

export function createLogger({ capacity = 500 } = {}) {
  const buf = [];
  const subs = new Set();

  function emit(level, event, fields) {
    const entry = { t: Date.now(), level, event: String(event), fields: fields || {} };
    buf.push(entry);
    if (buf.length > capacity) buf.splice(0, buf.length - capacity);
    for (const s of subs) {
      try { s(entry); } catch (err) { void err; }
    }
    return entry;
  }

  return {
    debug: (e, f) => emit('debug', e, f),
    info:  (e, f) => emit('info',  e, f),
    warn:  (e, f) => emit('warn',  e, f),
    error: (e, f) => emit('error', e, f),
    snapshot: () => buf.slice(),
    subscribe: (fn) => {
      subs.add(fn);
      return () => subs.delete(fn);
    },
  };
}

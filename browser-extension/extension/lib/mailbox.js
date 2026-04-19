export function createMailbox({ cap = 20, logger } = {}) {
  const buf = [];
  return {
    push(entry) {
      if (buf.length >= cap) {
        const dropped = buf.shift();
        logger?.warn?.('mailbox.drop', { type: dropped?.message?.type });
      }
      buf.push(entry);
    },
    flush(handler) {
      while (buf.length) {
        const e = buf.shift();
        try { handler(e); } catch (err) { void err; }
      }
    },
    size: () => buf.length,
  };
}

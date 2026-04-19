export function waitForSpaHydration(doc, { quietMs = 500, hardCapMs = 3000 } = {}) {
  return new Promise((resolve) => {
    let obs;
    let quietTimer;
    const target = (doc && (doc.body || doc.documentElement)) || null;
    const done = (reason) => {
      try { obs && obs.disconnect(); } catch (err) { void err; }
      clearTimeout(quietTimer);
      clearTimeout(hardTimer);
      resolve(reason);
    };
    if (typeof MutationObserver !== 'undefined' && target) {
      obs = new MutationObserver(() => {
        clearTimeout(quietTimer);
        quietTimer = setTimeout(() => done('quiet'), quietMs);
      });
      try { obs.observe(target, { childList: true, subtree: true }); } catch (err) { void err; }
    }
    quietTimer = setTimeout(() => done('quiet'), quietMs);
    const hardTimer = setTimeout(() => done('hardcap'), hardCapMs);
  });
}

export function validateApiUrl(value) {
  if (typeof value !== 'string' || !value.trim()) {
    return { ok: false, error: 'empty' };
  }
  let u;
  try {
    u = new URL(value.trim());
  } catch {
    return { ok: false, error: 'invalid' };
  }
  if (u.protocol !== 'http:' && u.protocol !== 'https:') {
    return { ok: false, error: 'bad_protocol' };
  }
  const canonical = u.toString().replace(/\/$/, '');
  return { ok: true, url: canonical };
}

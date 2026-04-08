export function getOrSet(cache: Map<string, any>, key: string, compute: () => any): any {
  if (cache.has(key)) {
    return cache.get(key);
  }
  const value = compute();
  cache.set(key, value);
  return value;
}

export function evictStale(cache: Map<string, Entry>, maxAge: number): number {
  let evicted = 0;
  const now = Date.now();
  for (const [key, entry] of cache) {
    if (now - entry.createdAt > maxAge) {
      cache.delete(key);
      evicted++;
    }
  }
  return evicted;
}

export function warmUp(cache: Map<string, any>, keys: string[], loader: (k: string) => any): void {
  for (const key of keys) {
    if (!cache.has(key)) {
      const value = loader(key);
      cache.set(key, value);
    }
  }
}

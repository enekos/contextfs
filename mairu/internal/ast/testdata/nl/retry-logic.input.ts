export async function withRetry(
  fn: () => Promise<any>,
  maxAttempts: number,
  delayMs: number
): Promise<any> {
  let lastError: Error | null = null;
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      const result = await fn();
      return result;
    } catch (e) {
      lastError = e as Error;
      if (attempt < maxAttempts - 1) {
        await sleep(delayMs * Math.pow(2, attempt));
      }
    }
  }
  throw lastError;
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

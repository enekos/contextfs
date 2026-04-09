## sleep (fn)

1. Returns a new `Promise` with `resolve => setTimeout(resolve, ms)`

## withRetry (fn)

1. Assigns `null` to let variable `lastError` of type `Error | null`
2. Loops with let attempt = 0; while attempt < maxAttempts; attempt++: Attempts to Assigns awaits calling `fn` to constant `result`; Returns `result`. If an error occurs (e), assigning `e as Error` to `lastError`; If `attempt` is less than `maxAttempts - 1`, awaits calling `sleep` with `delayMs * Math.pow(2, attempt)`
3. Throws `lastError`

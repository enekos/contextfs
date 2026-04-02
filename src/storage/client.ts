import { ContextManager } from "./contextManager";
import { config } from "../core/config";

export function createContextManager(): ContextManager {
  const url = config.meili.url;

  if (!url) {
    throw new Error("Please set MEILI_URL in your .env file or environment.");
  }

  return new ContextManager(url, config.meili.apiKey || undefined);
}

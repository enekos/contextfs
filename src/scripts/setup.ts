import { MeilisearchDB } from "../storage/meilisearchDB";
import { config } from "../core/config";

const url = config.meili.url;

if (!url) {
  console.error("Please set MEILI_URL in your .env file");
  process.exit(1);
}

const db = new MeilisearchDB(url, config.meili.apiKey || undefined);

async function main() {
  console.log("Resetting and initializing Meilisearch indices for Agent Context...");
  try {
    await db.resetIndices();
    await db.initIndices();
    console.log("Successfully reset and initialized indices!");
  } catch (err) {
    console.error("Failed to reset/initialize indices:", err);
  }
}

main();

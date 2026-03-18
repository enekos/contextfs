import { Client } from "@elastic/elasticsearch";
import { SKILLS_INDEX, MEMORIES_INDEX, CONTEXT_INDEX } from "../src/elasticDB";
import * as dotenv from "dotenv";
dotenv.config();

const client = new Client({
  node: process.env.ELASTIC_URL || "http://localhost:9200",
  ...(process.env.ELASTIC_USERNAME && process.env.ELASTIC_PASSWORD
    ? { auth: { username: process.env.ELASTIC_USERNAME, password: process.env.ELASTIC_PASSWORD } }
    : {}),
});

async function run() {
  for (const index of [CONTEXT_INDEX, MEMORIES_INDEX, SKILLS_INDEX]) {
    await client.deleteByQuery({
      index,
      query: { match_all: {} },
      refresh: true,
    }).catch(() => {});
  }
  console.log("Indices cleared!");
}
run();

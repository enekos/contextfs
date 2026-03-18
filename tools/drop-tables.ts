import { ElasticDB } from "../src/elasticDB";
import * as dotenv from "dotenv";
dotenv.config();

const db = new ElasticDB(
  process.env.ELASTIC_URL || "http://localhost:9200",
  process.env.ELASTIC_USERNAME && process.env.ELASTIC_PASSWORD
    ? { username: process.env.ELASTIC_USERNAME, password: process.env.ELASTIC_PASSWORD }
    : undefined
);

async function run() {
  await db.resetIndices();
  console.log("Indices dropped!");
}
run();

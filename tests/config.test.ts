import { describe, it, expect } from "vitest";
import { assertEmbeddingDimension, config } from "../mairu/contextfs/src/core/config";

describe("assertEmbeddingDimension", () => {
  const dimension = config.embedding.dimension;

  it("does not throw for correctly sized vector", () => {
    const vector = Array(dimension).fill(0);
    expect(() => assertEmbeddingDimension(vector, "test")).not.toThrow();
  });

  it("throws for undersized vector", () => {
    const vector = Array(dimension - 1).fill(0);
    expect(() => assertEmbeddingDimension(vector, "test-context")).toThrow(
      `Invalid embedding size for test-context. Expected ${dimension}, got ${dimension - 1}.`
    );
  });

  it("throws for oversized vector", () => {
    const vector = Array(dimension + 1).fill(0);
    expect(() => assertEmbeddingDimension(vector, "my-context")).toThrow(
      `Invalid embedding size for my-context. Expected ${dimension}, got ${dimension + 1}.`
    );
  });
});

describe("config.embedding", () => {
  it("returns model, dimension, and allowZeroEmbeddings", () => {
    const embedConfig = config.embedding;
    expect(embedConfig).toHaveProperty("model");
    expect(embedConfig).toHaveProperty("dimension");
    expect(embedConfig).toHaveProperty("allowZeroEmbeddings");
    expect(typeof embedConfig.model).toBe("string");
    expect(typeof embedConfig.dimension).toBe("number");
    expect(typeof embedConfig.allowZeroEmbeddings).toBe("boolean");
  });
});

describe("dream config", () => {
  it("has default dream settings", async () => {
    const { config } = await import("../mairu/contextfs/src/core/config");
    expect(config.dream.threshold).toBe(25);
    expect(config.dream.cooldownMs).toBe(4 * 60 * 60 * 1000);
    expect(config.dream.idleTimeoutMs).toBe(30 * 60 * 1000);
    expect(config.dream.enabled).toBe(true);
  });

  it("reads dream settings from env vars", async () => {
    process.env.DREAM_THRESHOLD = "50";
    process.env.DREAM_COOLDOWN = "2h";
    process.env.DREAM_IDLE_TIMEOUT = "15m";
    process.env.DREAM_ENABLED = "false";

    const { vi } = await import("vitest");
    vi.resetModules();
    const { config } = await import("../mairu/contextfs/src/core/config");
    expect(config.dream.threshold).toBe(50);
    expect(config.dream.cooldownMs).toBe(2 * 60 * 60 * 1000);
    expect(config.dream.idleTimeoutMs).toBe(15 * 60 * 1000);
    expect(config.dream.enabled).toBe(false);
  });
});
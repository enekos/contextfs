import { describe, it, expect, vi, beforeEach } from "vitest";

const mockCount = vi.fn();
const mockBulk = vi.fn();

vi.mock("@elastic/elasticsearch", () => ({
  Client: vi.fn().mockImplementation(() => ({
    count: mockCount,
    bulk: mockBulk,
    indices: { exists: vi.fn().mockResolvedValue(true), create: vi.fn(), delete: vi.fn() },
    index: vi.fn(),
    get: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    search: vi.fn(),
    updateByQuery: vi.fn(),
  })),
  HttpConnection: vi.fn(),
}));

import { ElasticDB, MEMORIES_INDEX } from "../src/storage/elasticDB";

describe("countByProject", () => {
  let db: ElasticDB;

  beforeEach(() => {
    vi.clearAllMocks();
    db = new ElasticDB("http://localhost:9200");
  });

  it("returns count for a project", async () => {
    mockCount.mockResolvedValue({ count: 42 });
    const result = await db.countByProject(MEMORIES_INDEX, "my-project");
    expect(result).toBe(42);
    expect(mockCount).toHaveBeenCalledWith({
      index: MEMORIES_INDEX,
      query: { term: { project: "my-project" } },
    });
  });

  it("returns 0 when no documents match", async () => {
    mockCount.mockResolvedValue({ count: 0 });
    const result = await db.countByProject(MEMORIES_INDEX, "empty-project");
    expect(result).toBe(0);
  });
});

describe("bulkIndex", () => {
  let db: ElasticDB;

  beforeEach(() => {
    vi.clearAllMocks();
    db = new ElasticDB("http://localhost:9200");
  });

  it("indexes multiple documents in one bulk call", async () => {
    mockBulk.mockResolvedValue({
      errors: false,
      items: [
        { index: { _id: "1", status: 201 } },
        { index: { _id: "2", status: 201 } },
      ],
    });

    const result = await db.bulkIndex([
      { index: MEMORIES_INDEX, id: "1", body: { content: "a" } },
      { index: MEMORIES_INDEX, id: "2", body: { content: "b" } },
    ]);

    expect(result.successful).toBe(2);
    expect(result.failed).toBe(0);
    expect(result.errors).toHaveLength(0);
  });

  it("reports per-item errors", async () => {
    mockBulk.mockResolvedValue({
      errors: true,
      items: [
        { index: { _id: "1", status: 201 } },
        { index: { _id: "2", status: 400, error: { reason: "bad mapping" } } },
      ],
    });

    const result = await db.bulkIndex([
      { index: MEMORIES_INDEX, id: "1", body: { content: "a" } },
      { index: MEMORIES_INDEX, id: "2", body: { content: "bad" } },
    ]);

    expect(result.successful).toBe(1);
    expect(result.failed).toBe(1);
    expect(result.errors[0]).toEqual({ id: "2", error: "bad mapping" });
  });

  it("returns all zeros for empty input", async () => {
    const result = await db.bulkIndex([]);
    expect(result.successful).toBe(0);
    expect(result.failed).toBe(0);
    expect(mockBulk).not.toHaveBeenCalled();
  });
});

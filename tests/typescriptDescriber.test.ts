import { describe, expect, it, beforeAll, afterAll } from "vitest";
import { TypeScriptDescriber } from "../mairu/contextfs/src/ast/typescriptDescriber";

describe("TypeScriptDescriber", () => {
  const describer = new TypeScriptDescriber();

  beforeAll(async () => {
    await describer.initParsers();
  });

  afterAll(() => {
    describer.deleteParsers();
  });

  it("extracts symbols and edges from TypeScript source", () => {
    const source = [
      "import { slugify } from './slug';",
      "const INTERNAL_SEED = 42;",
      "",
      "function normalize(name: string) {",
      "  return slugify(`${name}-${INTERNAL_SEED}`);",
      "}",
      "",
      "export function greet(name: string) {",
      "  return normalize(name);",
      "}",
      "",
      "export class UserService {",
      "  public run(input: string) {",
      "    this.bump();",
      "    return greet(input);",
      "  }",
      "  private bump() {",
      "    return normalize('x');",
      "  }",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/feature.ts", source);

    const symbolIds = result.symbols.map(s => s.id);
    expect(symbolIds).toContain("fn:greet");
    expect(symbolIds).toContain("fn:normalize");
    expect(symbolIds).toContain("cls:UserService");
    expect(symbolIds).toContain("mtd:UserService.run");
    expect(symbolIds).toContain("mtd:UserService.bump");
    expect(symbolIds).toContain("var:INTERNAL_SEED");

    const edgeKeys = result.edges.map(e => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("call:fn:greet->fn:normalize");
    expect(edgeKeys).toContain("call:mtd:UserService.run->mtd:UserService.bump");
    expect(edgeKeys).toContain("call:mtd:UserService.run->fn:greet");
    expect(edgeKeys).toContain("import:file->module:./slug");

    expect(result.imports).toContain("./slug");
  });

  it("generates NL descriptions for each function/method symbol", () => {
    const source = [
      "export function greet(name: string) {",
      "  const trimmed = name.trim();",
      "  if (!trimmed) {",
      "    throw new Error('Name required');",
      "  }",
      "  return `Hello ${trimmed}`;",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/greet.ts", source);

    const greetDesc = result.symbolDescriptions.get("fn:greet");
    expect(greetDesc).toBeDefined();
    expect(greetDesc).toContain("trimmed");
    expect(greetDesc).toMatch(/[Tt]hrows/);
    expect(greetDesc).toMatch(/[Rr]eturns/);
  });

  it("generates NL descriptions for class methods", () => {
    const source = [
      "export class Calculator {",
      "  add(a: number, b: number) {",
      "    return a + b;",
      "  }",
      "  divide(a: number, b: number) {",
      "    if (b === 0) {",
      "      throw new Error('Division by zero');",
      "    }",
      "    return a / b;",
      "  }",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/calc.ts", source);

    expect(result.symbolDescriptions.get("mtd:Calculator.add")).toBeDefined();
    const divideDesc = result.symbolDescriptions.get("mtd:Calculator.divide");
    expect(divideDesc).toBeDefined();
    expect(divideDesc).toMatch(/[Dd]ivision|zero/);
  });

  it("populates byteStart, byteEnd, and contentHash on symbols", () => {
    const source = [
      "export function hello() {",
      "  return 'world';",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/offsets.ts", source);
    const hello = result.symbols.find(s => s.id === "fn:hello")!;

    expect(hello.byteStart).toBeGreaterThanOrEqual(0);
    expect(hello.byteEnd).toBeGreaterThan(hello.byteStart);
    expect(hello.contentHash).toMatch(/^[0-9a-f]{40}$/);

    // Verify offsets match actual source text
    const sliced = source.slice(hello.byteStart, hello.byteEnd);
    expect(sliced).toContain("function hello");
    expect(sliced).toContain("return 'world'");
  });

  it("gives different contentHash for different function bodies", () => {
    const source = [
      "function a() { return 1; }",
      "function b() { return 2; }",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/hashes.ts", source);
    const a = result.symbols.find(s => s.id === "fn:a")!;
    const b = result.symbols.find(s => s.id === "fn:b")!;

    expect(a.contentHash).not.toBe(b.contentHash);
  });

  it("gives different contentHash for functions with different names but same body", () => {
    const source = [
      "function a() { return 1; }",
      "function b() { return 1; }",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/same.ts", source);
    const a = result.symbols.find(s => s.id === "fn:a")!;
    const b = result.symbols.find(s => s.id === "fn:b")!;

    // Different names but same body — hash covers full node text so they differ
    expect(a.contentHash).not.toBe(b.contentHash);
  });

  it("generates a file summary", () => {
    const source = [
      "export function greet(name: string) { return 'Hello ' + name; }",
      "export function farewell(name: string) { return 'Bye ' + name; }",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/greetings.ts", source);
    expect(result.fileSummary).toBeTruthy();
    expect(result.fileSummary).toMatch(/greet|farewell/i);
  });

  it("extracts symbols from empty file", () => {
    const result = describer.extractFileGraph("/tmp/test/empty.ts", "/* empty */");
    expect(result.symbols).toHaveLength(0);
    expect(result.edges).toHaveLength(0);
    expect(typeof result.fileSummary).toBe("string");
  });

  it("extracts JSDoc docstring from preceding comment", () => {
    const source = [
      "/** Validates an email address against RFC 5322 rules. */",
      "export function validate(email: string) {",
      "  return email.includes('@');",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/jsdoc.ts", source);
    const validate = result.symbols.find(s => s.id === "fn:validate")!;

    expect(validate.docstring).toBe("Validates an email address against RFC 5322 rules.");
  });

  it("extracts first sentence from multi-line JSDoc", () => {
    const source = [
      "/**",
      " * Fetches user data from the API. This function handles",
      " * retries and timeout logic internally.",
      " * @param userId - The user ID to fetch",
      " */",
      "export function fetchUser(userId: string) {",
      "  return fetch(`/api/users/${userId}`);",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/multiline.ts", source);
    const fetchUser = result.symbols.find(s => s.id === "fn:fetchUser")!;

    expect(fetchUser.docstring).toBe("Fetches user data from the API.");
  });

  it("extracts JSDoc for class methods", () => {
    const source = [
      "export class Service {",
      "  /** Starts the service and binds to the given port. */",
      "  start(port: number) {",
      "    return this.listen(port);",
      "  }",
      "  listen(port: number) {}",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/method-jsdoc.ts", source);
    const start = result.symbols.find(s => s.id === "mtd:Service.start")!;
    const listen = result.symbols.find(s => s.id === "mtd:Service.listen")!;

    expect(start.docstring).toBe("Starts the service and binds to the given port.");
    expect(listen.docstring).toBeUndefined();
  });

  it("extracts JSDoc for class declarations", () => {
    const source = [
      "/** Manages user sessions and authentication state. */",
      "export class SessionManager {",
      "  clear() {}",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/class-jsdoc.ts", source);
    const cls = result.symbols.find(s => s.id === "cls:SessionManager")!;

    expect(cls.docstring).toBe("Manages user sessions and authentication state.");
  });

  it("skips non-JSDoc comments", () => {
    const source = [
      "// This is a regular comment",
      "export function noDoc() { return 1; }",
      "",
      "/* Block comment but not JSDoc */",
      "export function alsoNoDoc() { return 2; }",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/no-jsdoc.ts", source);
    const noDoc = result.symbols.find(s => s.id === "fn:noDoc")!;
    const alsoNoDoc = result.symbols.find(s => s.id === "fn:alsoNoDoc")!;

    expect(noDoc.docstring).toBeUndefined();
    expect(alsoNoDoc.docstring).toBeUndefined();
  });

  it("handles JSDoc with decorator between comment and declaration", () => {
    const source = [
      "/** Handles incoming HTTP requests. */",
      "@Controller('/api')",
      "export class ApiController {",
      "  handle() {}",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/decorator-jsdoc.ts", source);
    const cls = result.symbols.find(s => s.id === "cls:ApiController")!;

    expect(cls.docstring).toBe("Handles incoming HTTP requests.");
  });

  it("returns undefined docstring for empty JSDoc", () => {
    const source = [
      "/** */",
      "export function empty() { return 1; }",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/empty-jsdoc.ts", source);
    const empty = result.symbols.find(s => s.id === "fn:empty")!;

    expect(empty.docstring).toBeUndefined();
  });
});

import { describe, expect, it } from "vitest";
import type { LanguageDescriber, FileGraphResult, LogicSymbol } from "../src/ast/languageDescriber";
import { sortSymbols } from "../src/ast/languageDescriber";
import { TypeScriptDescriber } from "../src/ast/typescriptDescriber";

describe("LanguageDescriber interface", () => {
  it("TypeScriptDescriber implements LanguageDescriber", () => {
    const describer: LanguageDescriber = new TypeScriptDescriber();
    expect(describer.languageId).toBe("typescript");
    expect(describer.extensions).toContain(".ts");
    expect(describer.extensions).toContain(".tsx");
    expect(describer.extensions).toContain(".js");
    expect(describer.extensions).toContain(".jsx");
    expect(describer.extensions).toContain(".mjs");
    expect(describer.extensions).toContain(".cjs");
    expect(typeof describer.extractFileGraph).toBe("function");
  });

  it("sorts template symbol kinds after script kinds", () => {
    const tplSymbol: LogicSymbol = {
      id: "tpl:App", kind: "tpl", name: "App", exported: false,
      parentId: null, params: [], complexity: "low",
      control: { async: false, branch: false, await: false, throw: false }, line: 1,
    };
    const fnSymbol: LogicSymbol = {
      id: "fn:setup", kind: "fn", name: "setup", exported: true,
      parentId: null, params: [], complexity: "low",
      control: { async: false, branch: false, await: false, throw: false }, line: 1,
    };
    const sorted = sortSymbols([tplSymbol, fnSymbol]);
    expect(sorted[0].kind).toBe("fn");
    expect(sorted[1].kind).toBe("tpl");
  });
});

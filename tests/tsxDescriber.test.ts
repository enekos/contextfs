import { describe, expect, it } from "vitest";
import { TsxDescriber } from "../src/ast/tsxDescriber";

describe("TsxDescriber", () => {
  const describer = new TsxDescriber();

  it("extracts script symbols AND template symbols from TSX", () => {
    const source = [
      "import React from 'react';",
      "",
      "function App() {",
      "  const isOpen = true;",
      "  return (",
      "    <div>",
      "      <Header />",
      "      {isOpen && <Modal />}",
      "    </div>",
      "  );",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/App.tsx", source);

    const symbolIds = result.symbols.map((s) => s.id);

    // Script symbols
    expect(symbolIds).toContain("fn:App");

    // Template symbols
    expect(symbolIds).toContain("tpl:App");
    expect(symbolIds).toContain("tpl-branch:App.if_isOpen");

    // Render edges
    const edgeKeys = result.edges.map((e) => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("render:tpl:App->type:Header");
    expect(edgeKeys).toContain("render:tpl-branch:App.if_isOpen->type:Modal");
  });

  it("extracts ternary as if/else branches", () => {
    const source = [
      "function Page() {",
      "  const loading = true;",
      "  return (",
      "    <div>",
      "      {loading ? <Spinner /> : <Content />}",
      "    </div>",
      "  );",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/Page.tsx", source);

    const symbolIds = result.symbols.map((s) => s.id);
    expect(symbolIds).toContain("tpl-branch:Page.if_loading");
    expect(symbolIds).toContain("tpl-branch:Page.else");

    const edgeKeys = result.edges.map((e) => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("render:tpl-branch:Page.if_loading->type:Spinner");
    expect(edgeKeys).toContain("render:tpl-branch:Page.else->type:Content");
  });

  it("extracts .map() as tpl-loop", () => {
    const source = [
      "function UserList({ users }) {",
      "  return (",
      "    <div>",
      "      {users.map(u => <UserCard />)}",
      "    </div>",
      "  );",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/UserList.tsx", source);

    const symbolIds = result.symbols.map((s) => s.id);
    expect(symbolIds).toContain("tpl-loop:UserList.for_users");

    const edgeKeys = result.edges.map((e) => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("render:tpl-loop:UserList.for_users->type:UserCard");
  });

  it("handles arrow function components", () => {
    const source = [
      "const Card = () => (",
      "  <div>",
      "    <Header />",
      "  </div>",
      ");",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/Card.tsx", source);

    const symbolIds = result.symbols.map((s) => s.id);
    expect(symbolIds).toContain("tpl:Card");

    const edgeKeys = result.edges.map((e) => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("render:tpl:Card->type:Header");
  });

  it("produces no template symbols for non-JSX functions", () => {
    const source = [
      "function add(a: number, b: number) {",
      "  return a + b;",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/math.tsx", source);

    const templateSymbols = result.symbols.filter((s) =>
      s.kind === "tpl" || s.kind === "tpl-branch" || s.kind === "tpl-loop" || s.kind === "tpl-slot"
    );
    expect(templateSymbols).toHaveLength(0);
  });

  it("handles nested branch inside map", () => {
    const source = [
      "function ItemList({ items }) {",
      "  return (",
      "    <div>",
      "      {items.map(i => (",
      "        <div>{i.active && <Badge />}</div>",
      "      ))}",
      "    </div>",
      "  );",
      "}",
    ].join("\n");

    const result = describer.extractFileGraph("/tmp/test/ItemList.tsx", source);

    const symbolIds = result.symbols.map((s) => s.id);
    expect(symbolIds).toContain("tpl-loop:ItemList.for_items");
    expect(symbolIds).toContain("tpl-branch:ItemList.if_i.active");

    const edgeKeys = result.edges.map((e) => `${e.kind}:${e.from}->${e.to}`);
    expect(edgeKeys).toContain("render:tpl-branch:ItemList.if_i.active->type:Badge");
  });
});

import { describe, it, expect } from "vitest";
import {
  walkTemplate,
  type TemplateNode,
} from "../mairu/contextfs/src/ast/templateWalker";

function makeNode(
  tag: string,
  opts: Partial<TemplateNode> = {},
): TemplateNode {
  return {
    tag,
    isComponent: opts.isComponent ?? /^[A-Z]/.test(tag),
    directives: opts.directives ?? [],
    children: opts.children ?? [],
    line: opts.line ?? 1,
    slotName: opts.slotName,
  };
}

describe("walkTemplate", () => {
  it("creates tpl root and render edges for child components", () => {
    const nodes: TemplateNode[] = [
      makeNode("div", {
        children: [
          makeNode("Header", { line: 2 }),
          makeNode("Footer", { line: 3 }),
        ],
      }),
    ];

    const result = walkTemplate("MyPage", nodes);

    // Root symbol
    const root = result.symbols.find((s) => s.id === "tpl:MyPage");
    expect(root).toBeDefined();
    expect(root!.kind).toBe("tpl");

    // Render edges for Header and Footer
    const renderEdges = result.edges.filter((e) => e.kind === "render");
    expect(renderEdges).toContainEqual({
      kind: "render",
      from: "tpl:MyPage",
      to: "type:Header",
    });
    expect(renderEdges).toContainEqual({
      kind: "render",
      from: "tpl:MyPage",
      to: "type:Footer",
    });

    // Root description mentions rendered components
    const rootDesc = result.descriptions.get("tpl:MyPage");
    expect(rootDesc).toBeDefined();
    expect(rootDesc).toContain("Header");
    expect(rootDesc).toContain("Footer");
  });

  it("creates tpl-branch symbols for v-if/v-else", () => {
    const nodes: TemplateNode[] = [
      makeNode("div", {
        children: [
          makeNode("Spinner", {
            line: 2,
            directives: [{ kind: "if", expression: "loading" }],
          }),
          makeNode("Content", {
            line: 3,
            directives: [{ kind: "else", expression: "" }],
          }),
        ],
      }),
    ];

    const result = walkTemplate("LoadingView", nodes);

    const ifSym = result.symbols.find(
      (s) => s.kind === "tpl-branch" && s.id.includes("if_loading"),
    );
    expect(ifSym).toBeDefined();

    const elseSym = result.symbols.find(
      (s) => s.kind === "tpl-branch" && s.id.includes("else"),
    );
    expect(elseSym).toBeDefined();

    // Description for the if branch
    const ifDesc = result.descriptions.get(ifSym!.id);
    expect(ifDesc).toBeDefined();
    expect(ifDesc).toContain("loading");
    expect(ifDesc).toContain("Spinner");
  });

  it("creates tpl-loop symbols for v-for", () => {
    const nodes: TemplateNode[] = [
      makeNode("div", {
        children: [
          makeNode("Card", {
            line: 2,
            directives: [{ kind: "for", expression: "item in items" }],
          }),
        ],
      }),
    ];

    const result = walkTemplate("CardList", nodes);

    const loopSym = result.symbols.find(
      (s) => s.kind === "tpl-loop" && s.id.includes("for_items"),
    );
    expect(loopSym).toBeDefined();

    const loopDesc = result.descriptions.get(loopSym!.id);
    expect(loopDesc).toBeDefined();
    expect(loopDesc).toContain("items");
    expect(loopDesc).toContain("Card");
  });

  it("creates tpl-slot symbols for slot elements", () => {
    const nodes: TemplateNode[] = [
      makeNode("div", {
        children: [makeNode("slot", { isComponent: false, line: 2 })],
      }),
    ];

    const result = walkTemplate("Wrapper", nodes);

    const slotSym = result.symbols.find((s) => s.kind === "tpl-slot");
    expect(slotSym).toBeDefined();
    expect(slotSym!.id).toContain("default");

    const slotEdge = result.edges.find((e) => e.kind === "slot");
    expect(slotEdge).toBeDefined();
    expect(slotEdge!.from).toBe("tpl:Wrapper");
  });

  it("creates named slot symbols", () => {
    const nodes: TemplateNode[] = [
      makeNode("div", {
        children: [
          makeNode("slot", {
            isComponent: false,
            line: 2,
            slotName: "header",
          }),
          makeNode("slot", { isComponent: false, line: 3 }),
        ],
      }),
    ];

    const result = walkTemplate("Layout", nodes);

    const slotSymbols = result.symbols.filter((s) => s.kind === "tpl-slot");
    expect(slotSymbols).toHaveLength(2);

    const headerSlot = slotSymbols.find((s) => s.id.includes("header"));
    expect(headerSlot).toBeDefined();

    const defaultSlot = slotSymbols.find((s) => s.id.includes("default"));
    expect(defaultSlot).toBeDefined();

    // Both should have slot edges
    const slotEdges = result.edges.filter((e) => e.kind === "slot");
    expect(slotEdges).toHaveLength(2);
  });

  it("handles v-show as tpl-branch", () => {
    const nodes: TemplateNode[] = [
      makeNode("div", {
        children: [
          makeNode("Tooltip", {
            line: 2,
            directives: [{ kind: "show", expression: "isVisible" }],
          }),
        ],
      }),
    ];

    const result = walkTemplate("HoverCard", nodes);

    const showSym = result.symbols.find(
      (s) => s.kind === "tpl-branch" && s.id.includes("show_isVisible"),
    );
    expect(showSym).toBeDefined();

    const desc = result.descriptions.get(showSym!.id);
    expect(desc).toBeDefined();
    expect(desc).toContain("isVisible");
  });

  it("handles nested branch inside loop", () => {
    const nodes: TemplateNode[] = [
      makeNode("div", {
        children: [
          makeNode("div", {
            line: 2,
            directives: [{ kind: "for", expression: "user in users" }],
            children: [
              makeNode("Badge", {
                line: 3,
                directives: [{ kind: "if", expression: "user.isAdmin" }],
              }),
            ],
          }),
        ],
      }),
    ];

    const result = walkTemplate("UserList", nodes);

    const loopSym = result.symbols.find(
      (s) => s.kind === "tpl-loop" && s.id.includes("for_users"),
    );
    expect(loopSym).toBeDefined();

    const branchSym = result.symbols.find(
      (s) => s.kind === "tpl-branch" && s.id.includes("if_user.isAdmin"),
    );
    expect(branchSym).toBeDefined();

    // The branch should be a child of the loop
    expect(branchSym!.parentId).toBe(loopSym!.id);

    // Render edge for Badge
    const renderEdge = result.edges.find(
      (e) => e.kind === "render" && e.to === "type:Badge",
    );
    expect(renderEdge).toBeDefined();

    // Descriptions
    const loopDesc = result.descriptions.get(loopSym!.id);
    expect(loopDesc).toContain("users");

    const branchDesc = result.descriptions.get(branchSym!.id);
    expect(branchDesc).toContain("user.isAdmin");
    expect(branchDesc).toContain("Badge");
  });
});

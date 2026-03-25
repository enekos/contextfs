import type { Node as SyntaxNode } from "web-tree-sitter";
import { TypeScriptDescriber } from "./typescriptDescriber";
import type { FileGraphResult } from "./languageDescriber";
import { sortSymbols, sortEdges } from "./languageDescriber";
import type { TemplateNode, TemplateDirective } from "./templateWalker";
import { walkTemplate } from "./templateWalker";

export class TsxDescriber extends TypeScriptDescriber {
  override readonly languageId = "tsx";
  override readonly extensions: ReadonlySet<string> = new Set([".tsx", ".jsx"]);

  override extractFileGraph(filePath: string, sourceText: string): FileGraphResult {
    const base = super.extractFileGraph(filePath, sourceText);

    // Re-parse to walk JSX (we need the tree again)
    const tree = this["parseSync"]("tsx", sourceText);
    const root = tree.rootNode;

    try {
      const allTemplateSymbols: FileGraphResult["symbols"] = [];
      const allTemplateEdges: FileGraphResult["edges"] = [];
      const allTemplateDescriptions = new Map<string, string>();

      // Find all function declarations that return JSX
      for (const child of root.namedChildren) {
        const node = this.unwrapExport(child);
        if (!node) continue;

        if (node.type === "function_declaration") {
          const fnName = node.childForFieldName("name")?.text ?? `anonymous_fn_${node.startPosition.row + 1}`;
          const templateNodes = this.extractJsxTemplateNodes(node);
          if (templateNodes.length === 0) continue;
          const result = walkTemplate(fnName, templateNodes);
          allTemplateSymbols.push(...result.symbols);
          allTemplateEdges.push(...result.edges);
          for (const [k, v] of result.descriptions) allTemplateDescriptions.set(k, v);
        }

        if (node.type === "lexical_declaration" || node.type === "variable_declaration") {
          for (const declarator of node.namedChildren) {
            if (declarator.type !== "variable_declarator") continue;
            const varName = declarator.childForFieldName("name")?.text ?? "";
            const value = declarator.childForFieldName("value");
            if (!value) continue;
            if (value.type !== "arrow_function" && value.type !== "function") continue;

            const templateNodes = this.extractJsxTemplateNodes(value);
            if (templateNodes.length === 0) continue;
            const result = walkTemplate(varName, templateNodes);
            allTemplateSymbols.push(...result.symbols);
            allTemplateEdges.push(...result.edges);
            for (const [k, v] of result.descriptions) allTemplateDescriptions.set(k, v);
          }
        }
      }

      if (allTemplateSymbols.length === 0) return base;

      const mergedDescriptions = new Map(base.symbolDescriptions);
      for (const [k, v] of allTemplateDescriptions) mergedDescriptions.set(k, v);

      return {
        symbols: sortSymbols([...base.symbols, ...allTemplateSymbols]),
        edges: sortEdges([...base.edges, ...allTemplateEdges]),
        imports: base.imports,
        symbolDescriptions: mergedDescriptions,
        fileSummary: base.fileSummary,
      };
    } finally {
      tree.delete();
    }
  }

  private unwrapExport(node: SyntaxNode): SyntaxNode | null {
    if (node.type === "export_statement") {
      return node.childForFieldName("declaration") ?? node.namedChildren.find(n =>
        n.type === "function_declaration" || n.type === "lexical_declaration" || n.type === "variable_declaration"
      ) ?? null;
    }
    return node;
  }

  private extractJsxTemplateNodes(fnNode: SyntaxNode): TemplateNode[] {
    const jsxElements: SyntaxNode[] = [];

    // Look for return statements containing JSX
    this.walkForReturnJsx(fnNode, jsxElements);

    // For arrow functions with expression body (implicit return)
    if (fnNode.type === "arrow_function") {
      const body = fnNode.childForFieldName("body");
      if (body && body.type !== "statement_block") {
        const jsx = this.findJsxRoot(body);
        if (jsx) jsxElements.push(jsx);
      }
    }

    const templateNodes: TemplateNode[] = [];
    for (const jsx of jsxElements) {
      templateNodes.push(...this.jsxToTemplateNodes(jsx));
    }
    return templateNodes;
  }

  private walkForReturnJsx(node: SyntaxNode, results: SyntaxNode[]): void {
    for (const child of node.namedChildren) {
      if (child.type === "return_statement") {
        for (const rc of child.namedChildren) {
          const jsx = this.findJsxRoot(rc);
          if (jsx) results.push(jsx);
        }
      } else {
        this.walkForReturnJsx(child, results);
      }
    }
  }

  private findJsxRoot(node: SyntaxNode): SyntaxNode | null {
    if (
      node.type === "jsx_element" ||
      node.type === "jsx_self_closing_element" ||
      node.type === "jsx_fragment"
    ) {
      return node;
    }
    if (node.type === "parenthesized_expression") {
      const inner = node.namedChildren[0];
      return inner ? this.findJsxRoot(inner) : null;
    }
    return null;
  }

  private jsxToTemplateNodes(node: SyntaxNode): TemplateNode[] {
    if (node.type === "jsx_element") {
      return [this.convertJsxElement(node)];
    }
    if (node.type === "jsx_self_closing_element") {
      return [this.convertJsxSelfClosing(node)];
    }
    if (node.type === "jsx_fragment") {
      return this.convertJsxChildren(node.namedChildren);
    }
    return [];
  }

  private convertJsxElement(node: SyntaxNode): TemplateNode {
    const openingElement = node.namedChildren.find((c) => c.type === "jsx_opening_element");
    const tagName = this.getJsxTagName(openingElement);
    const children = this.convertJsxChildren(
      node.namedChildren.filter(
        (c) => c.type !== "jsx_opening_element" && c.type !== "jsx_closing_element",
      ),
    );
    return {
      tag: tagName,
      isComponent: this.isComponentTag(tagName),
      directives: [],
      children,
      line: node.startPosition.row + 1,
    };
  }

  private convertJsxSelfClosing(node: SyntaxNode): TemplateNode {
    const tagName = this.getJsxTagName(node);
    return {
      tag: tagName,
      isComponent: this.isComponentTag(tagName),
      directives: [],
      children: [],
      line: node.startPosition.row + 1,
    };
  }

  private getJsxTagName(node: SyntaxNode | undefined): string {
    if (!node) return "unknown";
    // Look for identifier or member_expression child that's the tag name
    for (const child of node.namedChildren) {
      if (
        child.type === "identifier" ||
        child.type === "member_expression" ||
        child.type === "nested_identifier"
      ) {
        return child.text;
      }
    }
    return "unknown";
  }

  private convertJsxChildren(children: SyntaxNode[]): TemplateNode[] {
    const result: TemplateNode[] = [];
    for (const child of children) {
      if (child.type === "jsx_element") {
        result.push(this.convertJsxElement(child));
      } else if (child.type === "jsx_self_closing_element") {
        result.push(this.convertJsxSelfClosing(child));
      } else if (child.type === "jsx_fragment") {
        result.push(...this.convertJsxChildren(child.namedChildren));
      } else if (child.type === "jsx_expression") {
        result.push(...this.convertJsxExpression(child));
      }
    }
    return result;
  }

  private convertJsxExpression(expr: SyntaxNode): TemplateNode[] {
    const inner = expr.namedChildren[0];
    if (!inner) return [];

    // Pattern: {condition && <X/>} → if directive
    if (inner.type === "binary_expression") {
      const op = inner.childForFieldName("operator")?.text;
      if (op === "&&") {
        const condition = inner.childForFieldName("left")?.text ?? "";
        const right = inner.childForFieldName("right");
        if (right) {
          const childNodes = this.extractJsxFromExpression(right);
          if (childNodes.length > 0) {
            const directives: TemplateDirective[] = [{ kind: "if", expression: condition }];
            if (childNodes.length === 1) {
              return [{
                ...childNodes[0],
                directives: [...directives, ...childNodes[0].directives],
              }];
            }
            return [{
              tag: "div",
              isComponent: false,
              directives,
              children: childNodes,
              line: expr.startPosition.row + 1,
            }];
          }
        }
      }
    }

    // Pattern: {condition ? <X/> : <Y/>} → if/else directives
    if (inner.type === "ternary_expression") {
      const condition = inner.childForFieldName("condition")?.text ?? "";
      const consequence = inner.childForFieldName("consequence");
      const alternative = inner.childForFieldName("alternative");
      const result: TemplateNode[] = [];

      if (consequence) {
        const trueNodes = this.extractJsxFromExpression(consequence);
        if (trueNodes.length > 0) {
          const ifDirective: TemplateDirective = { kind: "if", expression: condition };
          if (trueNodes.length === 1) {
            result.push({
              ...trueNodes[0],
              directives: [ifDirective, ...trueNodes[0].directives],
            });
          } else {
            result.push({
              tag: "div", isComponent: false,
              directives: [ifDirective], children: trueNodes,
              line: consequence.startPosition.row + 1,
            });
          }
        }
      }

      if (alternative) {
        const falseNodes = this.extractJsxFromExpression(alternative);
        if (falseNodes.length > 0) {
          const elseDirective: TemplateDirective = { kind: "else", expression: "" };
          if (falseNodes.length === 1) {
            result.push({
              ...falseNodes[0],
              directives: [elseDirective, ...falseNodes[0].directives],
            });
          } else {
            result.push({
              tag: "div", isComponent: false,
              directives: [elseDirective], children: falseNodes,
              line: alternative.startPosition.row + 1,
            });
          }
        }
      }

      return result;
    }

    // Pattern: {items.map(item => <X/>)} → for directive
    if (inner.type === "call_expression") {
      return this.tryExtractMapExpression(inner, expr.startPosition.row + 1);
    }

    return [];
  }

  private tryExtractMapExpression(callExpr: SyntaxNode, fallbackLine: number): TemplateNode[] {
    const fn = callExpr.childForFieldName("function");
    if (!fn || fn.type !== "member_expression") return [];

    const property = fn.childForFieldName("property");
    if (property?.text !== "map") return [];

    const object = fn.childForFieldName("object");
    const iterableText = object?.text ?? "";

    const args = callExpr.childForFieldName("arguments");
    if (!args || args.namedChildren.length === 0) return [];

    const callback = args.namedChildren[0];
    let iteratorName = "item";
    let callbackBody: SyntaxNode | null = null;

    if (callback.type === "arrow_function" || callback.type === "function") {
      const params = callback.childForFieldName("parameters");
      if (params && params.namedChildren.length > 0) {
        const firstParam = params.namedChildren[0];
        iteratorName = firstParam.childForFieldName("name")?.text
          ?? firstParam.childForFieldName("pattern")?.text
          ?? firstParam.text;
      }
      callbackBody = callback.childForFieldName("body");
    }

    if (!callbackBody) return [];

    const childNodes = this.extractJsxFromExpression(callbackBody);
    if (childNodes.length === 0) return [];

    const forDirective: TemplateDirective = {
      kind: "for",
      expression: `${iteratorName} in ${iterableText}`,
    };

    if (childNodes.length === 1) {
      return [{
        ...childNodes[0],
        directives: [forDirective, ...childNodes[0].directives],
      }];
    }

    return [{
      tag: "div",
      isComponent: false,
      directives: [forDirective],
      children: childNodes,
      line: fallbackLine,
    }];
  }

  private extractJsxFromExpression(node: SyntaxNode): TemplateNode[] {
    if (node.type === "jsx_element") return [this.convertJsxElement(node)];
    if (node.type === "jsx_self_closing_element") return [this.convertJsxSelfClosing(node)];
    if (node.type === "jsx_fragment") return this.convertJsxChildren(node.namedChildren);
    if (node.type === "parenthesized_expression") {
      const inner = node.namedChildren[0];
      return inner ? this.extractJsxFromExpression(inner) : [];
    }
    // Handle block body (arrow function with block)
    if (node.type === "statement_block") {
      for (const stmt of node.namedChildren) {
        if (stmt.type === "return_statement") {
          for (const rc of stmt.namedChildren) {
            const result = this.extractJsxFromExpression(rc);
            if (result.length > 0) return result;
          }
        }
      }
      return [];
    }
    // Handle && and ternary inside map
    if (node.type === "binary_expression") {
      const op = node.childForFieldName("operator")?.text;
      if (op === "&&") {
        const condition = node.childForFieldName("left")?.text ?? "";
        const right = node.childForFieldName("right");
        if (right) {
          const childNodes = this.extractJsxFromExpression(right);
          if (childNodes.length > 0) {
            const directives: TemplateDirective[] = [{ kind: "if", expression: condition }];
            if (childNodes.length === 1) {
              return [{ ...childNodes[0], directives: [...directives, ...childNodes[0].directives] }];
            }
            return [{
              tag: "div", isComponent: false, directives,
              children: childNodes, line: node.startPosition.row + 1,
            }];
          }
        }
      }
    }
    if (node.type === "ternary_expression") {
      const condition = node.childForFieldName("condition")?.text ?? "";
      const trueNodes = this.extractJsxFromExpression(node.childForFieldName("consequence")!);
      const falseNodes = this.extractJsxFromExpression(node.childForFieldName("alternative")!);
      const result: TemplateNode[] = [];
      if (trueNodes.length > 0) {
        if (trueNodes.length === 1) {
          result.push({ ...trueNodes[0], directives: [{ kind: "if", expression: condition }, ...trueNodes[0].directives] });
        } else {
          result.push({ tag: "div", isComponent: false, directives: [{ kind: "if", expression: condition }], children: trueNodes, line: node.startPosition.row + 1 });
        }
      }
      if (falseNodes.length > 0) {
        if (falseNodes.length === 1) {
          result.push({ ...falseNodes[0], directives: [{ kind: "else", expression: "" }, ...falseNodes[0].directives] });
        } else {
          result.push({ tag: "div", isComponent: false, directives: [{ kind: "else", expression: "" }], children: falseNodes, line: node.startPosition.row + 1 });
        }
      }
      return result;
    }
    // Handle .map() calls
    if (node.type === "call_expression") {
      return this.tryExtractMapExpression(node, node.startPosition.row + 1);
    }
    return [];
  }

  private isComponentTag(tag: string): boolean {
    return /^[A-Z]/.test(tag) || tag.includes(".");
  }
}

import {
  Node,
  Project,
  SyntaxKind,
  type JsxElement,
  type JsxSelfClosingElement,
  type JsxChild,
  type JsxExpression,
} from "ts-morph";
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

    // Create a new ts-morph project to inspect JSX
    const project = new Project({
      compilerOptions: { allowJs: true, jsx: 2 /* React */ },
      useInMemoryFileSystem: true,
    });
    const sourceFile = project.createSourceFile(filePath, sourceText);

    const allTemplateSymbols: FileGraphResult["symbols"] = [];
    const allTemplateEdges: FileGraphResult["edges"] = [];
    const allTemplateDescriptions = new Map<string, string>();

    // Find all function declarations that return JSX
    for (const fn of sourceFile.getFunctions()) {
      const fnName = fn.getName() ?? `anonymous_fn_${fn.getStartLineNumber()}`;
      const templateNodes = this.extractJsxTemplateNodes(fn);
      if (templateNodes.length === 0) continue;

      const result = walkTemplate(fnName, templateNodes);
      allTemplateSymbols.push(...result.symbols);
      allTemplateEdges.push(...result.edges);
      for (const [k, v] of result.descriptions) allTemplateDescriptions.set(k, v);
    }

    // Find arrow function variable declarations that return JSX
    for (const decl of sourceFile.getVariableDeclarations()) {
      const initializer = decl.getInitializer();
      if (!initializer) continue;
      if (!Node.isArrowFunction(initializer) && !Node.isFunctionExpression(initializer)) continue;

      const varName = decl.getName();
      const templateNodes = this.extractJsxTemplateNodes(initializer);
      if (templateNodes.length === 0) continue;

      const result = walkTemplate(varName, templateNodes);
      allTemplateSymbols.push(...result.symbols);
      allTemplateEdges.push(...result.edges);
      for (const [k, v] of result.descriptions) allTemplateDescriptions.set(k, v);
    }

    if (allTemplateSymbols.length === 0) return base;

    // Merge base descriptions with template descriptions
    const mergedDescriptions = new Map(base.symbolDescriptions);
    for (const [k, v] of allTemplateDescriptions) mergedDescriptions.set(k, v);

    return {
      symbols: sortSymbols([...base.symbols, ...allTemplateSymbols]),
      edges: sortEdges([...base.edges, ...allTemplateEdges]),
      imports: base.imports,
      symbolDescriptions: mergedDescriptions,
      fileSummary: base.fileSummary,
    };
  }

  private extractJsxTemplateNodes(fnNode: Node): TemplateNode[] {
    // Find all top-level JSX returns in the function
    const jsxElements: Node[] = [];

    // Look for return statements containing JSX
    for (const ret of fnNode.getDescendantsOfKind(SyntaxKind.ReturnStatement)) {
      const expr = ret.getExpression();
      if (!expr) continue;
      const jsx = this.findJsxRoot(expr);
      if (jsx) jsxElements.push(jsx);
    }

    // For arrow functions with expression body (implicit return)
    if (Node.isArrowFunction(fnNode)) {
      const body = fnNode.getBody();
      if (body && !Node.isBlock(body)) {
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

  private findJsxRoot(node: Node): Node | null {
    if (Node.isJsxElement(node) || Node.isJsxSelfClosingElement(node) || Node.isJsxFragment(node)) {
      return node;
    }
    if (Node.isParenthesizedExpression(node)) {
      return this.findJsxRoot(node.getExpression());
    }
    return null;
  }

  private jsxToTemplateNodes(node: Node): TemplateNode[] {
    if (Node.isJsxElement(node)) {
      return [this.convertJsxElement(node)];
    }
    if (Node.isJsxSelfClosingElement(node)) {
      return [this.convertJsxSelfClosing(node)];
    }
    if (Node.isJsxFragment(node)) {
      return this.convertJsxChildren(node.getJsxChildren());
    }
    return [];
  }

  private convertJsxElement(node: JsxElement): TemplateNode {
    const tagName = node.getOpeningElement().getTagNameNode().getText();
    const children = this.convertJsxChildren(node.getJsxChildren());
    return {
      tag: tagName,
      isComponent: this.isComponentTag(tagName),
      directives: [],
      children,
      line: node.getStartLineNumber(),
    };
  }

  private convertJsxSelfClosing(node: JsxSelfClosingElement): TemplateNode {
    const tagName = node.getTagNameNode().getText();
    return {
      tag: tagName,
      isComponent: this.isComponentTag(tagName),
      directives: [],
      children: [],
      line: node.getStartLineNumber(),
    };
  }

  private convertJsxChildren(children: JsxChild[]): TemplateNode[] {
    const result: TemplateNode[] = [];
    for (const child of children) {
      if (Node.isJsxElement(child)) {
        result.push(this.convertJsxElement(child));
      } else if (Node.isJsxSelfClosingElement(child)) {
        result.push(this.convertJsxSelfClosing(child));
      } else if (Node.isJsxFragment(child)) {
        result.push(...this.convertJsxChildren(child.getJsxChildren()));
      } else if (Node.isJsxExpression(child)) {
        result.push(...this.convertJsxExpression(child));
      }
    }
    return result;
  }

  private convertJsxExpression(expr: JsxExpression): TemplateNode[] {
    const innerExpr = expr.getExpression();
    if (!innerExpr) return [];

    // Pattern: {condition && <X/>} → if directive
    if (Node.isBinaryExpression(innerExpr)) {
      const op = innerExpr.getOperatorToken().getText();
      if (op === "&&") {
        const condition = innerExpr.getLeft().getText();
        const right = innerExpr.getRight();
        const childNodes = this.extractJsxFromExpression(right);
        if (childNodes.length > 0) {
          const directives: TemplateDirective[] = [{ kind: "if", expression: condition }];
          // If the right side is a single JSX element, wrap it
          if (childNodes.length === 1) {
            return [{
              ...childNodes[0],
              directives: [...directives, ...childNodes[0].directives],
            }];
          }
          // Multiple children — wrap in a synthetic div
          return [{
            tag: "div",
            isComponent: false,
            directives,
            children: childNodes,
            line: expr.getStartLineNumber(),
          }];
        }
      }
    }

    // Pattern: {condition ? <X/> : <Y/>} → if/else directives
    if (Node.isConditionalExpression(innerExpr)) {
      const condition = innerExpr.getCondition().getText();
      const whenTrue = innerExpr.getWhenTrue();
      const whenFalse = innerExpr.getWhenFalse();
      const trueNodes = this.extractJsxFromExpression(whenTrue);
      const falseNodes = this.extractJsxFromExpression(whenFalse);
      const result: TemplateNode[] = [];

      if (trueNodes.length > 0) {
        const ifDirective: TemplateDirective = { kind: "if", expression: condition };
        if (trueNodes.length === 1) {
          result.push({
            ...trueNodes[0],
            directives: [ifDirective, ...trueNodes[0].directives],
          });
        } else {
          result.push({
            tag: "div",
            isComponent: false,
            directives: [ifDirective],
            children: trueNodes,
            line: whenTrue.getStartLineNumber(),
          });
        }
      }

      if (falseNodes.length > 0) {
        const elseDirective: TemplateDirective = { kind: "else", expression: "" };
        if (falseNodes.length === 1) {
          result.push({
            ...falseNodes[0],
            directives: [elseDirective, ...falseNodes[0].directives],
          });
        } else {
          result.push({
            tag: "div",
            isComponent: false,
            directives: [elseDirective],
            children: falseNodes,
            line: whenFalse.getStartLineNumber(),
          });
        }
      }

      return result;
    }

    // Pattern: {items.map(item => <X/>)} → for directive
    if (Node.isCallExpression(innerExpr)) {
      return this.tryExtractMapExpression(innerExpr, expr.getStartLineNumber());
    }

    return [];
  }

  private tryExtractMapExpression(callExpr: Node, fallbackLine: number): TemplateNode[] {
    if (!Node.isCallExpression(callExpr)) return [];

    const expression = callExpr.getExpression();
    if (!Node.isPropertyAccessExpression(expression)) return [];

    const methodName = expression.getName();
    if (methodName !== "map") return [];

    const iterableText = expression.getExpression().getText();
    const args = callExpr.getArguments();
    if (args.length === 0) return [];

    const callback = args[0];
    let iteratorName = "item";
    let callbackBody: Node | null = null;

    if (Node.isArrowFunction(callback)) {
      const params = callback.getParameters();
      if (params.length > 0) iteratorName = params[0].getName();
      callbackBody = callback.getBody();
    } else if (Node.isFunctionExpression(callback)) {
      const params = callback.getParameters();
      if (params.length > 0) iteratorName = params[0].getName();
      callbackBody = callback.getBody();
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

  private extractJsxFromExpression(node: Node): TemplateNode[] {
    if (Node.isJsxElement(node)) {
      return [this.convertJsxElement(node)];
    }
    if (Node.isJsxSelfClosingElement(node)) {
      return [this.convertJsxSelfClosing(node)];
    }
    if (Node.isJsxFragment(node)) {
      return this.convertJsxChildren(node.getJsxChildren());
    }
    if (Node.isParenthesizedExpression(node)) {
      return this.extractJsxFromExpression(node.getExpression());
    }
    // Handle block body (arrow function with block)
    if (Node.isBlock(node)) {
      for (const stmt of node.getStatements()) {
        if (Node.isReturnStatement(stmt)) {
          const expr = stmt.getExpression();
          if (expr) return this.extractJsxFromExpression(expr);
        }
      }
      return [];
    }
    // Handle expression with && or ternary inside map
    if (Node.isBinaryExpression(node)) {
      const op = node.getOperatorToken().getText();
      if (op === "&&") {
        const condition = node.getLeft().getText();
        const right = node.getRight();
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
            line: node.getStartLineNumber(),
          }];
        }
      }
    }
    if (Node.isConditionalExpression(node)) {
      const condition = node.getCondition().getText();
      const trueNodes = this.extractJsxFromExpression(node.getWhenTrue());
      const falseNodes = this.extractJsxFromExpression(node.getWhenFalse());
      const result: TemplateNode[] = [];
      if (trueNodes.length > 0) {
        if (trueNodes.length === 1) {
          result.push({ ...trueNodes[0], directives: [{ kind: "if", expression: condition }, ...trueNodes[0].directives] });
        } else {
          result.push({ tag: "div", isComponent: false, directives: [{ kind: "if", expression: condition }], children: trueNodes, line: node.getStartLineNumber() });
        }
      }
      if (falseNodes.length > 0) {
        if (falseNodes.length === 1) {
          result.push({ ...falseNodes[0], directives: [{ kind: "else", expression: "" }, ...falseNodes[0].directives] });
        } else {
          result.push({ tag: "div", isComponent: false, directives: [{ kind: "else", expression: "" }], children: falseNodes, line: node.getStartLineNumber() });
        }
      }
      return result;
    }
    // Handle .map() calls
    if (Node.isCallExpression(node)) {
      return this.tryExtractMapExpression(node, node.getStartLineNumber());
    }
    return [];
  }

  private isComponentTag(tag: string): boolean {
    // Component if starts with uppercase or contains a dot
    return /^[A-Z]/.test(tag) || tag.includes(".");
  }
}

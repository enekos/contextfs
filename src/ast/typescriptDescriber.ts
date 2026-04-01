import type { Node as SyntaxNode, Tree } from "web-tree-sitter";
import { Parser } from "web-tree-sitter";
import { createHash } from "crypto";
import type {
  LanguageDescriber,
  FileGraphResult,
  LogicSymbol,
  LogicEdge,
  LogicSymbolKind,
  ComplexityBucket,
} from "./languageDescriber";
import { sortSymbols, sortEdges } from "./languageDescriber";
import { describeStatements } from "./nlDescriber";
import { ParserPool, type SupportedLanguage } from "./parserPool";

export interface RawLogicGraph {
  symbols: LogicSymbol[];
  edges: LogicEdge[];
  imports: string[];
  callableNodes: CallableNodeRef[];
}

export interface CallableNodeRef {
  symbolId: string;
  className: string | null;
  node: SyntaxNode;
}

/** Map file extension to tree-sitter language id. */
function extToLang(filePath: string): SupportedLanguage {
  if (filePath.endsWith(".tsx") || filePath.endsWith(".jsx")) return "tsx";
  if (filePath.endsWith(".js") || filePath.endsWith(".mjs") || filePath.endsWith(".cjs")) return "javascript";
  return "typescript";
}

export class TypeScriptDescriber implements LanguageDescriber {
  readonly languageId: string = "typescript";
  readonly extensions: ReadonlySet<string> = new Set([".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"]);

  extractFileGraph(filePath: string, sourceText: string): FileGraphResult {
    const lang = extToLang(filePath);
    const tree = this.parseSync(lang, sourceText);
    const root = tree.rootNode;

    try {
      const rawGraph = this.extractRawLogicGraph(root);

      // Generate per-symbol NL descriptions
      const symbolDescriptions = new Map<string, string>();

      for (const callable of rawGraph.callableNodes) {
        symbolDescriptions.set(callable.symbolId, describeStatements(callable.node));
      }

      // Generate class summaries
      for (const child of root.namedChildren) {
        if (child.type === "class_declaration") {
          const className = child.childForFieldName("name")?.text ?? `anonymous_class_${child.startPosition.row + 1}`;
          const classId = `cls:${className}`;
          const superClass = this.getExtendsClause(child);
          const extendsStr = superClass ? ` extends ${superClass}` : "";
          const methodNames = this.getClassMethods(child).map((m) => m.childForFieldName("name")?.text ?? "");
          symbolDescriptions.set(
            classId,
            `Class \`${className}\`${extendsStr} with methods: ${methodNames.join(", ")}`,
          );
        }
      }

      // Generate file summary
      let fileSummary: string;
      if (rawGraph.symbols.length === 0) {
        fileSummary = "Empty or declaration-free source file.";
      } else {
        const exportedSymbols = rawGraph.symbols.filter((s) => s.exported);
        const names = exportedSymbols.map((s) => s.name);
        const kindCounts = new Map<string, number>();
        for (const s of exportedSymbols) {
          kindCounts.set(s.kind, (kindCounts.get(s.kind) ?? 0) + 1);
        }
        const kindSummary = Array.from(kindCounts.entries())
          .map(([kind, count]) => `${count} ${kind}${count > 1 ? "s" : ""}`)
          .join(", ");
        fileSummary = `File containing ${exportedSymbols.length} exported symbols (${kindSummary}): ${names.join(", ")}.`;
      }

      return {
        symbols: rawGraph.symbols,
        edges: rawGraph.edges,
        imports: rawGraph.imports,
        symbolDescriptions,
        fileSummary,
      };
    } finally {
      tree.delete();
    }
  }

  /** Synchronous parse using the pre-initialized parser pool. */
  protected parseSync(lang: SupportedLanguage, sourceText: string): Tree {
    // web-tree-sitter requires async init, but parse itself is sync once language is loaded.
    // We assume ParserPool.init() has been called at daemon startup.
    // For direct usage, we use a cached parser approach.
    const parser = this.getParserSync(lang);
    const tree = parser.parse(sourceText);
    if (!tree) throw new Error(`Failed to parse ${lang} source`);
    return tree;
  }

  private parserCache = new Map<SupportedLanguage, InstanceType<typeof Parser>>();

  private getParserSync(lang: SupportedLanguage): InstanceType<typeof Parser> {
    const cached = this.parserCache.get(lang);
    if (cached) return cached;
    throw new Error(
      `Parser for ${lang} not initialized. Call TypeScriptDescriber.initParsers() first.`,
    );
  }

  /** Initialize parsers for all needed languages. Must be called once before extractFileGraph. */
  async initParsers(): Promise<void> {
    await ParserPool.init();
    for (const lang of ["typescript", "tsx", "javascript"] as SupportedLanguage[]) {
      const parser = await ParserPool.getParser(lang);
      this.parserCache.set(lang, parser);
    }
  }

  /** Cleanup parsers when done. */
  deleteParsers(): void {
    for (const parser of this.parserCache.values()) {
      parser.delete();
    }
    this.parserCache.clear();
  }

  protected extractRawLogicGraph(root: SyntaxNode): RawLogicGraph {
    const symbols: LogicSymbol[] = [];
    const edgesMap = new Map<string, LogicEdge>();
    const nameToSymbolIds = new Map<string, string[]>();
    const symbolById = new Map<string, LogicSymbol>();
    const methodByClassAndName = new Map<string, string>();
    const methodIdsByName = new Map<string, string[]>();
    const callableNodes: CallableNodeRef[] = [];
    const moduleVariableByName = new Map<string, string>();

    const pushSymbol = (symbol: LogicSymbol) => {
      symbols.push(symbol);
      symbolById.set(symbol.id, symbol);
      const existing = nameToSymbolIds.get(symbol.name) ?? [];
      existing.push(symbol.id);
      nameToSymbolIds.set(symbol.name, existing);
    };

    const addEdge = (edge: LogicEdge) => {
      const key = `${edge.kind}|${edge.from}|${edge.to}`;
      if (!edgesMap.has(key)) {
        edgesMap.set(key, edge);
      }
    };

    // Process top-level declarations
    for (const child of root.namedChildren) {
      // Handle export wrappers
      const isExport = child.type === "export_statement";
      const declNode = isExport ? (child.childForFieldName("declaration") ?? child.namedChildren.find(n => this.isDeclarationType(n.type))) : child;
      if (!declNode) {
        // export { ... } from '...' style
        if (isExport) {
          const source = child.childForFieldName("source");
          if (source) {
            const moduleName = stripQuotes(source.text);
            if (moduleName) addEdge({ kind: "import", from: "file", to: `module:${moduleName}` });
          }
        }
        continue;
      }

      switch (declNode.type) {
        case "import_statement": {
          const source = declNode.childForFieldName("source");
          const moduleName = source ? stripQuotes(source.text) : "";
          if (moduleName) addEdge({ kind: "import", from: "file", to: `module:${moduleName}` });
          break;
        }

        case "class_declaration": {
          const className = declNode.childForFieldName("name")?.text ?? `anonymous_class_${declNode.startPosition.row + 1}`;
          const classId = `cls:${className}`;
          pushSymbol(this.makeSymbol(classId, "cls", className, isExport, null, [], declNode));

          const superClass = this.getExtendsClause(declNode);
          if (superClass) {
            addEdge({ kind: "extends", from: classId, to: `type:${superClass}` });
          }
          for (const impl of this.getImplementsClause(declNode)) {
            addEdge({ kind: "implements", from: classId, to: `type:${impl}` });
          }

          for (const method of this.getClassMethods(declNode)) {
            const methodName = method.childForFieldName("name")?.text ?? "";
            const methodId = `mtd:${className}.${methodName}`;
            const params = this.extractParams(method);
            pushSymbol(this.makeSymbol(methodId, "mtd", methodName, isExport, classId, params, method));
            methodByClassAndName.set(`${className}.${methodName}`, methodId);
            const existingMethodIds = methodIdsByName.get(methodName) ?? [];
            existingMethodIds.push(methodId);
            methodIdsByName.set(methodName, existingMethodIds);
            callableNodes.push({ symbolId: methodId, className, node: method });
          }
          break;
        }

        case "function_declaration": {
          const fnName = declNode.childForFieldName("name")?.text ?? `anonymous_fn_${declNode.startPosition.row + 1}`;
          const fnId = `fn:${fnName}`;
          const params = this.extractParams(declNode);
          pushSymbol(this.makeSymbol(fnId, "fn", fnName, isExport, null, params, declNode));
          callableNodes.push({ symbolId: fnId, className: null, node: declNode });
          break;
        }

        case "lexical_declaration":
        case "variable_declaration": {
          for (const declarator of declNode.namedChildren) {
            if (declarator.type !== "variable_declarator") continue;
            const variableName = declarator.childForFieldName("name")?.text ?? "";
            const symbolId = `var:${variableName}`;
            pushSymbol(this.makeSymbol(symbolId, "var", variableName, isExport, null, [], declarator));
            moduleVariableByName.set(variableName, symbolId);
          }
          break;
        }

        case "interface_declaration": {
          const name = declNode.childForFieldName("name")?.text ?? "";
          const symbolId = `iface:${name}`;
          pushSymbol(this.makeSymbol(symbolId, "iface", name, isExport, null, [], declNode));
          break;
        }

        case "enum_declaration": {
          const name = declNode.childForFieldName("name")?.text ?? "";
          const symbolId = `enum:${name}`;
          pushSymbol(this.makeSymbol(symbolId, "enum", name, isExport, null, [], declNode));
          break;
        }

        case "type_alias_declaration": {
          const name = declNode.childForFieldName("name")?.text ?? "";
          const symbolId = `type:${name}`;
          pushSymbol(this.makeSymbol(symbolId, "type", name, isExport, null, [], declNode));
          break;
        }
      }
    }

    // Resolve call edges and read/write edges for callables
    for (const callable of callableNodes) {
      this.collectCallEdges(callable, methodByClassAndName, methodIdsByName, nameToSymbolIds, symbolById, addEdge);
      this.collectVariableEdges(callable, moduleVariableByName, addEdge);
    }

    // Collect imports
    const imports = this.collectImports(root);

    return {
      symbols: sortSymbols(symbols),
      edges: sortEdges(Array.from(edgesMap.values())),
      imports,
      callableNodes,
    };
  }

  private isDeclarationType(type: string): boolean {
    return [
      "class_declaration", "function_declaration", "lexical_declaration",
      "variable_declaration", "interface_declaration", "enum_declaration",
      "type_alias_declaration",
    ].includes(type);
  }

  private getExtendsClause(classNode: SyntaxNode): string | null {
    // In tree-sitter TS, class heritage is in "class_heritage" or via children
    // Look for extends_clause
    for (const child of classNode.children) {
      if (child.type === "extends_clause") {
        // The value/type is the named child
        const value = child.namedChildren[0];
        return value?.text ?? null;
      }
    }
    return null;
  }

  private getImplementsClause(classNode: SyntaxNode): string[] {
    const result: string[] = [];
    for (const child of classNode.children) {
      if (child.type === "implements_clause") {
        for (const type of child.namedChildren) {
          result.push(type.text);
        }
      }
    }
    return result;
  }

  private getClassMethods(classNode: SyntaxNode): SyntaxNode[] {
    const body = classNode.childForFieldName("body");
    if (!body) return [];
    return body.namedChildren.filter((c) => c.type === "method_definition");
  }

  private extractParams(fnNode: SyntaxNode): string[] {
    const params = fnNode.childForFieldName("parameters");
    if (!params) return [];
    return params.namedChildren
      .map((p) => {
        // required_parameter, optional_parameter, rest_parameter
        const name = p.childForFieldName("pattern")?.text ?? p.childForFieldName("name")?.text ?? p.text;
        return name;
      })
      .filter((n) => n.length > 0);
  }

  private makeSymbol(
    id: string,
    kind: LogicSymbolKind,
    name: string,
    exported: boolean,
    parentId: string | null,
    params: string[],
    node: SyntaxNode,
  ): LogicSymbol {
    const isFnLike = this.isFunctionLikeNode(node);
    const control = {
      async: isFnLike ? this.isAsync(node) : false,
      branch: isFnLike ? this.hasBranching(node) : false,
      await: isFnLike ? this.hasDescendantOfType(node, "await_expression") : false,
      throw: isFnLike ? this.hasDescendantOfType(node, "throw_statement") : false,
    };

    return {
      id,
      kind,
      name,
      exported,
      parentId,
      params: [...params].sort((a, b) => a.localeCompare(b)),
      complexity: this.computeComplexityBucket(isFnLike ? node : null),
      control,
      line: node.startPosition.row + 1,
      byteStart: node.startIndex,
      byteEnd: node.endIndex,
      contentHash: createHash("sha1").update(node.text).digest("hex"),
    };
  }

  private isFunctionLikeNode(node: SyntaxNode): boolean {
    return (
      node.type === "function_declaration" ||
      node.type === "method_definition" ||
      node.type === "arrow_function" ||
      node.type === "function"
    );
  }

  private isAsync(node: SyntaxNode): boolean {
    // Check for "async" keyword child
    return node.children.some((c) => c.type === "async");
  }

  private hasBranching(node: SyntaxNode): boolean {
    const branchTypes = new Set([
      "if_statement", "switch_statement", "ternary_expression",
      "for_statement", "for_in_statement", "while_statement", "do_statement",
    ]);
    return this.hasDescendantOfTypes(node, branchTypes);
  }

  private hasDescendantOfType(node: SyntaxNode, type: string): boolean {
    for (const child of node.namedChildren) {
      if (child.type === type) return true;
      if (this.hasDescendantOfType(child, type)) return true;
    }
    return false;
  }

  private hasDescendantOfTypes(node: SyntaxNode, types: Set<string>): boolean {
    for (const child of node.namedChildren) {
      if (types.has(child.type)) return true;
      if (this.hasDescendantOfTypes(child, types)) return true;
    }
    return false;
  }

  private computeComplexityBucket(node: SyntaxNode | null): ComplexityBucket {
    if (!node) return "low";
    const count = this.countDescendantStatements(node);
    if (count >= 18) return "high";
    if (count >= 6) return "medium";
    return "low";
  }

  private countDescendantStatements(node: SyntaxNode): number {
    let count = 0;
    for (const child of node.namedChildren) {
      if (child.type.endsWith("_statement") || child.type.endsWith("_declaration")) {
        count++;
      }
      count += this.countDescendantStatements(child);
    }
    return count;
  }

  private collectCallEdges(
    callable: CallableNodeRef,
    methodByClassAndName: Map<string, string>,
    methodIdsByName: Map<string, string[]>,
    nameToSymbolIds: Map<string, string[]>,
    symbolById: Map<string, LogicSymbol>,
    addEdge: (edge: LogicEdge) => void,
  ): void {
    this.walkForType(callable.node, "call_expression", (callExpr) => {
      const targetId = this.resolveCallTarget(
        callExpr,
        callable.className,
        methodByClassAndName,
        methodIdsByName,
        nameToSymbolIds,
        symbolById,
      );
      if (targetId) {
        addEdge({ kind: "call", from: callable.symbolId, to: targetId });
      }
    });
  }

  private collectVariableEdges(
    callable: CallableNodeRef,
    moduleVariableByName: Map<string, string>,
    addEdge: (edge: LogicEdge) => void,
  ): void {
    this.walkForType(callable.node, "identifier", (identifier) => {
      const name = identifier.text;
      const variableId = moduleVariableByName.get(name);
      if (!variableId) return;
      if (this.isDeclarationIdentifier(identifier)) return;

      const isWrite = this.isWriteIdentifier(identifier);
      addEdge({
        kind: isWrite ? "write" : "read",
        from: callable.symbolId,
        to: variableId,
      });
    });
  }

  private walkForType(node: SyntaxNode, type: string, callback: (node: SyntaxNode) => void): void {
    for (const child of node.namedChildren) {
      if (child.type === type) callback(child);
      this.walkForType(child, type, callback);
    }
  }

  private resolveCallTarget(
    callExpr: SyntaxNode,
    callerClassName: string | null,
    methodByClassAndName: Map<string, string>,
    methodIdsByName: Map<string, string[]>,
    nameToSymbolIds: Map<string, string[]>,
    symbolById: Map<string, LogicSymbol>,
  ): string | null {
    const fn = callExpr.childForFieldName("function");
    if (!fn) return null;

    if (fn.type === "identifier") {
      const symbolIds = nameToSymbolIds.get(fn.text) ?? [];
      return this.pickBestCallableSymbolId(symbolIds, symbolById);
    }

    if (fn.type === "member_expression") {
      const property = fn.childForFieldName("property");
      const object = fn.childForFieldName("object");
      const methodName = property?.text ?? "";

      if (object?.text === "this" && callerClassName) {
        const ownMethod = methodByClassAndName.get(`${callerClassName}.${methodName}`);
        if (ownMethod) return ownMethod;
      }
      const candidateMethodIds = methodIdsByName.get(methodName) ?? [];
      if (candidateMethodIds.length === 1) return candidateMethodIds[0];
    }

    return null;
  }

  private pickBestCallableSymbolId(
    symbolIds: string[],
    symbolById: Map<string, LogicSymbol>,
  ): string | null {
    const ranked = symbolIds
      .map((id) => symbolById.get(id))
      .filter((s): s is LogicSymbol => !!s)
      .filter((s) => s.kind === "fn" || s.kind === "mtd")
      .sort((a, b) => {
        const kindWeight = (s: LogicSymbol) => (s.kind === "fn" ? 0 : 1);
        const diff = kindWeight(a) - kindWeight(b);
        if (diff !== 0) return diff;
        return a.id.localeCompare(b.id);
      });
    return ranked[0]?.id ?? null;
  }

  private isDeclarationIdentifier(identifier: SyntaxNode): boolean {
    const parent = identifier.parent;
    if (!parent) return false;
    const nameNode = parent.childForFieldName("name");
    if (nameNode && nameNode.id === identifier.id) {
      const declarationTypes = [
        "variable_declarator", "function_declaration", "method_definition",
        "class_declaration", "interface_declaration", "type_alias_declaration",
        "enum_declaration",
      ];
      return declarationTypes.includes(parent.type);
    }
    // Also check for "pattern" field (parameters)
    const patternNode = parent.childForFieldName("pattern");
    if (patternNode && patternNode.id === identifier.id) {
      return parent.type === "required_parameter" || parent.type === "optional_parameter";
    }
    return false;
  }

  private isWriteIdentifier(identifier: SyntaxNode): boolean {
    const parent = identifier.parent;
    if (!parent) return false;

    if (parent.type === "assignment_expression" || parent.type === "augmented_assignment_expression") {
      const left = parent.childForFieldName("left");
      if (left && left.id === identifier.id) return true;
    }

    if (parent.type === "update_expression") {
      const arg = parent.childForFieldName("argument");
      if (arg && arg.id === identifier.id) return true;
    }

    return false;
  }

  private collectImports(root: SyntaxNode): string[] {
    const modules = new Set<string>();
    for (const child of root.namedChildren) {
      if (child.type === "import_statement") {
        const source = child.childForFieldName("source");
        if (source) {
          const mod = stripQuotes(source.text);
          if (mod) modules.add(mod);
        }
      }
    }
    return Array.from(modules).sort((a, b) => a.localeCompare(b));
  }
}

function stripQuotes(s: string): string {
  return s.replace(/^['"]|['"]$/g, "");
}

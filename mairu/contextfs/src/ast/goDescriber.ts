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
  receiverType: string | null;
  node: SyntaxNode;
}

export class GoDescriber implements LanguageDescriber {
  readonly languageId: string = "go";
  readonly extensions: ReadonlySet<string> = new Set([".go"]);

  extractFileGraph(filePath: string, sourceText: string): FileGraphResult {
    const tree = this.parseSync("go", sourceText);
    const root = tree.rootNode;

    try {
      const rawGraph = this.extractRawLogicGraph(root);

      // Generate per-symbol NL descriptions
      const symbolDescriptions = new Map<string, string>();

      for (const callable of rawGraph.callableNodes) {
        // We reuse describeStatements, but note that it's tuned for JS/TS.
        // It provides a basic fallback for Go code blocks.
        symbolDescriptions.set(callable.symbolId, describeStatements(callable.node));
      }

      // Generate struct/interface summaries
      for (const symbol of rawGraph.symbols) {
        if (symbol.kind === "cls" || symbol.kind === "iface") {
          const kindStr = symbol.kind === "cls" ? "Struct" : "Interface";
          // Collect methods associated with this type
          const methods = rawGraph.symbols
            .filter((s) => s.parentId === symbol.id)
            .map((s) => s.name);
          const methodsStr = methods.length > 0 ? ` with methods: ${methods.join(", ")}` : "";
          symbolDescriptions.set(
            symbol.id,
            `${kindStr} \`${symbol.name}\`${methodsStr}`,
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

  protected parseSync(lang: SupportedLanguage, sourceText: string): Tree {
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
      `Parser for ${lang} not initialized. Call GoDescriber.initParsers() first.`,
    );
  }

  async initParsers(): Promise<void> {
    await ParserPool.init();
    const parser = await ParserPool.getParser("go");
    this.parserCache.set("go", parser);
  }

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
    const methodByReceiverAndName = new Map<string, string>();
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

    const isExported = (name: string) => /^[A-Z]/.test(name);

    // Find all type_spec, var_spec, const_spec, function_declaration, method_declaration
    this.walkForDeclarations(root, (node) => {
      switch (node.type) {
        case "import_spec": {
          const pathNode = node.childForFieldName("path");
          if (pathNode) {
            let pathText = pathNode.text.replace(/^["']|["']$/g, "");
            addEdge({ kind: "import", from: "file", to: `module:${pathText}` });
          }
          break;
        }

        case "type_spec": {
          const nameNode = node.childForFieldName("name");
          if (!nameNode) break;
          const typeName = nameNode.text;
          const typeNode = node.childForFieldName("type");
          let kind: LogicSymbolKind = "type";
          if (typeNode) {
            if (typeNode.type === "struct_type") kind = "cls";
            else if (typeNode.type === "interface_type") kind = "iface";
          }
          
          const typeId = `${kind}:${typeName}`;
          const exported = isExported(typeName);
          const docstring = this.extractDocstring(node.parent?.type === "type_declaration" ? node.parent : node);
          pushSymbol(this.makeSymbol(typeId, kind, typeName, exported, null, [], node, docstring));
          break;
        }

        case "method_declaration": {
          const nameNode = node.childForFieldName("name");
          if (!nameNode) break;
          const methodName = nameNode.text;
          const receiverNode = node.childForFieldName("receiver");
          const receiverType = this.findTypeIdentifier(receiverNode);
          
          const parentId = receiverType ? `cls:${receiverType}` : null; // Fallback to cls
          const methodId = receiverType ? `mtd:${receiverType}.${methodName}` : `mtd:${methodName}`;
          
          const exported = isExported(methodName);
          const docstring = this.extractDocstring(node);
          const params = this.extractParams(node);
          
          pushSymbol(this.makeSymbol(methodId, "mtd", methodName, exported, parentId, params, node, docstring));
          
          if (receiverType) {
             methodByReceiverAndName.set(`${receiverType}.${methodName}`, methodId);
          }
          const existingMethodIds = methodIdsByName.get(methodName) ?? [];
          existingMethodIds.push(methodId);
          methodIdsByName.set(methodName, existingMethodIds);
          callableNodes.push({ symbolId: methodId, receiverType, node });
          break;
        }

        case "function_declaration": {
          const nameNode = node.childForFieldName("name");
          if (!nameNode) break;
          const fnName = nameNode.text;
          const fnId = `fn:${fnName}`;
          const exported = isExported(fnName);
          const docstring = this.extractDocstring(node);
          const params = this.extractParams(node);
          
          pushSymbol(this.makeSymbol(fnId, "fn", fnName, exported, null, params, node, docstring));
          callableNodes.push({ symbolId: fnId, receiverType: null, node });
          break;
        }

        case "var_spec":
        case "const_spec": {
          const nameNodes = node.namedChildren.filter(c => c.type === "identifier");
          for (const nameNode of nameNodes) {
             const varName = nameNode.text;
             const varId = `var:${varName}`;
             const exported = isExported(varName);
             pushSymbol(this.makeSymbol(varId, "var", varName, exported, null, [], node));
             moduleVariableByName.set(varName, varId);
          }
          break;
        }
      }
    });

    for (const callable of callableNodes) {
      this.collectCallEdges(callable, methodByReceiverAndName, methodIdsByName, nameToSymbolIds, symbolById, addEdge);
      this.collectVariableEdges(callable, moduleVariableByName, addEdge);
    }

    const imports = this.collectImports(root);

    return {
      symbols: sortSymbols(symbols),
      edges: sortEdges(Array.from(edgesMap.values())),
      imports,
      callableNodes,
    };
  }

  private walkForDeclarations(node: SyntaxNode, callback: (node: SyntaxNode) => void) {
    if (["import_spec", "type_spec", "method_declaration", "function_declaration", "var_spec", "const_spec"].includes(node.type)) {
       callback(node);
       if (node.type === "method_declaration" || node.type === "function_declaration") return; // do not go inside bodies
    }
    for (const child of node.namedChildren) {
      this.walkForDeclarations(child, callback);
    }
  }

  private findTypeIdentifier(node: SyntaxNode | null): string | null {
     if (!node) return null;
     if (node.type === "type_identifier") return node.text;
     for (const child of node.namedChildren) {
       const t = this.findTypeIdentifier(child);
       if (t) return t;
     }
     return null;
  }

  private extractImports(node: SyntaxNode): string[] {
    const modules: string[] = [];
    if (node.type === "import_spec") {
      const pathNode = node.childForFieldName("path");
      if (pathNode) {
        modules.push(pathNode.text.replace(/^["']|["']$/g, ""));
      }
    }
    for (const child of node.namedChildren) {
      modules.push(...this.extractImports(child));
    }
    return modules;
  }

  private collectImports(root: SyntaxNode): string[] {
    const modules = new Set<string>();
    const importDecls = root.namedChildren.filter(c => c.type === "import_declaration");
    for (const child of importDecls) {
      this.extractImports(child).forEach(mod => modules.add(mod));
    }
    return Array.from(modules).sort((a, b) => a.localeCompare(b));
  }

  private extractParams(fnNode: SyntaxNode): string[] {
    const paramsNode = fnNode.childForFieldName("parameters");
    if (!paramsNode) return [];
    
    const params: string[] = [];
    for (const p of paramsNode.namedChildren) {
       if (p.type === "parameter_declaration") {
          const names = p.namedChildren.filter(c => c.type === "identifier").map(c => c.text);
          params.push(...names);
       }
    }
    return params;
  }

  private extractDocstring(node: SyntaxNode): string | undefined {
    // In Go, docstrings are comments immediately preceding the declaration.
    // However, tree-sitter often attaches them as previous siblings.
    let prev = node.previousSibling;
    const lines: string[] = [];
    while (prev && prev.type === "comment") {
      lines.push(prev.text.replace(/^\/\/\s*/, ""));
      prev = prev.previousSibling;
    }
    if (lines.length > 0) {
      lines.reverse();
      let text = lines.join(" ");
      return text.length > 200 ? text.slice(0, 200) + "..." : text;
    }
    return undefined;
  }

  private makeSymbol(
    id: string,
    kind: LogicSymbolKind,
    name: string,
    exported: boolean,
    parentId: string | null,
    params: string[],
    node: SyntaxNode,
    docstring?: string,
  ): LogicSymbol {
    const isFnLike = this.isFunctionLikeNode(node);
    const control = {
      async: isFnLike ? this.hasDescendantOfType(node, "go_statement") : false,
      branch: isFnLike ? this.hasBranching(node) : false,
      await: false, // Go uses goroutines, not async/await
      throw: isFnLike ? this.hasPanic(node) : false,
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
      startLine: node.startPosition.row,
      endLine: node.endPosition.row,
      contentHash: createHash("sha1").update(node.text).digest("hex"),
      ...(docstring ? { docstring } : {}),
    };
  }

  private isFunctionLikeNode(node: SyntaxNode): boolean {
    return node.type === "function_declaration" || node.type === "method_declaration";
  }

  private hasBranching(node: SyntaxNode): boolean {
    const branchTypes = new Set([
      "if_statement", "for_statement", "switch_statement", "select_statement"
    ]);
    return this.hasDescendantOfTypes(node, branchTypes);
  }

  private hasPanic(node: SyntaxNode): boolean {
    // Look for panic() calls
    let found = false;
    this.walkForType(node, "call_expression", (call) => {
       const fn = call.childForFieldName("function");
       if (fn && fn.text === "panic") found = true;
    });
    return found;
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
    methodByReceiverAndName: Map<string, string>,
    methodIdsByName: Map<string, string[]>,
    nameToSymbolIds: Map<string, string[]>,
    symbolById: Map<string, LogicSymbol>,
    addEdge: (edge: LogicEdge) => void,
  ): void {
    this.walkForType(callable.node, "call_expression", (callExpr) => {
      const targetId = this.resolveCallTarget(
        callExpr,
        callable.receiverType,
        methodByReceiverAndName,
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
    callerReceiverType: string | null,
    methodByReceiverAndName: Map<string, string>,
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

    if (fn.type === "selector_expression") {
      const field = fn.childForFieldName("field");
      const methodName = field?.text ?? "";

      // We cannot easily trace local variable types in Go AST without full type inference.
      // So we do best-effort heuristic.
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
    let current = identifier.parent;
    while (current) {
       if (["function_declaration", "method_declaration", "var_spec", "const_spec"].includes(current.type)) {
          const nameNode = current.childForFieldName("name");
          if (nameNode && nameNode.id === identifier.id) return true;
          // check var_spec/const_spec names (they can have multiple)
          if (current.type === "var_spec" || current.type === "const_spec") {
             for (const c of current.namedChildren) {
                if (c.type === "identifier" && c.id === identifier.id) return true;
             }
          }
       }
       current = current.parent;
    }
    return false;
  }

  private isWriteIdentifier(identifier: SyntaxNode): boolean {
    let current = identifier.parent;
    while (current) {
       if (current.type === "assignment_statement" || current.type === "short_var_declaration" || current.type === "inc_statement" || current.type === "dec_statement") {
          const leftNode = current.childForFieldName("left") || current.namedChildren[0];
          if (leftNode) {
             // For multiple assignment, it's an expression_list usually
             let found = false;
             this.walkForType(leftNode, "identifier", (id) => {
                if (id.id === identifier.id) found = true;
             });
             if (found) return true;
             if (leftNode.id === identifier.id) return true;
          }
       }
       current = current.parent;
    }
    return false;
  }
}

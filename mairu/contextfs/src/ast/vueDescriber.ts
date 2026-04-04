import type { Node as SyntaxNode } from "web-tree-sitter";
import { Parser } from "web-tree-sitter";
import type { LanguageDescriber, FileGraphResult } from "./languageDescriber";
import { sortSymbols, sortEdges } from "./languageDescriber";
import { TypeScriptDescriber } from "./typescriptDescriber";
import type { TemplateNode, TemplateDirective } from "./templateWalker";
import { walkTemplate } from "./templateWalker";
import { ParserPool } from "./parserPool";
import * as path from "path";

const PASCAL_CASE_RE = /^[A-Z][a-zA-Z0-9]+$/;

export class VueDescriber implements LanguageDescriber {
  readonly languageId = "vue";
  readonly extensions: ReadonlySet<string> = new Set([".vue"]);

  private tsDescriber = new TypeScriptDescriber();
  private parser: InstanceType<typeof Parser> | null = null;

  /** Initialize the Vue parser. Must be called once before extractFileGraph. */
  async initParsers(): Promise<void> {
    await ParserPool.init();
    this.parser = await ParserPool.getParser("vue");
    await this.tsDescriber.initParsers();
  }

  /** Cleanup parsers when done. */
  deleteParsers(): void {
    if (this.parser) {
      this.parser.delete();
      this.parser = null;
    }
    this.tsDescriber.deleteParsers();
  }

  extractFileGraph(filePath: string, sourceText: string): FileGraphResult {
    const componentName = path.basename(filePath, ".vue");

    if (!this.parser) {
      throw new Error("VueDescriber not initialized. Call initParsers() first.");
    }

    const tree = this.parser.parse(sourceText);
    if (!tree) throw new Error("Failed to parse Vue SFC");

    try {
      const root = tree.rootNode;

      // --- Script extraction ---
      let scriptResult: FileGraphResult | null = null;
      const importedComponents = new Set<string>();

      const scriptSetup = this.findScriptSetup(root);
      if (scriptSetup) {
        const scriptContent = this.extractBlockContent(scriptSetup, sourceText);
        const fakePath = filePath.replace(/\.vue$/, ".ts");
        scriptResult = this.tsDescriber.extractFileGraph(fakePath, scriptContent);
        this.scanComponentImports(scriptContent, importedComponents);
      }

      // --- Template extraction ---
      let templateResult: ReturnType<typeof walkTemplate> | null = null;

      const templateBlock = this.findTemplateBlock(root);
      if (templateBlock) {
        try {
          const templateNodes = this.convertVueNodes(templateBlock, importedComponents);
          templateResult = walkTemplate(componentName, templateNodes);
        } catch {
          // Malformed template — continue with script-only
        }
      }

      // --- Merge results ---
      const symbols = [
        ...(scriptResult?.symbols ?? []),
        ...(templateResult?.symbols ?? []),
      ];
      const edges = [
        ...(scriptResult?.edges ?? []),
        ...(templateResult?.edges ?? []),
      ];
      const imports = scriptResult?.imports ?? [];

      const symbolDescriptions = new Map<string, string>(
        scriptResult?.symbolDescriptions ?? [],
      );
      if (templateResult) {
        for (const [k, v] of templateResult.descriptions) {
          symbolDescriptions.set(k, v);
        }
      }

      // Build file summary
      const scriptSummary = scriptResult?.fileSummary ?? "";
      const templateDesc = templateResult?.descriptions.get(`tpl:${componentName}`) ?? "";
      const parts: string[] = [];
      if (scriptSummary) parts.push(scriptSummary);
      if (templateDesc) parts.push(templateDesc);
      const fileSummary =
        parts.length > 0
          ? parts.join(" ")
          : `Vue component \`${componentName}\`.`;

      return {
        symbols: sortSymbols(symbols),
        edges: sortEdges(edges),
        imports,
        symbolDescriptions,
        fileSummary,
      };
    } finally {
      tree.delete();
    }
  }

  /** Find <script setup> block in the Vue SFC tree. */
  private findScriptSetup(root: SyntaxNode): SyntaxNode | null {
    for (const child of root.namedChildren) {
      if (child.type === "script_element") {
        // Check for "setup" attribute
        const startTag = child.namedChildren.find(
          (c) => c.type === "start_tag",
        );
        if (startTag) {
          const hasSetup = startTag.namedChildren.some(
            (attr) => attr.type === "attribute" && attr.text.includes("setup"),
          );
          if (hasSetup) return child;
        }
      }
    }
    // Fallback: return any script_element
    for (const child of root.namedChildren) {
      if (child.type === "script_element") return child;
    }
    return null;
  }

  /** Find <template> block in the Vue SFC tree. */
  private findTemplateBlock(root: SyntaxNode): SyntaxNode | null {
    for (const child of root.namedChildren) {
      if (child.type === "template_element") return child;
    }
    return null;
  }

  /** Extract the text content of a Vue SFC block (between start and end tags). */
  private extractBlockContent(block: SyntaxNode, sourceText: string): string {
    // Find the raw_text child which contains the block content
    for (const child of block.namedChildren) {
      if (child.type === "raw_text") {
        return child.text;
      }
    }
    // Fallback: extract between start_tag end and end_tag start
    const startTag = block.namedChildren.find((c) => c.type === "start_tag");
    const endTag = block.namedChildren.find((c) => c.type === "end_tag");
    if (startTag && endTag) {
      return sourceText.slice(startTag.endIndex, endTag.startIndex);
    }
    return "";
  }

  private scanComponentImports(scriptContent: string, registry: Set<string>): void {
    const importRegex = /import\s+(?:(\w+)|{([^}]+)})\s+from\s+['"][^'"]+['"]/g;
    let match: RegExpExecArray | null;
    while ((match = importRegex.exec(scriptContent)) !== null) {
      const defaultImport = match[1];
      const namedImports = match[2];
      if (defaultImport && PASCAL_CASE_RE.test(defaultImport)) {
        registry.add(defaultImport);
      }
      if (namedImports) {
        for (const part of namedImports.split(",")) {
          const name = part.trim().split(/\s+as\s+/).pop()!.trim();
          if (PASCAL_CASE_RE.test(name)) {
            registry.add(name);
          }
        }
      }
    }
  }

  /** Convert Vue template tree-sitter nodes into TemplateNode[]. */
  private convertVueNodes(
    templateBlock: SyntaxNode,
    componentRegistry: Set<string>,
  ): TemplateNode[] {
    const result: TemplateNode[] = [];
    for (const child of templateBlock.namedChildren) {
      if (child.type === "element" || child.type === "self_closing_tag") {
        result.push(this.convertVueElement(child, componentRegistry));
      }
      // The template_element contains start_tag, (elements), end_tag
      // We might need to look deeper
      if (child.type === "start_tag" || child.type === "end_tag") continue;
      // Some Vue template ASTs have element children directly
    }
    // If no elements found at top level, look for elements inside the template content
    if (result.length === 0) {
      for (const child of templateBlock.namedChildren) {
        for (const grandchild of child.namedChildren) {
          if (grandchild.type === "element" || grandchild.type === "self_closing_tag") {
            result.push(this.convertVueElement(grandchild, componentRegistry));
          }
        }
      }
    }
    return result;
  }

  private convertVueElement(
    el: SyntaxNode,
    componentRegistry: Set<string>,
  ): TemplateNode {
    // Get tag name from start_tag or self_closing_tag
    const startTag = el.type === "self_closing_tag"
      ? el
      : el.namedChildren.find((c) => c.type === "start_tag");

    const tagNameNode = startTag?.namedChildren.find(
      (c) => c.type === "tag_name",
    );
    const tag = tagNameNode?.text ?? "unknown";
    const directives: TemplateDirective[] = [];
    let slotName: string | undefined;

    // Process attributes for directives
    if (startTag) {
      for (const attr of startTag.namedChildren) {
        if (attr.type === "directive_attribute") {
          const directive = this.convertDirective(attr);
          if (directive) directives.push(directive);
        }
        // Check for slot name attribute
        if (attr.type === "attribute" && tag === "slot") {
          const attrName = attr.namedChildren.find((c) => c.type === "attribute_name");
          const attrValue = attr.namedChildren.find((c) => c.type === "quoted_attribute_value");
          if (attrName?.text === "name" && attrValue) {
            slotName = attrValue.text.replace(/^['"]|['"]$/g, "");
          }
        }
      }
    }

    const isComponent = this.isComponentTag(tag, componentRegistry);
    const children = this.convertVueChildElements(el, componentRegistry);

    return {
      tag,
      isComponent,
      directives,
      children,
      line: el.startPosition.row + 1,
      ...(tag === "slot" ? { slotName } : {}),
    };
  }

  private convertVueChildElements(
    el: SyntaxNode,
    componentRegistry: Set<string>,
  ): TemplateNode[] {
    const result: TemplateNode[] = [];
    for (const child of el.namedChildren) {
      if (child.type === "element" || child.type === "self_closing_tag") {
        result.push(this.convertVueElement(child, componentRegistry));
      }
    }
    return result;
  }

  private convertDirective(attr: SyntaxNode): TemplateDirective | null {
    // directive_attribute has directive_name and possibly directive_value
    const nameNode = attr.namedChildren.find((c) => c.type === "directive_name");
    const valueNode = attr.namedChildren.find(
      (c) => c.type === "quoted_attribute_value" || c.type === "directive_value",
    );

    const name = nameNode?.text ?? "";
    let expression = valueNode?.text?.replace(/^['"]|['"]$/g, "") ?? "";

    // Parse directive name (v-if, v-for, v-show, v-else-if, v-else, :is)
    if (name === "v-if" || name === "if") return { kind: "if", expression };
    if (name === "v-else-if" || name === "else-if") return { kind: "else-if", expression };
    if (name === "v-else" || name === "else") return { kind: "else", expression: "" };
    if (name === "v-for" || name === "for") return { kind: "for", expression };
    if (name === "v-show" || name === "show") return { kind: "show", expression };

    return null;
  }

  private isComponentTag(tag: string, componentRegistry: Set<string>): boolean {
    if (PASCAL_CASE_RE.test(tag)) return true;
    if (tag.includes(".")) return true;
    if (componentRegistry.has(tag)) return true;
    return false;
  }
}

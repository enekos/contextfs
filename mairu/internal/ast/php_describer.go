package ast

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/php"
)

type PHPDescriber struct{}

func (d PHPDescriber) LanguageID() string   { return "php" }
func (d PHPDescriber) Extensions() []string { return []string{".php"} }

func (d PHPDescriber) ExtractFileGraph(_ string, source string) FileGraph {
	sourceBytes := []byte(source)
	parser := GetParser()
	defer PutParser(parser)
	parser.SetLanguage(php.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, sourceBytes)
	if tree != nil {
		defer tree.Close()
	}
	root := tree.RootNode()

	symbols := []LogicSymbol{}
	edgesMap := make(map[string]LogicEdge)
	symbolDescriptions := make(map[string]string)

	addEdge := func(edge LogicEdge) {
		k := edge.Kind + "|" + edge.From + "|" + edge.To
		if _, ok := edgesMap[k]; !ok {
			edgesMap[k] = edge
		}
	}

	imports := []string{}
	namespace := ""

	// We need to walk the tree to find namespace, uses, classes, functions
	walkForType(root, "namespace_definition", func(node *sitter.Node) {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			namespace = nameNode.Content(sourceBytes)
		}
	})

	walkForType(root, "namespace_use_clause", func(node *sitter.Node) {
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child.Type() == "qualified_name" || child.Type() == "name" {
				imports = append(imports, child.Content(sourceBytes))
				break
			}
		}
	})

	// fallback for imports if it's name directly in namespace_use_declaration
	walkForType(root, "namespace_use_declaration", func(node *sitter.Node) {
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child.Type() == "namespace_use_clause" {
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					imports = append(imports, nameNode.Content(sourceBytes))
				}
			}
		}
	})
	imports = dedupeStrings(imports)

	idsByName := map[string]string{}
	currentClass := ""

	// To extract symbols correctly, let's walk through declarations.
	// PHP AST structure usually has declarations at the top level or within a namespace block.
	var processDeclarations func(node *sitter.Node)
	processDeclarations = func(node *sitter.Node) {
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)

			switch child.Type() {
			case "class_declaration", "interface_declaration", "trait_declaration", "enum_declaration":
				nameNode := child.ChildByFieldName("name")
				name := "anonymous"
				if nameNode != nil {
					name = nameNode.Content(sourceBytes)
				}

				prefix := "cls"
				if child.Type() == "interface_declaration" {
					prefix = "iface"
				} else if child.Type() == "trait_declaration" {
					prefix = "trait"
				} else if child.Type() == "enum_declaration" {
					prefix = "enum"
				}

				symId := prefix + ":" + name
				if namespace != "" {
					symId = prefix + ":" + namespace + "\\" + name
				}

				symbols = append(symbols, LogicSymbol{
					ID:       symId,
					Kind:     prefix,
					Name:     name,
					Line:     int(child.StartPoint().Row) + 1,
					Exported: true, // In PHP, classes are generally globally accessible if included
				})
				symbolDescriptions[symId] = "Describes " + name
				idsByName[name] = symId

				// Process methods
				body := child.ChildByFieldName("body")
				if body == nil {
					body = child.ChildByFieldName("declarations")
				}
				if body == nil {
					for j := 0; j < int(child.NamedChildCount()); j++ {
						if child.NamedChild(j).Type() == "declaration_list" {
							body = child.NamedChild(j)
							break
						}
					}
				}
				if body != nil {
					prevClass := currentClass
					currentClass = name
					processDeclarations(body)
					currentClass = prevClass
				}

			case "function_definition":
				nameNode := child.ChildByFieldName("name")
				name := "anonymous_fn"
				if nameNode != nil {
					name = nameNode.Content(sourceBytes)
				}

				symId := "fn:" + name
				if namespace != "" {
					symId = "fn:" + namespace + "\\" + name
				}

				symbols = append(symbols, LogicSymbol{
					ID:          symId,
					Kind:        "fn",
					Name:        name,
					Line:        int(child.StartPoint().Row) + 1,
					Exported:    true,
					ControlFlow: SummarizeControlFlow(child, sourceBytes),
				})
				symbolDescriptions[symId] = "Describes " + name
				idsByName[name] = symId

			case "method_declaration":
				nameNode := child.ChildByFieldName("name")
				name := "anonymous_mtd"
				if nameNode != nil {
					name = nameNode.Content(sourceBytes)
				}

				className := currentClass
				symId := "mtd:" + className + "::" + name
				if namespace != "" {
					symId = "mtd:" + namespace + "\\" + className + "::" + name
				}

				// visibility
				exported := true
				for j := 0; j < int(child.ChildCount()); j++ {
					c := child.Child(j)
					if c.Type() == "visibility_modifier" {
						vis := c.Content(sourceBytes)
						if strings.Contains(vis, "private") || strings.Contains(vis, "protected") {
							exported = false
						}
					}
				}

				symbols = append(symbols, LogicSymbol{
					ID:          symId,
					Kind:        "mtd",
					Name:        name,
					Line:        int(child.StartPoint().Row) + 1,
					Exported:    exported,
					ControlFlow: SummarizeControlFlow(child, sourceBytes),
				})
				symbolDescriptions[symId] = "Describes " + className + "::" + name
				idsByName[name] = symId
				idsByName[className+"::"+name] = symId

			case "namespace_definition":
				body := child.ChildByFieldName("body")
				if body != nil {
					processDeclarations(body)
				}
			}
		}
	}

	processDeclarations(root)

	// Calls
	walkForType(root, "function_call_expression", func(callExpr *sitter.Node) {
		fnNode := callExpr.ChildByFieldName("function")
		if fnNode != nil {
			targetName := fnNode.Content(sourceBytes)
			// simplistic check if target exists
			if targetId, ok := idsByName[targetName]; ok {
				// We need to know who is calling. Find closest fn or mtd parent
				callerId := findEnclosingCallablePHP(callExpr, sourceBytes, namespace, currentClass)
				if callerId != "" && callerId != targetId {
					addEdge(LogicEdge{Kind: "call", From: callerId, To: targetId})
				}
			}
		}
	})

	walkForType(root, "member_call_expression", func(callExpr *sitter.Node) {
		nameNode := callExpr.ChildByFieldName("name")
		if nameNode != nil {
			targetName := nameNode.Content(sourceBytes)
			if targetId, ok := idsByName[targetName]; ok {
				callerId := findEnclosingCallablePHP(callExpr, sourceBytes, namespace, currentClass)
				if callerId != "" && callerId != targetId {
					addEdge(LogicEdge{Kind: "call", From: callerId, To: targetId})
				}
			}
		}
	})

	walkForType(root, "scoped_call_expression", func(callExpr *sitter.Node) {
		nameNode := callExpr.ChildByFieldName("name")
		scopeNode := callExpr.ChildByFieldName("scope")
		if nameNode != nil && scopeNode != nil {
			targetName := scopeNode.Content(sourceBytes) + "::" + nameNode.Content(sourceBytes)
			if targetId, ok := idsByName[targetName]; ok {
				callerId := findEnclosingCallablePHP(callExpr, sourceBytes, namespace, currentClass)
				if callerId != "" && callerId != targetId {
					addEdge(LogicEdge{Kind: "call", From: callerId, To: targetId})
				}
			}
		}
	})

	var finalEdges []LogicEdge
	for _, v := range edgesMap {
		finalEdges = append(finalEdges, v)
	}

	return FileGraph{
		FileSummary:        fmt.Sprintf("PHP file with %d symbols.", len(symbols)),
		Symbols:            SortSymbols(symbols),
		Edges:              SortEdges(finalEdges),
		Imports:            imports,
		SymbolDescriptions: symbolDescriptions,
	}
}

func findEnclosingCallablePHP(node *sitter.Node, sourceBytes []byte, namespace string, currentClass string) string {
	curr := node.Parent()
	for curr != nil {
		if curr.Type() == "function_definition" {
			nameNode := curr.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(sourceBytes)
				if namespace != "" {
					return "fn:" + namespace + "\\" + name
				}
				return "fn:" + name
			}
		} else if curr.Type() == "method_declaration" {
			nameNode := curr.ChildByFieldName("name")
			className := ""

			// Find enclosing class
			clsCurr := curr.Parent()
			for clsCurr != nil {
				if clsCurr.Type() == "class_declaration" || clsCurr.Type() == "trait_declaration" {
					clsNameNode := clsCurr.ChildByFieldName("name")
					if clsNameNode != nil {
						className = clsNameNode.Content(sourceBytes)
						break
					}
				}
				clsCurr = clsCurr.Parent()
			}

			if nameNode != nil {
				name := nameNode.Content(sourceBytes)
				if namespace != "" && className != "" {
					return "mtd:" + namespace + "\\" + className + "::" + name
				}
				if className != "" {
					return "mtd:" + className + "::" + name
				}
				return "mtd:anonymous::" + name
			}
		}
		curr = curr.Parent()
	}
	return ""
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

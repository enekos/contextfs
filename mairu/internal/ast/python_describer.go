package ast

import "regexp"

var rePyDef = regexp.MustCompile(`(?m)^\s*def\s+([A-Za-z_]\w*)\s*\(`)

type PythonDescriber struct{}

func (d PythonDescriber) LanguageID() string   { return "python" }
func (d PythonDescriber) Extensions() []string { return []string{".py"} }
func (d PythonDescriber) ExtractFileGraph(_ string, source string) FileGraph {
	symbols := []LogicSymbol{}
	for _, m := range rePyDef.FindAllStringSubmatch(source, -1) {
		symbols = append(symbols, LogicSymbol{ID: "fn:" + m[1], Name: m[1], Kind: "fn"})
	}
	descs := map[string]string{}
	for _, s := range symbols {
		descs[s.ID] = "Describes " + s.Name
	}
	return FileGraph{
		FileSummary:        "Python module with extracted symbols.",
		Symbols:            symbols,
		Edges:              []LogicEdge{},
		SymbolDescriptions: descs,
	}
}

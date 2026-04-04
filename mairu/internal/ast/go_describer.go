package ast

import "regexp"

var reGoFn = regexp.MustCompile(`(?m)^func\s+(?:\(\w+\s+\*?\w+\)\s*)?([A-Za-z_]\w*)\s*\(`)

type GoDescriber struct{}

func (d GoDescriber) LanguageID() string   { return "go" }
func (d GoDescriber) Extensions() []string { return []string{".go"} }
func (d GoDescriber) ExtractFileGraph(_ string, source string) FileGraph {
	symbols := []LogicSymbol{}
	for _, m := range reGoFn.FindAllStringSubmatch(source, -1) {
		symbols = append(symbols, LogicSymbol{ID: "fn:" + m[1], Name: m[1], Kind: "fn"})
	}
	return FileGraph{
		FileSummary: "Go file graph extracted.",
		Symbols:     symbols,
		Edges:       []LogicEdge{},
	}
}

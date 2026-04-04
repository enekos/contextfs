package ast

import "regexp"

var reVueSetup = regexp.MustCompile(`(?m)function\s+([A-Za-z_]\w*)\s*\(`)

type VueDescriber struct{}

func (d VueDescriber) LanguageID() string   { return "vue" }
func (d VueDescriber) Extensions() []string { return []string{".vue"} }
func (d VueDescriber) ExtractFileGraph(_ string, source string) FileGraph {
	symbols := []LogicSymbol{}
	for _, m := range reVueSetup.FindAllStringSubmatch(source, -1) {
		symbols = append(symbols, LogicSymbol{ID: "fn:" + m[1], Name: m[1], Kind: "fn"})
	}
	return FileGraph{
		FileSummary: "Vue component graph extracted.",
		Symbols:     symbols,
		Edges:       []LogicEdge{},
	}
}

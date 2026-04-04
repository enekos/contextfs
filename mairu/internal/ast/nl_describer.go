package ast

import "strings"

func DescribeSymbols(symbols []LogicSymbol, edges []LogicEdge) string {
	if len(symbols) == 0 {
		return ""
	}
	callsByFrom := map[string][]string{}
	for _, e := range edges {
		callsByFrom[e.From] = append(callsByFrom[e.From], e.To)
	}
	var sections []string
	for _, s := range symbols {
		lines := []string{"## " + s.Name, "Symbol kind: " + s.Kind}
		if s.Doc != "" {
			lines = append(lines, s.Doc)
		}
		if calls := callsByFrom[s.ID]; len(calls) > 0 {
			lines = append(lines, "Calls "+strings.Join(calls, ", "))
		}
		lines = append(lines, "Returns a value.")
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.Join(sections, "\n\n")
}

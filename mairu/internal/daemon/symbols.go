package daemon

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type symbol struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	Doc  string `json:"doc,omitempty"`
}

type edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

var (
	reFunction = regexp.MustCompile(`(?m)(?:export\s+)?function\s+([A-Za-z_]\w*)\s*\(`)
	reClass    = regexp.MustCompile(`(?m)(?:export\s+)?class\s+([A-Za-z_]\w*)`)
	reMethod   = regexp.MustCompile(`(?m)^\s*(?:public|private|protected)?\s*([A-Za-z_]\w*)\s*\(`)
	reGoFunc   = regexp.MustCompile(`(?m)^func\s+(?:\(\w+\s+\*?\w+\)\s*)?([A-Za-z_]\w*)\s*\(`)
	rePyDef    = regexp.MustCompile(`(?m)^\s*def\s+([A-Za-z_]\w*)\s*\(`)
	reCalls    = regexp.MustCompile(`([A-Za-z_]\w*)\s*\(`)
	reJSDoc    = regexp.MustCompile(`(?s)^\s*/\*\*(.*?)\*/`)
	reDocLine  = regexp.MustCompile(`(?m)^\s*\*\s?`)
)

func extractSymbols(src string) []symbol {
	var out []symbol
	for _, m := range reFunction.FindAllStringSubmatch(src, -1) {
		name := m[1]
		out = append(out, symbol{ID: "fn:" + name, Kind: "fn", Name: name, Doc: nearestDoc(src, name)})
	}
	for _, m := range reClass.FindAllStringSubmatch(src, -1) {
		name := m[1]
		out = append(out, symbol{ID: "cls:" + name, Kind: "cls", Name: name, Doc: nearestDoc(src, name)})
	}
	for _, m := range reGoFunc.FindAllStringSubmatch(src, -1) {
		name := m[1]
		out = append(out, symbol{ID: "fn:" + name, Kind: "fn", Name: name})
	}
	for _, m := range rePyDef.FindAllStringSubmatch(src, -1) {
		name := m[1]
		out = append(out, symbol{ID: "fn:" + name, Kind: "fn", Name: name})
	}
	seenClass := ""
	for _, m := range reClass.FindAllStringSubmatch(src, -1) {
		seenClass = m[1]
	}
	if seenClass != "" {
		for _, m := range reMethod.FindAllStringSubmatch(src, -1) {
			n := m[1]
			if n == "if" || n == "for" || n == "while" || n == "switch" || n == "catch" || n == "function" || n == "return" {
				continue
			}
			out = append(out, symbol{ID: "mtd:" + seenClass + "." + n, Kind: "mtd", Name: n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return dedupeSymbols(out)
}

func dedupeSymbols(in []symbol) []symbol {
	seen := map[string]bool{}
	var out []symbol
	for _, s := range in {
		if seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		out = append(out, s)
	}
	return out
}

func extractEdges(src string, symbols []symbol) []edge {
	if len(symbols) == 0 {
		return nil
	}
	idsByName := map[string]string{}
	for _, s := range symbols {
		idsByName[s.Name] = s.ID
	}
	var edges []edge
	for _, s := range symbols {
		for _, m := range reCalls.FindAllStringSubmatch(src, -1) {
			callee := m[1]
			to := idsByName[callee]
			if to == "" || to == s.ID {
				continue
			}
			edges = append(edges, edge{From: s.ID, To: to})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})
	return dedupeEdges(edges)
}

func dedupeEdges(in []edge) []edge {
	seen := map[string]bool{}
	var out []edge
	for _, e := range in {
		key := e.From + "->" + e.To
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	return out
}

func buildNLContent(symbols []symbol, edges []edge) string {
	if len(symbols) == 0 {
		return ""
	}
	byFrom := map[string][]string{}
	for _, e := range edges {
		byFrom[e.From] = append(byFrom[e.From], e.To)
	}
	var parts []string
	for _, s := range symbols {
		lines := []string{fmt.Sprintf("## [%s] %s", s.Kind, s.Name)}
		if s.Doc != "" {
			lines = append(lines, s.Doc)
		}
		if calls := byFrom[s.ID]; len(calls) > 0 {
			lines = append(lines, "Dependencies: "+strings.Join(calls, ", "))
		}
		parts = append(parts, strings.Join(lines, "\n"))
	}
	return strings.Join(parts, "\n\n")
}

func extractFileJSDoc(src string) string {
	match := reJSDoc.FindStringSubmatch(src)
	if len(match) < 2 {
		return ""
	}
	raw := strings.TrimSpace(reDocLine.ReplaceAllString(match[1], ""))
	if raw == "" {
		return ""
	}
	return raw
}

func nearestDoc(src, name string) string {
	idx := strings.Index(src, name)
	if idx <= 0 {
		return ""
	}
	prefix := src[:idx]
	last := strings.LastIndex(prefix, "/**")
	if last < 0 {
		return ""
	}
	chunk := prefix[last:]
	end := strings.Index(chunk, "*/")
	if end < 0 {
		return ""
	}
	body := chunk[:end+2]
	return extractFileJSDoc(body)
}

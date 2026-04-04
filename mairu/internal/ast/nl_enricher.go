package ast

import "strings"

func EnrichDescriptions(descriptions map[string]string, edges []LogicEdge) map[string]string {
	out := map[string]string{}
	for k, v := range descriptions {
		out[k] = v
	}
	for _, e := range edges {
		existing := out[e.From]
		if existing == "" {
			existing = "Describes symbol."
		}
		out[e.From] = existing + " Calls " + e.To + "."
	}
	for k, v := range out {
		out[k] = strings.TrimSpace(v)
	}
	return out
}

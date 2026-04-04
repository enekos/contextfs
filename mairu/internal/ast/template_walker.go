package ast

import "regexp"

type TemplateStats struct {
	Slots    int
	Branches int
	Loops    int
}

var (
	reSlot   = regexp.MustCompile(`<slot`)
	reBranch = regexp.MustCompile(`v-if=`)
	reLoop   = regexp.MustCompile(`v-for=`)
)

func WalkTemplate(source string) TemplateStats {
	return TemplateStats{
		Slots:    len(reSlot.FindAllStringIndex(source, -1)),
		Branches: len(reBranch.FindAllStringIndex(source, -1)),
		Loops:    len(reLoop.FindAllStringIndex(source, -1)),
	}
}

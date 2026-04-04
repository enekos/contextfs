package ast

type TSXDescriber struct{}

func (d TSXDescriber) LanguageID() string   { return "tsx" }
func (d TSXDescriber) Extensions() []string { return []string{".tsx", ".jsx"} }
func (d TSXDescriber) ExtractFileGraph(_ string, source string) FileGraph {
	g := BaseExtract(source)
	g.FileSummary = "TSX/JSX component graph extracted."
	return g
}

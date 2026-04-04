package ast

type TypeScriptDescriber struct{}

func (d TypeScriptDescriber) LanguageID() string   { return "typescript" }
func (d TypeScriptDescriber) Extensions() []string { return []string{".ts", ".js", ".mjs", ".cjs"} }
func (d TypeScriptDescriber) ExtractFileGraph(_ string, source string) FileGraph {
	g := BaseExtract(source)
	g.FileSummary = "TypeScript module with extracted symbols."
	return g
}

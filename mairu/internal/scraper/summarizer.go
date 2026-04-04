package scraper

func SummarizePage(p Page, maxChars int) Summary {
	if maxChars <= 0 {
		maxChars = 200
	}
	abstract := p.Content
	if len(abstract) > maxChars {
		abstract = abstract[:maxChars]
	}
	return Summary{
		URL:      p.URL,
		Abstract: abstract,
	}
}

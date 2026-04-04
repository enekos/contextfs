package scraper

type Page struct {
	URL     string
	Title   string
	Content string
}

type Summary struct {
	URL      string
	Abstract string
}

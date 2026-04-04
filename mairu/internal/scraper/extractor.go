package scraper

import (
	"regexp"
	"strings"
)

var (
	reTitle = regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	reTags  = regexp.MustCompile(`(?is)<[^>]+>`)
	reSpace = regexp.MustCompile(`\s+`)
)

func ExtractPage(url, html string) Page {
	title := ""
	if m := reTitle.FindStringSubmatch(html); len(m) > 1 {
		title = strings.TrimSpace(m[1])
	}
	text := strings.TrimSpace(reSpace.ReplaceAllString(reTags.ReplaceAllString(html, " "), " "))
	return Page{URL: url, Title: title, Content: text}
}

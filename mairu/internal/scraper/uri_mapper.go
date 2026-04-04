package scraper

import (
	"net/url"
	"strings"
)

func MapURLToContextURI(project, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "contextfs://" + project + "/invalid"
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		path = "root"
	}
	path = strings.ReplaceAll(path, "/", "-")
	return "contextfs://" + project + "/web/" + u.Host + "/" + path
}

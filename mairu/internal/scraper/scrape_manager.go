package scraper

type Manager struct {
	cache *Cache
}

func NewManager(cache *Cache) *Manager {
	if cache == nil {
		cache = NewCache()
	}
	return &Manager{cache: cache}
}

func (m *Manager) ProcessHTML(url, html string) (Page, Summary) {
	if cached, ok := m.cache.Get(url); ok {
		return cached, SummarizePage(cached, 200)
	}
	page := ExtractPage(url, html)
	m.cache.Put(page)
	return page, SummarizePage(page, 200)
}

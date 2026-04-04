package scraper

import "sync"

type Cache struct {
	mu   sync.Mutex
	data map[string]Page
}

func NewCache() *Cache {
	return &Cache{data: map[string]Page{}}
}

func (c *Cache) Get(url string) (Page, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data[url]
	return v, ok
}

func (c *Cache) Put(p Page) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[p.URL] = p
}

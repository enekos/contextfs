package llm

import (
	"sync"
)

// EmbeddingCache is a bounded, concurrency-safe in-memory cache for query
// embeddings.  It uses a simple LRU eviction strategy backed by a doubly-
// linked list so that both lookup and eviction are O(1).
//
// Typical embedding calls cost ~10 ms and a network round-trip to the
// Gemini API.  Caching the last N query vectors eliminates redundant calls
// for repeated or near-identical queries within a session.
type EmbeddingCache struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*cacheNode
	head     *cacheNode // most-recently-used sentinel
	tail     *cacheNode // least-recently-used sentinel
}

type cacheNode struct {
	key   string
	value []float32
	prev  *cacheNode
	next  *cacheNode
}

// NewEmbeddingCache creates a cache with the given capacity.
// capacity ≤ 0 disables caching (Get always misses, Put is a no-op).
func NewEmbeddingCache(capacity int) *EmbeddingCache {
	head := &cacheNode{}
	tail := &cacheNode{}
	head.next = tail
	tail.prev = head
	return &EmbeddingCache{
		capacity: capacity,
		items:    make(map[string]*cacheNode, capacity),
		head:     head,
		tail:     tail,
	}
}

// Get returns the cached embedding for key and true, or nil, false if absent.
func (c *EmbeddingCache) Get(key string) ([]float32, bool) {
	if c.capacity <= 0 {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	node, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.moveToFront(node)
	return node.value, true
}

// Put stores the embedding for key, evicting the least-recently-used entry
// if the cache is at capacity.
func (c *EmbeddingCache) Put(key string, value []float32) {
	if c.capacity <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if node, ok := c.items[key]; ok {
		node.value = value
		c.moveToFront(node)
		return
	}
	node := &cacheNode{key: key, value: value}
	c.items[key] = node
	c.insertFront(node)
	if len(c.items) > c.capacity {
		lru := c.tail.prev
		c.removeNode(lru)
		delete(c.items, lru.key)
	}
}

func (c *EmbeddingCache) insertFront(node *cacheNode) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

func (c *EmbeddingCache) removeNode(node *cacheNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (c *EmbeddingCache) moveToFront(node *cacheNode) {
	c.removeNode(node)
	c.insertFront(node)
}

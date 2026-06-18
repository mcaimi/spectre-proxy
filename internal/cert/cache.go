package cert

import (
	"crypto/tls"
	"sync"
	"time"
)

type cacheEntry struct {
	cert      *tls.Certificate
	expiresAt time.Time
}

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxSize,
	}
}

func (c *Cache) Get(hostname string) (*tls.Certificate, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[hostname]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.cert, true
}

func (c *Cache) Set(hostname string, cert *tls.Certificate, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[hostname] = &cacheEntry{
		cert:      cert,
		expiresAt: expiresAt,
	}
}

func (c *Cache) Delete(hostname string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, hostname)
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	first := true
	for key, entry := range c.entries {
		if first || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0

	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
			count++
		}
	}

	return count
}

package services

import (
	"sync"
	"time"
)

// ThemeCache is a thread-safe in-memory cache for generated theme CSS
type ThemeCache struct {
	mu    sync.RWMutex
	items map[string]*CacheItem
	ttl   time.Duration
}

// CacheItem represents a cached CSS string with expiration
type CacheItem struct {
	CSS       string
	ExpiresAt time.Time
}

// NewThemeCache creates a new theme cache with the specified TTL
func NewThemeCache(ttl time.Duration) *ThemeCache {
	cache := &ThemeCache{
		items: make(map[string]*CacheItem),
		ttl:   ttl,
	}

	// Start background cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached CSS string if it exists and hasn't expired
func (c *ThemeCache) Get(themeID string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[themeID]
	if !exists {
		return "", false
	}

	// Check if expired
	if time.Now().After(item.ExpiresAt) {
		return "", false
	}

	return item.CSS, true
}

// Set stores a CSS string in the cache with TTL
func (c *ThemeCache) Set(themeID, css string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[themeID] = &CacheItem{
		CSS:       css,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a cached item
func (c *ThemeCache) Delete(themeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, themeID)
}

// Clear removes all cached items
func (c *ThemeCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
}

// Size returns the number of cached items
func (c *ThemeCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// cleanupExpired runs periodically to remove expired items
func (c *ThemeCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()

		for id, item := range c.items {
			if now.After(item.ExpiresAt) {
				delete(c.items, id)
			}
		}

		c.mu.Unlock()
	}
}

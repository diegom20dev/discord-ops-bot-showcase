package platform

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type CacheEntry struct {
	Value     string
	ExpiresAt time.Time
}

type InMemoryCache struct {
	mu    sync.RWMutex
	store map[string]CacheEntry
}

func NewInMemoryCache() *InMemoryCache {
	cache := &InMemoryCache{
		store: make(map[string]CacheEntry),
	}
	go cache.cleanupExpired()
	return cache
}

func (c *InMemoryCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.store[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}

	if time.Now().After(entry.ExpiresAt) {
		return "", fmt.Errorf("key expired")
	}

	return entry.Value, nil
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
	return nil
}

func (c *InMemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.store {
			if now.After(entry.ExpiresAt) {
				delete(c.store, key)
			}
		}
		c.mu.Unlock()
	}
}

// Cache keys generator
func GetCaptureKey(captureID string) string {
	return fmt.Sprintf("capture:%s", captureID)
}

func GetCaptureSummaryKey(captureID string) string {
	return fmt.Sprintf("summary:%s", captureID)
}

func GetUserInboxKey(userID string) string {
	return fmt.Sprintf("inbox:%s", userID)
}

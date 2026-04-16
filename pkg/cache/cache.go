package cache

import (
	"context"
	"sync"
	"time"
)

type entry[T any] struct {
	value     T
	expiresAt time.Time
}

type MemoryCache[T any] struct {
	data map[string]entry[T]
	mu   sync.RWMutex
}

func NewMemoryCache[T any]() *MemoryCache[T] {
	return &MemoryCache[T]{
		data: make(map[string]entry[T]),
	}
}

func (c *MemoryCache[T]) Get(ctx context.Context, key string) (T, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()

	var zero T

	if !ok {
		return zero, false
	}

	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()

		return zero, false
	}

	return e.value, true
}

func (c *MemoryCache[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.data[key] = entry[T]{
		value:     value,
		expiresAt: expiresAt,
	}

	return nil
}

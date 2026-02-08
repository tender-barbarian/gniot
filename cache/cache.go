package cache

import (
	"context"
	"sync"

	"github.com/tender-barbarian/gniot/repository"
	gocrud "github.com/tender-barbarian/go-crud"
)

type Cache[M gocrud.Model] struct {
	mu      sync.Map
	cache   []M
	cacheMu sync.RWMutex
}

func NewCache[M gocrud.Model]() *Cache[M] {
	return &Cache[M]{}
}

func (c *Cache[M]) InvalidateCache(context.Context) {
	c.cacheMu.Lock()
	c.cache = nil
	c.cacheMu.Unlock()
	c.mu.Range(func(key, _ any) bool {
		c.mu.Delete(key)
		return true
	})
}

func (c *Cache[M]) GetIDByName(ctx context.Context, qr repository.Querier, table, name string) (int, error) {
	key := table + ":" + name

	// Fast path: check sync.Map cache
	if id, ok := c.mu.Load(key); ok {
		return id.(int), nil
	}

	// Cache miss: query DB
	id, err := qr.GetIDByName(ctx, table, name)
	if err != nil {
		return 0, err
	}

	// Store in cache
	c.mu.Store(key, id)
	return id, nil
}

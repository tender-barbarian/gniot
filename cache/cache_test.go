package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tender-barbarian/gniotek/repository/models"
)

type mockQuerier struct {
	nameToID  map[string]int
	err       error
	callCount int
}

func (m *mockQuerier) GetIDByName(_ context.Context, table, name string) (int, error) {
	m.callCount++
	if m.err != nil {
		return 0, m.err
	}
	id, ok := m.nameToID[table+":"+name]
	if !ok {
		return 0, fmt.Errorf("'%s' not found in %s", name, table)
	}
	return id, nil
}

func TestNewCache(t *testing.T) {
	c := NewCache[*models.Device]()
	require.NotNil(t, c)
	assert.Nil(t, c.cache)
}

func TestGetIDByName(t *testing.T) {
	ctx := context.Background()

	t.Run("cache miss queries DB and returns ID", func(t *testing.T) {
		qr := &mockQuerier{nameToID: map[string]int{"devices:sensor1": 42}}
		c := NewCache[*models.Device]()

		id, err := c.GetIDByName(ctx, qr, "devices", "sensor1")

		require.NoError(t, err)
		assert.Equal(t, 42, id)
		assert.Equal(t, int(1), qr.callCount)
	})

	t.Run("cache hit returns cached value without DB query", func(t *testing.T) {
		qr := &mockQuerier{nameToID: map[string]int{"devices:sensor1": 42}}
		c := NewCache[*models.Device]()

		// First call populates cache
		_, _ = c.GetIDByName(ctx, qr, "devices", "sensor1")
		// Second call should hit cache
		id, err := c.GetIDByName(ctx, qr, "devices", "sensor1")

		require.NoError(t, err)
		assert.Equal(t, 42, id)
		assert.Equal(t, int(1), qr.callCount)
	})

	t.Run("DB error is propagated and value is not cached", func(t *testing.T) {
		qr := &mockQuerier{err: fmt.Errorf("db connection lost")}
		c := NewCache[*models.Device]()

		id, err := c.GetIDByName(ctx, qr, "devices", "sensor1")

		assert.Error(t, err)
		assert.Equal(t, 0, id)
		assert.Contains(t, err.Error(), "db connection lost")

		// Verify nothing was cached — next call should query DB again
		qr.err = nil
		qr.nameToID = map[string]int{"devices:sensor1": 7}
		id, err = c.GetIDByName(ctx, qr, "devices", "sensor1")
		require.NoError(t, err)
		assert.Equal(t, 7, id)
		assert.Equal(t, int(2), qr.callCount)
	})

	t.Run("different names are cached independently", func(t *testing.T) {
		qr := &mockQuerier{nameToID: map[string]int{
			"devices:sensor1": 1,
			"devices:sensor2": 2,
			"actions:toggle":  10,
		}}
		c := NewCache[*models.Device]()

		id1, err := c.GetIDByName(ctx, qr, "devices", "sensor1")
		require.NoError(t, err)
		id2, err := c.GetIDByName(ctx, qr, "devices", "sensor2")
		require.NoError(t, err)
		id3, err := c.GetIDByName(ctx, qr, "actions", "toggle")
		require.NoError(t, err)

		assert.Equal(t, 1, id1)
		assert.Equal(t, 2, id2)
		assert.Equal(t, 10, id3)
		assert.Equal(t, int(3), qr.callCount)
	})
}

func TestInvalidateCache(t *testing.T) {
	ctx := context.Background()

	t.Run("clears sync.Map so next lookup queries DB", func(t *testing.T) {
		qr := &mockQuerier{nameToID: map[string]int{"devices:sensor1": 42}}
		c := NewCache[*models.Device]()

		// Populate cache
		_, _ = c.GetIDByName(ctx, qr, "devices", "sensor1")
		assert.Equal(t, int(1), qr.callCount)

		c.InvalidateCache(ctx)

		// Should query DB again after invalidation
		id, err := c.GetIDByName(ctx, qr, "devices", "sensor1")
		require.NoError(t, err)
		assert.Equal(t, 42, id)
		assert.Equal(t, int(2), qr.callCount)
	})

	t.Run("clears slice cache", func(t *testing.T) {
		c := NewCache[*models.Device]()
		c.cache = []*models.Device{{ID: 1, Name: "test"}}

		c.InvalidateCache(ctx)

		assert.Nil(t, c.cache)
	})
}

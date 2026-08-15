package core_storage_cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_SetGet(t *testing.T) {

	c := Init()
	require.NotNil(t, c)

	var (
		key   = "key"
		value = 24
	)

	c.Set(key, value)

	result, err := c.Get(key)
	require.NoError(t, err)

	assert.Equal(t, value, result)

}

func TestCache_NotFound(t *testing.T) {

	c := Init()
	require.NotNil(t, c)

	result, err := c.Get("nope")
	require.ErrorIs(t, err, ErrNotFound)

	assert.Nil(t, result)
}

func TestCache_SetOverwrite(t *testing.T) {
	c := Init()

	c.Set("key", 1)
	c.Set("key", 2)

	result, err := c.Get("key")

	require.NoError(t, err)
	assert.Equal(t, 2, result)
}

func TestCache_DifferentTypes(t *testing.T) {
	c := Init()

	testTypes := []struct {
		key   string
		value any
	}{
		{"int", 1},
		{"string", "hello"},
		{"struct", struct{ ID int }{1}},
	}

	for _, tt := range testTypes {
		c.Set(tt.key, tt.value)

		result, err := c.Get(tt.key)

		require.NoError(t, err)
		assert.Equal(t, tt.value, result)
	}
}

// go test -race
func TestCache_ConcurrentAccess(t *testing.T) {
	c := Init()

	done := make(chan struct{})

	for i := 0; i < 100; i++ {
		go func(i int) {
			c.Set("key", i)
			_, _ = c.Get("key")
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

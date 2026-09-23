package cache

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformcache "github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func newCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := platformcache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	return New(client, true, nil), mr
}

func key(inputs any) Key {
	return Key{UserID: 1004, Version: 3, Locale: "fa", Today: civildate.MustParse("2026-09-23"), Scope: "day:2026-09-23", Inputs: inputs}
}

func TestKeyFormat(t *testing.T) {
	k, err := key([]any{"x"}).String()
	require.NoError(t, err)
	parts := strings.Split(k, ":")
	require.Len(t, parts, 8)
	assert.Equal(t, []string{"cycle-engine", "1004", "3", "fa", "2026-09-23", "day", "2026-09-23"}, parts[:7])
	assert.Len(t, parts[7], 32)

	other, err := key([]any{"y"}).String()
	require.NoError(t, err)
	assert.NotEqual(t, k, other, "inputs are part of the key")
}

func TestRememberMissThenHit(t *testing.T) {
	c, mr := newCache(t)
	ctx := context.Background()
	calls := 0
	compute := func() ([]byte, error) { calls++; return []byte(`{"a":1}`), nil }

	v, err := c.Remember(ctx, key(1), compute)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(v))
	v, err = c.Remember(ctx, key(1), compute)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(v))
	assert.Equal(t, 1, calls, "second call is a hit")

	k, _ := key(1).String()
	assert.True(t, mr.Exists("ritme-go:"+k), "stored under the prefixed key")
	assert.Equal(t, TTL, mr.TTL("ritme-go:"+k))

	_, err = c.Remember(ctx, key(2), compute)
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "different inputs miss")
}

func TestRememberFallsBackWhenRedisIsDown(t *testing.T) {
	c, mr := newCache(t)
	mr.Close()
	calls := 0
	v, err := c.Remember(context.Background(), key(1), func() ([]byte, error) { calls++; return []byte(`[]`), nil })
	require.NoError(t, err)
	assert.Equal(t, `[]`, string(v))
	assert.Equal(t, 1, calls)
}

func TestRememberPropagatesComputeErrorsAndStoresNothing(t *testing.T) {
	c, mr := newCache(t)
	boom := errors.New("boom")
	_, err := c.Remember(context.Background(), key(1), func() ([]byte, error) { return nil, boom })
	require.ErrorIs(t, err, boom)
	assert.Empty(t, mr.Keys())
}

func TestDisabledAlwaysComputes(t *testing.T) {
	mr := miniredis.RunT(t)
	client := platformcache.NewFromClient(redis.NewClient(&redis.Options{Addr: mr.Addr()}), "ritme-go:")
	for _, c := range []*Cache{New(client, false, nil), New(nil, true, nil), nil} {
		calls := 0
		for range 2 {
			_, err := c.Remember(context.Background(), key(1), func() ([]byte, error) { calls++; return []byte(`1`), nil })
			require.NoError(t, err)
		}
		assert.Equal(t, 2, calls)
		assert.False(t, c.Enabled())
	}
	assert.Empty(t, mr.Keys())
}

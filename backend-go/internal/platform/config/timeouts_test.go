package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M9-10: listener / request timeouts from env, defaults above the longest regular route.
func TestLoad_HTTPTimeouts(t *testing.T) {
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.Equal(t, HTTPTimeouts{
		Read: 15 * time.Second, Write: 120 * time.Second, Idle: 60 * time.Second,
		Request: 5 * time.Second, Media: time.Hour,
	}, cfg.HTTP.Timeouts)
	// The assistant SSE stream may run 90 s + 10 s slack (internal/assistant StreamTimeout/requestSlack).
	assert.Greater(t, cfg.HTTP.Timeouts.Write, 100*time.Second)

	env := minimal()
	env["HTTP_READ_TIMEOUT"] = "20s"
	env["HTTP_WRITE_TIMEOUT"] = "180"
	env["HTTP_IDLE_TIMEOUT"] = "2m"
	env["HTTP_REQUEST_TIMEOUT"] = "7s"
	env["HTTP_MEDIA_TIMEOUT"] = "30m"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Equal(t, HTTPTimeouts{
		Read: 20 * time.Second, Write: 180 * time.Second, Idle: 2 * time.Minute,
		Request: 7 * time.Second, Media: 30 * time.Minute,
	}, cfg.HTTP.Timeouts)

	env = minimal()
	env["HTTP_WRITE_TIMEOUT"] = "never"
	env["HTTP_REQUEST_TIMEOUT"] = "-1s"
	_, err = LoadFrom(lookup(env))
	require.Error(t, err)
	assert.ErrorContains(t, err, "HTTP_WRITE_TIMEOUT")
	assert.ErrorContains(t, err, "HTTP_REQUEST_TIMEOUT")

	assert.Equal(t, HTTPTimeouts{Read: 15 * time.Second, Write: 120 * time.Second, Idle: 60 * time.Second,
		Request: 5 * time.Second, Media: time.Hour}, HTTPTimeouts{}.WithDefaults())
	assert.Equal(t, 9*time.Second, HTTPTimeouts{Write: 9 * time.Second}.WithDefaults().Write)
}

// T-M9-10: Redis pool size and socket timeouts from env.
func TestLoad_RedisPool(t *testing.T) {
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.Equal(t, 20, cfg.Redis.PoolSize)
	assert.Equal(t, 2*time.Second, cfg.Redis.DialTimeout)
	assert.Equal(t, 2*time.Second, cfg.Redis.ReadTimeout)
	assert.Equal(t, 2*time.Second, cfg.Redis.WriteTimeout)

	env := minimal()
	env["REDIS_POOL_SIZE"] = "50"
	env["REDIS_DIAL_TIMEOUT"] = "5s"
	env["REDIS_READ_TIMEOUT"] = "500ms"
	env["REDIS_WRITE_TIMEOUT"] = "3"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Equal(t, 50, cfg.Redis.PoolSize)
	assert.Equal(t, 5*time.Second, cfg.Redis.DialTimeout)
	assert.Equal(t, 500*time.Millisecond, cfg.Redis.ReadTimeout)
	assert.Equal(t, 3*time.Second, cfg.Redis.WriteTimeout)

	env = minimal()
	env["REDIS_POOL_SIZE"] = "0"
	_, err = LoadFrom(lookup(env))
	assert.ErrorContains(t, err, "REDIS_POOL_SIZE")

	assert.Equal(t, Redis{PoolSize: 20, DialTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second,
		WriteTimeout: 2 * time.Second}, Redis{}.WithDefaults())
}

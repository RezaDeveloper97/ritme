package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Sharing(t *testing.T) {
	// Outside production: no pepper needed (the public development default), no web origin.
	env := minimal()
	env["APP_ENV"] = "local"
	cfg, err := LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.False(t, cfg.Sharing.PepperMissing(cfg.App))
	assert.Empty(t, cfg.Sharing.WebURL)

	// Production without a pepper starts but disables the codes (fail closed).
	cfg, err = LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.True(t, cfg.Sharing.PepperMissing(cfg.App))

	env = minimal()
	env["SHARE_CODE_PEPPER"] = strings.Repeat("s", MinSharePepperLen)
	env["SHARE_WEB_URL"] = "https://web.ritme.app/"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.False(t, cfg.Sharing.PepperMissing(cfg.App))
	assert.Equal(t, "https://web.ritme.app", cfg.Sharing.WebURL)

	// Refused in production: a short pepper, a plain-http origin.
	env = minimal()
	env["SHARE_CODE_PEPPER"] = "short"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "SHARE_CODE_PEPPER")
	env = minimal()
	env["SHARE_WEB_URL"] = "http://web.ritme.app"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "SHARE_WEB_URL")
}

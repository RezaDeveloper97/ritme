package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// T-M12-07 (S-13): production refuses to start on the committed contract Passport key; other envs may use it.
func TestLoad_ProductionRefusesContractKey(t *testing.T) {
	keys := filepath.Join("..", "..", "..", "contract", "fixtures", "keys")
	env := minimal()
	env["STORAGE_PATH"] = keys
	_, err := LoadFrom(lookup(env))
	require.ErrorContains(t, err, "contract test key")

	for _, name := range []string{"local", "testing", "contract", "staging"} {
		env["APP_ENV"] = name
		_, err = LoadFrom(lookup(env))
		require.NoError(t, err, name)
	}

	// Production with no key file in STORAGE_PATH is not this check's business (LoadKeys fails later).
	env = minimal()
	env["STORAGE_PATH"] = t.TempDir()
	_, err = LoadFrom(lookup(env))
	require.NoError(t, err)
}

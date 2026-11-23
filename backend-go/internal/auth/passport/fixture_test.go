package passport_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
)

var contractKeyDir = filepath.Join("..", "..", "..", "contract", "fixtures", "keys")

// T-M12-07 (S-13): the committed contract key is refused in production and only there.
func TestCheckProductionKey(t *testing.T) {
	contract, err := passport.LoadKeys(contractKeyDir)
	require.NoError(t, err)

	// The embedded constant really is the fixture key's fingerprint (a stale constant would let it through).
	fp, err := passport.Fingerprint(contract.Public)
	require.NoError(t, err)
	assert.Equal(t, passport.ContractKeyFingerprint, fp)

	require.ErrorIs(t, passport.CheckProductionKey(contract.Public, "production"), passport.ErrContractKey)
	for _, env := range []string{"local", "testing", "contract", "staging", "", "Production "} {
		assert.NoError(t, passport.CheckProductionKey(contract.Public, env), env)
	}

	// Any other key is fine in production.
	other, err := passport.Fingerprint(&testKey.PublicKey)
	require.NoError(t, err)
	assert.NotEqual(t, passport.ContractKeyFingerprint, other)
	assert.NoError(t, passport.CheckProductionKey(&testKey.PublicKey, "production"))
}

func TestCheckStorageKey(t *testing.T) {
	require.ErrorIs(t, passport.CheckStorageKey(contractKeyDir, "production"), passport.ErrContractKey)
	assert.NoError(t, passport.CheckStorageKey(contractKeyDir, "local"))
	assert.NoError(t, passport.CheckStorageKey(contractKeyDir, "testing"))

	dir := t.TempDir()
	writeKeys(t, dir, testKey, &testKey.PublicKey)
	assert.NoError(t, passport.CheckStorageKey(dir, "production"), "a real key boots")

	// No key file: left to LoadKeys (the guard fails the start-up with its own error).
	assert.NoError(t, passport.CheckStorageKey(t.TempDir(), "production"))
	assert.NoError(t, passport.CheckStorageKey("", "production"))

	// A key file that is not PEM is an error in production.
	bad := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bad, passport.PublicKeyFile), []byte("junk"), 0o600))
	assert.Error(t, passport.CheckStorageKey(bad, "production"))
}

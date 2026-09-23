package cache_test

// Every contract case of the cycle groups (cycle, period, cycle-sweep) must pass with the engine
// cache off and on: correctness never depends on the cache. The harness (cmd/contract) boots the
// Go API on its own database and Redis prefix; CYCLE_ENGINE_CACHE reaches it through the
// environment (routes_cycle.go).

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContractWithCacheOnAndOff(t *testing.T) {
	if os.Getenv("TEST_DB_ADMIN_DSN") == "" || os.Getenv("TEST_REDIS_ADDR") == "" {
		t.Skip("TEST_DB_ADMIN_DSN / TEST_REDIS_ADDR not set (run via make test-int)")
	}
	if testing.Short() {
		t.Skip("contract run skipped in -short mode")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	for _, mode := range []string{"off", "on"} {
		t.Run("cache_"+mode, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), "go", "run", "./cmd/contract", "diff", "--routes", "cycle,period,cycle-sweep")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "CYCLE_ENGINE_CACHE="+mode)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "contract diff with CYCLE_ENGINE_CACHE=%s:\n%s", mode, out)
			t.Logf("CYCLE_ENGINE_CACHE=%s: %s", mode, lastLine(out))
		})
	}
}

func lastLine(b []byte) string {
	s := string(b)
	for len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '\n' {
			return s[i+1:]
		}
	}
	return s
}

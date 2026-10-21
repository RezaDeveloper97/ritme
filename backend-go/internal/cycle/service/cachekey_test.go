package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M9-03 B1: an approved engine.* clinical-config version valid on the target day changes the cycle cache key; a
// vitals key does not.
func TestCacheKey_EngineConfigFingerprint(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	day := civildate.MustParse("2026-10-21")
	sn := &Snapshot{UserID: 7, From: day, To: day, Today: day}
	key := func() string {
		t.Helper()
		k, err := sn.cacheKey(context.Background(), "fa", "day:"+day.String(), false)
		require.NoError(t, err)
		s, err := k.String()
		require.NoError(t, err)
		return s
	}

	clinicalconfig.Reset()
	base := key()

	clinicalconfig.SetVersions([]clinicalconfig.Version{{
		Key: clinicalconfig.KeyBPCrisis, Version: 2, Value: clinicalconfig.Value{"systolic": 170},
		EffectiveFrom: civildate.MustParse("2026-10-01"),
	}})
	assert.Equal(t, base, key(), "a vitals key does not change the cycle key")

	clinicalconfig.SetVersions([]clinicalconfig.Version{{
		Key: clinicalconfig.KeyPeriodHardCap, Version: 2, Value: clinicalconfig.Value{"days": 10},
		EffectiveFrom: civildate.MustParse("2026-11-01"),
	}})
	assert.Equal(t, base, key(), "an engine version not yet valid on the target day keeps the key")

	clinicalconfig.SetVersions([]clinicalconfig.Version{{
		Key: clinicalconfig.KeyPeriodHardCap, Version: 2, Value: clinicalconfig.Value{"days": 10},
		EffectiveFrom: civildate.MustParse("2026-10-01"),
	}})
	v2 := key()
	assert.NotEqual(t, base, v2, "an engine.* version valid on the day changes the key")

	clinicalconfig.SetVersions([]clinicalconfig.Version{
		{Key: clinicalconfig.KeyPeriodHardCap, Version: 2, Value: clinicalconfig.Value{"days": 10}, EffectiveFrom: civildate.MustParse("2026-10-01")},
		{Key: clinicalconfig.KeyPeriodHardCap, Version: 3, Value: clinicalconfig.Value{"days": 11}, EffectiveFrom: civildate.MustParse("2026-10-15")},
	})
	assert.NotEqual(t, v2, key(), "a newer engine version changes the key again")
}

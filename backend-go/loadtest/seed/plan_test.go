package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var today = time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)

func TestModeCounts_2000(t *testing.T) {
	got := map[Mode]int{}
	for i := 0; i < 2000; i++ {
		got[ModeOf(i)]++
	}
	assert.Equal(t, map[Mode]int{ModeTTC: 300, ModePregnant: 100, ModePostpartum: 80, ModePill: 100, ModeCycle: 1420}, got)
}

func TestMonthsRangeAndMobile(t *testing.T) {
	lo, hi := 99, 0
	seen := map[string]bool{}
	for i := 0; i < 2000; i++ {
		m := MonthsOf(i)
		lo, hi = min(lo, m), max(hi, m)
		mob := Mobile(i)
		require.Len(t, mob, 11)
		require.False(t, seen[mob])
		seen[mob] = true
	}
	assert.Equal(t, MinMonths, lo)
	assert.Equal(t, MaxMonths, hi)
}

func TestBuild_Deterministic(t *testing.T) {
	for _, i := range []int{0, 15, 20, 24, 50} {
		assert.Equal(t, Build(i, today), Build(i, today), "user %d", i)
	}
}

func TestBuild_PerMode(t *testing.T) {
	ttc, preg, pp, pill, cyc := Build(0, today), Build(15, today), Build(20, today), Build(24, today), Build(50, today)
	require.Equal(t, ModeTTC, ttc.Mode)
	assert.NotEmpty(t, ttc.Fertility)
	hasBBT, hasLH := false, false
	for _, l := range ttc.Logs {
		hasBBT = hasBBT || l.BBT > 0
	}
	for _, f := range ttc.Fertility {
		hasLH = hasLH || f.LH == "positive"
	}
	assert.True(t, hasBBT && hasLH)

	require.NotNil(t, preg.LMP)
	assert.Equal(t, 280, int(preg.DueDate.Sub(*preg.LMP).Hours()/24))
	require.NotNil(t, pp.BirthDate)
	assert.True(t, pp.BirthDate.Before(today))
	require.NotNil(t, pill.PillStarted)
	assert.Len(t, pill.PillLogs, 61)
	for _, u := range []User{cyc, pill, pp, preg} {
		assert.Empty(t, u.Fertility, u.Mode)
		assert.NotEmpty(t, u.Periods, u.Mode)
	}
}

func TestBuild_HistoryAndLogsStayInWindow(t *testing.T) {
	for _, i := range []int{3, 7, 11, 40} {
		u := Build(i, today)
		for k, p := range u.Periods {
			assert.False(t, p.Start.After(today))
			if k > 0 {
				assert.True(t, p.Start.Before(u.Periods[k-1].Start), "newest first, strictly older")
			}
		}
		for _, l := range u.Logs {
			assert.False(t, l.Date.After(today))
		}
	}
}

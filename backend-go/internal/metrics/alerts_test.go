package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/metrics"
)

func TestExceeds_Boundaries(t *testing.T) {
	// OCR 12 % no / 12.01 % yes; MAE 2.5 no / 2.51 yes.
	assert.False(t, metrics.Exceeds(1200.0/10000.0, 0.12))
	assert.True(t, metrics.Exceeds(1201.0/10000.0, 0.12))
	assert.False(t, metrics.Exceeds(2.5, 2.5))
	assert.True(t, metrics.Exceeds(2.51, 2.5))
}

func TestDropVsMean_Boundaries(t *testing.T) {
	prior := []float64{0.10, 0.10, 0.10}
	d, ok := metrics.DropVsMean(0.0701, prior) // 29.9 % lower
	assert.True(t, ok)
	assert.False(t, metrics.Exceeds(d, 0.30))
	d, _ = metrics.DropVsMean(0.0699, prior) // 30.1 % lower
	assert.True(t, metrics.Exceeds(d, 0.30))
	_, ok = metrics.DropVsMean(0.05, nil)
	assert.False(t, ok)
	_, ok = metrics.DropVsMean(0.05, []float64{0, 0})
	assert.False(t, ok)
}

func TestRisingStreak(t *testing.T) {
	seven := []float64{1, 2, 3, 4, 5, 6, 7, 8}     // 7 rises
	eight := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}  // 8 rises
	broken := []float64{1, 2, 3, 4, 5, 6, 7, 8, 8} // flat last day
	assert.Equal(t, 7, metrics.RisingStreak(seven))
	assert.False(t, metrics.Exceeds(float64(metrics.RisingStreak(seven)), 7))
	assert.Equal(t, 8, metrics.RisingStreak(eight))
	assert.True(t, metrics.Exceeds(float64(metrics.RisingStreak(eight)), 7))
	assert.Equal(t, 0, metrics.RisingStreak(broken))
	assert.Equal(t, 0, metrics.RisingStreak(nil))
}

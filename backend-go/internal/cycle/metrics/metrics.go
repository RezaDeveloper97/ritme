// Package metrics is the port of CycleMetricsCalculator + CycleMetrics
// (backend/app/Services/HealthEngine/CycleMetricsCalculator.php, CycleMetrics.php): the three-layer
// (profile / calculated / effective) cycle length and period duration of task.md §6–9, §12.
//
// Rules: only is_confirmed rows count, oldest→newest. A cycle length is the gap between two
// consecutive confirmed starts, valid in 21–45 (outliers only raise flags). A period duration is
// end−start+1 of a closed period, valid in 2–10. The calculated value is the median of the last ≤3
// valid records (even count: PHP round() of the mean, half away from zero); the effective value is
// calculated → profile (0/NULL = missing, floored at 1) → default 28/5; with Profile.LengthsManual (B-N1-09)
// the profile comes first.
//
// # Test mapping (backend/tests/Unit/CycleMetricsCalculatorTest.php → TestCycleMetricsCalculator)
//
//	test_falls_back_to_system_default_when_no_data                     → same-named subtest
//	test_uses_median_of_two_valid_cycles_over_the_profile              → same-named subtest
//	test_single_valid_cycle_is_used_as_is_and_variability_is_null      → same-named subtest
//	test_uses_median_of_last_three_valid_cycles                        → same-named subtest
//	test_outlier_cycles_are_excluded_from_the_median                   → same-named subtest
//	test_period_duration_is_calculated_independently_of_cycle_length   → same-named subtest
//	test_open_and_overlong_periods_do_not_feed_the_duration_median     → same-named subtest
//	test_unconfirmed_periods_are_ignored                               → same-named subtest
//	test_regularity_reads_the_range_of_the_last_three_cycles           → same-named subtest
package metrics

import (
	"math"
	"slices"
	"sort"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

const (
	validCycleMin     = 21
	validCycleMax     = 45
	validDurationMin  = 2
	validDurationMax  = 10
	recentWindow      = 3
	variabilityWindow = 6

	// DefaultCycleLength is the system default when neither history nor profile has a value.
	DefaultCycleLength = 28
	// DefaultPeriodDuration is the system default period duration.
	DefaultPeriodDuration = 5
)

// Metrics is the immutable snapshot of CycleMetrics.php. Nullable PHP ints are *int.
type Metrics struct {
	ProfileCycleLength        *int
	ProfilePeriodDuration     *int
	CalculatedCycleLength     *int
	CalculatedPeriodDuration  *int
	EffectiveCycleLength      int
	EffectivePeriodDuration   int
	CycleLengthSource         enums.EffectiveSource
	PeriodDurationSource      enums.EffectiveSource
	ValidCyclesCount          int
	ValidPeriodDurationsCount int
	RegularityStatus          enums.RegularityStatus
	Variability               enums.CycleVariability
	// RecentValidCycleLengths are the last ≤3 valid lengths, oldest→newest.
	RecentValidCycleLengths []int
	// CycleVariabilityRange is max−min of the last ≤3 valid lengths; nil with fewer than 2 (§27).
	CycleVariabilityRange *int
	HasShortCycleOutlier  bool
	HasLongCycleOutlier   bool
	// OnlyOutlierHistory: cycle gaps exist but every one of them is an outlier (§28.3).
	OnlyOutlierHistory bool
}

// ProfileValues is the `profile_values` block.
type ProfileValues struct {
	CycleLength    *int `json:"cycle_length"`
	PeriodDuration *int `json:"period_duration"`
}

// CalculatedValues is the `calculated_values` block.
type CalculatedValues struct {
	CycleLength    *int `json:"cycle_length"`
	PeriodDuration *int `json:"period_duration"`
	BasedOnCycles  *int `json:"based_on_cycles"`
}

// EffectiveValues is the `effective_values` block.
type EffectiveValues struct {
	CycleLength          int                   `json:"cycle_length"`
	PeriodDuration       int                   `json:"period_duration"`
	Source               enums.EffectiveSource `json:"source"`
	PeriodDurationSource enums.EffectiveSource `json:"period_duration_source"`
}

// Layers is CycleMetrics::toApiArray(): the spec §19 three-layer values.
type Layers struct {
	ProfileValues    ProfileValues    `json:"profile_values"`
	CalculatedValues CalculatedValues `json:"calculated_values"`
	EffectiveValues  EffectiveValues  `json:"effective_values"`
}

// Layers returns the three-layer values block (CycleMetrics::toApiArray).
func (m Metrics) Layers() Layers {
	var basedOn *int
	if m.ValidCyclesCount > 0 {
		n := m.ValidCyclesCount
		basedOn = &n
	}
	return Layers{
		ProfileValues: ProfileValues{CycleLength: m.ProfileCycleLength, PeriodDuration: m.ProfilePeriodDuration},
		CalculatedValues: CalculatedValues{
			CycleLength:    m.CalculatedCycleLength,
			PeriodDuration: m.CalculatedPeriodDuration,
			BasedOnCycles:  basedOn,
		},
		EffectiveValues: EffectiveValues{
			CycleLength:          m.EffectiveCycleLength,
			PeriodDuration:       m.EffectivePeriodDuration,
			Source:               m.CycleLengthSource,
			PeriodDurationSource: m.PeriodDurationSource,
		},
	}
}

// ConfirmedOldestFirst returns the is_confirmed rows sorted by start ascending. The sort is stable
// (PHP 8 sortBy keeps the input order of equal starts).
func ConfirmedOldestFirst(histories []model.History) []model.History {
	out := make([]model.History, 0, len(histories))
	for _, h := range histories {
		if h.IsConfirmed {
			out = append(out, h)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].PeriodStart.Before(out[j].PeriodStart) })
	return out
}

// Calculate is CycleMetricsCalculator::calculate. histories may be in any order; profile nil = no
// profile row.
func Calculate(histories []model.History, profile *model.Profile) Metrics {
	confirmed := ConfirmedOldestFirst(histories)

	validLengths, short, long, onlyOutliers := cycleLengths(confirmed)
	validDurations := validPeriodDurations(confirmed)

	recentCycles := lastN(validLengths, recentWindow)
	recentDurations := lastN(validDurations, recentWindow)

	calculatedCycle := median(recentCycles)
	calculatedDuration := median(recentDurations)

	var profileCycle, profileDuration *int
	if profile != nil && profile.CycleDuration != nil && *profile.CycleDuration != 0 {
		v := *profile.CycleDuration
		profileCycle = &v
	}
	if profile != nil && profile.PeriodDuration != nil && *profile.PeriodDuration != 0 {
		v := *profile.PeriodDuration
		profileDuration = &v
	}

	resolve := resolveEffective
	if profile != nil && profile.LengthsManual {
		resolve = resolveManual // B-N1-09: the user set the lengths herself
	}
	effCycle, cycleSource := resolve(calculatedCycle, profileCycle, DefaultCycleLength)
	effDuration, durationSource := resolve(calculatedDuration, profileDuration, DefaultPeriodDuration)

	return Metrics{
		ProfileCycleLength:        profileCycle,
		ProfilePeriodDuration:     profileDuration,
		CalculatedCycleLength:     calculatedCycle,
		CalculatedPeriodDuration:  calculatedDuration,
		EffectiveCycleLength:      effCycle,
		EffectivePeriodDuration:   effDuration,
		CycleLengthSource:         cycleSource,
		PeriodDurationSource:      durationSource,
		ValidCyclesCount:          len(validLengths),
		ValidPeriodDurationsCount: len(validDurations),
		RegularityStatus:          enums.RegularityStatusFromCycleLengths(recentCycles),
		Variability:               variability(lastN(validLengths, variabilityWindow)),
		RecentValidCycleLengths:   slices.Clone(recentCycles),
		CycleVariabilityRange:     variabilityRange(recentCycles),
		HasShortCycleOutlier:      short,
		HasLongCycleOutlier:       long,
		OnlyOutlierHistory:        onlyOutliers,
	}
}

// cycleLengths: gaps between consecutive confirmed starts, split into valid lengths and outlier flags.
func cycleLengths(confirmed []model.History) (valid []int, short, long, onlyOutliers bool) {
	valid = []int{}
	total := 0
	for i := 1; i < len(confirmed); i++ {
		length := confirmed[i-1].PeriodStart.DiffDays(confirmed[i].PeriodStart)
		total++
		switch {
		case length < validCycleMin:
			short = true
		case length > validCycleMax:
			long = true
		default:
			valid = append(valid, length)
		}
	}
	return valid, short, long, total > 0 && len(valid) == 0
}

// validPeriodDurations: end−start+1 of closed confirmed periods within 2–10, oldest→newest.
func validPeriodDurations(confirmed []model.History) []int {
	out := []int{}
	for _, h := range confirmed {
		if !h.HasEnd() || h.PeriodEnd.Before(h.PeriodStart) {
			continue
		}
		duration := h.PeriodStart.DiffDays(h.PeriodEnd) + 1
		if duration >= validDurationMin && duration <= validDurationMax {
			out = append(out, duration)
		}
	}
	return out
}

func resolveEffective(calculated, profile *int, def int) (int, enums.EffectiveSource) {
	if calculated != nil {
		return *calculated, enums.EffectiveSourceRecentValidCycles
	}
	if profile != nil {
		return max(*profile, 1), enums.EffectiveSourceProfile
	}
	return def, enums.EffectiveSourceDefault
}

// resolveManual is the order when «خودکار از داده‌ها» is off (B-N1-09): profile → calculated → default.
func resolveManual(calculated, profile *int, def int) (int, enums.EffectiveSource) {
	if profile != nil {
		return max(*profile, 1), enums.EffectiveSourceProfile
	}
	return resolveEffective(calculated, nil, def)
}

// median of values (nil for none); an even count averages the middle pair with PHP round().
func median(values []int) *int {
	if len(values) == 0 {
		return nil
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mid := len(sorted) / 2
	var m int
	if len(sorted)%2 == 0 {
		m = int(phpround.Round(float64(sorted[mid-1]+sorted[mid])/2, 0))
	} else {
		m = sorted[mid]
	}
	return &m
}

func variabilityRange(recent []int) *int {
	if len(recent) < 2 {
		return nil
	}
	r := slices.Max(recent) - slices.Min(recent)
	return &r
}

// variability: population standard deviation of the lengths (fewer than 2 → regular).
func variability(lengths []int) enums.CycleVariability {
	if len(lengths) < 2 {
		return enums.CycleVariabilityRegular
	}
	sum := 0
	for _, l := range lengths {
		sum += l
	}
	mean := float64(sum) / float64(len(lengths))
	sq := 0.0
	for _, l := range lengths {
		sq += (float64(l) - mean) * (float64(l) - mean)
	}
	return enums.CycleVariabilityFromStdDev(math.Sqrt(sq / float64(len(lengths))))
}

func lastN(s []int, n int) []int {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

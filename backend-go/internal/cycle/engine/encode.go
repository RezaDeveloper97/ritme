package engine

import "strconv"

// The §2.35 JSON encoder (T-M9-21). It appends to a caller-owned buffer (the caller pools it; the engine keeps no
// package state), with a fixed key order:
//
//	date, cycle_day, main_phase, subphase, fertility_level, days_to_ovulation, days_to_period, days_late,
//	anchors{…8}, metrics{…4}, clinical_flags{…3}, [future_cycles], suggest_pregnancy_test, confidence,
//	confidence_reasons, data_quality, resolution_source, is_predicted, requires_user_input, warnings,
//	limits{…4} (E-24), future_k, [today_ui] (E-17)
//
// Every §2.35 key keeps its spec position; the additions (limits, future_k, today_ui) come after the last spec key.
// Dates are ISO YYYY-MM-DD; NoDay, Opt none, SuppressionNone, BannerNone and a null card or window are null.
// warnings and confidence_reasons are arrays in enum order, [] when empty. future_cycles is written only when
// Status.FutureCycles is non-nil (the today payload, State.Today): an empty list is [] (E-04, E-14, E-25), never
// null; Resolve and Range statuses (nil) omit the key. today_ui is written only when ui is non-nil.
// Diagnostics (ignored skips) are not part of the payload: §2.35 has no such field.

// AppendJSON appends the JSON object of st (and its today_ui block when ui is non-nil) to b.
func AppendJSON(b []byte, st *Status, ui *TodayUI) []byte {
	b = append(b, `{"date":`...)
	b = appendDay(b, st.Date)
	b = append(b, `,"cycle_day":`...)
	b = appendOptInt(b, st.CycleDay)
	b = appendKeyStr(b, `,"main_phase":`, st.MainPhase.String())
	b = appendKeyStr(b, `,"subphase":`, st.Subphase.String())
	b = appendKeyStr(b, `,"fertility_level":`, st.FertilityLevel.String())
	b = append(b, `,"days_to_ovulation":`...)
	b = appendOptInt(b, st.DaysToOvulation)
	b = append(b, `,"days_to_period":`...)
	b = appendOptInt(b, st.DaysToPeriod)
	b = append(b, `,"days_late":`...)
	b = appendOptInt(b, st.DaysLate)

	a := &st.Anchors
	b = append(b, `,"anchors":{"current_period_start":`...)
	b = appendDay(b, a.CurrentPeriodStart)
	b = appendKeyStr(b, `,"current_period_start_source":`, a.CurrentPeriodStartSource.String())
	b = append(b, `,"current_period_end":`...)
	b = appendDay(b, a.CurrentPeriodEnd)
	b = appendKeyStr(b, `,"current_period_end_source":`, a.CurrentPeriodEndSource.String())
	b = append(b, `,"current_period_end_is_confirmed":`...)
	b = strconv.AppendBool(b, a.CurrentPeriodEndIsConfirmed)
	b = append(b, `,"predicted_next_period_start":`...)
	b = appendDay(b, a.PredictedNextPeriodStart)
	b = append(b, `,"estimated_ovulation_date":`...)
	b = appendDay(b, a.EstimatedOvulationDate)
	b = appendKeyStr(b, `,"ovulation_source":`, a.OvulationSource.String())

	m := &st.Metrics
	b = append(b, `},"metrics":{"effective_cycle_length":`...)
	b = strconv.AppendInt(b, int64(m.EffectiveCycleLength), 10)
	b = append(b, `,"effective_period_length":`...)
	b = strconv.AppendInt(b, int64(m.EffectivePeriodLength), 10)
	b = append(b, `,"cycle_variability":`...)
	b = appendOptInt(b, m.CycleVariability)
	b = appendKeyStr(b, `,"cycle_regularity":`, m.CycleRegularity.String())

	f := &st.ClinicalFlags
	b = append(b, `},"clinical_flags":{"frequency":`...)
	if v, ok := f.Frequency.Get(); ok {
		b = appendStr(b, v.String())
	} else {
		b = append(b, "null"...)
	}
	b = append(b, `,"duration":`...)
	if v, ok := f.Duration.Get(); ok {
		b = appendStr(b, v.String())
	} else {
		b = append(b, "null"...)
	}
	b = append(b, `,"absent":`...)
	if v, ok := f.Absent.Get(); ok {
		b = strconv.AppendBool(b, v)
	} else {
		b = append(b, "null"...)
	}
	b = append(b, '}')

	if st.FutureCycles != nil {
		b = append(b, `,"future_cycles":[`...)
		for i := range st.FutureCycles {
			if i > 0 {
				b = append(b, ',')
			}
			b = appendFutureCycle(b, &st.FutureCycles[i])
		}
		b = append(b, ']')
	}

	b = append(b, `,"suggest_pregnancy_test":`...)
	b = strconv.AppendBool(b, st.SuggestPregnancyTest)
	b = appendKeyStr(b, `,"confidence":`, st.Confidence.String())
	b = append(b, `,"confidence_reasons":[`...)
	first := true
	for r := range st.ConfidenceReasons.All() {
		if !first {
			b = append(b, ',')
		}
		first = false
		b = appendStr(b, r.String())
	}
	b = append(b, ']')
	b = appendKeyStr(b, `,"data_quality":`, st.DataQuality.String())
	b = appendKeyStr(b, `,"resolution_source":`, st.ResolutionSource.String())
	b = append(b, `,"is_predicted":`...)
	b = strconv.AppendBool(b, st.IsPredicted())
	b = append(b, `,"requires_user_input":`...)
	b = strconv.AppendBool(b, st.RequiresUserInput)
	b = append(b, `,"warnings":[`...)
	first = true
	for w := range st.Warnings.All() {
		if !first {
			b = append(b, ',')
		}
		first = false
		b = appendStr(b, w.String())
	}
	b = append(b, ']')

	l := &st.Limits
	b = append(b, `,"limits":{"fertility_suppressed":`...)
	b = strconv.AppendBool(b, l.FertilitySuppressed)
	b = append(b, `,"suppression_reason":`...)
	b = appendEnumOrNull(b, l.SuppressionReason.String())
	b = append(b, `,"engine_suspended":`...)
	b = strconv.AppendBool(b, l.EngineSuspended)
	b = append(b, `,"prompt_pregnancy_mode":`...)
	b = strconv.AppendBool(b, l.PromptPregnancyMode)
	b = append(b, `},"future_k":`...)
	b = strconv.AppendInt(b, int64(st.FutureK), 10)

	if ui != nil {
		b = append(b, `,"today_ui":`...)
		b = appendTodayUI(b, ui)
	}
	return append(b, '}')
}

func appendFutureCycle(b []byte, c *FutureCycle) []byte {
	b = append(b, `{"k":`...)
	b = strconv.AppendInt(b, int64(c.K), 10)
	b = append(b, `,"period_start":`...)
	b = appendDay(b, c.PeriodStart)
	b = append(b, `,"period_end":`...)
	b = appendDay(b, c.PeriodEnd)
	b = append(b, `,"fertile_start":`...)
	b = appendDay(b, c.FertileStart)
	b = append(b, `,"fertile_end":`...)
	b = appendDay(b, c.FertileEnd)
	b = append(b, `,"ovulation":`...)
	b = appendDay(b, c.Ovulation)
	b = appendKeyStr(b, `,"ovulation_source":`, c.OvulationSource.String())
	b = appendKeyStr(b, `,"confidence":`, c.Confidence.String())
	return append(b, '}')
}

func appendTodayUI(b []byte, ui *TodayUI) []byte {
	b = appendKeyStr(b, `{"ring_action":`, ui.RingAction.String())
	b = append(b, `,"banner":`...)
	b = appendEnumOrNull(b, ui.Banner.String())
	b = append(b, `,"show_pregnancy_test_button":`...)
	b = strconv.AppendBool(b, ui.ShowPregnancyTestButton)
	b = append(b, `,"prediction_card":`...)
	c, ok := ui.PredictionCard.Get()
	if !ok {
		b = append(b, "null"...)
		return append(b, '}')
	}
	b = append(b, `{"period":`...)
	b = appendRange(b, c.Period)
	b = append(b, `,"pms":`...)
	b = appendRange(b, c.PMS)
	b = append(b, `,"fertile":`...)
	if r, ok := c.Fertile.Get(); ok {
		b = appendRange(b, r)
	} else {
		b = append(b, "null"...)
	}
	b = append(b, `,"ovulation":`...)
	if o, ok := c.Ovulation.Get(); ok {
		b = append(b, `{"date":`...)
		b = appendDay(b, o.Date)
		b = appendKeyStr(b, `,"source":`, o.Source.String())
		b = append(b, '}')
	} else {
		b = append(b, "null"...)
	}
	b = append(b, `,"cycle_length":`...)
	b = strconv.AppendInt(b, int64(c.CycleLength), 10)
	b = appendKeyStr(b, `,"cycle_regularity":`, c.CycleRegularity.String())
	return append(b, "}}"...)
}

func appendRange(b []byte, r DayRange) []byte {
	b = append(b, `{"start":`...)
	b = appendDay(b, r.Start)
	b = append(b, `,"end":`...)
	b = appendDay(b, r.End)
	return append(b, '}')
}

// appendDay writes "YYYY-MM-DD", or null for NoDay and a day outside the years 0..9999 (not representable in ISO).
func appendDay(b []byte, d Day) []byte {
	n := len(b)
	b = append(b, '"')
	if c := d.AppendISO(b); len(c) > len(b) {
		return append(c, '"')
	}
	return append(b[:n], "null"...)
}

func appendOptInt(b []byte, o Opt[int]) []byte {
	if v, ok := o.Get(); ok {
		return strconv.AppendInt(b, int64(v), 10)
	}
	return append(b, "null"...)
}

// appendStr writes an enum spelling; the spellings are ASCII identifiers, so nothing needs escaping.
func appendStr(b []byte, s string) []byte {
	b = append(b, '"')
	b = append(b, s...)
	return append(b, '"')
}

func appendKeyStr(b []byte, key, s string) []byte { return appendStr(append(b, key...), s) }

// appendEnumOrNull writes an enum whose "none" member spells "" (SuppressionNone, BannerNone) as null.
func appendEnumOrNull(b []byte, s string) []byte {
	if s == "" {
		return append(b, "null"...)
	}
	return appendStr(b, s)
}

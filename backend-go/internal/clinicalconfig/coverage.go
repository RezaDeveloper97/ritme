package clinicalconfig

// Coverage keys (T-M10-16, SPEC v6 §3.1–§3.5, §5.8; DECISIONS.md H-07, H-08, H-09, H-12, H-17). Version 1 is the
// constant or seed each key replaces (today's behaviour), or the SPEC table for a key nothing read before. Nothing
// reads these keys yet: the consumers switch in T-M10-18 … T-M10-22. TestCoverageKeys_PinnedToTheirOrigin names the
// file:line each value comes from.

// Keys.
const (
	KeyPregnancyEDD             = "pregnancy.edd"
	KeyPregnancyEDDCycleCorrect = "pregnancy.edd.cycle_correction"
	KeyPregnancyTrimester       = "pregnancy.trimester"
	KeyPregnancyRedating        = "pregnancy.redating" // rows
	KeyPregnancyIOM             = "pregnancy.iom"      // rows
	KeyPregnancyIOMFirstTri     = "pregnancy.iom.first_trimester"
	KeyPregnancySpotting        = "pregnancy.spotting"
	KeyEPDSSchedule             = "epds.schedule"

	KeyLabHb              = "lab.hb"
	KeyLabFerritinInflame = "lab.ferritin.inflamed"
	KeyLabVitD            = "lab.vitd"
	KeyLabTSHPregnancy    = "lab.tsh.pregnancy"
	KeyLabHbA1c           = "lab.hba1c"
	KeyLabOCR             = "lab.ocr"
	KeyLabBorderline      = "lab.borderline"

	KeyBMIWHO          = "bmi.who"
	KeyGlucoseContexts = "glucose.contexts"
	KeyGlucoseDiabetic = "glucose.diabetic"

	KeyChildGrowth          = "child.growth"
	KeyChildVaccineSchedule = "child.vaccine_schedule" // rows
	KeyScreeningUSPSTF      = "screening.uspstf"       // rows

	// Postpartum: the day after the birth from which the log sheet offers `flow` and the bleeding-log bridge builds
	// periods again; earlier bleeding is lochia (T-M9-28, DECISIONS.md I-05, H-14 advisor-pending).
	KeyPostpartumFlowFromDay = "postpartum.flow_from_day"
)

const (
	srcEDD        = "Naegele's rule, LMP + 280 days; correction by cycle length optional, off by default (ACOG Committee Opinion 700, Methods for Estimating the Due Date, 2017; SPEC v6 §3.1)"
	srcTrimester  = "ACOG Committee Opinion 700 (2017): first trimester to 13+6, second 14+0 to 27+6, third from 28+0 (SPEC v6 §3.1)"
	srcRedating   = "ACOG CO 700 (2017) Table 1: re-date to the scan when the LMP–US difference is more than the limit; the band is the GA by LMP on the scan day (footnote: based on LMP), = SPEC v6 §3.1 «GA at the scan»" //nolint:gosec // G101: a citation, not a credential
	srcIOM        = "IOM 2009, Weight Gain During Pregnancy, Table S-1 (singleton; weekly = 2nd/3rd trimester kg/week; first-trimester gain 0.5–2 kg); SPEC v6 §3.1"
	srcSpotting   = "Ritme pregnancy alerts v1 (spotting is urgent until week 12, critical_symptom.spotting_until_week); SPEC v6 §3.1 has no week limit"
	srcEPDSPlan   = "Cox JL et al. 1987 (EPDS, full check every 2 weeks, SPEC v6 §3.1); Ritme short check-in (EPDS-3) every 7 days"
	srcHb         = "WHO 2024 guideline on haemoglobin cutoffs (non-pregnant < 12, pregnant < 11, 2nd trimester < 10.5 g/dL); SPEC v6 §3.3"
	srcFerritinIn = "WHO 2020 guideline on serum ferritin (< 70 ng/mL with inflammation); SPEC v6 §3.3"
	srcVitD       = "IOM 2011 Dietary Reference Intakes for Calcium and Vitamin D (< 12 deficient, ≥ 20 sufficient ng/mL); SPEC v6 §3.3"
	srcTSH        = "Alexander EK et al., 2017 ATA thyroid & pregnancy guideline, Thyroid 2017;27:315–89 (upper 4.0 mIU/L without trimester ranges); SPEC v6 §3.3"
	srcHbA1c      = "ADA Standards of Care 2025 §2 (HbA1c 5.7–6.4 % prediabetes, ≥ 6.5 % diabetes); SPEC v6 §3.2"
	srcOCR        = "Ritme product rule (DECISIONS.md H-08, user-accepted): values read below 0.9 confidence wait for the user's confirmation; SPEC v6 §3.3"
	srcBorderline = "Ritme lab statuses v1 (borderline = within 10 % outside the range, DECISIONS.md H-07 advisor-pending; enabled 0 removes the class)"
	srcBMI        = "WHO BMI classification (< 18.5 underweight, 25 overweight, 30 obese; a boundary lands in the higher band); SPEC v6 §3.2"
	srcGlucoseCtx = "DECISIONS.md H-09 (advisor-pending): 0 = today's post-meal bands 140/200 for post-meal, random and bedtime; 1 = SPEC v6 §3.2 rule"
	srcGlucoseDM  = "ADA Standards of Care 2025 §6: preprandial (fasting / pre-meal) 80–130, peak post-meal < 180 mg/dL; replaceable by the doctor's target (SPEC v6 §3.2)"
	srcGrowth     = "WHO Child Growth Standards (normal band P3–P97); crossing 2 major percentile lines between two measurements → discuss (SPEC v6 §3.5)"
	srcVaccines   = "وزارت بهداشت — برنامه کشوری ایمن\u200cسازی (today's child_vaccines catalog ages, migration 00031); a child follows the version in force on the birth date (H-17)"
	srcPPFlow     = "ACOG Committee Opinion 736, Optimizing Postpartum Care (2018): the puerperium (lochia) lasts about 6 weeks; flow logged from day 42 is a period (DECISIONS.md I-05, H-14 advisor-pending)"
	srcScreening  = "USPSTF cervical 2018, breast 2024, hypertension 2021, prediabetes/diabetes 2021; American Dental Association (SPEC v6 §3.4)"
)

// Screening test kinds of a cervical record (checkup_records.test_kind, T-M10-17).
var screeningTestKinds = []string{"cytology", "hpv", "cotest"}

var coverageDefs = []Def{
	// Pregnancy dating (consumers: T-M10-18).
	{Key: KeyPregnancyEDD, Group: GroupPregnancy, Unit: "day", Source: srcEDD, Fields: one("naegele_days", 280, 266, 294)},
	{Key: KeyPregnancyEDDCycleCorrect, Group: GroupPregnancy, Unit: "flag", Source: srcEDD, Fields: one("enabled", 0, 0, 1)},
	{Key: KeyPregnancyTrimester, Group: GroupPregnancy, Unit: "day", Source: srcTrimester,
		Fields: []Field{{"t2_from_days", 98, 84, 112}, {"t3_from_days", 196, 182, 210}},
		Order:  [][2]string{{"t2_from_days", "t3_from_days"}}},
	// ga_until_days: the last day (inclusive, GA by LMP on the scan day) of the band; null = open (28+0 on).
	// max_diff_days: the LMP dating is replaced when |GA by scan − GA by LMP| is more than this.
	{Key: KeyPregnancyRedating, Group: GroupPregnancy, Unit: "day", Source: srcRedating, Kind: KindRows,
		Table: &Table{
			Columns: []Column{
				{Name: "ga_until_days", Type: ColNumber, Integer: true, Min: 0, Max: 300, Nullable: true},
				{Name: "max_diff_days", Type: ColNumber, Integer: true, Min: 1, Max: 42},
			},
			MinRows: 1, MaxRows: 12, SortBy: "ga_until_days", StrictSort: true,
			Default: [][]any{{62, 5}, {97, 7}, {111, 7}, {153, 10}, {195, 14}, {nil, 21}},
		}},
	// bmi_from inclusive, bmi_to exclusive (null = open); total kg; weekly kg/week in the 2nd and 3rd trimester.
	{Key: KeyPregnancyIOM, Group: GroupPregnancy, Unit: "kg/m² · kg · kg/week", Source: srcIOM, Kind: KindRows,
		Table: &Table{
			Columns: []Column{
				{Name: "bmi_from", Type: ColNumber, Min: 0, Max: 60},
				{Name: "bmi_to", Type: ColNumber, Min: 10, Max: 60, Nullable: true},
				{Name: "total_lo", Type: ColNumber, Min: 0, Max: 40},
				{Name: "total_hi", Type: ColNumber, Min: 0, Max: 40},
				{Name: "weekly_lo", Type: ColNumber, Min: 0, Max: 2},
				{Name: "weekly_hi", Type: ColNumber, Min: 0, Max: 2},
			},
			MinRows: 1, MaxRows: 10, SortBy: "bmi_from", StrictSort: true,
			Range:    &Range{From: "bmi_from", To: "bmi_to", Contiguous: true},
			RowOrder: [][2]string{{"total_lo", "total_hi"}, {"weekly_lo", "weekly_hi"}},
			Default: [][]any{
				{0, 18.5, 12.5, 18, 0.44, 0.58},
				{18.5, 25, 11.5, 16, 0.35, 0.50},
				{25, 30, 7, 11.5, 0.23, 0.33},
				{30, nil, 5, 9, 0.17, 0.27},
			},
		}},
	{Key: KeyPregnancyIOMFirstTri, Group: GroupPregnancy, Unit: "kg", Source: srcIOM,
		Fields: []Field{{"lo", 0.5, 0, 5}, {"hi", 2, 0, 5}}, Order: [][2]string{{"lo", "hi"}}},
	{Key: KeyPregnancySpotting, Group: GroupPregnancy, Unit: "week", Source: srcSpotting, Fields: one("until_week", 12, 0, 42)},
	{Key: KeyEPDSSchedule, Group: GroupPregnancy, Unit: "day", Source: srcEPDSPlan,
		Fields: []Field{{"full_days", 14, 7, 56}, {"short_days", 7, 3, 28}},
		Order:  [][2]string{{"short_days", "full_days"}}},

	// Labs (consumers: T-M10-20).
	{Key: KeyLabHb, Group: GroupLab, Unit: "g/dL", Source: srcHb,
		Fields: []Field{{"nonpregnant_low", 12, 8, 15}, {"pregnant_low", 11, 8, 15}, {"pregnant_t2_low", 10.5, 8, 15}}},
	{Key: KeyLabFerritinInflame, Group: GroupLab, Unit: "ng/mL", Source: srcFerritinIn, Fields: one("low", 70, 15, 200)},
	{Key: KeyLabVitD, Group: GroupLab, Unit: "ng/mL", Source: srcVitD,
		Fields: []Field{{"deficient_below", 12, 5, 50}, {"sufficient_from", 20, 5, 50}},
		Order:  [][2]string{{"deficient_below", "sufficient_from"}}},
	{Key: KeyLabTSHPregnancy, Group: GroupLab, Unit: "mIU/L", Source: srcTSH, Fields: one("upper", 4.0, 2, 10)},
	{Key: KeyLabHbA1c, Group: GroupLab, Unit: "%", Source: srcHbA1c,
		Fields: []Field{{"prediabetes_from", 5.7, 5, 8}, {"diabetes_from", 6.5, 5, 8}},
		Order:  [][2]string{{"prediabetes_from", "diabetes_from"}}},
	{Key: KeyLabOCR, Group: GroupLab, Unit: "confidence", Source: srcOCR, Fields: one("min_confidence", 0.9, 0.5, 1)},
	{Key: KeyLabBorderline, Group: GroupLab, Unit: "flag / fraction", Source: srcBorderline,
		Fields: []Field{{"enabled", 1, 0, 1}, {"margin", 0.10, 0, 0.5}}},

	// Vitals (consumers: T-M10-19; bmi.who also T-M10-21).
	{Key: KeyBMIWHO, Group: GroupVitals, Unit: "kg/m²", Source: srcBMI,
		Fields: []Field{{"underweight_below", 18.5, 15, 22}, {"overweight_from", 25, 22, 30}, {"obese_from", 30, 27, 40}},
		Order:  [][2]string{{"underweight_below", "overweight_from"}, {"overweight_from", "obese_from"}}},
	{Key: KeyGlucoseContexts, Group: GroupVitals, Unit: "flag", Source: srcGlucoseCtx, Fields: one("spec", 0, 0, 1)},
	{Key: KeyGlucoseDiabetic, Group: GroupVitals, Unit: "mg/dL", Source: srcGlucoseDM,
		Fields: []Field{{"fasting_lo", 80, 60, 120}, {"fasting_hi", 130, 100, 200}, {"post_meal_max", 180, 120, 250}},
		Order:  [][2]string{{"fasting_lo", "fasting_hi"}}},

	// Children (consumers: T-M10-22).
	{Key: KeyChildGrowth, Group: GroupChild, Unit: "percentile / lines", Source: srcGrowth,
		Fields: []Field{{"low_pct", 3, 0.1, 15}, {"high_pct", 97, 85, 99.9}, {"cross_lines", 2, 1, 4}}},
	// vaccine_code = the child_vaccines catalog code (child_vaccine_doses.dose_code); dose = the ordinal of that code
	// within its vaccine (the code prefix before "_": opv_0, opv_m4, opv_booster_1, opv_booster_2 are opv 1…4).
	{Key: KeyChildVaccineSchedule, Group: GroupChild, Unit: "month", Source: srcVaccines, Kind: KindRows,
		Table: &Table{
			Columns: []Column{
				{Name: "vaccine_code", Type: ColString, MaxLen: 64},
				{Name: "dose", Type: ColNumber, Integer: true, Min: 1, Max: 10},
				{Name: "age_months", Type: ColNumber, Integer: true, Min: 0, Max: 216},
			},
			MinRows: 1, MaxRows: 100, SortBy: "age_months", Unique: []string{"vaccine_code"},
			Default: [][]any{
				{"bcg", 1, 0}, {"hepb_1", 1, 0}, {"opv_0", 1, 0},
				{"penta_1", 1, 2}, {"polio_m2", 1, 2}, {"pcv_1", 1, 2}, {"rota_1", 1, 2},
				{"penta_2", 2, 4}, {"opv_m4", 2, 4}, {"pcv_2", 2, 4}, {"rota_2", 2, 4},
				{"penta_3", 3, 6}, {"polio_m6", 2, 6}, {"pcv_3", 3, 6},
				{"mmr_1", 1, 12}, {"pcv_booster", 4, 12},
				{"dtp_booster_1", 1, 18}, {"opv_booster_1", 3, 18}, {"mmr_2", 2, 18},
				{"dtp_booster_2", 2, 72}, {"opv_booster_2", 4, 72},
			},
		}},

	// Screening (consumer: T-M10-21). type_key = the checkup_types key; ages inclusive (null = open); last_test_kind
	// null = any; min_bmi null = no BMI condition. Bands of one type and test kind never overlap.
	{Key: KeyScreeningUSPSTF, Group: GroupScreening, Unit: "year / month / kg/m²", Source: srcScreening, Kind: KindRows,
		Table: &Table{
			Columns: []Column{
				{Name: "type_key", Type: ColString, MaxLen: 64},
				{Name: "age_min", Type: ColNumber, Integer: true, Min: 0, Max: 120, Nullable: true},
				{Name: "age_max", Type: ColNumber, Integer: true, Min: 0, Max: 120, Nullable: true},
				{Name: "interval_months", Type: ColNumber, Integer: true, Min: 1, Max: 240},
				{Name: "last_test_kind", Type: ColEnum, Enum: screeningTestKinds, Nullable: true},
				{Name: "min_bmi", Type: ColNumber, Min: 10, Max: 60, Nullable: true},
			},
			MinRows: 1, MaxRows: 50,
			Range: &Range{From: "age_min", To: "age_max", ToInclusive: true, GroupBy: []string{"type_key", "last_test_kind"}},
			Default: [][]any{
				{"pap_smear", 21, 29, 36, "cytology", nil},
				{"pap_smear", 30, 65, 36, "cytology", nil},
				{"pap_smear", 30, 65, 60, "hpv", nil},
				{"pap_smear", 30, 65, 60, "cotest", nil},
				{"mammography", 40, 74, 24, nil, nil},
				{"bp_screening", 18, 39, 36, nil, nil},
				{"bp_screening", 40, nil, 12, nil, nil},
				{"glucose_screening", 35, 70, 36, nil, 25},
				{"dentist", nil, nil, 6, nil, nil},
			},
		}},
	// Postpartum flow (T-M9-28, I-05, H-14): days after the birth (birth day = day 0) from which `flow` is offered in
	// postpartum mode and the bleeding-log bridge builds periods again (consumer: internal/healthlog).
	{Key: KeyPostpartumFlowFromDay, Group: GroupPregnancy, Unit: "day", Source: srcPPFlow, Fields: one("day", 42, 21, 84)},
}

package revision

// EngineTables are the engine-input tables (docs/engine-v6/README.md §3): a write to any of them must bump the
// writer's engine revision in the same transaction.
var EngineTables = []string{
	"cycle_histories", "cycle_skips", "daily_health_logs", "health_log_entries", "fertility_logs", "user_profiles",
	"cycle_preferences", "user_life_profiles", "contraception_methods", "pregnancy_profiles", "postpartum_profiles",
}

// Entry is the manifest line of one sqlc write query on an engine table: either the integration test that proves
// the write request bumps engine_rev by exactly 1 (Bumps), or why the query needs no bump (Exempt). Exactly one is set.
type Entry struct {
	Bumps  string
	Exempt string
}

// Manifest lists every sqlc write query on an EngineTables table, keyed "<db/queries dir>.<query name>".
// TestManifest_CoversEveryEngineWrite fails when a write query is missing, an entry names a query that no longer
// exists, or a Bumps test does not exist. A new writer of an engine table adds its query here with the test that
// proves its bump — the transaction owner calls Bump (package doc).
var Manifest = map[string]Entry{
	// cycle_histories / user_profiles.last_period_start — the period button (periods.Service.tx bumps).
	"cycle.InsertPeriod":                 {Bumps: "TestBump_Periods"},
	"cycle.UpdatePeriod":                 {Bumps: "TestBump_Periods"},
	"cycle.DeleteEstimatesExcept":        {Bumps: "TestBump_Periods"},
	"cycle.DeletePeriod":                 {Bumps: "TestBump_Periods"},
	"cycle.UpdateProfileLastPeriodStart": {Bumps: "TestBump_Periods"},
	// cycle_skips — the «پریود نداشتم» endpoint (skips.Service.tx bumps; T-M9-29, E-15).
	"cycle.InsertCycleSkip": {Bumps: "TestBump_CycleSkips"},
	"cycle.DeleteCycleSkip": {Bumps: "TestBump_CycleSkips"},

	// daily_health_logs / health_log_entries / the bleeding-log bridge — healthlog.Service.inTxLocked bumps when it
	// owns the transaction; fertility.Save and conditions (PBAC, pain) bump on theirs.
	"healthlog.InsertDailyHealthLog":         {Bumps: "TestBump_HealthLogV1"},
	"healthlog.UpdateDailyHealthLog":         {Bumps: "TestBump_HealthLogV1"},
	"healthlog.DeleteDailyHealthLog":         {Bumps: "TestBump_HealthLogV1"},
	"healthlog.InsertLogEntry":               {Bumps: "TestBump_HealthLogV2"},
	"healthlog.DeleteLogEntriesOn":           {Bumps: "TestBump_HealthLogV2"},
	"healthlog.DeleteLogEntryParam":          {Bumps: "TestBump_HealthLogV2"},
	"healthlog.DeleteLogEntrySlot":           {Bumps: "TestBump_HealthLogV1"},
	"healthlog.InsertCycleHistory":           {Bumps: "TestBump_HealthLogV1"},
	"healthlog.UpdateBleedingLogPeriod":      {Bumps: "TestBump_HealthLogV1"},
	"healthlog.DeleteBleedingLogPeriod":      {Bumps: "TestBump_HealthLogV2"},
	"healthlog.UpdateProfileLastPeriodStart": {Bumps: "TestBump_HealthLogV1"},
	"healthlog.MarkProfileRecalculated":      {Exempt: "calculation_version / recalculation status only, the client-visible signal (T-M9-30 Scope 4); it runs inside a bumping health-log write"},
	"healthlog.UpdateCycleHistoryEnd":        {Exempt: "no caller since the T-M9-28 bridge (dead query)"},
	"healthlog.UpdateCycleHistoryStart":      {Exempt: "no caller since the T-M9-28 bridge (dead query)"},
	"babylog.UpsertFeedsCount":               {Exempt: "baby.feeds_count entry: a category the engine never reads (inputs.ListEngineInputMarkers reads measurements lh_test / bbt / pregnancy_test only)"},
	"babylog.DeleteFeedsCount":               {Exempt: "baby.feeds_count entry: a category the engine never reads (inputs.ListEngineInputMarkers reads measurements lh_test / bbt / pregnancy_test only)"},

	// fertility_logs — fertility.Save.
	"fertility.UpsertFertilityLog": {Bumps: "TestBump_FertilityDay"},
	"fertility.DeleteFertilityLog": {Bumps: "TestBump_FertilityDay"},

	// user_profiles / cycle_preferences / user_life_profiles — profile.Service.EngineTx.
	"profile.InsertProfile":                {Bumps: "TestBump_ProfileSave"},
	"profile.UpdateProfileAttributes":      {Bumps: "TestBump_ProfileSave"},
	"profile.SetOnboardingLastPeriodStart": {Bumps: "TestBump_ProfileSave"},
	"profile.MarkProfileRecalculated":      {Exempt: "calculation_version / recalculation status only, the client-visible signal (T-M9-30 Scope 4); POST /cycle/recalculate changes no input"},
	"profile.SetProfileTimezone":           {Exempt: "timezone is not engine state: it moves the user-local today, which is part of the ETag (D-126); no caller"},
	"profile.StoreUserTimezone":            {Exempt: "timezone is not engine state: it moves the user-local today, which is part of the ETag (D-126); written after the response by usertz"},
	"profile.UpsertCyclePreferences":       {Bumps: "TestBump_CycleSettings"},
	"profile.UpsertLifeProfile":            {Bumps: "TestBump_LifeModeSwitch"},
	"admin.CreateProfilePlan":              {Bumps: "TestBump_AdminUserEdit"},
	"admin.UpdateProfilePlan":              {Bumps: "TestBump_AdminUserEdit"},

	// pregnancy_profiles — pregnancy.engineWrite (handlers), loss.Record, healthrecord dating; postpartum activation.
	"pregnancy.InsertProfile":              {Bumps: "TestBump_Pregnancy"},
	"pregnancy.UpdateProfile":              {Bumps: "TestBump_Pregnancy"},
	"pregnancy.MarkFetalMovementFelt":      {Exempt: "fetal_movement_felt / first_fetal_movement_date only: neither is an engine input (the engine reads pregnancy_mode and the dating)"},
	"postpartum.ClosePregnancyAsDelivered": {Bumps: "TestBump_Postpartum"},

	// postpartum_profiles — postpartum.Service.engineTx.
	"postpartum.UpsertPostpartumProfile":    {Bumps: "TestBump_Postpartum"},
	"postpartum.SetLifeModePostpartum":      {Bumps: "TestBump_Postpartum"},
	"postpartum.SyncPostpartumLegacyGoal":   {Bumps: "TestBump_Postpartum"},
	"postpartum.SetPostpartumBreastfeeding": {Bumps: "TestBump_Postpartum"},

	// contraception_methods / user_life_profiles.track_contraception — contraception.Service.inTx.
	"contraception.UpsertMethod":          {Bumps: "TestBump_Contraception"},
	"contraception.DeleteMethod":          {Bumps: "TestBump_Contraception"},
	"contraception.SetTrackContraception": {Bumps: "TestBump_Contraception"},

	// user_life_profiles mode columns — IVF and menopause.
	"ivf.SetIVFSwitch":                 {Bumps: "TestBump_IVFAndMenopause"},
	"menopause.UpsertMenopauseProfile": {Bumps: "TestBump_IVFAndMenopause"},
}

// AnalysisTables are the analysis-only input tables (T-M10-28, D-177): the engine never reads them, the analysis
// reports do, so a write to any of them must bump the writer's analysis_rev (BumpAnalysis / AnalysisTx) in the same
// transaction. Weight lives in health_log_entries / user_profiles (engine tables: Bump moves analysis_rev too) and in
// pregnancy_weekly_logs (listed here).
var AnalysisTables = []string{
	"vital_readings", "lab_reports", "lab_markers", "epds_checks", "postpartum_recovery_logs",
	"baby_feeds", "baby_sleeps", "baby_diapers", "baby_med_doses",
	"pregnancy_weekly_logs", "pregnancy_symptom_logs", "pregnancy_daily_extras", "pregnancy_fetal_movements",
}

// AnalysisManifest lists every sqlc write query on an AnalysisTables table, like Manifest: the integration test that
// proves the write request moves analysis_rev (Bumps), or why it needs no bump (Exempt).
// TestAnalysisManifest_CoversEveryAnalysisWrite fails when a write query is missing or an entry is stale.
var AnalysisManifest = map[string]Entry{
	// vital_readings — vitals.Service (Create / Update / Delete through revision.AnalysisTx).
	"vitals.InsertReading": {Bumps: "TestBumpAnalysis_Vitals"},
	"vitals.UpdateReading": {Bumps: "TestBumpAnalysis_Vitals"},
	"vitals.DeleteReading": {Bumps: "TestBumpAnalysis_Vitals"},

	// lab_reports / lab_markers — labs.Service: the writes that change what the trends and the hub card read (a
	// verified, not failed lab and its markers, labs.trendRows).
	"labs.CreateLabReport":           {Bumps: "TestBumpAnalysis_Labs"},
	"labs.CreateLabMarker":           {Bumps: "TestBumpAnalysis_Labs"},
	"labs.UpdateLabMarker":           {Bumps: "TestBumpAnalysis_Labs"},
	"labs.ConfirmLabMarker":          {Bumps: "TestBumpAnalysis_Labs"},
	"labs.DeleteLabMarker":           {Bumps: "TestBumpAnalysis_Labs"},
	"labs.DeleteLabReport":           {Bumps: "TestBumpAnalysis_Labs"},
	"labs.UpdateLabMeta":             {Bumps: "TestBumpAnalysis_Labs"},
	"labs.MarkLabVerified":           {Bumps: "TestBumpAnalysis_Labs"},
	"labs.DeleteExtractedLabMarkers": {Exempt: "extraction of an unverified upload: the trends and the hub read verified labs only (labs.trendRows)"},
	"labs.SetLabExtracted":           {Exempt: "extraction of an unverified upload: the trends and the hub read verified labs only (labs.trendRows)"},
	"labs.SetLabStatus":              {Exempt: "processing status: only `failed` drops a lab from the trends, and failLab runs on unverified extractions only"},
	"labs.SetLabInterpretation":      {Exempt: "the lab's own summary: no analysis report reads it"},
	"labs.SetLabFeedback":            {Exempt: "the user's 👍/👎 on the summary: no analysis report reads it"},
	"labs.TouchLab":                  {Exempt: "updated_at only, inside a marker edit that bumps"},
	"labs.ClearLabQuota":             {Exempt: "the Plus refund claim: no analysis report reads it"},

	// epds_checks / postpartum_recovery_logs — postpartum.Service.SaveCheck / SaveRecovery.
	"postpartum.UpsertEpdsCheck":   {Bumps: "TestBumpAnalysis_Postpartum"},
	"postpartum.UpsertRecoveryLog": {Bumps: "TestBumpAnalysis_Postpartum"},

	// baby_feeds / baby_sleeps / baby_diapers — babylog.Service (owner writes only; the owner's revision moves).
	"babylog.InsertFeed":    {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.UpdateFeed":    {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.DeleteFeed":    {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.InsertSleep":   {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.StopSleep":     {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.UpdateSleep":   {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.DeleteSleep":   {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.InsertDiaper":  {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.UpdateDiaper":  {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.DeleteDiaper":  {Bumps: "TestBumpAnalysis_BabyLog"},
	"babylog.InsertMedDose": {Exempt: "no caller (dead query); the hub's baby days read feeds, sleeps and diapers only"},
	"babylog.UpdateMedDose": {Exempt: "no caller (dead query); the hub's baby days read feeds, sleeps and diapers only"},
	"babylog.DeleteMedDose": {Exempt: "no caller (dead query); the hub's baby days read feeds, sleeps and diapers only"},

	// pregnancy logs — v1 handlers (analysisWrite), the v2 day log (its save transaction) and the kick counter.
	"pregnancy.InsertWeeklyLog":           {Bumps: "TestBumpAnalysis_PregnancyLogs"},
	"pregnancy.UpdateWeeklyLog":           {Bumps: "TestBumpAnalysis_PregnancyLogs"},
	"pregnancy.InsertSymptomLog":          {Bumps: "TestBumpAnalysis_PregnancyLogs"},
	"pregnancy.UpdateSymptomLog":          {Bumps: "TestBumpAnalysis_PregnancyLogs"},
	"pregnancy.DeleteSymptomLog":          {Bumps: "TestBumpAnalysis_PregnancyLogs"},
	"pregnancy.UpsertDailyExtras":         {Bumps: "TestBumpAnalysis_PregnancyLogs"},
	"pregnancy.InsertFetalMovement":       {Bumps: "TestBumpAnalysis_PregnancyLogs"},
	"pregnancy.UpdateFetalMovement":       {Bumps: "TestBumpAnalysis_PregnancyLogs"},
	"pregnancy.RecordFlowBleedingSymptom": {Exempt: "runs inside the health-log write that revision.Bump bumps (engine_rev and analysis_rev); bleeding is not a pregnancy-analysis symptom"},
	"pregnancy.DeleteDailyExtras":         {Exempt: "no caller (dead query)"},
}

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

package main

// Persona is a ContractFixtureSeeder persona (backend/database/seeders/ContractFixtureSeeder.php,
// docs/go-migration/contract.md → Fixture data). Keep in sync with PERSONAS there.
type Persona struct {
	Key    string
	UserID int
	Mobile string
	// Blocked personas cannot log in (verify-otp → 403), so they have no session.
	Blocked bool
}

// Personas in seeder order.
var Personas = []Persona{
	{Key: "no_profile", UserID: 1001, Mobile: "09900000001"},
	{Key: "profile_no_history", UserID: 1002, Mobile: "09900000002"},
	{Key: "onboarding_declared", UserID: 1003, Mobile: "09900000003"},
	{Key: "regular", UserID: 1004, Mobile: "09900000004"},
	{Key: "irregular", UserID: 1005, Mobile: "09900000005"},
	{Key: "short_outlier", UserID: 1006, Mobile: "09900000006"},
	{Key: "open_period_day3", UserID: 1007, Mobile: "09900000007"},
	{Key: "open_period_day11", UserID: 1008, Mobile: "09900000008"},
	{Key: "open_period_day13", UserID: 1009, Mobile: "09900000009"},
	{Key: "overdue_10", UserID: 1010, Mobile: "09900000010"},
	{Key: "overdue_20", UserID: 1011, Mobile: "09900000011"},
	{Key: "ttc", UserID: 1012, Mobile: "09900000012"},
	{Key: "premium", UserID: 1013, Mobile: "09900000013"},
	{Key: "pregnant_lmp_w8", UserID: 1014, Mobile: "09900000014"},
	{Key: "pregnant_ultrasound_w26", UserID: 1015, Mobile: "09900000015"},
	{Key: "pregnant_manual_w14", UserID: 1016, Mobile: "09900000016"},
	{Key: "blocked", UserID: 1017, Mobile: "09900000017", Blocked: true},
	{Key: "engaged", UserID: 1018, Mobile: "09900000018"},
}

// ContractClientID is the fixed id the recorder gives the Passport personal access
// client after every reseed (Passport creates it with a random UUID). It is the
// `aud` of every token.
const ContractClientID = "0199c0de-0000-7000-8000-00000c0ffee1"

// normaliseSQL runs after `contract-reset`: it pins the values Laravel generates from
// the real clock or randomness, so dump.sql is byte-identical across recordings.
// The admin hash is bcrypt("contract-admin") (the seeded contract admin password).
const normaliseSQL = `
UPDATE oauth_clients SET id='` + ContractClientID + `', secret=NULL,
  created_at='2026-09-23 09:00:00', updated_at='2026-09-23 09:00:00';
UPDATE admins SET password='$2y$12$KWbOe/tVkPIgiEnivl4Pae97brtPZYsY.JaoozUl3RcLt5EG3hlDG',
  created_at='2026-09-23 09:00:00', updated_at='2026-09-23 09:00:00';
UPDATE languages SET created_at='2026-09-23 09:00:00', updated_at='2026-09-23 09:00:00';
`

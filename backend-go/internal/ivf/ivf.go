// Package ivf is IVF treatment tracking (CB-IVF-01, canvas boards nbl_IVF_Home / _Meds / _Scan / _TWW): treatment
// cycles with their six stages, the injection schedule (trigger with an exact time, site rotation over 8 sites,
// inventory with days of supply), ultrasound scans (follicles per ovary per size bin, endometrium, E2), the
// two-week wait (mood check-in, luteal-support doses, beta countdown) and the outcome.
//
// Go only (deviations.md D-51). Built on bloom and the care reminders, never parallel to them:
//   - starting a cycle switches bloom's «IVF/IUI» flag (`user_life_profiles.ivf_iui`, B-N2-03) on;
//   - every IVF medicine IS a care medication reminder (`reminders`, internal/care meta) — its injection reminder,
//     GET /care/today and the companion `meds` section see it like any medicine; a dose taken IS a care intake;
//   - cycle dates (next scan, retrieval, transfer, beta) create care appointments linked in `ivf_reminders`;
//   - «همدمت هم در جریان باشد» is a per-cycle consent flag; what a companion sees goes through the B-N4-02 grants
//     on `meds` / `appointments` (no new companion section).
//
// Clinical copy and lists (stages, protocols, injection sites, medicine presets, notes, danger signs) are catalog
// groups (Group* constants, needs_review). Health data: every read and write is scoped to the caller.
package ivf

import (
	"errors"
	"slices"
)

// Cycle stages, in timeline order (ivf_cycles.stage, catalog `ivf_stages` codes).
const (
	StagePrep      = "prep"
	StageStim      = "stim"
	StageRetrieval = "retrieval"
	StageTransfer  = "transfer"
	StageTWW       = "tww"
	StageTest      = "test"
)

// Stages is the StepTimeline order.
var Stages = []string{StagePrep, StageStim, StageRetrieval, StageTransfer, StageTWW, StageTest}

// Medicine roles (ivf_meds.role).
const (
	RoleStimulation   = "stimulation"
	RoleSuppression   = "suppression"
	RoleTrigger       = "trigger"
	RoleLutealSupport = "luteal_support"
	RoleOther         = "other"
)

// Roles are the accepted roles.
var Roles = []string{RoleStimulation, RoleSuppression, RoleTrigger, RoleLutealSupport, RoleOther}

// Medicine routes (ivf_meds.route).
const (
	RouteSubcutaneous  = "subcutaneous"
	RouteIntramuscular = "intramuscular"
	RouteOral          = "oral"
	RouteVaginal       = "vaginal"
	RouteOther         = "other"
)

// Routes are the accepted routes.
var Routes = []string{RouteSubcutaneous, RouteIntramuscular, RouteOral, RouteVaginal, RouteOther}

// IsInjection reports whether route is an injection (the dose may carry an injection site).
func IsInjection(route string) bool { return route == RouteSubcutaneous || route == RouteIntramuscular }

// StockUnits are what the inventory counts (ivf_meds.stock_unit; «قلم» = pen).
var StockUnits = []string{"pen", "vial", "ampoule", "prefilled_syringe", "box", "other"}

// Moods are the two-week-wait check-in chips (ivf_tww_logs.mood).
var Moods = []string{"calm", "hopeful", "worried", "tired"}

// Outcomes of a cycle.
const (
	OutcomePositive  = "positive"
	OutcomeNegative  = "negative"
	OutcomeCancelled = "cancelled"
)

// Outcomes are the accepted results.
var Outcomes = []string{OutcomePositive, OutcomeNegative, OutcomeCancelled}

// Next steps after the outcome (the client's routes: bloom pregnancy setup, the loss path, a new cycle).
const (
	NextPregnancySetup = "pregnancy_setup"
	NextLoss           = "loss"
	NextNewCycle       = "new_cycle"
)

// NextSteps are the follow-ups offered for an outcome: positive → pregnancy setup; negative → the loss path or
// another cycle; cancelled → another cycle.
func NextSteps(outcome string) []string {
	switch outcome {
	case OutcomePositive:
		return []string{NextPregnancySetup}
	case OutcomeNegative:
		return []string{NextLoss, NextNewCycle}
	default:
		return []string{NextNewCycle}
	}
}

// E2Units are the estradiol units (ivf_scans.e2_unit); the first is the default.
var E2Units = []string{"pg_ml", "pmol_l"}

// Bins are the follicle size bins of a scan, in board order (<10, 10–14, 15–17, ≥18 mm).
var Bins = []string{"lt_10", "10_14", "15_17", "18_plus"}

// Catalog groups (admin-editable, fa + en, needs_review).
const (
	GroupStages    = "ivf_stages"
	GroupProtocols = "ivf_protocols"
	GroupSites     = "ivf_injection_sites"
	GroupPresets   = "ivf_med_presets"
	GroupGuidance  = "ivf_guidance"
	GroupDanger    = "ivf_danger_signs"
)

// Limits and defaults [needs clinical review where clinical].
const (
	// LowSupplyDays: the inventory is low when it lasts this many days or fewer (and runs out before the
	// medicine's end date).
	LowSupplyDays = 3
	// MaxTimes caps the daily dose times of a medicine (care's own cap).
	MaxTimes = 4
	// MaxStockUnits / MaxDosesPerUnit bound the inventory fields.
	MaxStockUnits   = 999
	MaxDosesPerUnit = 100
	// MaxFollicles bounds one bin of one ovary.
	MaxFollicles = 60
	// BetaHour is the hour (Tehran) of the beta blood-test appointment created from beta_on.
	BetaHour = 8
	// MaxCycleNumber bounds ivf_cycles.number.
	MaxCycleNumber = 50
)

// Appointment kinds the cycle dates create (ivf_reminders.kind), in display order.
const (
	KindScan      = "scan"
	KindRetrieval = "retrieval"
	KindTransfer  = "transfer"
	KindBeta      = "beta"
)

// Errors the handlers map to responses.
var (
	ErrNoCycle         = errors.New("ivf: no open cycle")
	ErrCycleOpen       = errors.New("ivf: a cycle is already open")
	ErrMedNotFound     = errors.New("ivf: medicine not found")
	ErrMedicationLimit = errors.New("ivf: care medication limit reached")
)

// FieldError is a 422 on one field with a lang key (validation.<Key>).
type FieldError struct {
	Field, Key string
}

func (e *FieldError) Error() string { return "ivf: " + e.Field + ": " + e.Key }

func fieldErr(field, key string) error { return &FieldError{Field: field, Key: key} }

func stageIndex(stage string) int { return slices.Index(Stages, stage) }

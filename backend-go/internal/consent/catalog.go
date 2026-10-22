// Package consent is the versioned consent records of Ritme (B-N6-05): one catalog of consent codes, each with the
// version of its text in force and that text per language, the user's answer per code (`user_consents`, shared
// with the B-N1-12 privacy toggles) and the check AI features run before they call a provider.
//
// Ownership: the B-N1-12 endpoints GET/PUT /profile/consents (internal/profile) keep their handlers, store and
// response shape; they only take the code list and the current versions from here, so a privacy toggle grants the
// text in force. This package adds the text-aware GET /consents, GET /consents/{code} and PUT /consents/{code}
// (accept a given version / withdraw) and the gate (Service.Require → *RequiredError → 403 consent_required).
//
// Versioning: bumping a code's Version (and adding the new text under texts.<code>.<version> in lang/*) makes
// every earlier acceptance stale — features answer 403 consent_required with reason "outdated" until the user
// accepts the new text. Old texts stay in the files (what a user accepted can always be shown again).
//
// Ledger (T-M10-23, D-174; SPEC v6 §4.2): every grant and withdrawal is also an append-only `consent_events` row
// (ledger.go), written with the `user_consents` projection change in one transaction. The mandatory signup consents
// (terms, health_data_processing) are catalog codes too: verify-otp records them for a new account, existing users
// accept them once from GET /consents/pending (H-18). Evidence-only codes (Evidence) record scoped shares — assistant
// item switches, telemed shares, share links — without any projection or gate.
package consent

// Consent codes. Every consent is opt-in: no row, a row with granted = 0, or an outdated version means «no».
const (
	AILabAnalysis    = "ai_lab_analysis"   // lab sheets read and explained by the AI (B-N6-06)
	AssistantProfile = "assistant_profile" // the assistant may use age, mode and medications (B-N7-06)
	AnonymousStats   = "anonymous_stats"   // anonymous, aggregated usage statistics
	AIAssistant      = "ai_assistant"      // chatting with the AI assistant at all (B-N7-06)
	AIVoiceLog       = "ai_voice_log"      // voice logging audio sent to an STT provider (B-N3-05)
	AIDocuments      = "ai_documents"      // medical documents read by the AI (CB-REC-02 and later)

	// Mandatory signup consents (T-M10-23): accepted on the signup screen (verify-otp `consents`), or once from the
	// pending sheet by users who signed up before they existed. Nothing is gated on them server-side (H-18).
	Terms                = "terms"                  // the terms of use and the privacy policy
	HealthDataProcessing = "health_data_processing" // processing the health data the user enters to run the service
)

// Evidence-only consent codes (T-M10-23): scoped events in the ledger, no user_consents row, no text version of their
// own beyond 1, never a gate (telemed AuthorizeShare and the share-link token stay the gates).
const (
	AssistantProfileItems = "assistant_profile_items" // which profile items the assistant may see (scope {items})
	TelemedShare          = "telemed_share"           // data shared with one booking's doctor (scope {booking_id, doctor_id, items})
	ShareLink             = "share_link"              // a doctor report share link (scope {link_id, expires_at, items})
)

// Definition is one consent: its code and the version of the text in force.
type Definition struct {
	Code    string
	Version int
	// AI marks a consent that lets data reach an AI provider. Granting it always names the version of the text the
	// client showed (B-N6-05b, M5): PUT /consents/{code} {version} and PUT /profile/consents {versions: {code: n}}.
	AI bool
	// Mandatory marks a signup consent: listed by GET /consents/pending until the version in force is accepted, not
	// by GET /consents (the feature consents).
	Mandatory bool
}

// catalog is the single source of consent codes and versions, in display order.
var catalog = []Definition{
	{Code: AILabAnalysis, Version: 1, AI: true},
	{Code: AssistantProfile, Version: 1, AI: true},
	{Code: AnonymousStats, Version: 1},
	{Code: AIAssistant, Version: 1, AI: true},
	{Code: AIVoiceLog, Version: 1, AI: true},
	{Code: AIDocuments, Version: 1, AI: true},
	{Code: Terms, Version: 1, Mandatory: true},
	{Code: HealthDataProcessing, Version: 1, Mandatory: true},
}

// evidence are the evidence-only codes with the version their events carry.
var evidence = []Definition{
	{Code: AssistantProfileItems, Version: 1},
	{Code: TelemedShare, Version: 1},
	{Code: ShareLink, Version: 1},
}

// Mandatory returns the mandatory signup consents in display order.
func Mandatory() []Definition {
	var out []Definition
	for _, d := range catalog {
		if d.Mandatory {
			out = append(out, d)
		}
	}
	return out
}

// lookupEvidence returns the definition of an evidence-only code.
func lookupEvidence(code string) (Definition, bool) {
	for _, d := range evidence {
		if d.Code == code {
			return d, true
		}
	}
	return Definition{}, false
}

// Catalog returns a copy of the definitions in display order.
func Catalog() []Definition { return append([]Definition(nil), catalog...) }

// Lookup returns the definition of code.
func Lookup(code string) (Definition, bool) {
	for _, d := range catalog {
		if d.Code == code {
			return d, true
		}
	}
	return Definition{}, false
}

// IsAI reports whether code is an AI consent (granting it needs an explicit version).
func IsAI(code string) bool {
	d, _ := Lookup(code)
	return d.AI
}

// CurrentVersion is the version in force of code (0 for an unknown code).
func CurrentVersion(code string) int {
	d, _ := Lookup(code)
	return d.Version
}

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
package consent

// Consent codes. Every consent is opt-in: no row, a row with granted = 0, or an outdated version means «no».
const (
	AILabAnalysis    = "ai_lab_analysis"   // lab sheets read and explained by the AI (B-N6-06)
	AssistantProfile = "assistant_profile" // the assistant may use age, mode and medications (B-N7-06)
	AnonymousStats   = "anonymous_stats"   // anonymous, aggregated usage statistics
	AIAssistant      = "ai_assistant"      // chatting with the AI assistant at all (B-N7-06)
	AIVoiceLog       = "ai_voice_log"      // voice logging audio sent to an STT provider (B-N3-05)
	AIDocuments      = "ai_documents"      // medical documents read by the AI (CB-REC-02 and later)
)

// Definition is one consent: its code and the version of the text in force.
type Definition struct {
	Code    string
	Version int
}

// catalog is the single source of consent codes and versions, in display order.
var catalog = []Definition{
	{Code: AILabAnalysis, Version: 1},
	{Code: AssistantProfile, Version: 1},
	{Code: AnonymousStats, Version: 1},
	{Code: AIAssistant, Version: 1},
	{Code: AIVoiceLog, Version: 1},
	{Code: AIDocuments, Version: 1},
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

// CurrentVersion is the version in force of code (0 for an unknown code).
func CurrentVersion(code string) int {
	d, _ := Lookup(code)
	return d.Version
}

// Package translations embeds the seed UI-string bundles the frontend renders from
// (<code>/<namespace>.json). They are a verbatim copy of frontend/messages/<code>/
// (frontend/CLAUDE.md §6.4; TestSeedMatchesFrontendMessages fails on drift); refresh with
//
//	cp ../frontend/messages/<code>/*.json resources/translations/<code>/
//
// and re-record internal/i18n/testdata/messages_*.json. Since T-M2-31 this seed is ahead of
// Laravel's backend/resources/translations (deviation D-23).
//
// internal/i18n.TranslationStore layers the live, admin-edited files from
// STORAGE_PATH/app/translations/<code>/ on top of this seed.
package translations

import "embed"

// FS holds <code>/<namespace>.json.
//
//go:embed */*.json
var FS embed.FS

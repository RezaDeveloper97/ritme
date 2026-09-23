// Package translations embeds the seed UI-string bundles the frontend renders from
// (<code>/<namespace>.json). They are a verbatim copy of backend/resources/translations
// (kept in sync with frontend/messages by `php artisan translations:import`); refresh with
//
//	rm -rf backend-go/resources/translations/{fa,en} && cp -R backend/resources/translations/{fa,en} backend-go/resources/translations/
//
// internal/i18n.TranslationStore layers the live, admin-edited files from
// STORAGE_PATH/app/translations/<code>/ on top of this seed.
package translations

import "embed"

// FS holds <code>/<namespace>.json.
//
//go:embed */*.json
var FS embed.FS

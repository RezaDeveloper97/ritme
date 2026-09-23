// Package lang embeds Laravel's translation files converted to JSON
// (<locale>/<group>.json, made by convert.php in this directory). The Go translator
// lives in internal/i18n/lang.
package lang

import "embed"

// FS holds <locale>/<group>.json.
//
//go:embed */*.json
var FS embed.FS

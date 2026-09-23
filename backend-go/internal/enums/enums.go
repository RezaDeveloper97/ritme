// Package enums holds the Laravel string-backed enums ported to Go.
//
// zz_generated_*.go files are produced by cmd/enumgen from backend/app/Enums/*.php and
// backend/app/Services/MessageSystem/Enums/*.php (value/label/description/icon tables, PHP case
// order, exact backed values). Behaviour is hand-ported in *_logic.go with the PHP file:line cited.
//
// Regenerate from backend-go/ with: go generate ./internal/enums/...
// Parity testdata (testdata/php_enums.json) is refreshed with:
//
//	php internal/enums/testdata/dump_enums.php ../backend > internal/enums/testdata/php_enums.json
package enums

//go:generate go run ../../cmd/enumgen -backend ../../../backend -out .

// Option is one {value, label} pair for a picker (PHP options($locale) rows).
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type labeled interface {
	~string
	Label(locale string) string
}

func valuesOf[E ~string](cases []E) []string {
	out := make([]string, len(cases))
	for i, c := range cases {
		out[i] = string(c)
	}
	return out
}

func optionsOf[E labeled](cases []E, locale string) []Option {
	out := make([]Option, len(cases))
	for i, c := range cases {
		out[i] = Option{Value: string(c), Label: c.Label(locale)}
	}
	return out
}

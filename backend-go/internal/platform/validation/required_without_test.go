package validation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestRequiredWithout(t *testing.T) {
	rules := validation.Rules{
		validation.F("body", "required_without:file_id", "nullable", "string"),
		validation.F("file_id", "nullable", "integer"),
	}
	for _, tc := range []struct {
		name  string
		data  map[string]any
		fails bool
	}{
		{"both missing", map[string]any{}, true},
		{"body empty, file null", map[string]any{"body": "", "file_id": nil}, true},
		{"body only", map[string]any{"body": "x"}, false},
		{"file only", map[string]any{"file_id": int64(3)}, false},
		{"both", map[string]any{"body": "x", "file_id": int64(3)}, false},
	} {
		data := phpval.NewMap()
		for k, v := range tc.data {
			data.Set(k, v)
		}
		v := validation.Make(lang.Default(), "en", data, rules)
		assert.Equal(t, tc.fails, v.Fails(), tc.name)
		if tc.fails {
			assert.Equal(t, "The body field is required when file id is not present.", v.Errors().Summary(), tc.name)
		}
	}
}

package sanitizer

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type phpCase struct {
	Input string  `json:"input"`
	Clean *string `json:"clean"`
	Plain *string `json:"plain"`
}

// loadPHPCases reads the PHP recordings (testdata/dump_sanitizer.php): the curated
// corpus (seeded article copy, CKEditor output, XSS vectors, malformed markup) and a
// generated differential-fuzz corpus of random tag / attribute / character soup.
func loadPHPCases(t *testing.T) []phpCase {
	t.Helper()
	var all []phpCase
	for _, f := range []string{"testdata/php_sanitizer.json", "testdata/php_fuzz.json"} {
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		var cases []phpCase
		require.NoError(t, json.Unmarshal(raw, &cases))
		require.NotEmpty(t, cases, f)
		all = append(all, cases...)
	}
	return all
}

func opt(s string, ok bool) *string {
	if !ok {
		return nil
	}
	return &s
}

// TestCleanMatchesPHP compares Clean with HtmlSanitizer::clean on the recorded corpus
// (seeded article copy, CKEditor output, XSS vectors, malformed markup).
func TestCleanMatchesPHP(t *testing.T) {
	for i, c := range loadPHPCases(t) {
		got := opt(Clean(c.Input))
		assert.Equal(t, c.Clean, got, "case %d clean(%q)", i, c.Input)
	}
}

// TestPlainTextMatchesPHP compares PlainText with HtmlSanitizer::toPlainText.
func TestPlainTextMatchesPHP(t *testing.T) {
	for i, c := range loadPHPCases(t) {
		got := opt(PlainText(c.Input))
		assert.Equal(t, c.Plain, got, "case %d plain(%q)", i, c.Input)
	}
}

func TestCleanNeverKeepsExecutableMarkup(t *testing.T) {
	for _, in := range []string{
		`<script>alert(1)</script>`, `<p onclick="x">a</p>`, `<a href="javascript:alert(1)">a</a>`,
		`<iframe src="x"></iframe>`, `<img src=x onerror=alert(1)>`, `<svg onload=alert(1)>`,
	} {
		out, _ := Clean(in)
		assert.NotContains(t, out, "script", in)
		assert.NotContains(t, out, "onclick", in)
		assert.NotContains(t, out, "onerror", in)
		assert.NotContains(t, out, "iframe", in)
		assert.NotContains(t, out, "javascript", in)
		assert.NotContains(t, out, "svg", in)
	}
}

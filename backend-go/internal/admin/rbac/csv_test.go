package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"user.block":      "user.block",
		"=HYPERLINK(1)":   "'=HYPERLINK(1)",
		" =cmd|' /C calc": "' =cmd|' /C calc", // leading space (B-N9-02b L7)
		"\n\t+1":          "'\n\t+1",
		"\t1":             "'\t1",
		"-2":              "'-2",
		"@SUM(A1)":        "'@SUM(A1)",
		"  plain":         "  plain",
	}
	for in, want := range cases {
		assert.Equal(t, want, csvSafe(in), "%q", in)
	}
}

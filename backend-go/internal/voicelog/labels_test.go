package voicelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// B-N3-14b (N3 stage smoke B-8): chip numbers use the language's digits.
func TestNumDigits(t *testing.T) {
	assert.Equal(t, "۵۸٫۵", num(58.5, "fa"))
	assert.Equal(t, "58.5", num(58.5, "en"))
	assert.Equal(t, "۳", num(3, "fa"))
	assert.Equal(t, "۰۸:۳۰", digits("08:30", "fa"))
	assert.Equal(t, "08:30", digits("08:30", "ar"))
}

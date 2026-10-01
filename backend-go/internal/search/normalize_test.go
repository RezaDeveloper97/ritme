package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// zwnj is the Persian zero-width non-joiner (the «half space»).
const zwnj = "\u200c"

func TestNormalize_PersianVariants(t *testing.T) {
	cases := map[string]string{
		"ي":                     "ی", // Arabic yeh
		"ى":                     "ی", // alef maksura
		"ك":                     "ک", // Arabic kaf
		"كيست تخمدان":           "کیست تخمدان",
		"ثبت" + zwnj + "های من": "ثبت های من", // ZWNJ is a word break
		"  درد   پريود  ":       "درد پریود",
		"۱۲۳ ٤٥٦":               "123 456", // Persian and Arabic-Indic digits
		"آب":                    "اب",
		"أإٱ":                   "ااا",
		"خانۀ":                  "خانه",
		"دَرْد":                 "درد",         // diacritics dropped
		"دردـــــ":              "درد",         // tatweel dropped
		"«درد» زیر-دلم؟":        "درد زیر دلم", // punctuation is a separator
		"Pap Smear":             "pap smear",
		"\u200fدرد\u200e":       "درد", // bidi marks dropped
		"":                      "",
		zwnj + zwnj + "  ،.":    "",
		"می" + zwnj + "شود":     "می شود",
		"آب":                   "اب", // alef + combining madda
		"ی\u200dکی":             "ی کی",
	}
	for in, want := range cases {
		assert.Equal(t, want, Normalize(in), "%q", in)
	}
}

func TestMatcher_Ranks(t *testing.T) {
	m := NewMatcher("درد")
	assert.Equal(t, 3, m.Match("درد پریود؛ کی عادی است؟"))
	assert.Equal(t, 2, m.Match("دفترچه درد اندومتریوز"))
	assert.Equal(t, 0, m.Match("یوگای ملایم"))

	// Arabic letters and ZWNJ / space differences in either side still match.
	assert.Positive(t, NewMatcher("ثبت هاي").Match("ثبت"+zwnj+"های من"))
	assert.Positive(t, NewMatcher("میشود").Match("می"+zwnj+"شود"))
	assert.Positive(t, NewMatcher("می شود").Match("میشود"))
	assert.Positive(t, NewMatcher("كيست").Match("کیست"))
	assert.Positive(t, NewMatcher("۱۱۵").Match("تماس با 115"))

	// Every word must occur, order free.
	assert.Equal(t, 1, NewMatcher("پریود درد").Match("درد پریود"))
	assert.Equal(t, 0, NewMatcher("پریود سر").Match("درد پریود"))
}

func TestMatcher_EmptyAndBest(t *testing.T) {
	assert.True(t, NewMatcher(" "+zwnj+" ،").Empty())
	assert.Equal(t, 0, NewMatcher("").Match("anything"))
	assert.Equal(t, 6, NewMatcher("قرص آهن").Len()) // spaces ignored

	m := NewMatcher("کگل")
	assert.Equal(t, 0, m.Best("کف لگن"))
	assert.Equal(t, 1, m.Best("کف لگن", "برنامه ۸ هفته\u200cای", "کگل"))
	assert.Equal(t, 4, NewMatcher("کف").Best("کف لگن", "کگل"))
}

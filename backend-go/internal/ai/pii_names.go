package ai

import (
	"strings"
	"unicode"
)

// Cross-script name matching (B-N6-05b, M4). A user named «فاطمه حسینی» may write «Fatemeh» in a chat, and one
// registered as «Sara Mohammadi» may write «محمدی». Exact transliteration is ambiguous (Persian omits short
// vowels, Latin spellings vary), so both sides are reduced to a consonant skeleton: Latin digraphs folded
// (kh, gh, sh, ch, zh, th, ph), every vowel and vowel carrier dropped (a e i o u y w v / ا و ی ع ء), letters that
// sound alike merged (ث س ص → s, ذ ز ض ظ → z, ت ط → t, غ ق → q, ح ه → h), doubles collapsed and a final h
// dropped (Fatemeh / فاطمه / Fatema). A text word is redacted when its skeleton equals the skeleton of one of the
// user's name words written in the other script. Names whose skeleton has fewer than minSkeleton consonants
// (Sara, Reza) are matched only literally: their skeletons collide with common words (سر, رز).

// minSkeleton is the shortest skeleton matched across scripts.
const minSkeleton = 3

// persianSkeleton maps Persian letters to skeleton consonants; letters not listed are dropped (vowel carriers).
var persianSkeleton = map[rune]rune{
	'ب': 'b', 'پ': 'p', 'ت': 't', 'ث': 's', 'ج': 'j', 'چ': 'c', 'ح': 'h', 'خ': 'x', 'د': 'd', 'ذ': 'z', 'ر': 'r',
	'ز': 'z', 'ژ': 'j', 'س': 's', 'ش': '$', 'ص': 's', 'ض': 'z', 'ط': 't', 'ظ': 'z', 'غ': 'q', 'ف': 'f', 'ق': 'q',
	'ک': 'k', 'گ': 'g', 'ل': 'l', 'م': 'm', 'ن': 'n', 'ه': 'h',
}

// latinDigraphs are folded before single letters (order matters: longest first).
var latinDigraphs = strings.NewReplacer("kh", "x", "gh", "q", "sh", "$", "ch", "C", "zh", "j", "th", "t", "ph", "f", "ck", "k")

// latinSkeleton maps single Latin letters; vowels and a e i o u y w v are dropped.
var latinSkeleton = map[rune]rune{
	'b': 'b', 'p': 'p', 't': 't', 's': 's', 'j': 'j', 'c': 'k', 'x': 'x', 'd': 'd', 'z': 'z', 'r': 'r', 'q': 'q',
	'f': 'f', 'k': 'k', 'g': 'g', 'l': 'l', 'm': 'm', 'n': 'n', 'h': 'h',
	'$': '$', // from "sh"
	'C': 'c', // from "ch" (a lone c is a hard k)
}

type script int

const (
	scriptOther script = iota
	scriptLatin
	scriptPersian
)

// wordScript is the script of a word of letters (normalized runes).
func wordScript(w []rune) script {
	latin, persian := false, false
	for _, r := range w {
		switch {
		case r < unicode.MaxASCII && unicode.IsLetter(r):
			latin = true
		case unicode.Is(unicode.Arabic, r):
			persian = true
		}
	}
	switch {
	case latin && !persian:
		return scriptLatin
	case persian && !latin:
		return scriptPersian
	}
	return scriptOther
}

// skeleton is the consonant skeleton of a word (see the comment at the top of the file).
func skeleton(w []rune, s script) string {
	var out []rune
	push := func(r rune) {
		if len(out) == 0 || out[len(out)-1] != r {
			out = append(out, r)
		}
	}
	switch s {
	case scriptLatin:
		for _, r := range latinDigraphs.Replace(strings.ToLower(string(w))) {
			if m, ok := latinSkeleton[r]; ok {
				push(m)
			}
		}
	case scriptPersian:
		for _, r := range w {
			if m, ok := persianSkeleton[r]; ok {
				push(m)
			}
		}
	default:
		return ""
	}
	if n := len(out); n > 0 && out[n-1] == 'h' {
		out = out[:n-1]
	}
	return string(out)
}

// crossScriptNames redacts the words of the text whose skeleton matches a name word written in the other script.
func (r *redactor) crossScriptNames(names []string) {
	want := map[script]map[string]bool{scriptLatin: {}, scriptPersian: {}}
	for _, n := range names {
		for _, w := range strings.FieldsFunc(string(normalizeRunes([]rune(n))), func(r rune) bool { return !isWordRune(r) }) {
			rs := []rune(w)
			s := wordScript(rs)
			if s == scriptOther {
				continue
			}
			if sk := skeleton(rs, s); len([]rune(sk)) >= minSkeleton {
				other := scriptPersian
				if s == scriptPersian {
					other = scriptLatin
				}
				want[other][sk] = true
			}
		}
	}
	if len(want[scriptLatin]) == 0 && len(want[scriptPersian]) == 0 {
		return
	}
	for i := 0; i < len(r.nr); {
		if !isWordRune(r.nr[i]) || isDigit(r.nr[i]) {
			i++
			continue
		}
		j := i
		for j < len(r.nr) && isWordRune(r.nr[j]) {
			j++
		}
		w := r.nr[i:j]
		if s := wordScript(w); s != scriptOther && want[s][skeleton(w, s)] && r.free(i, j) {
			r.mark(i, j, RedactedName)
		}
		i = j
	}
}

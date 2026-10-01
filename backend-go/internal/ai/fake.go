package ai

import (
	"bytes"
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Fake is the deterministic provider (dev, tests, stage). It never touches the network.
//
// Transcribe picks a fixture keyed by the input: a recording containing the ASCII marker
// "RITME-FAKE:<key>" gets fixture <key> (tests craft such payloads); any other recording — a real browser
// recording on stage — gets the "default" fixture, the artboard's sentence. Fixture "silence" is an empty
// transcript and "error" fails with ErrUpstream.
//
// ParseLog is a small rule-based matcher: a colloquial lexicon for the artboard phrases (pain locations and
// intensity, weight, sleep) plus label matching of the vocabulary's item / multi labels, so custom items
// work too. Same input → same output.
type Fake struct{}

// NewFake builds the fake provider.
func NewFake() *Fake { return &Fake{} }

// FakeModel is the model name the fake reports in its usage.
const FakeModel = "fixtures-v1"

// FakeMarker prefixes a fixture key inside a recording.
const FakeMarker = "RITME-FAKE:"

// FakeTranscripts are the fixtures, by key and language ("en" is the fallback language).
var FakeTranscripts = map[string]map[string]string{
	"default": {
		"fa": "از صبح دلم درد می\u200cکنه، متوسط، یه کم هم نفخ دارم و بی\u200cحوصله\u200cام",
		"en": "My belly has been hurting since this morning, moderate, I'm a bit bloated and feeling bored",
	},
	"weight": {
		"fa": "وزنم امروز ۵۸٫۵ کیلو بود و دیشب خوب خوابیدم",
		"en": "I weighed 58.5 kilos today and slept well last night",
	},
	"headache": {
		"fa": "سرم خیلی درد می\u200cکنه و حالت تهوع دارم",
		"en": "I have a severe headache and nausea",
	},
	"custom": {
		"fa": "امروز دلتنگم و خسته\u200cام",
		"en": "I feel homesick and tired today",
	},
	"silence": {"fa": "", "en": ""},
}

var fakeKey = regexp.MustCompile(`RITME-FAKE:([a-z_]{1,32})`)

// Transcribe implements Transcriber.
func (f *Fake) Transcribe(_ context.Context, req TranscribeRequest) (Transcript, Usage, error) {
	u := Usage{Provider: "fake", Model: FakeModel}
	key := "default"
	if bytes.Contains(req.Audio.Data, []byte(FakeMarker)) {
		if m := fakeKey.FindSubmatch(req.Audio.Data); m != nil {
			key = string(m[1])
		}
	}
	if key == "error" {
		return Transcript{}, u, ErrUpstream
	}
	fx, ok := FakeTranscripts[key]
	if !ok {
		fx = FakeTranscripts["default"]
	}
	lang := req.Language
	text, ok := fx[lang]
	if !ok {
		lang, text = "en", fx["en"]
	}
	u.InputTokens = len(req.Audio.Data) / 1024
	u.OutputTokens = utf8.RuneCountInString(text) / 4
	return Transcript{Text: text, Language: lang}, u, nil
}

// lexRule maps a phrase family to a slot.
type lexRule struct {
	pattern *regexp.Regexp
	key     string
	value   any // nil → the slot's default (items: detected level)
}

var fakeLexicon = []lexRule{
	{regexp.MustCompile(`دلم درد|دل ?درد|شکمم درد|درد شکم|دل پیچه|belly|stomach ?ache|abdomen|cramp`), "pain.location.abdomen", nil},
	{regexp.MustCompile(`سرم درد|سر ?درد|headache`), "pain.location.head", nil},
	{regexp.MustCompile(`کمرم درد|کمر ?درد|back ?pain|backache`), "pain.location.back", nil},
	{regexp.MustCompile(`نفخ|bloat`), "symptoms.digestive.bloating", nil},
	{regexp.MustCompile(`تهوع|nause`), "symptoms.digestive.nausea", nil},
	{regexp.MustCompile(`خستگی|خسته\x{200c}?ام|tired|fatigue`), "symptoms.general.fatigue", nil},
	{regexp.MustCompile(`بیحوصله|bored`), "mood.moods.bored", true},
	{regexp.MustCompile(`زودرنج|irritable`), "mood.moods.irritable", true},
	{regexp.MustCompile(`مضطرب|استرس|anxious`), "mood.moods.anxious", true},
	{regexp.MustCompile(`خوشحال|happy`), "mood.moods.happy", true},
	{regexp.MustCompile(`خوب خوابیدم|slept well`), "sleep.quality", "good"},
	{regexp.MustCompile(`بد خوابیدم|slept badly|slept poorly`), "sleep.quality", "poor"},
}

var (
	fakeWeight   = regexp.MustCompile(`(?:وزن\S*|weigh\S*)\D{0,12}(\d{2,3}(?:[.]\d{1,2})?)`)
	levelSevere  = regexp.MustCompile(`شدید|خیلی|severe|really bad`)
	levelMild    = regexp.MustCompile(`خفیف|یه ?کم|کمی|mild|a little`)
	levelMid     = regexp.MustCompile(`متوسط|moderate`)
	fakeDiacrit  = regexp.MustCompile(`[\x{064B}-\x{0652}\x{200C}-\x{200F}]`)
	fakeSpaces   = regexp.MustCompile(`\s+`)
	fakeArabicYa = strings.NewReplacer("ي", "ی", "ى", "ی", "ك", "ک", "٫", ".", ",", " ")
)

// NormalizeText folds what speech-to-text writes so phrases match whatever the keyboard: lower case, Arabic
// ي/ك → Persian ی/ک, ZWNJ and diacritics removed, Persian / Arabic digits → ASCII, «٫» → ".".
func NormalizeText(s string) string {
	s = strings.ToLower(fakeArabicYa.Replace(s))
	s = fakeDiacrit.ReplaceAllString(s, "")
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		}
		return r
	}, s)
	return strings.TrimSpace(fakeSpaces.ReplaceAllString(s, " "))
}

// ParseLog implements LogParser.
func (f *Fake) ParseLog(_ context.Context, req LogParseRequest) ([]Candidate, Usage, error) {
	u := Usage{Provider: "fake", Model: FakeModel, InputTokens: utf8.RuneCountInString(req.Text) / 4}
	text := NormalizeText(req.Text)
	vocab := map[string]VocabEntry{}
	for _, v := range req.Vocabulary {
		vocab[v.Key] = v
	}
	level := ""
	switch {
	case levelSevere.MatchString(text):
		level = "severe"
	case levelMid.MatchString(text):
		level = "moderate"
	case levelMild.MatchString(text):
		level = "mild"
	}
	var out []Candidate
	seen := map[string]bool{}
	add := func(key string, value any, conf float64) {
		v, ok := vocab[key]
		if !ok || seen[key] {
			return
		}
		if value == nil {
			value = itemValue(v, level)
		}
		seen[key] = true
		out = append(out, Candidate{Key: key, Value: value, Confidence: conf})
	}
	for _, r := range fakeLexicon {
		if r.pattern.MatchString(text) {
			add(r.key, r.value, 0.9)
		}
	}
	if m := fakeWeight.FindStringSubmatch(text); m != nil {
		if n, err := strconv.ParseFloat(m[1], 64); err == nil {
			add("measurements.weight", n, 0.85)
		}
	}
	// Item / multi labels said verbatim (custom items included). Single options are left to the lexicon:
	// their labels («متوسط», «زیاد») are too generic to match on their own.
	for _, v := range req.Vocabulary {
		if v.Type != "items" && v.Type != "multi" {
			continue
		}
		label := NormalizeText(lastLabel(v.Label))
		if utf8.RuneCountInString(label) < 3 || !strings.Contains(text, label) {
			continue
		}
		if v.Type == "multi" {
			add(v.Key, true, 0.7)
		} else {
			add(v.Key, nil, 0.7)
		}
	}
	u.OutputTokens = len(out) * 12
	return out, u, nil
}

// itemValue is an items slot's level for the detected intensity (or its default), a multi's true.
func itemValue(v VocabEntry, level string) any {
	switch v.Type {
	case "multi", "bool":
		return true
	case "items":
		codes := make([]string, len(v.Values))
		for i, x := range v.Values {
			codes[i] = x.Code
		}
		// yes/no symptoms are logged as "yes" (the intensity words describe the pain); graded slots take it
		if slices.Contains(codes, "yes") {
			return "yes"
		}
		if level != "" && slices.Contains(codes, level) {
			return level
		}
		if slices.Contains(codes, "moderate") {
			return "moderate"
		}
		if len(codes) > 0 {
			return codes[0]
		}
	}
	return nil
}

// lastLabel is the leaf of a "category › param › item" label.
func lastLabel(label string) string {
	if i := strings.LastIndex(label, "›"); i >= 0 {
		return strings.TrimSpace(label[i+len("›"):])
	}
	return label
}

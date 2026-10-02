package ai

import (
	"bytes"
	"context"
	"fmt"
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
// intensity, weight, sleep, menopause symptoms and triggers) plus label matching of the vocabulary's item / multi
// labels, so custom items work too, plus the canvas fields (CB-VOICE-01: hot flashes, pain-diary score / analgesic,
// pill taken / missed, bladder leak / night voids) and the mood ambiguity chooser. Same input → same output.
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
	// CB-VOICE-01: canvas items (each written through its own service on commit).
	"menopause": {
		"fa": "دیشب سه بار گرگرفتگی داشتم با عرق شبانه، امروز هم ذهنم مه\u200cآلوده و یه قهوه خوردم",
		"en": "I had three hot flashes last night with night sweats, brain fog today and I had a coffee",
	},
	"pain_diary": {
		"fa": "از دیشب زیر دلم درد می\u200cکنه، حدود شش از ده. ساعت ده یه ایبوپروفن خوردم که کمی کمک کرد. نتونستم برم سر کار",
		"en": "My lower belly has hurt since last night, about six out of ten. I took an ibuprofen at ten and it helped a little. I couldn't go to work",
	},
	"pill": {
		"fa": "قرصم رو ساعت نه خوردم",
		"en": "I took my pill at nine",
	},
	"pelvic": {
		"fa": "امروز موقع سرفه یه کم ادرارم نشت کرد و شب دو بار برای دستشویی بیدار شدم",
		"en": "I leaked a little urine when I coughed today and got up twice at night to pee",
	},
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
	// alts are the other slots the words may mean (ambiguity chooser).
	alts []string
}

var fakeLexicon = []lexRule{
	{regexp.MustCompile(`دلم درد|دل ?درد|شکمم درد|درد شکم|دل پیچه|belly|stomach ?ache|abdomen|cramp`), "pain.location.abdomen", nil, nil},
	{regexp.MustCompile(`سرم درد|سر ?درد|headache`), "pain.location.head", nil, nil},
	{regexp.MustCompile(`کمرم درد|کمر ?درد|back ?pain|backache`), "pain.location.back", nil, nil},
	{regexp.MustCompile(`نفخ|bloat`), "symptoms.digestive.bloating", nil, nil},
	{regexp.MustCompile(`تهوع|nause`), "symptoms.digestive.nausea", nil, nil},
	{regexp.MustCompile(`خستگی|خسته\x{200c}?ام|tired|fatigue`), "symptoms.general.fatigue", nil, nil},
	{regexp.MustCompile(`بیحوصله|bored`), "mood.moods.bored", true, []string{"mood.moods.sad", "symptoms.general.fatigue"}},
	{regexp.MustCompile(`زودرنج|irritable`), "mood.moods.irritable", true, nil},
	{regexp.MustCompile(`مضطرب|استرس|anxious`), "mood.moods.anxious", true, nil},
	{regexp.MustCompile(`خوشحال|happy`), "mood.moods.happy", true, nil},
	{regexp.MustCompile(`خوب خوابیدم|slept well`), "sleep.quality", "good", nil},
	{regexp.MustCompile(`بد خوابیدم|slept badly|slept poorly`), "sleep.quality", "poor", nil},
	// menopause symptoms and triggers (CB-MENO-01 taxonomy items; only offered in menopause mode)
	{regexp.MustCompile(`عرق شبانه|تعریق شبانه|night ?sweat`), "symptoms.general.night_sweats", nil, nil},
	{regexp.MustCompile(`مه ?آلود|حواس ?پرت|فراموشکار|brain ?fog|foggy`), "symptoms.general.brain_fog", nil, nil},
	{regexp.MustCompile(`تپش ?قلب|palpitation`), "symptoms.general.palpitations", nil, nil},
	{regexp.MustCompile(`افسرده|دلمرده|low mood|depressed`), "symptoms.general.low_mood", nil, nil},
	{regexp.MustCompile(`بیخوابی|خوابم نبرد|insomnia|couldn'?t sleep`), "symptoms.general.insomnia", nil, nil},
	{regexp.MustCompile(`درد مفاصل|مفاصلم درد|زانوهام درد|joint ?pain|joints? (?:hurt|ache)`), "pain.location.joints", nil, nil},
	{regexp.MustCompile(`خشکی واژن|vaginal dryness`), "urogenital.symptoms.vaginal_dryness", nil, nil},
	{regexp.MustCompile(`میل جنسی\S* کم|بیمیل|low libido|no sex drive`), "urogenital.symptoms.low_libido", nil, nil},
	{regexp.MustCompile(`لکه ?بینی|spotting`), "bleeding.presence", "spotting", nil},
	{regexp.MustCompile(`لکه ?بینی|spotting`), "bleeding.spotting", true, nil},
	{regexp.MustCompile(`قهوه|کافئین|coffee|caffeine`), "menopause.triggers.caffeine", true, nil},
	{regexp.MustCompile(`غذای تند|ادویه|spicy`), "menopause.triggers.spicy_food", true, nil},
	{regexp.MustCompile(`اتاق گرم|هوای گرم|warm room|hot room`), "menopause.triggers.warm_room", true, nil},
	{regexp.MustCompile(`چای داغ|نوشیدنی داغ|hot drink|hot tea`), "menopause.triggers.hot_drink", true, nil},
	{regexp.MustCompile(`مسکن|ایبوپروفن|استامینوفن|ژلوفن|مفنامیک|ناپروکسن|painkiller|ibuprofen|paracetamol|acetaminophen|naproxen`), "pain.relief.painkiller", true, nil},
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
	numbered := numberWords(text)
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
	score := -1
	if m := fakeScore.FindStringSubmatch(numbered); m != nil {
		score, _ = strconv.Atoi(m[1])
		if score >= 0 && score <= 10 {
			level = scoreLevel(score)
		} else {
			score = -1
		}
	}
	var out []Candidate
	seen := map[string]bool{}
	add := func(key string, value any, conf float64, alts ...string) bool {
		v, ok := vocab[key]
		if !ok || seen[key] {
			return false
		}
		if value == nil {
			value = itemValue(v, level)
		}
		seen[key] = true
		var keep []string
		for _, a := range alts {
			if _, ok := vocab[a]; ok {
				keep = append(keep, a)
			}
		}
		out = append(out, Candidate{Key: key, Value: value, Confidence: conf, Alternatives: keep})
		return true
	}
	// Canvas fields first: a hot flash / a leak said in so many words goes to its diary, and the matching log
	// symptom is then not suggested a second time.
	fakeCanvas(text, numbered, score, add, seen)
	for _, r := range fakeLexicon {
		if r.pattern.MatchString(text) {
			add(r.key, r.value, 0.9, r.alts...)
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

// Canvas phrases (CB-VOICE-01). Counts, scores and clock times are read from the text with number words turned
// into digits (numberWords).
var (
	fakeFlash      = regexp.MustCompile(`گرگرفتگی|گر گرفتم|hot ?flash`)
	fakeFlashCount = regexp.MustCompile(`(\d{1,2}) ?(?:بار|تا|دفعه) ?گرگرفتگی|(\d{1,2}) hot ?flash|hot ?flash\S* (\d{1,2}) times`)
	fakeFlashNight = regexp.MustCompile(`دیشب|نصف ?شب|نیمه ?شب|last night|at night|during the night`)
	fakeScore      = regexp.MustCompile(`(\d{1,2}) ?(?:از|out of|/) ?10`)
	fakeAnalgesic  = regexp.MustCompile(`ایبوپروفن|استامینوفن|ژلوفن|مفنامیک|ناپروکسن|ibuprofen|paracetamol|acetaminophen|naproxen|mefenamic`)
	fakeClock      = regexp.MustCompile(`(?:ساعت|\bat) (\d{1,2})(?:[:.](\d{2}))?`)
	fakeEffectNo   = regexp.MustCompile(`کمک نکرد|اثر نکرد|اثری نداشت|didn'?t help|did not help|no help`)
	fakeEffectBit  = regexp.MustCompile(`(?:کمی|یه ?کم|یک کم) (?:کمک|اثر)|helped (?:a little|a bit|somewhat)`)
	fakeEffectYes  = regexp.MustCompile(`کمک کرد|اثر کرد|بهتر شدم|helped|it worked`)
	fakeMissed     = regexp.MustCompile(`نتونستم برم (?:سر کار|دانشگاه|مدرسه|کلاس)|(?:سر کار|دانشگاه|مدرسه) نرفتم|نرفتم (?:سر کار|دانشگاه|مدرسه)|couldn'?t go to (?:work|school|class)|missed (?:work|school|class)`)
	fakePillMissed = regexp.MustCompile(`قرصم(?: رو| را|و)? (?:یادم رفت|نخوردم|جا موند|جا ماند)|یادم رفت قرصم|forgot (?:to take )?my pill|missed my pill|didn'?t take my pill`)
	fakePillTaken  = regexp.MustCompile(`قرصم(?: رو| را|و)? (?:\S+ ){0,3}?خوردم|قرص (?:ضد ?بارداری|پیشگیری)\S*(?: رو| را)? (?:\S+ ){0,3}?خوردم|took my (?:birth control |contraceptive )?pill`)
	fakeLeak       = regexp.MustCompile(`نشت|بی ?اختیاری|چکه کرد|leak|incontinen`)
	fakeLeakCough  = regexp.MustCompile(`سرفه|عطسه|خنده|خندیدم|ورزش|cough|sneez|laugh|exercis`)
	fakeLeakUrge   = regexp.MustCompile(`فوری|ناگهانی|نرسیدم|urgen|couldn'?t make it`)
	fakeNightVoids = regexp.MustCompile(`(\d{1,2}) ?(?:بار|دفعه) (?:\S+ ){0,3}?(?:دستشویی|ادرار|توالت)|(?:got up|woke up) (\d{1,2}) times (?:at night|during the night)|(\d{1,2}) times at night to (?:pee|urinate)`)
)

// fakeCanvas adds the canvas fields the text states. score is the "n out of 10" found (−1 when none).
func fakeCanvas(text, numbered string, score int, add func(string, any, float64, ...string) bool, seen map[string]bool) {
	if fakeFlash.MatchString(text) {
		n := 1
		if m := fakeFlashCount.FindStringSubmatch(numbered); m != nil {
			n = firstInt(m[1:])
		}
		if add("hot_flash.count", float64(n), 0.9) {
			seen["symptoms.general.hot_flashes"] = true
			if fakeFlashNight.MatchString(text) {
				add("hot_flash.night", true, 0.85)
			}
		}
	}
	if score >= 0 {
		add("pain_diary.score", float64(score), 0.9)
	}
	if m := fakeAnalgesic.FindString(text); m != "" && add("pain_diary.analgesic", m, 0.85) {
		if c := fakeClock.FindStringSubmatch(numbered); c != nil {
			h, _ := strconv.Atoi(c[1])
			mm := "00"
			if c[2] != "" {
				mm = c[2]
			}
			if h <= 23 {
				add("pain_diary.analgesic_time", fmt.Sprintf("%02d:%s", h, mm), 0.8)
			}
		}
		switch {
		case fakeEffectNo.MatchString(text):
			add("pain_diary.analgesic_effect", "no", 0.8)
		case fakeEffectBit.MatchString(text):
			add("pain_diary.analgesic_effect", "a_little", 0.8)
		case fakeEffectYes.MatchString(text):
			add("pain_diary.analgesic_effect", "helped", 0.8)
		}
	}
	if fakeMissed.MatchString(text) {
		add("pain_diary.missed_activity", true, 0.85)
	}
	switch {
	case fakePillMissed.MatchString(text):
		add("pill.status", "missed", 0.9)
	case fakePillTaken.MatchString(text):
		add("pill.status", "taken", 0.9)
	}
	if fakeLeak.MatchString(text) {
		kind := "unexplained"
		switch {
		case fakeLeakCough.MatchString(text):
			kind = "cough"
		case fakeLeakUrge.MatchString(text):
			kind = "urgency"
		}
		if add("bladder.leak", kind, 0.9) {
			seen["urogenital.symptoms.leakage"] = true
		}
	}
	if m := fakeNightVoids.FindStringSubmatch(numbered); m != nil {
		add("bladder.night_voids", float64(firstInt(m[1:])), 0.85)
	}
}

func firstInt(groups []string) int {
	for _, g := range groups {
		if n, err := strconv.Atoi(g); err == nil {
			return n
		}
	}
	return 1
}

// scoreLevel maps a 0–10 pain score onto the taxonomy pain level (as conditions.LevelFor does).
func scoreLevel(score int) string {
	switch {
	case score >= 7:
		return "severe"
	case score >= 4:
		return "moderate"
	default:
		return "mild"
	}
}

// fakeNumbers are the number words a count, a score or a clock time is said with.
var fakeNumbers = map[string]string{
	"یک": "1", "دو": "2", "سه": "3", "چهار": "4", "پنج": "5", "شش": "6", "شیش": "6", "هفت": "7", "هشت": "8",
	"نه": "9", "ده": "10", "یازده": "11", "دوازده": "12",
	"one": "1", "two": "2", "three": "3", "four": "4", "five": "5", "six": "6", "seven": "7", "eight": "8",
	"nine": "9", "ten": "10", "eleven": "11", "twelve": "12", "once": "1 times", "twice": "2 times", "thrice": "3 times",
}

// numberWords replaces whole number words of a normalized text with digits («شش از ده» → «6 از 10»).
func numberWords(text string) string {
	words := strings.Fields(text)
	for i, w := range words {
		core := strings.TrimRight(w, "،.؛!?")
		if d, ok := fakeNumbers[core]; ok {
			words[i] = d + w[len(core):]
		}
	}
	return strings.Join(words, " ")
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

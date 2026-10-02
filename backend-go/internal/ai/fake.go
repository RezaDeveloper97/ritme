package ai

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
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
// pill taken / missed, bladder leak / night voids) and the mood ambiguity chooser, plus bleeding flow / presence
// (CB-VOICE-03b). Labels match as whole words outside the phrases the lexicon already read. Same input → same output.
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

// painAdverbs are the intensity / filler words said between a body part and «درد» («دلم خیلی درد میکنه»); the
// sentence-wide level words still decide the level.
const (
	painAdverbs   = `(?:(?:خیلی|یه ?کم|یکم|کمی|یکمی|شدید|بدجور|هم|باز|هنوز|یه ذره) ){0,2}`
	painAdverbsEn = `(?:(?:really|so|very|still|a bit|a little|kind of) ){0,2}`
)

var fakeLexicon = []lexRule{
	{regexp.MustCompile(`(?:دلم|شکمم) ` + painAdverbs + `درد|دل ?درد|درد شکم|دل پیچه|belly|stomach ?ache|abdomen|cramp`), "pain.location.abdomen", nil, nil},
	{regexp.MustCompile(`سرم ` + painAdverbs + `درد|سر ?درد|headache|head ` + painAdverbsEn + `(?:hurts|aches)`), "pain.location.head", nil, nil},
	{regexp.MustCompile(`کمرم ` + painAdverbs + `درد|کمر ?درد|back ?pain|backache|back ` + painAdverbsEn + `(?:hurts|aches)`), "pain.location.back", nil, nil},
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
	{regexp.MustCompile(`مه ?آلود|حواس\S* ?پرت|فراموشکار|تمرکز\S* (?:ندارم|نداشتم|کمه|کم شده)|نمی ?تونم تمرکز|brain ?fog|foggy|can'?t (?:concentrate|focus)|trouble (?:concentrating|focusing)`), "symptoms.general.brain_fog", nil, nil},
	{regexp.MustCompile(`تپش ?قلب|palpitation`), "symptoms.general.palpitations", nil, nil},
	{regexp.MustCompile(`افسرده|دلمرده|low mood|depressed`), "symptoms.general.low_mood", nil, nil},
	{regexp.MustCompile(`بیخوابی|خوابم نبرد|insomnia|couldn'?t sleep`), "symptoms.general.insomnia", nil, nil},
	{regexp.MustCompile(`درد مفاصل|(?:مفاصلم|زانوهام|زانوم) ` + painAdverbs + `درد|joint ?pain|joints? (?:hurt|ache)`), "pain.location.joints", nil, nil},
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
	// used are the spans the lexicon understood: label matching below does not read them again («خشکی واژن» is
	// vaginal dryness, not also «رابطه › خشکی»). Only rules whose slot the user has claim their words.
	var used [][2]int
	for _, r := range fakeLexicon {
		if r.pattern.MatchString(text) {
			add(r.key, r.value, 0.9, r.alts...)
			if _, ok := vocab[r.key]; ok {
				for _, m := range r.pattern.FindAllStringIndex(text, -1) {
					used = append(used, [2]int{m[0], m[1]})
				}
			}
		}
	}
	fakeBleeding(text, add)
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
		if utf8.RuneCountInString(label) < 3 || !wordAt(text, label, used) {
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

// Bleeding (CB-VOICE-03b): «پریودم شروع شد», «خونریزیم زیاده», "heavy period". Cycle mode logs bleeding.flow (the
// amount said, else medium), menopause mode bleeding.presence = bleeding; «خونریزی ندارم» logs presence none
// (menopause) and no flow, as does a late period («پریودم دیر کرده»). Spotting is the lexicon's.
var (
	fakeBleed     = regexp.MustCompile(`پریود|خونریزی|قاعدگی|قاعده شدم|عادت ماهانه|period|bleeding|menstruat`)
	fakeBleedNone = regexp.MustCompile(`(?:خونریزی|پریود)\S* (?:\S+ )?(?:ندارم|نداشتم|نشدم|نشده|نیستم|نکردم|نیومده|نیامده|دیر کرده|عقب افتاده)` +
		`|no (?:bleeding|period)|not bleeding|didn'?t bleed|period (?:is |was )?late|late period|missed (?:my )?period` +
		`|period (?:hasn'?t|didn'?t|has not|did not) (?:started|come)`)
	fakeBleedAmt = regexp.MustCompile(`(?:خونریزی|پریود|قاعدگی)\S* (?:\S+ ){0,2}?(خیلی زیاد|خیلی شدید|خیلی کم|زیاد|شدید|سنگین|متوسط|معمولی|کم|خفیف)(?:ه|ی)?(?:[\s،.؛!?]|$)` +
		`|(?:period|bleeding|flow) (?:is |was |has been )?(?:\S+ ){0,1}?(very heavy|heavy|medium|moderate|normal|light)\b` +
		`|\b(very heavy|heavy|medium|moderate|normal|light) (?:period|bleeding|flow)`)
	fakeFlowCode = map[string]string{
		"خیلی زیاد": "very_heavy", "خیلی شدید": "very_heavy", "very heavy": "very_heavy",
		"زیاد": "heavy", "شدید": "heavy", "سنگین": "heavy", "heavy": "heavy",
		"متوسط": "medium", "معمولی": "medium", "medium": "medium", "moderate": "medium", "normal": "medium",
		"خیلی کم": "light", "کم": "light", "خفیف": "light", "light": "light",
	}
)

func fakeBleeding(text string, add func(string, any, float64, ...string) bool) {
	if !fakeBleed.MatchString(text) {
		return
	}
	if fakeBleedNone.MatchString(text) {
		add("bleeding.presence", "none", 0.85)
		return
	}
	add("bleeding.presence", "bleeding", 0.9)
	if m := fakeBleedAmt.FindStringSubmatch(text); m != nil {
		add("bleeding.flow", fakeFlowCode[firstNonEmpty(m[1:])], 0.85)
		return
	}
	add("bleeding.flow", "medium", 0.6) // «پریودم شروع شد»: a period, amount not said
}

func firstNonEmpty(groups []string) string {
	for _, g := range groups {
		if g != "" {
			return g
		}
	}
	return ""
}

// labelSuffixes are the Persian / English endings a label may carry and still be the same word («دلتنگم», «ترشی»).
var labelSuffixes = []string{"هایم", "هام", "های", "ها", "مون", "تون", "شون", "یم", "ام", "م", "ت", "ش", "ه", "ی", "ing", "es", "ed", "s"}

// wordAt reports whether label occurs in text as a whole word (a known suffix allowed), outside the used spans:
// «ترش» is not in «بیشترش».
func wordAt(text, label string, used [][2]int) bool {
	for from := 0; ; {
		i := strings.Index(text[from:], label)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(label)
		from = start + 1
		if r, _ := utf8.DecodeLastRuneInString(text[:start]); start > 0 && isWordRune(r) {
			continue
		}
		rest := text[end:]
		for _, suf := range labelSuffixes {
			if strings.HasPrefix(rest, suf) {
				if r, _ := utf8.DecodeRuneInString(rest[len(suf):]); rest[len(suf):] == "" || !isWordRune(r) {
					rest = rest[len(suf):]
					break
				}
			}
		}
		if r, _ := utf8.DecodeRuneInString(rest); rest != "" && isWordRune(r) {
			continue
		}
		overlaps := false
		for _, u := range used {
			if start < u[1] && end > u[0] {
				overlaps = true
				break
			}
		}
		if !overlaps {
			return true
		}
	}
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r)
}

// Canvas phrases (CB-VOICE-01). Counts, scores and clock times are read from the text with number words turned
// into digits (numberWords).
var (
	fakeFlash = regexp.MustCompile(`گرگرفتگی|گر گرفتم|hot ?flash`)
	// «N بار گرگرفتگی», also with a connector between («دو بار با گرگرفتگی … پریدم», «سه بار دچار گرگرفتگی شدم»)
	fakeFlashCount = regexp.MustCompile(`(\d{1,2}) ?(?:بار|تا|دفعه) ?(?:(?:با|از|هم|دیگه|دچار|بخاطر|به خاطر) )?(?:گرگرفتگی|گر ?گرفتم)|(\d{1,2}) hot ?flash|hot ?flash\S* (\d{1,2}) times`)
	fakeFlashNight = regexp.MustCompile(`دیشب|نصف ?شب|نیمه ?شب|last night|at night|during the night`)
	fakeScore      = regexp.MustCompile(`(\d{1,2}) ?(?:از|out of|/) ?10`)
	fakeAnalgesic  = regexp.MustCompile(`ایبوپروفن|استامینوفن|ژلوفن|مفنامیک|ناپروکسن|ibuprofen|paracetamol|acetaminophen|naproxen|mefenamic`)
	// a clock time with an optional part-of-day word right before «ساعت» or right after the time (clockHour)
	fakeClock = regexp.MustCompile(`(?:(صبح|بامداد|ظهر|بعد ?از ?ظهر|عصر|شب|نصف ?شب) )?(?:ساعت|\bat) (\d{1,2})(?:[:.](\d{2}))?` +
		`(?: (صبح|بامداد|ظهر|بعد ?از ?ظهر|عصر|شب|نصف ?شب|am|pm|a\.m\.|p\.m\.|in the morning|in the afternoon|in the evening|at night|tonight)(?:[\s،.؛!?]|$))?`)
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
			h, _ := strconv.Atoi(c[2])
			mm := "00"
			if c[3] != "" {
				mm = c[3]
			}
			if h = clockHour(h, c[1]+c[4]); h >= 0 {
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

// Part-of-day words of a clock time (fakeClock).
var (
	partMorning   = regexp.MustCompile(`صبح|بامداد|am|a\.m\.|morning`)
	partAfternoon = regexp.MustCompile(`ظهر|عصر|pm|p\.m\.|afternoon|evening`)
	partNight     = regexp.MustCompile(`شب|night|tonight`)
)

// clockHour turns a spoken hour into 0–23 (−1 when it is not one). part is the part-of-day word said with it.
//
// Rule (CB-VOICE-03b): «ساعت دو» / "at two" without a part-of-day word is a pain-diary time said during the day,
// so a bare 1–6 is read as the afternoon (13:00–18:00) — people are awake and taking painkillers then far more than
// at 01:00–06:00 — while a bare 7–12 stays as said (07:00 … 12:00, the morning dose). A part-of-day word wins:
// morning keeps the hour (12 am → 00), afternoon / evening adds 12 to 1–11, night adds 12 to 6–11 and makes
// 12 → 00 (1–5 «شب» stays after midnight). Hours 13–23 are already unambiguous.
func clockHour(h int, part string) int {
	switch {
	case h > 23:
		return -1
	case h > 12:
		return h
	case partNight.MatchString(part):
		if h == 12 {
			return 0
		}
		if h >= 6 {
			return h + 12
		}
		return h
	case partAfternoon.MatchString(part):
		if h < 12 {
			return h + 12
		}
		return h
	case partMorning.MatchString(part):
		if h == 12 {
			return 0
		}
		return h
	case h >= 1 && h <= 6:
		return h + 12
	}
	return h
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

package voicelog_test

// CB-VOICE-03 accuracy fixtures: 10 realistic Persian sentences (5 cycle-mode, 5 menopause-mode) run end to end
// through POST /logs/voice with the fake provider (transcript injected via the RITME-FAKE marker), then saved the way
// the client does — log items through PUT /logs/days with voice_params, diary items through POST /logs/voice/commit.
// "want" is what a careful human would log from the sentence; "miss" / "extra" pin the fake parser's known gaps so
// the table in docs/qa/canvas/voice.md stays honest. The real Gemini provider is never called.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/conditions"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

type accuracyCase struct {
	id, mode, text string
	want           map[string]any // target:category.param.item → value
	miss           []string       // wanted keys the fake parser does not produce (or produces with another value)
	extra          []string       // keys it produces that a human would not log
}

var accuracyCases = []accuracyCase{
	{id: "c1", mode: "cycle", text: "امروز پریودم شروع شد، خونریزیم زیاده و دلم خیلی درد می\u200cکنه",
		want: map[string]any{"log:bleeding.flow.": "heavy", "log:pain.location.abdomen": "severe"},
		miss: []string{"log:bleeding.flow.", "log:pain.location.abdomen"}},
	{id: "c2", mode: "cycle", text: "از صبح کمرم درد می\u200cکنه، هفت از ده. ساعت دو یه ژلوفن خوردم ولی اثر نکرد",
		want: map[string]any{"log:pain.location.back": "severe", "log:pain.relief.painkiller": true,
			"pain_diary:pain_diary.score.": 7.0, "pain_diary:pain_diary.analgesic.": "ژلوفن",
			"pain_diary:pain_diary.analgesic_time.": "14:00", "pain_diary:pain_diary.analgesic_effect.": "no"},
		miss: []string{"pain_diary:pain_diary.analgesic_time."}},
	{id: "c3", mode: "cycle", text: "قرص ضد بارداریم رو امروز صبح خوردم، یه کم هم سرم درد می\u200cکنه",
		want: map[string]any{"pill:pill.status.": "taken", "log:pain.location.head": "mild"}},
	{id: "c4", mode: "cycle", text: "امروز وقتی عطسه کردم یه کم ادرارم نشت کرد، شب هم سه بار برای دستشویی بیدار شدم",
		want: map[string]any{"bladder:bladder.leak.": "cough", "bladder:bladder.night_voids.": 3.0}},
	{id: "c5", mode: "cycle", text: "امروز خیلی بی\u200cحوصله\u200cام و نفخ دارم، دیشب هم بد خوابیدم",
		want: map[string]any{"log:mood.moods.bored": true, "log:symptoms.digestive.bloating": "yes", "log:sleep.quality.": "poor"}},
	{id: "m1", mode: "menopause", text: "امروز پنج بار گرگرفتگی داشتم، بیشترش بعد از چای داغ",
		want:  map[string]any{"hot_flash:hot_flash.count.": 5.0, "log:menopause.triggers.hot_drink": true},
		extra: []string{"log:appetite_energy.cravings.sour"}},
	{id: "m2", mode: "menopause", text: "دیشب دو بار با گرگرفتگی و عرق شبانه از خواب پریدم و بی\u200cخوابی داشتم",
		want: map[string]any{"hot_flash:hot_flash.count.": 2.0, "hot_flash:hot_flash.night.": true,
			"log:symptoms.general.night_sweats": "yes", "log:symptoms.general.insomnia": "yes"},
		miss: []string{"hot_flash:hot_flash.count."}},
	{id: "m3", mode: "menopause", text: "زانوهام درد می\u200cکنه، حدود پنج از ده، یه استامینوفن خوردم و کمک کرد",
		want: map[string]any{"log:pain.location.joints": "moderate", "log:pain.relief.painkiller": true}},
	{id: "m4", mode: "menopause", text: "وقتی خندیدم یه کم ادرارم چکه کرد و خشکی واژن هم اذیتم می\u200cکنه",
		want:  map[string]any{"bladder:bladder.leak.": "cough", "log:urogenital.symptoms.vaginal_dryness": "yes"},
		extra: []string{"log:sex.symptoms.dryness"}},
	{id: "m5", mode: "menopause", text: "امروز بی\u200cحوصله\u200cام، تمرکز ندارم و حواسم پرته",
		want: map[string]any{"log:mood.moods.bored": true, "log:symptoms.general.brain_fog": "yes"},
		miss: []string{"log:symptoms.general.brain_fog"}},
}

type accSuggestion struct {
	key, target, category, param, item string
	value                              any
	options                            int
}

func accSuggestions(r resp) []accSuggestion {
	list, _ := r.data()["suggestions"].([]any)
	out := []accSuggestion{}
	for _, x := range list {
		m := x.(map[string]any)
		item, _ := m["item"].(string)
		opts, _ := m["options"].([]any)
		s := accSuggestion{target: m["target"].(string), category: m["category"].(string), param: m["param"].(string),
			item: item, value: m["value"], options: len(opts)}
		s.key = s.target + ":" + s.category + "." + s.param + "." + item
		out = append(out, s)
	}
	return out
}

func TestVoice_AccuracyFixtures(t *testing.T) {
	e, _ := setupVoice(t, fakeClient)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

	totalWant, totalHit, exact := 0, 0, 0
	for i, c := range accuracyCases {
		key := "acc_" + string(rune('a'+i))
		ai.FakeTranscripts[key] = map[string]string{"fa": c.text}
		t.Cleanup(func() { delete(ai.FakeTranscripts, key) })

		uid, tok := e.user(t, fmt.Sprintf("0912000%04d", 300+i), true)
		if c.mode == "menopause" {
			_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, gender, created_at, updated_at) VALUES (?, 'menopause', 'female', NOW(), NOW())`, uid)
			require.NoError(t, err)
		} else { // cycle user with the endo programme and a combined pill: every cycle-mode diary is offered
			require.NoError(t, e.conds.Enrol(ctx, uid, conditions.ProgramEndo, civildate.MustParse("2026-09-01"), now))
			require.NoError(t, e.contr.SaveMethod(ctx, uid, contraception.Input{Method: contraception.MethodCombinedPill,
				PackType: "21_7", PackStartedOn: civildate.MustParse("2026-09-16")}, now, "fa"))
		}

		r := e.voice(t, tok, "fa", key)
		require.Equal(t, 200, r.status, r.raw)
		got := accSuggestions(r)
		gotMap := map[string]any{}
		for _, s := range got {
			gotMap[s.key] = s.value
		}

		// accuracy: a wanted key counts when it is suggested with the wanted value
		var misses, extras []string
		for k, v := range c.want {
			if g, ok := gotMap[k]; ok && fmt.Sprint(g) == fmt.Sprint(v) {
				totalHit++
			} else {
				misses = append(misses, k)
			}
		}
		for k := range gotMap {
			if _, ok := c.want[k]; !ok {
				extras = append(extras, k)
			}
		}
		totalWant += len(c.want)
		sort.Strings(misses)
		sort.Strings(extras)
		wantMiss := append([]string{}, c.miss...)
		sort.Strings(wantMiss)
		wantExtra := append([]string{}, c.extra...)
		sort.Strings(wantExtra)
		assert.Equal(t, nilIfEmpty(wantMiss), nilIfEmpty(misses), "%s misses (got %v)", c.id, gotMap)
		assert.Equal(t, nilIfEmpty(wantExtra), nilIfEmpty(extras), "%s extras (got %v)", c.id, gotMap)
		if len(misses) == 0 && len(extras) == 0 {
			exact++
		}

		// save like the client: log items first (PUT /logs/days with voice_params), then the diaries
		cats := map[string]any{}
		var params, diary []string
		for _, s := range got {
			if s.target != "log" {
				diary = append(diary, fmt.Sprintf(`{"category":%q,"param":%q,"value":%s}`, s.category, s.param, mustJSON(t, s.value)))
				continue
			}
			cat, _ := cats[s.category].(map[string]any)
			if cat == nil {
				cat = map[string]any{}
				cats[s.category] = cat
			}
			switch {
			case s.item == "":
				cat[s.param] = s.value
			case s.value == true: // multi: the chosen item codes
				list, _ := cat[s.param].([]string)
				cat[s.param] = append(list, s.item)
			default:
				p, _ := cat[s.param].(map[string]any)
				if p == nil {
					p = map[string]any{}
					cat[s.param] = p
				}
				p[s.item] = s.value
			}
			params = append(params, s.category+"."+s.param)
		}
		logOK, commitOK := "—", "—"
		if len(cats) > 0 {
			put := e.do(t, "PUT", "/api/v1/logs/days/2026-09-23", tok, "fa", "application/json", strings.NewReader(
				mustJSON(t, map[string]any{"categories": cats, "voice_params": params})))
			assert.Equal(t, 200, put.status, "%s PUT: %s", c.id, put.raw)
			logOK = fmt.Sprint(put.status)
		}
		if len(diary) > 0 {
			cm := e.commit(t, tok, "fa", `{"date":"2026-09-23","items":[`+strings.Join(diary, ",")+`]}`)
			assert.Equal(t, 200, cm.status, "%s commit: %s", c.id, cm.raw)
			commitOK = fmt.Sprint(cm.status)
		}
		var gotKeys []string
		for _, s := range got {
			v := fmt.Sprint(s.value)
			if s.options > 0 {
				v += fmt.Sprintf(" (+%d options)", s.options)
			}
			gotKeys = append(gotKeys, s.key+"="+v)
		}
		t.Logf("ACC %s | %s | got: %s | miss: %v | extra: %v | PUT %s | commit %s",
			c.id, c.mode, strings.Join(gotKeys, ", "), misses, extras, logOK, commitOK)
	}
	assert.Equal(t, 22, totalHit, "wanted items found")
	assert.Equal(t, 27, totalWant)
	t.Logf("ACC total: %d/%d wanted items (%.0f%%), %d/%d sentences exact",
		totalHit, totalWant, 100*float64(totalHit)/float64(totalWant), exact, len(accuracyCases))
}

func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

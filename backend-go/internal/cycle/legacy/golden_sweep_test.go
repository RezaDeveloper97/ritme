package legacy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Characterization tests: Laravel's cycle-sweep goldens (contract/golden/cycle-sweep, recorded by
// T-M2-05 with a frozen clock) replayed through the port. `calculation` of every /cycle/date step
// (all personas × 60 days × fa/en) and `calculations` + `month_summary` of every /cycle/month step
// (3 months × full/calendar × fa/en) must equal the recorded value — semantic JSON equality, no
// allow-list. Inputs come from the fixture the goldens were recorded against (contract/fixtures/dump.sql).

const (
	goldenDir  = "../../../contract/golden/cycle-sweep"
	fixtureSQL = "../../../contract/fixtures/dump.sql"
)

var _ view.BaseCalc = Calculation{}

// sweepPersonas mirrors cmd/contract/personas.go (ContractFixtureSeeder order; blocked has no session).
var sweepPersonas = map[string]int64{
	"no_profile": 1001, "profile_no_history": 1002, "onboarding_declared": 1003, "regular": 1004,
	"irregular": 1005, "short_outlier": 1006, "open_period_day3": 1007, "open_period_day11": 1008,
	"open_period_day13": 1009, "overdue_10": 1010, "overdue_20": 1011, "ttc": 1012, "premium": 1013,
	"pregnant_lmp_w8": 1014, "pregnant_ultrasound_w26": 1015, "pregnant_manual_w14": 1016, "engaged": 1018,
}

type goldenStep struct {
	Request struct {
		URL            string `json:"url"`
		AcceptLanguage string `json:"accept_language"`
		Now            string `json:"now"`
	} `json:"request"`
	Status int `json:"status"`
	Body   struct {
		Data struct {
			Calculation  json.RawMessage `json:"calculation"`
			Calculations json.RawMessage `json:"calculations"`
			MonthSummary json.RawMessage `json:"month_summary"`
		} `json:"data"`
	} `json:"body"`
}

var monthURL = regexp.MustCompile(`/cycle/month/(\d+)/(\d+)`)

func TestLegacyGoldenSweep(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(goldenDir, "*.json"))
	require.NoError(t, err)
	if len(files) == 0 {
		t.Skip("cycle-sweep goldens not recorded (T-M2-05)")
	}
	fx := loadFixture(t)
	ctx := context.Background()
	days, months, rewritten := 0, 0, 0

	for _, file := range files {
		parts := strings.Split(filepath.Base(file), ".") // <kind>.<persona>.<locale>.json
		require.Len(t, parts, 4, file)
		kind, persona, locale := parts[0], parts[1], parts[2]
		userID, known := sweepPersonas[persona]
		require.True(t, known, "unknown persona %q", persona)

		t.Run(kind+"."+persona+"."+locale, func(t *testing.T) {
			raw, err := os.ReadFile(file)
			require.NoError(t, err)
			var g struct{ Steps []goldenStep }
			require.NoError(t, json.Unmarshal(raw, &g))
			require.NotEmpty(t, g.Steps)

			for _, step := range g.Steps {
				require.Equal(t, 200, step.Status, step.Request.URL)
				now, err := time.Parse(time.RFC3339, step.Request.Now)
				require.NoError(t, err)
				engine := New(Input{
					Profile:   fx.profiles[userID],
					Histories: fx.histories[userID],
					Logs:      fx.logs[userID],
					Today:     civildate.InTehran(now),
					Tips:      recommendation.New(fx.recs),
				})

				switch kind {
				case "date":
					date := civildate.MustParse(step.Request.URL[strings.LastIndex(step.Request.URL, "/")+1:])
					calc, err := engine.CalculateForDisplay(ctx, date)
					require.NoError(t, err)
					want := applyD30(t, engine, date, step.Body.Data.Calculation, step.Request.AcceptLanguage)
					if !assertJSON(t, want, calc.Localize(step.Request.AcceptLanguage), step.Request.URL) {
						return
					}
					days++
					if !bytes.Equal(want, step.Body.Data.Calculation) {
						rewritten++
					}
				case "month_full", "month_calendar":
					m := monthURL.FindStringSubmatch(step.Request.URL)
					require.NotNil(t, m, step.Request.URL)
					year, _ := strconv.Atoi(m[1])
					month, _ := strconv.Atoi(m[2])
					start := civildate.New(year, time.Month(month), 1)
					var calcs []Calculation
					for d := start; d.Month == start.Month; d = d.AddDays(1) {
						c, err := engine.CalculateForDate(ctx, d, kind == "month_full")
						require.NoError(t, err)
						calcs = append(calcs, c)
					}
					if !assertJSON(t, step.Body.Data.Calculations, calcs, step.Request.URL) ||
						!assertJSON(t, step.Body.Data.MonthSummary, SummarizeMonth(calcs), step.Request.URL+" summary") {
						return
					}
					months++
				default:
					t.Fatalf("unknown golden kind %q", kind)
				}
			}
		})
	}
	t.Logf("%d golden days (%d rewritten by D-30) and %d golden months compared across %d files", days, rewritten, months, len(files))
	assert.Positive(t, days)
	assert.Positive(t, rewritten, "the fixture has O + 1 days")
	assert.Positive(t, months)
}

// applyD30 rewrites Laravel's /cycle/date calculation to the §19 display window Go serves since
// T-M2-36 (deviations.md D-30): on the legacy O + 1 day (phase ovulation after the ovulation day)
// the phase is luteal, is_fertile_window false, the fertility_status flag is gone, phase_info names
// the luteal phase and daily_tips are the early-luteal ones (luteal + early_luteal, same log; the
// emitted sub-phase stays post_ovulation). Every other
// day, and every other field, is compared unchanged.
func applyD30(t *testing.T, e *Engine, date civildate.Date, raw json.RawMessage, locale string) json.RawMessage {
	t.Helper()
	var want map[string]any
	require.NoError(t, json.Unmarshal(raw, &want))
	day, _ := want["cycle_day"].(float64)
	ovulation, _ := want["estimated_ovulation_day"].(float64)
	if want["phase"] != string(enums.CyclePhaseOvulation) || day <= ovulation {
		return raw
	}
	sub := enums.CycleSubphase(want["subphase"].(string))
	luteal := enums.CyclePhaseLuteal
	want["phase"] = string(luteal)
	want["is_fertile_window"] = false

	flags, _ := want["text_flags"].(map[string]any)
	delete(flags, "fertility_status")
	phaseInfo := func(l string) string {
		prefix := map[string]string{"en": "Current phase: ", "fa": "فاز فعلی: "}[l]
		return prefix + luteal.Label(l) + " (" + sub.Label(l) + ")"
	}
	if _, bilingual := flags["phase_info"].(map[string]any); bilingual {
		flags["phase_info"] = map[string]any{"en": phaseInfo("en"), "fa": phaseInfo("fa")}
	} else {
		flags["phase_info"] = phaseInfo(locale)
	}

	tips, err := e.dailyTips(context.Background(), luteal, enums.CycleSubphaseEarlyLuteal, e.in.Logs[date])
	require.NoError(t, err)
	var tipsJSON any
	b, err := json.Marshal(recommendation.Localize(tips, locale))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &tipsJSON))
	want["daily_tips"] = tipsJSON

	out, err := json.Marshal(want)
	require.NoError(t, err)
	return out
}

func assertJSON(t *testing.T, want json.RawMessage, got any, msg string) bool {
	t.Helper()
	require.NotEmpty(t, want, msg)
	b, err := json.Marshal(got)
	require.NoError(t, err)
	var w, h any
	require.NoError(t, json.Unmarshal(want, &w))
	require.NoError(t, json.Unmarshal(b, &h))
	return assert.Equal(t, w, h, msg)
}

// ---------------------------------------------------------------------------
// Fixture

type fixture struct {
	histories map[int64][]model.History
	profiles  map[int64]*model.Profile
	logs      map[int64]map[civildate.Date]*DailyLog
	recs      *sliceSource
}

type sliceSource struct{ rows []recommendation.Row }

func (s *sliceSource) ActiveRows(context.Context) ([]recommendation.Row, error) {
	var out []recommendation.Row
	for _, r := range s.rows {
		if r.IsActive {
			out = append(out, r)
		}
	}
	slices.SortStableFunc(out, func(a, b recommendation.Row) int {
		if a.SortOrder != b.SortOrder {
			return a.SortOrder - b.SortOrder
		}
		return int(a.ID - b.ID)
	})
	return out, nil
}

func (s *sliceSource) AnyExists(context.Context) (bool, error) { return len(s.rows) > 0, nil }

var createTable = regexp.MustCompile("^CREATE TABLE `([a-z_]+)`")
var columnDef = regexp.MustCompile("^  `([a-z_]+)` ")

// loadFixture reads the tables the legacy engine consumes from the contract dump.
func loadFixture(t *testing.T) fixture {
	t.Helper()
	f, err := os.Open(fixtureSQL)
	require.NoError(t, err)
	defer f.Close()

	fx := fixture{
		histories: map[int64][]model.History{},
		profiles:  map[int64]*model.Profile{},
		logs:      map[int64]map[civildate.Date]*DailyLog{},
		recs:      &sliceSource{},
	}
	columns := map[string][]string{}
	creating, table := "", ""

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		if m := createTable.FindStringSubmatch(line); m != nil {
			creating = m[1]
			continue
		}
		if creating != "" {
			if m := columnDef.FindStringSubmatch(line); m != nil {
				columns[creating] = append(columns[creating], m[1])
				continue
			}
			if strings.HasPrefix(line, ")") {
				creating = ""
			}
			continue
		}
		if strings.HasPrefix(line, "INSERT INTO `") {
			table = strings.TrimSuffix(strings.TrimPrefix(line, "INSERT INTO `"), "` VALUES")
			continue
		}
		if !strings.HasPrefix(line, "(") {
			table = ""
			continue
		}
		cols := columns[table]
		switch table {
		case "cycle_histories", "user_profiles", "daily_health_logs", "recommendations":
		default:
			continue
		}
		vals := parseTuple(t, line)
		require.Len(t, vals, len(cols), "%s: %s", table, line)
		row := map[string]*string{}
		for i, c := range cols {
			row[c] = vals[i]
		}
		switch table {
		case "cycle_histories":
			h := model.History{
				ID:             mustInt64(t, row["id"]),
				PeriodStart:    civildate.MustParse(*row["period_start_date"]),
				IsConfirmed:    *row["is_confirmed"] == "1",
				IsEstimated:    *row["is_estimated"] == "1",
				Source:         *row["source"],
				CycleLength:    optInt(t, row["cycle_length"]),
				BleedingLength: optInt(t, row["bleeding_length"]),
			}
			if row["period_end_date"] != nil {
				h.PeriodEnd = civildate.MustParse(*row["period_end_date"])
			}
			if row["data_quality_flags"] != nil {
				require.NoError(t, json.Unmarshal([]byte(*row["data_quality_flags"]), &h.DataQualityFlags))
			}
			uid := mustInt64(t, row["user_id"])
			fx.histories[uid] = append(fx.histories[uid], h)
		case "user_profiles":
			pr := &model.Profile{
				PeriodDuration: optInt(t, row["period_duration"]),
				CycleDuration:  optInt(t, row["cycle_duration"]),
			}
			if row["birthday"] != nil {
				pr.Birthday = civildate.MustParse(*row["birthday"])
			}
			if row["last_period_start"] != nil {
				pr.LastPeriodStart = civildate.MustParse(*row["last_period_start"])
			}
			fx.profiles[mustInt64(t, row["user_id"])] = pr
		case "daily_health_logs":
			uid := mustInt64(t, row["user_id"])
			if fx.logs[uid] == nil {
				fx.logs[uid] = map[civildate.Date]*DailyLog{}
			}
			fx.logs[uid][civildate.MustParse(*row["log_date"])] = dailyLogFromRow(t, cols, row)
		case "recommendations":
			fx.recs.rows = append(fx.recs.rows, recommendation.Row{
				ID:             mustInt64(t, row["id"]),
				Key:            row["key"],
				Type:           *row["type"],
				Title:          rawJSON(row["title"]),
				Text:           rawJSON(row["text"]),
				CyclePhase:     row["cycle_phase"],
				CycleSubphases: rawJSON(row["cycle_subphases"]),
				SymptomTrigger: row["symptom_trigger"],
				IsActive:       *row["is_active"] == "1",
				SortOrder:      int(mustInt64(t, row["sort_order"])),
			})
		}
	}
	require.NoError(t, sc.Err())
	require.NotEmpty(t, fx.histories)
	require.NotEmpty(t, fx.profiles)
	require.NotEmpty(t, fx.logs)
	require.NotEmpty(t, fx.recs.rows)
	return fx
}

// DailyHealthLog casts (backend/app/Models/DailyHealthLog.php:200).
var (
	logBoolCols = []string{
		"has_clots", "spotting", "diarrhea", "constipation", "food_craving", "vaginal_dryness",
		"vaginal_burning", "vaginal_itching", "vaginal_smell_change", "acne", "oily_skin", "hair_loss",
		"swelling", "fatigue", "dizziness", "hot_flashes", "chills", "discharge_itching",
		"discharge_burning", "frequent_urination",
	}
	logIntCols     = []string{"id", "user_id", "heart_rate", "systolic_pressure", "diastolic_pressure", "exercise_duration"}
	logDecimalCols = map[string]int{"weight": 2, "basal_body_temperature": 2, "blood_sugar": 1}
	logArrayCols   = []string{"moods", "exercise_type", "sexual_activities", "medications"}
	logTimeCols    = []string{"created_at", "updated_at"}
)

// dailyLogFromRow builds the engine input plus DailyHealthLog::toArray() for one dump row.
func dailyLogFromRow(t *testing.T, cols []string, row map[string]*string) *DailyLog {
	t.Helper()
	src := jsonx.NewObject()
	for _, c := range cols {
		v := row[c]
		switch {
		case v == nil:
			src.Set(c, nil)
		case slices.Contains(logBoolCols, c):
			src.Set(c, *v == "1")
		case slices.Contains(logIntCols, c):
			src.Set(c, mustInt64(t, v))
		case logDecimalCols[c] > 0:
			src.Set(c, jsonx.MustDecimal(*v, logDecimalCols[c]))
		case slices.Contains(logArrayCols, c):
			src.Set(c, json.RawMessage(*v))
		case slices.Contains(logTimeCols, c):
			ts, err := time.ParseInLocation(time.DateTime, *v, civildate.Tehran)
			require.NoError(t, err)
			src.Set(c, jsonx.DateTime(ts))
		default:
			src.Set(c, *v)
		}
	}

	boolPtr := func(c string) *bool {
		if row[c] == nil {
			return nil
		}
		b := *row[c] == "1"
		return &b
	}
	list := func(c string) []string {
		if row[c] == nil {
			return nil
		}
		var out []string
		if json.Unmarshal([]byte(*row[c]), &out) != nil {
			return nil
		}
		return out
	}
	return &DailyLog{
		Spotting:             boolPtr("spotting"),
		VaginalDryness:       boolPtr("vaginal_dryness"),
		Fatigue:              boolPtr("fatigue"),
		DischargeTexture:     row["discharge_texture"],
		OvarianPainIntensity: row["ovarian_pain_intensity"],
		BloatingIntensity:    row["bloating_intensity"],
		HeadacheIntensity:    row["headache_intensity"],
		PelvicPainIntensity:  row["pelvic_pain_intensity"],
		StomachAcheIntensity: row["stomach_ache_intensity"],
		SleepQuality:         row["sleep_quality"],
		SexualDesire:         row["sexual_desire"],
		SexualActivities:     list("sexual_activities"),
		Moods:                list("moods"),
		Source:               src,
	}
}

func rawJSON(v *string) json.RawMessage {
	if v == nil {
		return nil
	}
	return json.RawMessage(*v)
}

// parseTuple splits one mysqldump "(v1,'v2',NULL,...)," row; NULL → nil.
func parseTuple(t *testing.T, line string) []*string {
	t.Helper()
	s := strings.TrimSuffix(strings.TrimSuffix(line, ";"), ",")
	require.True(t, strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")"), line)
	s = s[1 : len(s)-1]

	var out []*string
	var cur strings.Builder
	quoted, inQuote := false, false
	flush := func() {
		v := cur.String()
		if !quoted && v == "NULL" {
			out = append(out, nil)
		} else {
			out = append(out, &v)
		}
		cur.Reset()
		quoted = false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inQuote && c == '\\' && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
		case inQuote && c == '\'':
			inQuote = false
		case !inQuote && c == '\'':
			inQuote, quoted = true, true
		case !inQuote && c == ',':
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}

func mustInt64(t *testing.T, v *string) int64 {
	t.Helper()
	require.NotNil(t, v)
	n, err := strconv.ParseInt(*v, 10, 64)
	require.NoError(t, err)
	return n
}

func optInt(t *testing.T, v *string) *int {
	t.Helper()
	if v == nil {
		return nil
	}
	n, err := strconv.Atoi(*v)
	require.NoError(t, err, fmt.Sprint(*v))
	return &n
}

package view

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// TestCycleViewGoldenSweep replays Laravel's /cycle/date sweep (contract/golden/cycle-sweep,
// recorded by T-M2-05: every persona × fa/en × 60 consecutive days around CONTRACT_TODAY) through
// Build and requires cycle_view to equal the recorded value (semantic JSON equality).
//
// Inputs come from the same fixture the goldens were recorded against (contract/fixtures/dump.sql:
// cycle_histories + user_profiles). The legacy part (calculation.cycle_day / subphase — the
// HealthDataEngine port is T-M2-14) is taken from the golden's own `calculation` block.

const (
	goldenDir  = "../../../contract/golden/cycle-sweep"
	fixtureSQL = "../../../contract/fixtures/dump.sql"
)

// sweepPersonas mirrors cmd/contract/personas.go (ContractFixtureSeeder order; blocked has no session).
var sweepPersonas = map[string]int64{
	"no_profile": 1001, "profile_no_history": 1002, "onboarding_declared": 1003, "regular": 1004,
	"irregular": 1005, "short_outlier": 1006, "open_period_day3": 1007, "open_period_day11": 1008,
	"open_period_day13": 1009, "overdue_10": 1010, "overdue_20": 1011, "ttc": 1012, "premium": 1013,
	"pregnant_lmp_w8": 1014, "pregnant_ultrasound_w26": 1015, "pregnant_manual_w14": 1016, "engaged": 1018,
}

type goldenCalc struct {
	day *int
	sub string
}

func (g goldenCalc) CycleDay() (int, bool) {
	if g.day == nil {
		return 0, false
	}
	return *g.day, true
}

func (g goldenCalc) Subphase() enums.CycleSubphase { return enums.CycleSubphase(g.sub) }

type goldenFile struct {
	Steps []struct {
		Request struct {
			URL            string `json:"url"`
			AcceptLanguage string `json:"accept_language"`
			Now            string `json:"now"`
		} `json:"request"`
		Status int `json:"status"`
		Body   struct {
			Data struct {
				Calculation struct {
					CycleDay *int    `json:"cycle_day"`
					Subphase *string `json:"subphase"`
				} `json:"calculation"`
				CycleView json.RawMessage `json:"cycle_view"`
			} `json:"data"`
		} `json:"body"`
	} `json:"steps"`
}

func TestCycleViewGoldenSweep(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(goldenDir, "date.*.json"))
	require.NoError(t, err)
	if len(files) == 0 {
		t.Skip("cycle-sweep goldens not recorded (T-M2-05)")
	}

	histories, profiles := loadFixture(t)
	steps := 0

	for _, file := range files {
		parts := strings.Split(filepath.Base(file), ".") // date.<persona>.<locale>.json
		require.Len(t, parts, 4, file)
		persona := parts[1]
		userID, known := sweepPersonas[persona]
		require.True(t, known, "unknown persona %q", persona)

		t.Run(persona+"."+parts[2], func(t *testing.T) {
			raw, err := os.ReadFile(file)
			require.NoError(t, err)
			var g goldenFile
			require.NoError(t, json.Unmarshal(raw, &g))
			require.NotEmpty(t, g.Steps)

			for _, step := range g.Steps {
				require.Equal(t, 200, step.Status, step.Request.URL)
				selected := civildate.MustParse(step.Request.URL[strings.LastIndex(step.Request.URL, "/")+1:])
				now, err := time.Parse(time.RFC3339, step.Request.Now)
				require.NoError(t, err)

				calc := goldenCalc{day: step.Body.Data.Calculation.CycleDay}
				if step.Body.Data.Calculation.Subphase != nil {
					calc.sub = *step.Body.Data.Calculation.Subphase
				}

				view := Build(histories[userID], profiles[userID], selected, civildate.InTehran(now), step.Request.AcceptLanguage, calc)
				got, err := json.Marshal(view)
				require.NoError(t, err)

				var want, have any
				require.NoError(t, json.Unmarshal(step.Body.Data.CycleView, &want))
				require.NoError(t, json.Unmarshal(got, &have))
				if !assert.Equal(t, want, have, "%s %s", step.Request.URL, step.Request.AcceptLanguage) {
					return
				}
				steps++
			}
		})
	}
	t.Logf("%d golden days compared across %d persona×locale files", steps, len(files))
}

// loadFixture reads cycle_histories (newest first per user, as HealthDataEngine loads them) and
// user_profiles from the contract dump.
func loadFixture(t *testing.T) (map[int64][]model.History, map[int64]*model.Profile) {
	t.Helper()
	f, err := os.Open(fixtureSQL)
	require.NoError(t, err)
	defer f.Close()

	histories := map[int64][]model.History{}
	profiles := map[int64]*model.Profile{}
	table := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "INSERT INTO `cycle_histories`"):
			table = "cycle_histories"
			continue
		case strings.HasPrefix(line, "INSERT INTO `user_profiles`"):
			table = "user_profiles"
			continue
		case !strings.HasPrefix(line, "("):
			table = ""
			continue
		}
		if table == "" {
			continue
		}
		row := parseTuple(t, line)
		switch table {
		case "cycle_histories":
			uid := mustInt64(t, row[1])
			h := model.History{
				ID:          mustInt64(t, row[0]),
				PeriodStart: civildate.MustParse(*row[2]),
				IsConfirmed: *row[6] == "1",
				IsEstimated: *row[7] == "1",
				Source:      *row[8],
			}
			if row[3] != nil {
				h.PeriodEnd = civildate.MustParse(*row[3])
			}
			h.CycleLength = optInt(t, row[4])
			h.BleedingLength = optInt(t, row[5])
			if row[9] != nil {
				require.NoError(t, json.Unmarshal([]byte(*row[9]), &h.DataQualityFlags))
			}
			histories[uid] = append(histories[uid], h)
		case "user_profiles":
			uid := mustInt64(t, row[1])
			pr := &model.Profile{
				PeriodDuration: optInt(t, row[5]),
				CycleDuration:  optInt(t, row[6]),
				Goal:           *row[8],
			}
			if row[2] != nil {
				pr.Birthday = civildate.MustParse(*row[2])
			}
			if row[7] != nil {
				pr.LastPeriodStart = civildate.MustParse(*row[7])
			}
			profiles[uid] = pr
		}
	}
	require.NoError(t, sc.Err())
	require.NotEmpty(t, histories)
	require.NotEmpty(t, profiles)

	for uid := range histories {
		histories[uid] = newestFirst(histories[uid])
	}
	return histories, profiles
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

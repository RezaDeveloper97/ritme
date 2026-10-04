package service_test

// B-N2-11b (N2 stage smoke B-3) + CB-TEEN-04b: a stored teen / menopause life mode keeps /cycle/today|date
// off the fertile window — no «پنجره باروری» / ovulation / pregnancy-chance copy in the daily card, the
// text flags or the daily tips on any day of the cycle — while the same user without a stored mode still
// gets it (contract payloads unchanged).

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestDayJSON_NoFertilityCopyForTeenAndMenopause(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "..", "contract", "fixtures", "dump.sql"))
	svc := service.New(db, cache.New(nil, false, nil))
	ctx := context.Background()
	const uid = 1004 // regular
	today := civildate.MustParse("2026-09-01")

	// copyOf is the day's user-facing copy: the daily card, the calculation's text flags and daily tips
	// (CB-TEEN-04b) — enum values and numbers are data, not copy.
	copyOf := func(raw []byte) string {
		var body struct {
			Calc struct {
				Flags map[string]any `json:"text_flags"`
				Tips  []struct {
					Type  string `json:"type"`
					Text  string `json:"text"`
					Title string `json:"title"`
				} `json:"daily_tips"`
			} `json:"calculation"`
			View struct {
				Card struct {
					Title          string   `json:"title"`
					Subtitle       string   `json:"subtitle"`
					FertilityLabel string   `json:"fertility_label"`
					Badges         []string `json:"badges"`
				} `json:"daily_card"`
			} `json:"cycle_view"`
		}
		require.NoError(t, json.Unmarshal(raw, &body), string(raw))
		parts := []string{body.View.Card.Title, body.View.Card.Subtitle, body.View.Card.FertilityLabel}
		parts = append(parts, body.View.Card.Badges...)
		for _, v := range body.Calc.Flags {
			if s, ok := v.(string); ok {
				parts = append(parts, s)
			}
		}
		for _, tip := range body.Calc.Tips {
			parts = append(parts, tip.Type, tip.Title, tip.Text)
		}
		return strings.ToLower(strings.Join(parts, " "))
	}
	fertilityWords := []string{"باروری", "تخمک", "بارداری", "اوج", "fertil", "ovulat", "pregnan", "conceive", "peak"}
	mentionsFertility := func(locale string) bool {
		for d := today; d.Compare(today.AddDays(35)) < 0; d = d.AddDays(1) {
			p, err := svc.DayJSON(ctx, uid, d, today, locale)
			require.NoError(t, err)
			text := copyOf(p.JSON)
			for _, w := range fertilityWords {
				if strings.Contains(text, w) {
					return true
				}
			}
		}
		return false
	}

	for _, locale := range []string{"fa", "en"} {
		assert.True(t, mentionsFertility(locale), "a cycle user's card talks about the fertile window (%s)", locale)
	}
	for _, mode := range []string{"teen", "menopause"} {
		_, err := db.ExecContext(ctx, "DELETE FROM user_life_profiles WHERE user_id = ?", uid)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, "INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, ?, NOW(), NOW())", uid, mode)
		require.NoError(t, err)
		for _, locale := range []string{"fa", "en"} {
			assert.False(t, mentionsFertility(locale), "%s %s", mode, locale)
		}
	}
}

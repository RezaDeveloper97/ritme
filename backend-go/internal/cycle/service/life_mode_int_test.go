package service_test

// B-N2-11b (N2 stage smoke B-3): a stored teen / menopause life mode keeps /cycle/today|date's
// daily card off the fertile window — no «پنجره باروری» / ovulation copy on any day of the cycle —
// while the same user without a stored mode still gets it (contract payloads unchanged).

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

	mentionsFertility := func(locale string) bool {
		for d := today; d.Compare(today.AddDays(35)) < 0; d = d.AddDays(1) {
			p, err := svc.DayJSON(ctx, uid, d, today, locale)
			require.NoError(t, err)
			var body struct {
				View struct {
					Card struct {
						Title    string `json:"title"`
						Subtitle string `json:"subtitle"`
					} `json:"daily_card"`
				} `json:"cycle_view"`
			}
			require.NoError(t, json.Unmarshal(p.JSON, &body), string(p.JSON))
			text := strings.ToLower(body.View.Card.Title + " " + body.View.Card.Subtitle)
			for _, w := range []string{"باروری", "تخمک", "fertil", "ovulat"} {
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

package home

// On the contract fixtures every persona's page builds without a single section failing, in
// every locale and on a few dates; the snapshot feeds both the sections and the message system.

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/home/store"
	msgstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

func TestMain(m *testing.M) { testdb.Main(m) }

var personas = []uint64{1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1018}

func TestEveryPersonaBuildsWithoutSectionFailures(t *testing.T) {
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewHandlers(Deps{
		Queries: store.New(db), Messages: msgstore.New(db), Pregnancy: pstore.New(db),
		AppURL: "https://api.ritme.app", Logger: logger,
	}, cycleservice.New(db, nil), clock.Real{})
	today := civildate.MustParse("2026-09-23")
	ctx := context.Background()

	for _, uid := range personas {
		for _, locale := range []string{"fa", "en", "ar"} {
			for _, date := range []string{"2026-09-23", "2026-09-30", "2026-08-01"} {
				user := &auth.User{ID: uid}
				hc, err := h.newContext(ctx, user, civildate.MustParse(date), today, locale, "fa")
				require.NoError(t, err)
				sections := h.page.Build(hc)
				require.NotEmpty(t, sections, "user %d %s %s", uid, locale, date)
				first, _ := sections[0].Get("key")
				assert.Equal(t, "header", first)
				_, err = sections[0].MarshalJSON()
				require.NoError(t, err)
			}
		}
	}
	assert.Empty(t, logs.String(), "no section failed to build")
}

package loss

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/loss/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var t0 = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

func body(t *testing.T, js string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(js))
	require.NoError(t, err)
	return v.(phpval.Map)
}

func status(err error) int {
	var fe *httpx.FailError
	if errors.As(err, &fe) {
		return fe.Status
	}
	return 0
}

func TestNoteBox_RoundTripAndBinding(t *testing.T) {
	b, err := NewNoteBox(nil, nil, false)
	require.NoError(t, err)
	sealed, err := b.Seal("فقط برای خودم", 7, 3)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(sealed, "v1:"+KeyID(devNoteKey[:])+":"), "versioned, names its key")
	assert.NotContains(t, sealed, "خودم")
	plain, err := b.Open(sealed, 7, 3)
	require.NoError(t, err)
	assert.Equal(t, "فقط برای خودم", plain)

	again, err := b.Seal("فقط برای خودم", 7, 3)
	require.NoError(t, err)
	assert.NotEqual(t, sealed, again, "random nonce per write")

	for _, c := range []struct{ user, loss uint64 }{{8, 3}, {7, 4}} {
		_, err := b.Open(sealed, c.user, c.loss)
		assert.ErrorIs(t, err, ErrNoteUnreadable, "bound to the owner and row")
	}
	newKey := []byte(strings.Repeat("k", 32))
	other, err := NewNoteBox(newKey, nil, false)
	require.NoError(t, err)
	_, err = other.Open(sealed, 7, 3)
	assert.ErrorIs(t, err, ErrNoteUnreadable, "another key")
	_, err = b.Open("plain text", 7, 3)
	assert.ErrorIs(t, err, ErrNoteUnreadable)
	_, err = b.Open("v1:"+KeyID(devNoteKey[:])+":!!", 7, 3)
	assert.ErrorIs(t, err, ErrNoteUnreadable)

	// Rotation: the new key seals, the previous one still opens older notes.
	rotated, err := NewNoteBox(newKey, [][]byte{devNoteKey[:]}, false)
	require.NoError(t, err)
	plain, err = rotated.Open(sealed, 7, 3)
	require.NoError(t, err)
	assert.Equal(t, "فقط برای خودم", plain)
	fresh, err := rotated.Seal("x", 7, 3)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(fresh, "v1:"+KeyID(newKey)+":"))
	_, err = b.Open(fresh, 7, 3)
	assert.ErrorIs(t, err, ErrNoteUnreadable, "the old box does not know the new key")

	_, err = NewNoteBox([]byte("short"), nil, false)
	assert.Error(t, err)
	disabled, err := NewNoteBox(nil, nil, true)
	require.NoError(t, err)
	assert.True(t, disabled.Disabled())
	assert.False(t, b.Disabled())
	var none *NoteBox
	assert.True(t, none.Disabled())
}

func TestValidateRecord(t *testing.T) {
	in, err := validateRecord(body(t, `{}`), "en", t0)
	require.NoError(t, err)
	assert.Equal(t, RecordInput{Type: TypeUnspecified}, in, "everything optional: she answers only what she wants to")
	in, err = validateRecord(body(t, `{"type":null,"occurred_on":null}`), "en", t0)
	require.NoError(t, err)
	assert.Equal(t, RecordInput{Type: TypeUnspecified, HasType: true, HasDate: true}, in, "explicit null clears on a correction")

	in, err = validateRecord(body(t, `{"type":"ectopic","occurred_on":"2026-09-10","notify_companion":true,"extra":"x"}`), "en", t0)
	require.NoError(t, err)
	assert.Equal(t, TypeEctopic, in.Type)
	assert.Equal(t, civildate.NullDate{Date: civildate.MustParse("2026-09-10"), Valid: true}, in.OccurredOn)
	assert.True(t, in.NotifyCompanion)

	for _, js := range []string{`{"type":"other"}`, `{"occurred_on":"2026-09-24"}`, `{"occurred_on":"2025-09-01"}`,
		`{"occurred_on":"20-09-2026"}`, `{"notify_companion":"maybe"}`} {
		_, err := validateRecord(body(t, js), "fa", t0)
		assert.Equal(t, 422, status(err), js)
	}
}

func TestValidateFollowup(t *testing.T) {
	in, err := validateFollowup(body(t, `{}`), "en", t0)
	require.NoError(t, err)
	assert.Equal(t, FollowupInput{}, in)

	in, err = validateFollowup(body(t, `{"bleeding_stopped":false,"beta_next_on":null,"visit_at":"2026-10-07 11:30"}`), "en", t0)
	require.NoError(t, err)
	require.NotNil(t, in.BleedingStopped)
	assert.False(t, *in.BleedingStopped)
	require.NotNil(t, in.BetaNextOn)
	assert.False(t, in.BetaNextOn.Valid, "null clears")
	require.NotNil(t, in.VisitAt)
	assert.Equal(t, time.Date(2026, 10, 7, 11, 30, 0, 0, civildate.Tehran), in.VisitAt.Time)
	assert.Nil(t, in.BetaNegative)

	for _, js := range []string{`{"beta_next_on":"2026-09-22"}`, `{"visit_at":"2026-09-23 09:00"}`, `{"visit_at":"tomorrow"}`,
		`{"beta_negative":"x"}`} {
		_, err := validateFollowup(body(t, js), "en", t0)
		assert.Equal(t, 422, status(err), js)
	}
}

func TestValidateMoodNoteNextStep(t *testing.T) {
	m, d, err := validateMood(body(t, `{"mood":"a_bit_better"}`), "en", t0)
	require.NoError(t, err)
	assert.Equal(t, "a_bit_better", m)
	assert.Equal(t, civildate.MustParse("2026-09-23"), d)
	for _, js := range []string{`{}`, `{"mood":"happy"}`, `{"mood":"sad","date":"2026-09-24"}`, `{"mood":"sad","date":"2026-09-01"}`} {
		_, _, err := validateMood(body(t, js), "en", t0)
		assert.Equal(t, 422, status(err), js)
	}

	n, err := validateNote(body(t, `{"note":"  hi  "}`), "en", t0)
	require.NoError(t, err)
	assert.Equal(t, "hi", n)
	n, err = validateNote(body(t, `{"note":null}`), "en", t0)
	require.NoError(t, err)
	assert.Empty(t, n)
	_, err = validateNote(body(t, `{}`), "en", t0)
	assert.Equal(t, 422, status(err), "note must be present")
	_, err = validateNote(body(t, `{"note":"`+strings.Repeat("a", MaxNoteLen+1)+`"}`), "en", t0)
	assert.Equal(t, 422, status(err))

	c, err := validateNextStep(body(t, `{"choice":"nothing"}`), "en", t0)
	require.NoError(t, err)
	assert.Equal(t, NextNothing, c)
	_, err = validateNextStep(body(t, `{"choice":"pregnancy"}`), "en", t0)
	assert.Equal(t, 422, status(err))
}

func TestBleedingOf(t *testing.T) {
	ns := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
	assert.Empty(t, bleedingOf(nil))
	assert.Equal(t, "light", bleedingOf([]taxonomy.Entry{{Category: "bleeding", Param: "spotting"}, {Category: "bleeding", Param: "flow", Code: ns("light")}}))
	assert.Equal(t, "spotting", bleedingOf([]taxonomy.Entry{{Category: "bleeding", Param: "spotting"}}))
	assert.Empty(t, bleedingOf([]taxonomy.Entry{{Category: "pain", Param: "flow", Code: ns("heavy")}}))
}

func TestHelpers(t *testing.T) {
	assert.Equal(t, "b", pickText(map[string]string{"en": "b"}, "fa", "en"))
	assert.Equal(t, "a", pickText(map[string]string{"fa": "a", "en": "b"}, "fa", "en"))
	assert.Equal(t, "x", pickText(map[string]string{"zz": "x"}, "fa", "en"))
	assert.Empty(t, pickText(map[string]string{}, "fa", "en"))
	assert.Equal(t, []uint64{1, 2, 3}, mergeIDs([]uint64{1, 2}, []uint64{2, 3}))
	assert.False(t, encodeIDs(nil).Valid)
	assert.Equal(t, []uint64{4, 5}, decodeIDs(encodeIDs([]uint64{4, 5})))
}

func TestStateJSON(t *testing.T) {
	raw, err := jsonx.Marshal(StateJSON(State{}), 0)
	require.NoError(t, err)
	assert.JSONEq(t, `{"loss":null,"losses_count":0,"recurrent_hint":false,"followup":null,"mood":null,"note":null}`, string(raw))

	l := store.PregnancyLoss{ID: 9, LossType: TypeChemical, CreatedAt: sql.NullTime{Time: t0, Valid: true},
		ContentStoppedAt: sql.NullTime{Time: t0, Valid: true}}
	st := State{Loss: &l, Count: 2, Today: civildate.InTehran(t0), Now: t0,
		Moods: []store.ListLossMoodsSinceRow{{LogDate: civildate.InTehran(t0), Mood: "sad"}}}
	raw, err = jsonx.Marshal(StateJSON(st), 0)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.Equal(t, true, m["recurrent_hint"])
	assert.Equal(t, "2026-10-07", m["followup"].(map[string]any)["visit"].(map[string]any)["suggested_on"], "from the record day without a date")
	assert.Equal(t, "sad", m["mood"].(map[string]any)["today"])
	assert.Equal(t, "2026-09-23T10:00:00+03:30", m["loss"].(map[string]any)["created_at"])
	assert.NotContains(t, string(raw), "private_note")
}

// Every language file has the same keys as English (the fallback).
func TestLangFiles(t *testing.T) {
	keys := func(code string) []string {
		b, err := fs.ReadFile(langFS, "lang/"+code+"/loss.json")
		require.NoError(t, err)
		var m map[string]map[string]any
		require.NoError(t, json.Unmarshal(b, &m))
		var out []string
		for g, kv := range m {
			for k := range kv {
				out = append(out, g+"."+k)
			}
		}
		return out
	}
	en := keys("en")
	entries, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	for _, e := range entries {
		assert.ElementsMatch(t, en, keys(e.Name()), e.Name())
	}
	assert.Equal(t, "Follow-up saved", T("messages.followup_saved", "en"))
	assert.NotEmpty(t, attributes("fa"))
}

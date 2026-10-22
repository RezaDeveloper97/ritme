package consent

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/consent/store"
)

// Every catalog consent has the text of its version in force in every language file, and every language file has
// the same messages.
func TestCatalogTextsComplete(t *testing.T) {
	dirs, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	require.NotEmpty(t, dirs)
	b, err := bundles()
	require.NoError(t, err)
	for _, d := range dirs {
		f := b[d.Name()]
		for _, def := range Catalog() {
			txt, ok := f.Texts[def.Code][strconv.Itoa(def.Version)]
			require.True(t, ok, "%s: %s v%d", d.Name(), def.Code, def.Version)
			assert.NotEmpty(t, txt.Title)
			assert.NotEmpty(t, txt.Body)
			assert.NotEmpty(t, txt.Points)
		}
		assert.Len(t, f.Messages, len(b[fallbackLocale].Messages), d.Name())
	}
	assert.Equal(t, "Consent saved", T("saved", "xx"), "unknown language → English")
}

func TestCatalogCodesUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range Catalog() {
		assert.False(t, seen[d.Code], d.Code)
		seen[d.Code] = true
		assert.Positive(t, d.Version)
	}
	assert.Equal(t, 0, CurrentVersion("nope"))
}

type memStore struct {
	rows   map[string]store.GetConsentRow
	events []store.InsertConsentEventParams
	err    error
}

func (m *memStore) InsertConsentEvent(_ context.Context, a store.InsertConsentEventParams) error {
	m.events = append(m.events, a)
	return m.err
}

func (m *memStore) ListConsentEvents(context.Context, store.ListConsentEventsParams) ([]store.ListConsentEventsRow, error) {
	out := make([]store.ListConsentEventsRow, 0, len(m.events))
	for i := len(m.events) - 1; i >= 0; i-- {
		e := m.events[i]
		out = append(out, store.ListConsentEventsRow{ID: uint64(i + 1), Consent: e.Consent, Version: e.Version, //nolint:gosec // test index
			Action: e.Action, Scope: e.Scope, Source: e.Source, CreatedAt: e.Now})
	}
	return out, m.err
}

func (m *memStore) GetConsent(_ context.Context, a store.GetConsentParams) (store.GetConsentRow, error) {
	if m.err != nil {
		return store.GetConsentRow{}, m.err
	}
	r, ok := m.rows[a.Consent]
	if !ok {
		return r, sql.ErrNoRows
	}
	return r, nil
}

func (m *memStore) ListConsents(context.Context, uint64) ([]store.ListConsentsRow, error) {
	var out []store.ListConsentsRow
	for _, r := range m.rows {
		out = append(out, store.ListConsentsRow(r))
	}
	return out, m.err
}

func (m *memStore) GrantConsent(_ context.Context, a store.GrantConsentParams) error {
	m.rows[a.Consent] = store.GetConsentRow{Consent: a.Consent, Granted: true, Version: a.Version, GrantedAt: a.Now}
	return m.err
}

func (m *memStore) RevokeConsent(_ context.Context, a store.RevokeConsentParams) error {
	r := m.rows[a.Consent]
	r.Consent, r.Granted, r.RevokedAt = a.Consent, false, a.Now
	m.rows[a.Consent] = r
	return m.err
}

func TestService_RequireAcceptWithdraw(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	m := &memStore{rows: map[string]store.GetConsentRow{}}
	s := NewServiceWith(m)

	var req *RequiredError
	require.ErrorAs(t, s.Require(ctx, 1, AIAssistant), &req)
	assert.Equal(t, RequiredError{Code: AIAssistant, Version: 1, Reason: ReasonMissing}, *req)

	require.ErrorIs(t, s.Accept(ctx, 1, AIAssistant, 2, now), ErrStaleVersion)
	require.ErrorIs(t, s.Accept(ctx, 1, "nope", 1, now), ErrUnknown)
	require.NoError(t, s.Accept(ctx, 1, AIAssistant, 1, now))
	require.NoError(t, s.Accept(ctx, 1, AIAssistant, 1, now), "same version again: no-op")
	require.NoError(t, s.Require(ctx, 1, AIAssistant))
	require.Len(t, m.events, 1, "a repeated accept writes no second event")

	// a B-N1-12 grant without a version (or an older one) is outdated
	m.rows[AILabAnalysis] = store.GetConsentRow{Consent: AILabAnalysis, Granted: true}
	require.ErrorAs(t, s.Require(ctx, 1, AILabAnalysis), &req)
	assert.Equal(t, ReasonOutdated, req.Reason)

	require.NoError(t, s.WithdrawFrom(ctx, 1, AIAssistant, SourceSettings, now))
	require.Len(t, m.events, 2)
	assert.Equal(t, ActionRevoke, m.events[1].Action)
	assert.EqualValues(t, 1, m.events[1].Version, "the version withdrawn")
	require.NoError(t, s.WithdrawFrom(ctx, 1, AIVoiceLog, SourceSettings, now))
	require.Len(t, m.events, 2, "«no» to what was never granted is not an event")
	require.Error(t, s.AcceptFrom(ctx, 1, AIAssistant, 1, "elsewhere", now), "unknown source")
	require.ErrorAs(t, s.Require(ctx, 1, AIAssistant), &req)
	assert.Equal(t, ReasonMissing, req.Reason)

	list, err := s.List(ctx, 1)
	require.NoError(t, err)
	require.Len(t, list, len(Catalog())-len(Mandatory()), "the signup consents are listed by Pending")
	assert.Equal(t, AILabAnalysis, list[0].Code, "display order")

	m.err = errors.New("db down")
	err = s.Require(ctx, 1, AIAssistant)
	require.Error(t, err)
	assert.NotErrorAs(t, err, &req, "a store error is not a consent answer (fail closed upstream)")
	require.ErrorIs(t, s.Require(ctx, 1, "nope"), ErrUnknown)
}

func TestMandatoryAndPending(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	codes := []string{}
	for _, d := range Mandatory() {
		codes = append(codes, d.Code)
	}
	assert.Equal(t, []string{Terms, HealthDataProcessing}, codes)

	m := &memStore{rows: map[string]store.GetConsentRow{}}
	s := NewServiceWith(m)
	p, err := s.Pending(ctx, 1)
	require.NoError(t, err)
	require.Len(t, p, 2)
	assert.Equal(t, ReasonMissing, p[0].Reason())

	sg := NewSignup(s)
	require.Equal(t, []Requirement{{Terms, 1}, {HealthDataProcessing, 1}}, sg.Required())
	require.ErrorIs(t, sg.Record(ctx, 1, map[string]int{Terms: 1}, now), ErrStaleVersion, "every code needs its version")
	m.rows, m.events = map[string]store.GetConsentRow{}, nil
	require.NoError(t, sg.Record(ctx, 1, map[string]int{Terms: 1, HealthDataProcessing: 1}, now))
	require.Len(t, m.events, 2)
	for _, e := range m.events {
		assert.Equal(t, SourceSignup, e.Source)
		assert.Equal(t, ActionGrant, e.Action)
	}
	p, err = s.Pending(ctx, 1)
	require.NoError(t, err)
	assert.Empty(t, p)
	assert.NotEqual(t, "signup_required", sg.RequiredMessage("fa"))

	// an outdated acceptance is pending again
	m.rows[Terms] = store.GetConsentRow{Consent: Terms, Granted: true, Version: sql.NullInt16{Int16: 0, Valid: true}}
	p, err = s.Pending(ctx, 1)
	require.NoError(t, err)
	require.Len(t, p, 1)
	assert.Equal(t, ReasonOutdated, p[0].Reason())
}

func TestEvidenceTexts(t *testing.T) {
	b, err := bundles()
	require.NoError(t, err)
	for loc, f := range b {
		for _, d := range evidence {
			txt, ok := f.Texts[d.Code][strconv.Itoa(d.Version)]
			require.True(t, ok, "%s: %s", loc, d.Code)
			assert.NotEmpty(t, txt.Title)
		}
	}
	assert.True(t, validSource(SourceBackfill))
	assert.False(t, validSource("x"))
}

// verify-otp gets the signup consents through auth.SignupConsents (auth cannot import this package).
var _ auth.SignupConsents = Signup{}

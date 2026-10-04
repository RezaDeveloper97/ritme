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
	rows map[string]store.GetConsentRow
	err  error
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
	require.NoError(t, s.Require(ctx, 1, AIAssistant))

	// a B-N1-12 grant without a version (or an older one) is outdated
	m.rows[AILabAnalysis] = store.GetConsentRow{Consent: AILabAnalysis, Granted: true}
	require.ErrorAs(t, s.Require(ctx, 1, AILabAnalysis), &req)
	assert.Equal(t, ReasonOutdated, req.Reason)

	require.NoError(t, s.Withdraw(ctx, 1, AIAssistant, now))
	require.ErrorAs(t, s.Require(ctx, 1, AIAssistant), &req)
	assert.Equal(t, ReasonMissing, req.Reason)

	list, err := s.List(ctx, 1)
	require.NoError(t, err)
	require.Len(t, list, len(Catalog()))
	assert.Equal(t, AILabAnalysis, list[0].Code, "display order")

	m.err = errors.New("db down")
	err = s.Require(ctx, 1, AIAssistant)
	require.Error(t, err)
	assert.NotErrorAs(t, err, &req, "a store error is not a consent answer (fail closed upstream)")
	require.ErrorIs(t, s.Require(ctx, 1, "nope"), ErrUnknown)
}

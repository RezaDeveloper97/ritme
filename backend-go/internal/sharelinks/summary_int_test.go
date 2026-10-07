package sharelinks_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/sharelinks"
	"github.com/ritme/backend-go/internal/sharelinks/store"
)

func summaryReq() sharelinks.SummaryRequest {
	return sharelinks.SummaryRequest{Report: healthrecord.ReportRequest{Range: healthrecord.ReportRange6M, Sections: all}}
}

// CB-REC-03: the 24h summary at the service level — record built for the share audience, code and pepper, expiry,
// purge of both ciphertexts, access log rows going with their link, the kill switch.
func TestSummary_ShareAudienceExpiryPurgeAndPepper(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid, _ := e.user(t, "09120000301", false)
	e.seed(t, uid)
	svc := sharelinks.NewService(store.New(e.db), healthrecord.NewService(e.db, nil), quiet).WithDB(e.db).
		WithCodes([]byte("pepper-0123456789-0123456789-0123456789"), false)

	created, err := svc.CreateSummary(ctx, uid, summaryReq(), fixed, "en", "fa")
	require.NoError(t, err)
	assert.Equal(t, fixed.Add(24*time.Hour), created.Summary.ExpiresAt)
	assert.Equal(t, 0, created.Documents)

	opened, err := svc.OpenCode(ctx, sharelinks.FormatCode(created.Code), fixed.Add(time.Hour),
		sharelinks.ViewerOf("", "curl/8"))
	require.NoError(t, err)
	raw, err := opened.Report.MarshalJSON()
	require.NoError(t, err)
	assertNoIDs(t, string(raw))
	assert.Contains(t, string(raw), `"audience":"share"`)
	assert.Contains(t, string(raw), `"documents":[]`, "no document unless explicitly picked")

	// Another pepper (or a rotated one) opens nothing; neither does the bloom report path for the code.
	other := sharelinks.NewService(store.New(e.db), healthrecord.NewService(e.db, nil), quiet).WithDB(e.db).
		WithCodes([]byte("another-pepper-0123456789-0123456789"), false)
	_, err = other.OpenCode(ctx, created.Code, fixed.Add(time.Hour), sharelinks.Viewer{})
	assert.ErrorIs(t, err, sharelinks.ErrNotFound)

	// Kill switch: production without a pepper.
	off := sharelinks.NewService(store.New(e.db), healthrecord.NewService(e.db, nil), quiet).WithDB(e.db).WithCodes(nil, true)
	_, err = off.CreateSummary(ctx, uid, summaryReq(), fixed, "en", "fa")
	assert.ErrorIs(t, err, sharelinks.ErrCodesDisabled)
	_, err = off.OpenCode(ctx, created.Code, fixed, sharelinks.Viewer{})
	assert.ErrorIs(t, err, sharelinks.ErrCodesDisabled)

	// Expiry: 404 by code and by token; the purge wipes both ciphertexts; the access log goes with the row.
	after := fixed.Add(24*time.Hour + time.Second)
	_, err = svc.OpenCode(ctx, created.Code, after, sharelinks.Viewer{})
	assert.ErrorIs(t, err, sharelinks.ErrNotFound)
	_, err = svc.Open(ctx, created.Token, after)
	assert.ErrorIs(t, err, sharelinks.ErrNotFound, "a summary never answers 410")
	require.NoError(t, svc.Purge(ctx, after))
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM health_share_links WHERE id = ? AND payload IS NULL AND code_payload IS NULL`,
		created.Summary.ID).Scan(&n))
	assert.Equal(t, 1, n)
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM health_share_link_views WHERE share_link_id = ?`, created.Summary.ID).Scan(&n))
	assert.Equal(t, 1, n)
	require.NoError(t, svc.Purge(ctx, after.Add(sharelinks.RetainAfterExpiry+time.Hour)))
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM health_share_link_views WHERE share_link_id = ?`, created.Summary.ID).Scan(&n))
	assert.Equal(t, 0, n, "the access log is deleted with its link")
}

// Report links (bloom) keep their 410 and now also write the access log.
func TestSummary_ReportLinksLogAccess(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	uid, _ := e.user(t, "09120000302", false)
	created, err := e.svc.Create(ctx, uid, healthrecord.ReportRequest{Range: healthrecord.ReportRange3M, Sections: []string{"basics"}},
		fixed, "en", "fa")
	require.NoError(t, err)
	_, err = e.svc.OpenAs(ctx, created.Token, fixed, sharelinks.ViewerOf("", "Mozilla/5.0 (Windows NT 10.0) Chrome/126.0 Safari/537.36"))
	require.NoError(t, err)
	entries, err := e.svc.Access(ctx, uid, 0, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, sharelinks.KindReport, entries[0].Kind)
	assert.Equal(t, sharelinks.ViaLink, entries[0].Via)
	assert.Equal(t, sharelinks.DeviceDesktop, entries[0].Device)
	assert.Equal(t, sharelinks.BrowserChrome, entries[0].Browser)
	_, err = e.svc.Open(ctx, created.Token, fixed.Add(sharelinks.LinkTTL))
	assert.ErrorIs(t, err, sharelinks.ErrExpired)

	other, _ := e.user(t, "09120000303", false)
	_, err = e.svc.Access(ctx, other, created.Link.ID, 10)
	assert.ErrorIs(t, err, sharelinks.ErrNotFound)
	_, err = e.svc.RevokeSummary(ctx, uid, created.Link.ID, fixed)
	assert.ErrorIs(t, err, sharelinks.ErrNotFound, "a report link is not a summary")
}

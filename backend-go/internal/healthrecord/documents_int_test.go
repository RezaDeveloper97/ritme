package healthrecord_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// fakeLabs is the labs source of the timeline: ready lab rows per user (the real RecordLabs is user-scoped and tested
// in internal/labs).
type fakeLabs map[uint64][]*jsonx.OrderedMap

func (f fakeLabs) RecordLabs(_ context.Context, userID uint64, _, _ string, limit int) ([]*jsonx.OrderedMap, error) {
	rows := f[userID]
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func labRow(id uint64, date string) *jsonx.OrderedMap {
	return jsonx.Obj("id", id, "category", "blood", "title", "CBC", "date", date, "marker_count", 14,
		"attention_count", 1, "all_normal", false, "attention", []any{})
}

type docEnv struct {
	*env
	files *files.Service
	docs  *healthrecord.Documents
	labs  fakeLabs
}

var docPDF = []byte("%PDF-1.4\n1 0 obj << /Sono 1 >> endobj\n%%EOF")

func setupDocs(t *testing.T) *docEnv {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))

	vault, err := files.NewVault(t.TempDir(), files.DevKey("ritme-file-dev-key"), nil, false)
	require.NoError(t, err)
	fs := files.NewService(files.Options{DB: db, Vault: vault, BaseURL: "https://api.test"})
	labs := fakeLabs{}
	docs := healthrecord.NewDocuments(db, fs, labs)
	h := healthrecord.NewDocumentHandlers(docs, clock.Fixed(fixed))
	rec := healthrecord.NewHandlers(healthrecord.NewService(db, nil), clock.Fixed(fixed))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	p := "/api/v1/health-record"
	app.Put(p+"/basics", locale, guard, rec.UpdateBasics)
	app.Get(p+"/categories", locale, guard, h.Categories)
	app.Get(p+"/timeline", locale, guard, h.Timeline)
	app.Get(p+"/extras", locale, guard, h.ShowExtras)
	app.Put(p+"/extras", locale, guard, h.UpdateExtras)
	app.Post(p+"/documents", locale, guard, h.StoreDocument)
	app.Get(p+"/documents/:id", locale, guard, h.ShowDocument)
	app.Put(p+"/documents/:id", locale, guard, h.UpdateDocument)
	app.Delete(p+"/documents/:id", locale, guard, h.DestroyDocument)
	return &docEnv{
		env:   &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)},
		files: fs, docs: docs, labs: labs,
	}
}

func (e *docEnv) upload(t *testing.T, uid uint64, purpose string) uint64 {
	t.Helper()
	f, err := e.files.Put(context.Background(), uid, purpose, docPDF, fixed)
	require.NoError(t, err)
	return f.ID
}

func id(t *testing.T, r response) string {
	t.Helper()
	v, ok := r.data()["id"].(float64)
	require.True(t, ok, r.raw)
	return strconv.FormatUint(uint64(v), 10)
}

func errorKeys(r response) []string {
	errs, _ := r.body["errors"].(map[string]any)
	var keys []string
	for k := range errs {
		keys = append(keys, k)
	}
	return keys
}

const docs = "/api/v1/health-record/documents"

func TestRecordDocuments_Unauthenticated(t *testing.T) {
	e := setupDocs(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/health-record/categories"}, {http.MethodGet, "/api/v1/health-record/timeline"},
		{http.MethodGet, "/api/v1/health-record/extras"}, {http.MethodPut, "/api/v1/health-record/extras"},
		{http.MethodPost, docs}, {http.MethodGet, docs + "/1"}, {http.MethodPut, docs + "/1"},
		{http.MethodDelete, docs + "/1"},
	} {
		r := e.do(t, c.method, c.path, "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.Equal(t, "unauthenticated", r.body["error_code"])
	}
}

func TestRecordDocuments_CRUD(t *testing.T) {
	e := setupDocs(t)
	uid, tok := e.user(t, "09120000001")
	f1, f2 := e.upload(t, uid, files.PurposeRecordDocument), e.upload(t, uid, files.PurposeRecordDocument)

	r := e.do(t, http.MethodPost, docs, tok, map[string]any{"kind": "xray", "date": "2030-01-01", "file_ids": "x"})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.ElementsMatch(t, []string{"kind", "date", "file_ids"}, errorKeys(r))

	r = e.do(t, http.MethodPost, docs, tok, map[string]any{"kind": "imaging", "date": "2026-09-24", "ended_on": "2026-09-01"})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []string{"ended_on"}, errorKeys(r))

	r = e.do(t, http.MethodPost, docs, tok, map[string]any{"kind": "imaging", "file_ids": []uint64{f1, f1}})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []string{"file_ids.1"}, errorKeys(r))

	r = e.do(t, http.MethodPost, docs, tok, map[string]any{
		"kind": "imaging", "title": "Pregnancy ultrasound", "date": "2026-09-24", "centre": "Imaging centre",
		"doctor": "Dr. A", "note": "8w6d", "file_ids": []uint64{f1, f2},
	})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "Document added", r.body["message"])
	d := r.data()
	assert.Equal(t, "imaging", d["kind"])
	assert.Equal(t, "2026-09-24", d["date"])
	assert.Equal(t, "manual", d["review_state"])
	assert.Nil(t, d["extracted"])
	assert.Equal(t, []any{}, d["where_used"])
	fl, _ := d["files"].([]any)
	require.Len(t, fl, 2)
	f0, _ := fl[0].(map[string]any)
	assert.Equal(t, float64(f1), f0["id"])
	assert.Regexp(t, `^https://api.test/api/v1/files/\d+/download\?expires=\d+&signature=`, f0["url"])
	docID := id(t, r)

	// The same file cannot go on a second document.
	r = e.do(t, http.MethodPost, docs, tok, map[string]any{"kind": "visit", "file_ids": []uint64{f2}})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []string{"file_ids.0"}, errorKeys(r))
	assert.Contains(t, r.raw, "already attached")

	// Partial PUT: absent keys keep, file_ids replaces and the dropped file is deleted.
	r = e.do(t, http.MethodPut, docs+"/"+docID, tok, map[string]any{"centre": nil, "file_ids": []uint64{f2}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Nil(t, d["centre"])
	assert.Equal(t, "Dr. A", d["doctor"])
	assert.Equal(t, "Pregnancy ultrasound", d["title"])
	fl, _ = d["files"].([]any)
	assert.Len(t, fl, 1)
	_, err := e.files.Get(context.Background(), uid, f1)
	require.ErrorIs(t, err, files.ErrNotFound)

	// Review: confirm moves needs_review → confirmed (and nothing else).
	e.exec(t, `UPDATE record_documents SET review_state = 'needs_review', extracted = '{"ga_weeks":8}' WHERE id = ?`, docID)
	r = e.do(t, http.MethodGet, docs+"/"+docID, tok, nil)
	require.Equal(t, http.StatusOK, r.status)
	assert.Equal(t, map[string]any{"ga_weeks": float64(8)}, r.data()["extracted"])
	r = e.do(t, http.MethodPut, docs+"/"+docID, tok, map[string]any{"confirm": true})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "confirmed", r.data()["review_state"])

	// «Where used»: links written by the Go API of the owning task.
	n, err := strconv.ParseUint(docID, 10, 64)
	require.NoError(t, err)
	require.NoError(t, e.docs.AddLink(context.Background(), uid, n, healthrecord.LinkClaim, 42, healthrecord.LinkWaiting, fixed))
	require.NoError(t, e.docs.AddLink(context.Background(), uid, n, healthrecord.LinkClaim, 42, healthrecord.LinkAttached, fixed))
	require.ErrorIs(t, e.docs.AddLink(context.Background(), uid, n, "insurer", 1, healthrecord.LinkAttached, fixed),
		healthrecord.ErrLinkTarget)
	require.ErrorIs(t, e.docs.AddLink(context.Background(), uid, n, healthrecord.LinkPregnancy, 999, healthrecord.LinkApplied, fixed),
		healthrecord.ErrLinkTarget)
	r = e.do(t, http.MethodGet, docs+"/"+docID, tok, nil)
	used, _ := r.data()["where_used"].([]any)
	require.Len(t, used, 1)
	u0, _ := used[0].(map[string]any)
	assert.Equal(t, "claim", u0["type"])
	assert.Equal(t, "attached", u0["state"])

	r = e.do(t, http.MethodDelete, docs+"/"+docID, tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	_, err = e.files.Get(context.Background(), uid, f2)
	require.ErrorIs(t, err, files.ErrNotFound, "the document's files go with it")
	r = e.do(t, http.MethodGet, docs+"/"+docID, tok, nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "record_document_not_found", r.body["error_code"])
}

func TestRecordDocuments_UserIsolationAndFileOwnership(t *testing.T) {
	e := setupDocs(t)
	a, tokA := e.user(t, "09120000001")
	b, tokB := e.user(t, "09120000002")
	fa := e.upload(t, a, files.PurposeRecordDocument)
	claim := e.upload(t, b, files.PurposeClaimDocument)
	fb := e.upload(t, b, files.PurposeRecordDocument)

	r := e.do(t, http.MethodPost, docs, tokA, map[string]any{"kind": "visit", "date": "2026-09-20", "file_ids": []uint64{fa}})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	docA := id(t, r)
	require.NoError(t, e.docs.SaveExtras(context.Background(), a, healthrecord.ExtrasInput{
		SetSurgeries: true, Surgeries: []healthrecord.Surgery{{Title: "Cesarean"}},
	}, fixed))

	// B cannot read, change or delete A's document: the uniform 404.
	for _, c := range []struct {
		method string
		body   any
	}{{http.MethodGet, nil}, {http.MethodPut, map[string]any{"title": "x"}}, {http.MethodDelete, nil}} {
		r = e.do(t, c.method, docs+"/"+docA, tokB, c.body)
		assert.Equal(t, http.StatusNotFound, r.status, c.method)
		assert.Equal(t, "record_document_not_found", r.body["error_code"])
	}
	// B cannot attach A's file, nor a file of another purpose (her own claim document).
	for _, f := range []uint64{fa, claim} {
		r = e.do(t, http.MethodPost, docs, tokB, map[string]any{"kind": "other", "file_ids": []uint64{f}})
		require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
		assert.Equal(t, []string{"file_ids.0"}, errorKeys(r))
		assert.Contains(t, r.raw, "This file was not found.")
	}
	// Nor move her own file onto A's document, nor A's file onto her own.
	r = e.do(t, http.MethodPost, docs, tokB, map[string]any{"kind": "other", "file_ids": []uint64{fb}})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	docB := id(t, r)
	r = e.do(t, http.MethodPut, docs+"/"+docB, tokB, map[string]any{"file_ids": []uint64{fb, fa}})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []string{"file_ids.1"}, errorKeys(r))
	nA, _ := strconv.ParseUint(docA, 10, 64)
	require.ErrorIs(t, e.docs.AddLink(context.Background(), b, nA, healthrecord.LinkClaim, 1, healthrecord.LinkAttached, fixed),
		healthrecord.ErrDocumentNotFound)

	// B's timeline, counts and extras show none of A's rows.
	r = e.do(t, http.MethodGet, "/api/v1/health-record/timeline", tokB, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	months, _ := r.data()["months"].([]any)
	require.Len(t, months, 1)
	m0, _ := months[0].(map[string]any)
	items, _ := m0["items"].([]any)
	require.Len(t, items, 1)
	it, _ := items[0].(map[string]any)
	assert.Equal(t, docB, strconv.FormatFloat(it["id"].(float64), 'f', -1, 64))
	r = e.do(t, http.MethodGet, "/api/v1/health-record/categories", tokB, nil)
	assert.Equal(t, float64(1), r.data()["documents_count"])
	r = e.do(t, http.MethodGet, "/api/v1/health-record/extras", tokB, nil)
	assert.Nil(t, r.data()["surgeries"])

	// A's file is untouched by B's attempts.
	_, err := e.files.Get(context.Background(), a, fa)
	require.NoError(t, err)
}

func TestRecordDocuments_TimelineAndCategories(t *testing.T) {
	e := setupDocs(t)
	uid, tok := e.user(t, "09120000001")
	e.labs[uid] = []*jsonx.OrderedMap{labRow(7, "2026-09-09"), labRow(6, "2025-03-01")}
	for _, d := range []struct{ kind, date string }{
		{"imaging", "2026-09-24"}, {"prescription", "2026-09-23"}, {"visit", "2026-08-31"}, {"hospital", "2025-03-04"},
		{"visit", "2026-09-24"},
	} {
		r := e.do(t, http.MethodPost, docs, tok, map[string]any{"kind": d.kind, "date": d.date})
		require.Equal(t, http.StatusCreated, r.status, r.raw)
	}
	r := e.do(t, http.MethodPost, docs, tok, map[string]any{"kind": "other"}) // no date: filed under today
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	e.exec(t, `UPDATE record_documents SET review_state = 'needs_review' WHERE kind = 'prescription'`)
	// Private loss data and care appointments live elsewhere; the timeline never reads them.
	e.exec(t, `INSERT INTO reminders (user_id, type, title, recurrence, is_active, meta, created_at, updated_at)
		VALUES (?, 'appointment', 'Follow-up', 'none', 1, '{"private":true}', '2026-09-20 09:00:00', '2026-09-20 09:00:00')`, uid)

	r = e.do(t, http.MethodGet, "/api/v1/health-record/timeline", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	months, _ := r.data()["months"].([]any)
	var keys []string
	var kinds []string
	for _, m := range months {
		mm, _ := m.(map[string]any)
		keys = append(keys, mm["key"].(string))
		for _, it := range mm["items"].([]any) {
			im, _ := it.(map[string]any)
			assert.Contains(t, []any{"document", "lab"}, im["type"])
			kinds = append(kinds, im["kind"].(string))
		}
	}
	// 2026-10-06 = 14 Mehr 1405; 2026-09-24 = 2 Mehr; 2026-09-23 = 1 Mehr; 2026-09-09 / 08-31 = Shahrivar; 2025-03 =
	// Esfand 1403.
	assert.Equal(t, []string{"1405-07", "1405-06", "1403-12"}, keys)
	assert.Equal(t, []string{"other", "visit", "imaging", "prescription", "lab", "visit", "hospital", "lab"}, kinds)
	m0, _ := months[0].(map[string]any)
	assert.Equal(t, "2026-09-23", m0["start"])
	assert.Nil(t, r.data()["next_before"])

	// Filters and pages that never split a day.
	r = e.do(t, http.MethodGet, "/api/v1/health-record/timeline?kind=lab", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Len(t, r.data()["months"], 2)
	r = e.do(t, http.MethodGet, "/api/v1/health-record/timeline?limit=2", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-09-24", r.data()["next_before"], "the 2nd row's day is complete on this page")
	m0, _ = r.data()["months"].([]any)[0].(map[string]any)
	assert.Len(t, m0["items"], 3)
	r = e.do(t, http.MethodGet, "/api/v1/health-record/timeline?limit=2&before=2026-09-24", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	m0, _ = r.data()["months"].([]any)[0].(map[string]any)
	first, _ := m0["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "prescription", first["kind"])
	r = e.do(t, http.MethodGet, "/api/v1/health-record/timeline?kind=fax&limit=500", tok, nil)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.ElementsMatch(t, []string{"kind", "limit"}, errorKeys(r))

	r = e.do(t, http.MethodGet, "/api/v1/health-record/categories", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	cats := map[string]float64{}
	for _, c := range r.data()["categories"].([]any) {
		cm, _ := c.(map[string]any)
		cats[cm["key"].(string)] = cm["count"].(float64)
	}
	assert.Equal(t, map[string]float64{"labs": 2, "imaging": 1, "visit": 2, "prescription": 1, "hospital": 1, "other": 1,
		"surgeries": 0, "family_history": 0}, cats)
	assert.Equal(t, float64(6), r.data()["documents_count"])
	assert.Equal(t, float64(8), r.data()["all_count"])
	assert.Equal(t, float64(1), r.data()["needs_review_count"])
}

func TestRecordExtras(t *testing.T) {
	e := setupDocs(t)
	_, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodGet, "/api/v1/health-record/extras", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, map[string]any{"allergies": nil, "allergies_on_emergency_card": true, "surgeries": nil,
		"family_history": nil}, r.data())

	r = e.do(t, http.MethodPut, "/api/v1/health-record/extras", tok, map[string]any{
		"allergies_on_emergency_card": "maybe",
		"surgeries":                   []any{map[string]any{"date": "2030-01-01"}},
		"family_history":              []any{map[string]any{"condition": "Diabetes", "relative": "neighbour"}},
	})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.ElementsMatch(t, []string{"allergies_on_emergency_card", "surgeries.0.title", "surgeries.0.date",
		"family_history.0.relative"}, errorKeys(r))

	r = e.do(t, http.MethodPut, "/api/v1/health-record/extras", tok, map[string]any{
		"allergies_on_emergency_card": false,
		"surgeries":                   []any{map[string]any{"title": " Cesarean ", "date": "2025-03-04"}},
		"family_history":              []any{map[string]any{"condition": "Diabetes", "relative": "mother"}},
	})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"allergies":null,"allergies_on_emergency_card":false,
		"surgeries":[{"title":"Cesarean","date":"2025-03-04"}],
		"family_history":[{"condition":"Diabetes","relative":"mother"}]}`, mustJSON(t, r.data()))

	// bloom's basics edit keeps the extras, and the extras edit keeps the allergies.
	r = e.do(t, http.MethodPut, "/api/v1/health-record/basics", tok, map[string]any{"allergies": []string{"Penicillin"}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodPut, "/api/v1/health-record/extras", tok, map[string]any{"family_history": []any{}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"allergies":["Penicillin"],"allergies_on_emergency_card":false,
		"surgeries":[{"title":"Cesarean","date":"2025-03-04"}],"family_history":[]}`, mustJSON(t, r.data()))
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := jsonx.Marshal(v, 0)
	require.NoError(t, err)
	return string(b)
}

func TestRecordDocuments_CapHoldsUnderConcurrentCreates(t *testing.T) {
	e := setupDocs(t)
	uid, _ := e.user(t, "09120000001")
	e.docs.WithMaxDocuments(3)
	const workers = 8
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.docs.Create(context.Background(), uid, healthrecord.DocumentInput{Kind: healthrecord.KindVisit}, fixed)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	ok, capped := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, healthrecord.ErrTooManyDocuments):
			capped++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	assert.Equal(t, 3, ok)
	assert.Equal(t, workers-3, capped)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM record_documents WHERE user_id = ?`, uid).Scan(&n))
	assert.Equal(t, 3, n)
}

func TestRecordDocuments_VanishedFileIsNotFound(t *testing.T) {
	e := setupDocs(t)
	uid, _ := e.user(t, "09120000001")
	doc, err := e.docs.Create(context.Background(), uid, healthrecord.DocumentInput{Kind: healthrecord.KindOther}, fixed)
	require.NoError(t, err)
	// A file deleted between the ownership check and the insert: the FK error (1452) is file_not_found, not a 500.
	err = e.docs.AttachFilesForTest(context.Background(), uid, doc.ID, []uint64{987654}, fixed)
	var fe *healthrecord.FileError
	require.ErrorAs(t, err, &fe)
	assert.Equal(t, healthrecord.FileNotFound, fe.Code)
	assert.Equal(t, 0, fe.Index)
}

func TestRecordDocuments_UpdateAndDeleteRemoveFileRowsInTransaction(t *testing.T) {
	e := setupDocs(t)
	uid, _ := e.user(t, "09120000001")
	f1, f2 := e.upload(t, uid, files.PurposeRecordDocument), e.upload(t, uid, files.PurposeRecordDocument)
	doc, err := e.docs.Create(context.Background(), uid, healthrecord.DocumentInput{Kind: healthrecord.KindImaging,
		FileIDs: []uint64{f1, f2}}, fixed)
	require.NoError(t, err)
	kept, err := e.files.Get(context.Background(), uid, f1)
	require.NoError(t, err)

	_, err = e.docs.Update(context.Background(), uid, doc.ID, healthrecord.DocumentInput{Kind: healthrecord.KindImaging,
		SetFiles: true, FileIDs: []uint64{f1}}, fixed)
	require.NoError(t, err)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM files WHERE id = ?`, f2).Scan(&n))
	assert.Zero(t, n, "the dropped file row is deleted in the update transaction")
	_, err = e.files.Read(kept)
	require.NoError(t, err, "the kept file still opens")

	require.NoError(t, e.docs.Delete(context.Background(), uid, doc.ID))
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM files WHERE user_id = ?`, uid).Scan(&n))
	assert.Zero(t, n)
	_, err = e.files.Read(kept)
	require.Error(t, err, "the blob is removed after commit")
	require.ErrorIs(t, e.docs.Delete(context.Background(), uid, doc.ID), healthrecord.ErrDocumentNotFound)
}

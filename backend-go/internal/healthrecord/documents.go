package healthrecord

// Record documents (canvas-build CB-REC-01, D-70; boards nbl_Rec_Home, nbl_Rec_Timeline, nbl_Rec_Doc): every document
// kind of the record besides lab sheets (which stay in internal/labs and only appear in the timeline), with their files
// (internal/files rows of purpose record_document), the AI step's extracted JSON (empty until CB-REC-02), a review state
// and «where used» links (insurance claims, the pregnancy section). Owner-only like the rest of the record: every query
// carries the user id, a foreign id is the same 404 as an unknown one, and nothing here is logged.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Document kinds (lab sheets are KindLab in the timeline only).
const (
	KindImaging      = "imaging"
	KindVisit        = "visit"
	KindPrescription = "prescription"
	KindHospital     = "hospital"
	KindOther        = "other"
	KindLab          = "lab"
)

// DocumentKinds are the record_documents.kind values, in board order.
var DocumentKinds = []string{KindImaging, KindVisit, KindPrescription, KindHospital, KindOther}

// Review states: manual (typed in, nothing to review), pending (extraction queued, CB-REC-02), needs_review (extracted,
// waiting for the user), confirmed (the user checked it), failed (the extraction failed; the user fills it in).
const (
	ReviewManual      = "manual"
	ReviewPending     = "pending"
	ReviewNeedsReview = "needs_review"
	ReviewConfirmed   = "confirmed"
	ReviewFailed      = "failed"
)

// ReviewStates are every review state.
var ReviewStates = []string{ReviewManual, ReviewPending, ReviewNeedsReview, ReviewConfirmed, ReviewFailed}

// Link targets («این سند کجا استفاده شده؟») and their states. Links are written by the Go code of the owning task
// (CB-INS claims, CB-REC-02 pregnancy dating), never by a client.
const (
	LinkClaim     = "claim"
	LinkPregnancy = "pregnancy"

	LinkAttached = "attached" // the document is attached to the target
	LinkWaiting  = "waiting"  // the target waits for this document («منتظر همین مدرک»)
	LinkApplied  = "applied"  // the target was updated from this document («سن بارداری از همین سند به‌روز شد»)
)

// LinkTypes and LinkStates are the accepted link values.
var (
	LinkTypes  = []string{LinkClaim, LinkPregnancy}
	LinkStates = []string{LinkAttached, LinkWaiting, LinkApplied}
)

// Document limits.
const (
	MaxDocuments       = 500
	MaxDocumentFiles   = 10
	MaxDocumentText    = 120
	MaxDocumentNote    = 1000
	DefaultTimelineMax = 30
	MaxTimelineLimit   = 100
	// timelineLabs caps the lab sheets the timeline and the counts read (bloom's RecordLabs, newest first).
	timelineLabs = 1000
)

// Document errors.
var (
	// ErrDocumentNotFound: no such document of this user (a foreign id is the same 404).
	ErrDocumentNotFound = errors.New("healthrecord: document not found")
	// ErrTooManyDocuments: the user already has MaxDocuments documents.
	ErrTooManyDocuments = errors.New("healthrecord: too many documents")
	// ErrLinkTarget: the link target is not the user's (or the type / state is unknown).
	ErrLinkTarget = errors.New("healthrecord: invalid link target")
	// ErrDocumentBusy: an extraction of the document is pending (CB-REC-02): no kind change, no confirm.
	ErrDocumentBusy = errors.New("healthrecord: document extraction pending")
)

// FileError rejects file_ids[Index]: Code file_not_found (unknown, another user's, or not a record_document) or
// file_taken (attached to another document).
type FileError struct {
	Index int
	Code  string
}

func (e *FileError) Error() string { return "healthrecord: file " + e.Code }

// File error codes.
const (
	FileNotFound = "file_not_found"
	FileTaken    = "file_taken"
)

// FileStore is the slice of internal/files the documents use (*files.Service).
type FileStore interface {
	Get(ctx context.Context, owner, id uint64) (files.File, error)
	RemoveBlob(f files.File) error
	Link(f files.File, now time.Time, ttl time.Duration) (files.Link, error)
}

// Documents is the record documents service.
type Documents struct {
	db              *sql.DB
	q               *store.Queries
	files           FileStore
	labs            LabsSource
	maxDocs         int64
	onDeletePending PendingDeleteHook
}

// PendingDeleteHook is told, after the commit, that a document was deleted while its extraction was pending (CB-REC-02:
// the extraction package gives a never-run job's reserved Plus use back). extracted is the row's stored JSON.
type PendingDeleteHook func(ctx context.Context, userID uint64, extracted db.NullRawJSON)

// OnDeletePending installs the hook (nil = none).
func (s *Documents) OnDeletePending(h PendingDeleteHook) *Documents {
	s.onDeletePending = h
	return s
}

// NewDocuments wires the service. labs may be nil (no lab rows in the timeline and counts).
func NewDocuments(dbtx *sql.DB, fs FileStore, labs LabsSource) *Documents {
	return &Documents{db: dbtx, q: store.New(dbtx), files: fs, labs: labs, maxDocs: MaxDocuments}
}

// WithMaxDocuments overrides MaxDocuments (tests).
func (s *Documents) WithMaxDocuments(n int64) *Documents {
	s.maxDocs = n
	return s
}

// DocumentInput is a validated document (a full replace; the handler merges a partial PUT onto the stored row).
type DocumentInput struct {
	Kind                        string
	Title, Centre, Doctor, Note string         // "" = none
	Date, EndedOn               civildate.Date // zero = unknown
	SetFiles                    bool           // FileIDs replaces the files (always true on create)
	FileIDs                     []uint64
	Confirm                     bool // the user confirmed the extracted values (needs_review → confirmed)
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func nullDate(d civildate.Date) civildate.NullDate {
	return civildate.NullDate{Date: d, Valid: !d.IsZero()}
}

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

// Get is one document of the user.
func (s *Documents) Get(ctx context.Context, userID, id uint64) (store.RecordDocument, error) {
	row, err := s.q.GetRecordDocument(ctx, store.GetRecordDocumentParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return row, ErrDocumentNotFound
	}
	if err != nil {
		return row, fmt.Errorf("healthrecord: get document: %w", err)
	}
	return row, nil
}

// checkFiles makes sure every id is the user's own record_document file, not attached to another document than
// except (0 = any document).
func (s *Documents) checkFiles(ctx context.Context, userID uint64, ids []uint64, except uint64) error {
	for i, id := range ids {
		f, err := s.files.Get(ctx, userID, id)
		if errors.Is(err, files.ErrNotFound) || (err == nil && f.Purpose != files.PurposeRecordDocument) {
			return &FileError{Index: i, Code: FileNotFound}
		}
		if err != nil {
			return fmt.Errorf("healthrecord: document file: %w", err)
		}
		doc, err := s.q.GetRecordFileDocument(ctx, store.GetRecordFileDocumentParams{FileID: id, UserID: userID})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("healthrecord: document file: %w", err)
		}
		if err == nil && doc != except {
			return &FileError{Index: i, Code: FileTaken}
		}
	}
	return nil
}

// MySQL / MariaDB errors on attaching a file: ER_DUP_ENTRY (a concurrent request attached the same file) and
// ER_NO_REFERENCED_ROW_2 (the file was deleted between the ownership check and the insert).
const (
	errDuplicate = 1062
	errNoParent  = 1452
)

func mysqlErr(err error) uint16 {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number
	}
	return 0
}

func insertFiles(ctx context.Context, q *store.Queries, userID, docID uint64, ids []uint64, now time.Time) error {
	for i, id := range ids {
		if err := q.InsertRecordDocumentFile(ctx, store.InsertRecordDocumentFileParams{
			UserID: userID, DocumentID: docID, FileID: id, Position: uint8(i), Now: nullTime(now), //nolint:gosec // ≤ MaxDocumentFiles
		}); err != nil {
			switch mysqlErr(err) {
			case errDuplicate:
				return &FileError{Index: i, Code: FileTaken}
			case errNoParent:
				return &FileError{Index: i, Code: FileNotFound}
			}
			return fmt.Errorf("healthrecord: attach file: %w", err)
		}
	}
	return nil
}

func (s *Documents) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("healthrecord: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("healthrecord: commit: %w", err)
	}
	return nil
}

// Create adds a document with its files (ErrTooManyDocuments, *FileError).
func (s *Documents) Create(ctx context.Context, userID uint64, in DocumentInput, now time.Time) (store.RecordDocument, error) {
	if err := s.checkFiles(ctx, userID, in.FileIDs, 0); err != nil {
		return store.RecordDocument{}, err
	}
	var id uint64
	err := s.inTx(ctx, func(q *store.Queries) error {
		// The user row lock serialises this user's creates, so the cap below cannot be overshot by concurrent requests.
		if _, err := q.LockRecordOwner(ctx, userID); err != nil {
			return fmt.Errorf("healthrecord: lock owner: %w", err)
		}
		n, err := q.CountRecordDocuments(ctx, userID)
		if err != nil {
			return fmt.Errorf("healthrecord: count documents: %w", err)
		}
		if n >= s.maxDocs {
			return ErrTooManyDocuments
		}
		last, err := q.InsertRecordDocument(ctx, store.InsertRecordDocumentParams{
			UserID: userID, Kind: in.Kind, Title: nullStr(in.Title), DocumentDate: nullDate(in.Date),
			EndedOn: nullDate(in.EndedOn), Centre: nullStr(in.Centre), Doctor: nullStr(in.Doctor),
			Note: nullStr(in.Note), ReviewState: ReviewManual, Now: nullTime(now),
		})
		if err != nil {
			return fmt.Errorf("healthrecord: insert document: %w", err)
		}
		id = uint64(last) //nolint:gosec // auto-increment id
		return insertFiles(ctx, q, userID, id, in.FileIDs, now)
	})
	if err != nil {
		return store.RecordDocument{}, err
	}
	return s.Get(ctx, userID, id)
}

// Update replaces a document of the user (ErrDocumentNotFound, *FileError). Files dropped from the list are deleted
// (they were uploaded for this document).
func (s *Documents) Update(ctx context.Context, userID, id uint64, in DocumentInput, now time.Time) (store.RecordDocument, error) {
	cur, err := s.Get(ctx, userID, id)
	if err != nil {
		return cur, err
	}
	if in.SetFiles {
		if err := s.checkFiles(ctx, userID, in.FileIDs, id); err != nil {
			return cur, err
		}
	}
	var dropped []files.File
	err = s.inTx(ctx, func(q *store.Queries) error {
		if err := lockDocument(ctx, q, userID, id); err != nil {
			return err
		}
		// The review state comes from the locked row: an extraction job (CB-REC-02) may have moved it since cur was
		// read. While a job is pending the kind cannot change and nothing can be confirmed (security audit L1).
		locked, err := q.GetRecordDocument(ctx, store.GetRecordDocumentParams{ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("healthrecord: document: %w", err)
		}
		state := locked.ReviewState
		if state == ReviewPending && (in.Kind != locked.Kind || in.Confirm) {
			return ErrDocumentBusy
		}
		if in.Confirm && state == ReviewNeedsReview {
			state = ReviewConfirmed
		}
		// The row is locked and the user's (lockDocument); 0 affected rows only means nothing changed.
		_, err = q.UpdateRecordDocument(ctx, store.UpdateRecordDocumentParams{
			ID: id, UserID: userID, Kind: in.Kind, Title: nullStr(in.Title), DocumentDate: nullDate(in.Date),
			EndedOn: nullDate(in.EndedOn), Centre: nullStr(in.Centre), Doctor: nullStr(in.Doctor),
			Note: nullStr(in.Note), ReviewState: state, Now: nullTime(now),
		})
		if err != nil {
			return fmt.Errorf("healthrecord: update document: %w", err)
		}
		if !in.SetFiles {
			return nil
		}
		old, err := q.LockRecordDocumentFiles(ctx, store.LockRecordDocumentFilesParams{DocumentID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("healthrecord: document files: %w", err)
		}
		if err := q.DeleteRecordDocumentFiles(ctx, store.DeleteRecordDocumentFilesParams{DocumentID: id, UserID: userID}); err != nil {
			return fmt.Errorf("healthrecord: detach files: %w", err)
		}
		if err := insertFiles(ctx, q, userID, id, in.FileIDs, now); err != nil {
			return err
		}
		var keep []store.LockRecordDocumentFilesRow
		for _, f := range old {
			if !slices.Contains(in.FileIDs, f.ID) {
				keep = append(keep, f)
			}
		}
		dropped, err = deleteDetached(ctx, q, userID, keep)
		return err
	})
	if err != nil {
		return cur, err
	}
	s.removeBlobs(dropped)
	return s.Get(ctx, userID, id)
}

// lockDocument locks the user's document row for the rest of the transaction (ErrDocumentNotFound).
func lockDocument(ctx context.Context, q *store.Queries, userID, id uint64) error {
	_, err := q.LockRecordDocument(ctx, store.LockRecordDocumentParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDocumentNotFound
	}
	if err != nil {
		return fmt.Errorf("healthrecord: lock document: %w", err)
	}
	return nil
}

// deleteDetached deletes, inside the caller's transaction, the `files` rows of rows that no document uses any more,
// and returns them so their blobs can be removed after commit.
func deleteDetached(ctx context.Context, q *store.Queries, userID uint64, rows []store.LockRecordDocumentFilesRow,
) ([]files.File, error) {
	var out []files.File
	for _, r := range rows {
		n, err := q.DeleteDetachedRecordFile(ctx, store.DeleteDetachedRecordFileParams{
			ID: r.ID, UserID: sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // G115: user ids fit int64
		})
		if err != nil {
			return nil, fmt.Errorf("healthrecord: delete file: %w", err)
		}
		if n > 0 {
			out = append(out, files.File{ID: r.ID, Owner: userID, Purpose: r.Purpose, Path: r.Path})
		}
	}
	return out, nil
}

// removeBlobs removes the blobs of files whose rows were deleted with a committed transaction. Best effort: a failure
// leaves an unreferenced encrypted blob that the files orphan sweep removes; nothing is logged (health data).
func (s *Documents) removeBlobs(fs []files.File) {
	for _, f := range fs {
		_ = s.files.RemoveBlob(f)
	}
}

// Delete removes a document of the user, its links and its files in one transaction (the document row locked
// first); the blobs go after commit.
func (s *Documents) Delete(ctx context.Context, userID, id uint64) error {
	var (
		dropped []files.File
		pending db.NullRawJSON
	)
	err := s.inTx(ctx, func(q *store.Queries) error {
		if err := lockDocument(ctx, q, userID, id); err != nil {
			return err
		}
		row, err := q.GetRecordDocument(ctx, store.GetRecordDocumentParams{ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("healthrecord: document: %w", err)
		}
		if row.ReviewState == ReviewPending {
			pending = row.Extracted
		}
		rows, err := q.LockRecordDocumentFiles(ctx, store.LockRecordDocumentFilesParams{DocumentID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("healthrecord: document files: %w", err)
		}
		n, err := q.DeleteRecordDocument(ctx, store.DeleteRecordDocumentParams{ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("healthrecord: delete document: %w", err)
		}
		if n == 0 {
			return ErrDocumentNotFound
		}
		dropped, err = deleteDetached(ctx, q, userID, rows) // record_document_files went with the document (FK cascade)
		return err
	})
	if err != nil {
		return err
	}
	s.removeBlobs(dropped)
	if pending.Valid && s.onDeletePending != nil {
		s.onDeletePending(context.WithoutCancel(ctx), userID, pending)
	}
	return nil
}

// AddLink records that the user's document docID is used by a target (Go API for CB-INS claims and the CB-REC-02
// pregnancy dating hook): ErrDocumentNotFound, ErrLinkTarget. A pregnancy target must be the user's own pregnancy
// profile; a claim target is checked by its owning task before it calls this. Calling it again updates the state.
func (s *Documents) AddLink(ctx context.Context, userID, docID uint64, targetType string, targetID uint64, state string,
	now time.Time,
) error {
	if !slices.Contains(LinkTypes, targetType) || !slices.Contains(LinkStates, state) || targetID == 0 {
		return ErrLinkTarget
	}
	if _, err := s.Get(ctx, userID, docID); err != nil {
		return err
	}
	if targetType == LinkPregnancy {
		_, err := s.q.GetRecordPregnancyProfileID(ctx, store.GetRecordPregnancyProfileIDParams{ID: targetID, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLinkTarget
		}
		if err != nil {
			return fmt.Errorf("healthrecord: link target: %w", err)
		}
	}
	if err := s.q.UpsertRecordDocumentLink(ctx, store.UpsertRecordDocumentLinkParams{
		UserID: userID, DocumentID: docID, TargetType: targetType, TargetID: targetID, State: state, Now: nullTime(now),
	}); err != nil {
		return fmt.Errorf("healthrecord: link document: %w", err)
	}
	return nil
}

// RemoveLink drops a link of the user's document (ErrDocumentNotFound when there was none).
func (s *Documents) RemoveLink(ctx context.Context, userID, docID uint64, targetType string, targetID uint64) error {
	n, err := s.q.DeleteRecordDocumentLink(ctx, store.DeleteRecordDocumentLinkParams{
		DocumentID: docID, UserID: userID, TargetType: targetType, TargetID: targetID,
	})
	if err != nil {
		return fmt.Errorf("healthrecord: unlink document: %w", err)
	}
	if n == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

// --- views ----------------------------------------------------------------------------------------------------------

func strVal(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func dateVal(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date.String()
}

func timeVal(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.DateTime(t.Time)
}

// Detail is GET /health-record/documents/{id} (nbl_Rec_Doc): the document, its files with short-lived signed links,
// the extracted values and «where used».
func (s *Documents) Detail(ctx context.Context, userID uint64, doc store.RecordDocument, now time.Time) (*jsonx.OrderedMap, error) {
	ids, err := s.q.ListRecordDocumentFiles(ctx, store.ListRecordDocumentFilesParams{DocumentID: doc.ID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("healthrecord: document files: %w", err)
	}
	fileList := []*jsonx.OrderedMap{}
	for _, id := range ids {
		f, err := s.files.Get(ctx, userID, id)
		if errors.Is(err, files.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("healthrecord: document file: %w", err)
		}
		var url, exp any
		if link, err := s.files.Link(f, now, files.DefaultLinkTTL); err == nil { // no link while storage is disabled
			url = link.URL
			if link.ExpiresAt != nil {
				exp = jsonx.ISO8601(link.ExpiresAt.In(civildate.Tehran))
			}
		}
		fileList = append(fileList, jsonx.Obj("id", f.ID, "mime", f.MIME, "size_bytes", f.Size, "url", url,
			"url_expires_at", exp))
	}
	links, err := s.q.ListRecordDocumentLinks(ctx, store.ListRecordDocumentLinksParams{DocumentID: doc.ID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("healthrecord: document links: %w", err)
	}
	used := []*jsonx.OrderedMap{}
	for _, l := range links {
		used = append(used, jsonx.Obj("type", l.TargetType, "target_id", l.TargetID, "state", l.State,
			"updated_at", timeVal(l.UpdatedAt)))
	}
	extracted := publicExtracted(doc.Extracted) // CB-REC-02: without the job bookkeeping
	return jsonx.Obj(
		"id", doc.ID,
		"kind", doc.Kind,
		"title", strVal(doc.Title),
		"date", dateVal(doc.DocumentDate),
		"ended_on", dateVal(doc.EndedOn),
		"centre", strVal(doc.Centre),
		"doctor", strVal(doc.Doctor),
		"note", strVal(doc.Note),
		"review_state", doc.ReviewState,
		"extracted", extracted,
		"files", fileList,
		"where_used", used,
		"created_at", timeVal(doc.CreatedAt),
		"updated_at", timeVal(doc.UpdatedAt),
	), nil
}

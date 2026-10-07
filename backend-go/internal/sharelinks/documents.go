package sharelinks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/files"
	hrstore "github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Documents in a 24h summary (CB-REC-03): only the record documents (CB-REC-01) the owner explicitly picks when she
// creates the code — never all of them, never by default. The snapshot keeps their descriptive fields (kind, title,
// dates, centre, doctor — no free-text note, no extracted values) and file ids; every public open signs fresh
// short-lived links (files.DefaultLinkTTL) for the files that still exist and still belong to the owner, so a file
// she deleted is gone from the link too. The snapshot's file_ids list is never returned (the signed download URL
// itself names the file, as every signed link of internal/files does).

// MaxSharedDocuments caps the documents of one summary.
const MaxSharedDocuments = 10

// ErrDocument: a picked id is not one of the owner's documents (the handler answers 422 on document_ids.N).
type ErrDocument struct{ Index int }

func (e *ErrDocument) Error() string {
	return fmt.Sprintf("sharelinks: document %d not found", e.Index)
}

// SharedDocument is a document frozen into a summary snapshot.
type SharedDocument struct {
	Kind, Title, Centre, Doctor string
	Date, EndedOn               civildate.Date
	FileIDs                     []uint64
}

// DocumentSource reads the owner's documents (RecordDocuments on internal/healthrecord's store).
type DocumentSource interface {
	SharedDocuments(ctx context.Context, userID uint64, ids []uint64) ([]SharedDocument, error)
}

// FileLinker is the slice of internal/files the public open signs links with (*files.Service).
type FileLinker interface {
	Get(ctx context.Context, owner, id uint64) (files.File, error)
	Link(f files.File, now time.Time, ttl time.Duration) (files.Link, error)
}

// RecordDocuments is DocumentSource on the record documents tables (reads only, scoped by user id).
type RecordDocuments struct{ q *hrstore.Queries }

// NewRecordDocuments wires the source on db.
func NewRecordDocuments(db hrstore.DBTX) *RecordDocuments {
	return &RecordDocuments{q: hrstore.New(db)}
}

func str(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

// SharedDocuments loads ids of userID in order (*ErrDocument for a foreign or unknown id).
func (r *RecordDocuments) SharedDocuments(ctx context.Context, userID uint64, ids []uint64) ([]SharedDocument, error) {
	out := make([]SharedDocument, 0, len(ids))
	for i, id := range ids {
		d, err := r.q.GetRecordDocument(ctx, hrstore.GetRecordDocumentParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &ErrDocument{Index: i}
		}
		if err != nil {
			return nil, fmt.Errorf("sharelinks: document: %w", err)
		}
		fileIDs, err := r.q.ListRecordDocumentFiles(ctx, hrstore.ListRecordDocumentFilesParams{DocumentID: d.ID, UserID: userID})
		if err != nil {
			return nil, fmt.Errorf("sharelinks: document files: %w", err)
		}
		sd := SharedDocument{Kind: d.Kind, Title: str(d.Title), Centre: str(d.Centre), Doctor: str(d.Doctor),
			FileIDs: fileIDs}
		if d.DocumentDate.Valid {
			sd.Date = d.DocumentDate.Date
		}
		if d.EndedOn.Valid {
			sd.EndedOn = d.EndedOn.Date
		}
		out = append(out, sd)
	}
	return out, nil
}

func orNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func dateOrNil(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

// snapshotJSON is the frozen form of the documents (file ids stay server-side).
func snapshotDocuments(docs []SharedDocument) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(docs))
	for _, d := range docs {
		ids := d.FileIDs
		if ids == nil {
			ids = []uint64{}
		}
		out = append(out, jsonx.Obj("kind", d.Kind, "title", orNil(d.Title), "date", dateOrNil(d.Date),
			"ended_on", dateOrNil(d.EndedOn), "centre", orNil(d.Centre), "doctor", orNil(d.Doctor), "file_ids", ids))
	}
	return out
}

// publicDocuments turns the snapshot's documents into the viewer's form: file ids replaced by fresh signed links
// of the owner's files that still exist.
func (s *Service) publicDocuments(ctx context.Context, ownerID uint64, raw any, now time.Time) ([]*jsonx.OrderedMap, error) {
	_, docs := phpval.Entries(raw)
	out := make([]*jsonx.OrderedMap, 0, len(docs))
	for _, d := range docs {
		m, ok := d.(phpval.Map)
		if !ok {
			continue
		}
		get := func(k string) any { v, _ := m.Get(k); return v }
		fileList := []*jsonx.OrderedMap{}
		_, ids := phpval.Entries(get("file_ids"))
		for _, idv := range ids {
			if s.files == nil {
				break
			}
			id, ok := toUint(idv)
			if !ok {
				continue
			}
			f, err := s.files.Get(ctx, ownerID, id)
			if errors.Is(err, files.ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("sharelinks: document file: %w", err)
			}
			link, err := s.files.Link(f, now, files.DefaultLinkTTL)
			if err != nil { // storage disabled: the document stays listed without its file
				continue
			}
			var exp any
			if link.ExpiresAt != nil {
				exp = jsonx.ISO8601(link.ExpiresAt.In(civildate.Tehran))
			}
			fileList = append(fileList, jsonx.Obj("mime", f.MIME, "size_bytes", f.Size, "url", link.URL,
				"url_expires_at", exp))
		}
		out = append(out, jsonx.Obj("kind", get("kind"), "title", get("title"), "date", get("date"),
			"ended_on", get("ended_on"), "centre", get("centre"), "doctor", get("doctor"), "files", fileList))
	}
	return out, nil
}

func toUint(v any) (uint64, bool) {
	switch x := v.(type) {
	case int64:
		if x > 0 {
			return uint64(x), true
		}
	case float64:
		if x > 0 && x == float64(uint64(x)) {
			return uint64(x), true
		}
	case int:
		if x > 0 {
			return uint64(x), true
		}
	}
	return 0, false
}

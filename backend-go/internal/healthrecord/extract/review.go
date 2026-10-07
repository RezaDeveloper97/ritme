package extract

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/healthrecord/extract/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// ReviewInput is the user's check of the extracted values (validated by the handler against the kind's schema).
type ReviewInput struct {
	// Fields overrides values by key: a value replaces the extracted one, nil clears it; absent keys keep what was
	// read.
	Fields map[string]any
	// SetItems replaces the prescription rows with Items (absent = keep the rows as read).
	SetItems bool
	Items    []map[string]any
}

// Columns the review copies onto the document (when the reviewed value is a valid one for the column).
var columnKeys = []string{"date", "ended_on", "centre", "doctor"}

// Review confirms the user's document (needs_review → confirmed) with the corrected values; date, end date, centre and
// doctor are copied to the document's own columns. ErrNotFound, ErrNotInReview.
func (s *Service) Review(ctx context.Context, userID, docID uint64, in ReviewInput, now time.Time) error {
	today := civildate.InTehran(now)
	return s.inTx(ctx, func(q *store.Queries) error {
		doc, err := lockDoc(ctx, q, userID, docID)
		if err != nil {
			return err
		}
		st := parseStored(doc.Extracted)
		if doc.ReviewState != healthrecord.ReviewNeedsReview || st == nil || st.Status != StatusDone {
			return ErrNotInReview
		}
		reviewed := map[string]any{}
		for k, v := range st.Fields {
			reviewed[k] = v.Value
		}
		for k, v := range in.Fields {
			if v == nil {
				delete(reviewed, k)
				continue
			}
			reviewed[k] = v
		}
		items := make([]map[string]any, 0, len(st.Items))
		if in.SetItems {
			items = append(items, in.Items...)
		} else {
			for _, row := range st.Items {
				m := map[string]any{}
				for k, v := range row {
					m[k] = v.Value
				}
				items = append(items, m)
			}
		}
		st.Reviewed, st.ReviewedItems, st.ReviewedAt = reviewed, items, strPtr(isoTime(now))
		st.Dating = nil
		raw, err := st.raw()
		if err != nil {
			return err
		}
		p := store.ApplyExtractReviewParams{DocumentDate: doc.DocumentDate, EndedOn: doc.EndedOn, Centre: doc.Centre,
			Doctor: doc.Doctor, ReviewState: healthrecord.ReviewConfirmed, Extracted: raw, Now: nullTime(now), ID: docID, UserID: userID}
		for _, k := range columnKeys {
			v, ok := reviewed[k]
			if !ok {
				continue
			}
			sv, _ := v.(string)
			switch k {
			case "date", "ended_on":
				d, err := civildate.Parse(sv)
				if err != nil || d.After(today) || d.Before(civildate.New(1950, time.January, 1)) {
					continue
				}
				if k == "date" {
					p.DocumentDate = civildate.NullDate{Date: d, Valid: true}
				} else if doc.Kind == healthrecord.KindHospital {
					p.EndedOn = civildate.NullDate{Date: d, Valid: true}
				}
			case "centre", "doctor":
				if sv == "" {
					continue
				}
				if utf8.RuneCountInString(sv) > healthrecord.MaxDocumentText {
					sv = string([]rune(sv)[:healthrecord.MaxDocumentText])
				}
				if k == "centre" {
					p.Centre = sql.NullString{String: sv, Valid: true}
				} else {
					p.Doctor = sql.NullString{String: sv, Valid: true}
				}
			}
		}
		if p.EndedOn.Valid && p.DocumentDate.Valid && p.EndedOn.Date.Before(p.DocumentDate.Date) {
			p.EndedOn = doc.EndedOn
		}
		if _, err := q.ApplyExtractReview(ctx, p); err != nil {
			return fmt.Errorf("extract: review: %w", err)
		}
		return nil
	})
}

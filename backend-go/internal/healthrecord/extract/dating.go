package extract

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/healthrecord/extract/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// Dating offer states.
const (
	OfferOffered     = "offered"
	OfferApplied     = DatingApplied
	OfferDismissed   = DatingDismissed
	OfferUnavailable = "unavailable"
)

// Reasons an offer is unavailable.
const (
	ReasonNotImaging     = "not_imaging"     // only imaging documents date a pregnancy
	ReasonNotConfirmed   = "not_confirmed"   // the user has not confirmed the extracted values yet
	ReasonNoDating       = "no_dating"       // no scan date, gestational age or due date was read
	ReasonOutOfRange     = "out_of_range"    // implausible: a future scan, a gestational age outside 4–43 weeks, a past pregnancy
	ReasonNotPregnant    = "not_pregnant"    // the user is not in pregnancy mode
	ReasonOtherPregnancy = "other_pregnancy" // more than 6 weeks off the current dating: another pregnancy's scan
	ReasonSameDating     = "same_dating"     // the pregnancy is already dated from this scan
)

// Plausibility bounds of a scan's gestational age (days).
const (
	minScanGA       = 4 * 7
	maxScanGA       = 43*7 - 1
	termDays        = 280
	maxDatingDrift  = 42 // days between the scan's due date and the current one, beyond which it is another pregnancy
	pastPregnancyDD = 28 // a scan whose due date is this many days past is a past pregnancy
)

// Offer is the dating offer of a document (GET /health-record/documents/{id}/dating).
type Offer struct {
	State    string
	Reason   string // when State is unavailable
	ScanDate civildate.Date
	// AtScan is the gestational age at the scan (days); Due the due date it implies.
	AtScan int
	Due    civildate.Date
	Today  civildate.Date
	// Current dating of the pregnancy (CurrentDue zero when it has none).
	CurrentSource string
	CurrentDue    civildate.Date
	// ProfileID is the pregnancy profile the offer applies to (0 when not pregnant).
	ProfileID uint64
}

// ErrDatingUnavailable: the offer is not open (its state / reason say why).
type ErrDatingUnavailable struct{ Offer Offer }

func (e *ErrDatingUnavailable) Error() string {
	return "extract: dating offer " + e.Offer.State + " " + e.Offer.Reason
}

func number(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok && !math.IsNaN(f) && !math.IsInf(f, 0)
}

func dateOf(v any) (civildate.Date, bool) {
	s, ok := v.(string)
	if !ok {
		return civildate.Date{}, false
	}
	d, err := civildate.Parse(s)
	return d, err == nil
}

// offerFor works the offer out from the document and the user's pregnancy profile (nil = not pregnant).
func offerFor(doc store.RecordDocument, p *pregstore.PregnancyProfile, today civildate.Date) Offer {
	o := Offer{State: OfferUnavailable, Today: today}
	st := parseStored(doc.Extracted)
	switch {
	case doc.Kind != healthrecord.KindImaging:
		o.Reason = ReasonNotImaging
		return o
	case st == nil || doc.ReviewState != healthrecord.ReviewConfirmed:
		o.Reason = ReasonNotConfirmed
		return o
	}
	vals := st.values()
	scan, ok := dateOf(vals["date"])
	if !ok && doc.DocumentDate.Valid {
		scan, ok = doc.DocumentDate.Date, true
	}
	if !ok {
		o.Reason = ReasonNoDating
		return o
	}
	o.ScanDate = scan
	if w, ok := number(vals["ga_weeks"]); ok {
		d, _ := number(vals["ga_days"])
		o.AtScan = -1 // implausible → out_of_range below; clamped before any int conversion (audit L6)
		if w >= 0 && w <= 60 && d >= 0 && d <= 6 {
			o.AtScan = int(math.Round(w))*7 + int(math.Round(d))
		}
	} else if edd, ok := dateOf(vals["edd"]); ok {
		o.AtScan = termDays - scan.DiffDays(edd)
	} else {
		o.Reason = ReasonNoDating
		return o
	}
	if o.AtScan < minScanGA || o.AtScan > maxScanGA || scan.After(today) {
		o.Reason = ReasonOutOfRange
		return o
	}
	o.Due = scan.AddDays(termDays - o.AtScan)
	if o.Due.AddDays(pastPregnancyDD).Before(today) {
		o.Reason = ReasonOutOfRange
		return o
	}
	if p != nil {
		o.ProfileID = p.ID
		o.CurrentSource = p.AgeSource.String
		if due, ok := pregnancy.CurrentDue(p, today); ok {
			o.CurrentDue = due
		}
	}
	if st.Dating != nil {
		o.State = *st.Dating
		return o
	}
	if p == nil {
		o.Reason = ReasonNotPregnant
		return o
	}
	if !o.CurrentDue.IsZero() {
		if diff := o.CurrentDue.DiffDays(o.Due); diff > maxDatingDrift || diff < -maxDatingDrift {
			o.Reason = ReasonOtherPregnancy
			return o
		}
		if o.CurrentSource == "ultrasound" && p.UltrasoundDate.Valid && p.UltrasoundDate.Date == scan &&
			int(p.UltrasoundWeeks.Int32)*7+int(p.UltrasoundDays.Int32) == o.AtScan {
			o.Reason = ReasonSameDating
			return o
		}
	}
	o.State = OfferOffered
	return o
}

// activeProfile is the user's pregnancy profile in pregnancy mode, nil when not pregnant (or no querier is wired).
func activeProfile(ctx context.Context, pq pregstore.Querier, userID uint64) (*pregstore.PregnancyProfile, error) {
	if pq == nil {
		return nil, nil
	}
	p, err := pregnancy.ActiveProfile(ctx, pq, userID)
	if errors.Is(err, pregnancy.ErrNotPregnant) {
		return nil, nil
	}
	return p, err
}

// DatingOffer is the dating offer of the user's document. ErrNotFound.
func (s *Service) DatingOffer(ctx context.Context, userID, docID uint64, today civildate.Date) (Offer, error) {
	doc, err := s.q.GetExtractDocument(ctx, store.GetExtractDocumentParams{ID: docID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Offer{}, ErrNotFound
	}
	if err != nil {
		return Offer{}, fmt.Errorf("extract: document: %w", err)
	}
	p, err := activeProfile(ctx, s.preg, userID)
	if err != nil {
		return Offer{}, err
	}
	return offerFor(doc, p, today), nil
}

// setDating records the offer's outcome on the document in ONE transaction (security audit MEDIUM-1): the document
// row and the pregnancy profile row are locked FOR UPDATE, the offer is worked out from the locked rows, apply (the
// re-dating, through a tx-bound pregnancy querier) runs, and the outcome + the «where used» link are written — all
// or nothing.
func (s *Service) setDating(ctx context.Context, userID, docID uint64, outcome string, now time.Time,
	apply func(pq pregstore.Querier, o Offer) error,
) (Offer, error) {
	today := civildate.InTehran(now)
	var out Offer
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("extract: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)
	doc, err := lockDoc(ctx, q, userID, docID)
	if err != nil {
		return out, err
	}
	var pq pregstore.Querier
	if s.preg != nil {
		pq = pregstore.New(tx)
		if _, err := q.LockExtractPregnancyProfile(ctx, userID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return out, fmt.Errorf("extract: lock pregnancy: %w", err)
		}
	}
	p, err := activeProfile(ctx, pq, userID)
	if err != nil {
		return out, err
	}
	o := offerFor(doc, p, today)
	if o.State != OfferOffered {
		return out, &ErrDatingUnavailable{Offer: o}
	}
	if apply != nil {
		if err := apply(pq, o); err != nil {
			return out, err
		}
		if err := q.UpsertExtractDocumentLink(ctx, store.UpsertExtractDocumentLinkParams{UserID: userID, DocumentID: docID,
			TargetType: healthrecord.LinkPregnancy, TargetID: o.ProfileID, State: healthrecord.LinkApplied, Now: nullTime(now),
		}); err != nil {
			return out, fmt.Errorf("extract: link pregnancy: %w", err)
		}
		if p, err := activeProfile(ctx, pq, userID); err == nil && p != nil {
			if due, ok := pregnancy.CurrentDue(p, today); ok {
				o.CurrentDue, o.CurrentSource = due, p.AgeSource.String
			}
		}
	}
	st := parseStored(doc.Extracted)
	st.Dating = strPtr(outcome)
	raw, err := st.raw()
	if err != nil {
		return out, err
	}
	if _, err := q.SetExtractState(ctx, store.SetExtractStateParams{ReviewState: doc.ReviewState, Extracted: raw,
		Now: nullTime(now), ID: docID, UserID: userID}); err != nil {
		return out, fmt.Errorf("extract: dating: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return out, fmt.Errorf("extract: commit: %w", err)
	}
	o.State = outcome
	return o, nil
}

// ApplyDating re-dates the user's pregnancy from the document's scan — only on the user's explicit confirm — and links
// the document to the pregnancy («سن بارداری از همین سند به‌روز شد»). ErrNotFound, *ErrDatingUnavailable.
func (s *Service) ApplyDating(ctx context.Context, userID, docID uint64, now time.Time) (Offer, error) {
	return s.setDating(ctx, userID, docID, DatingApplied, now, func(pq pregstore.Querier, o Offer) error {
		_, err := pregnancy.ApplyUltrasoundDating(ctx, pq, userID, pregnancy.UltrasoundDating{
			ScanDate: o.ScanDate, Weeks: o.AtScan / 7, Days: o.AtScan % 7,
		}, now)
		return err
	})
}

// DismissDating closes the offer without touching the pregnancy. ErrNotFound, *ErrDatingUnavailable.
func (s *Service) DismissDating(ctx context.Context, userID, docID uint64, now time.Time) (Offer, error) {
	return s.setDating(ctx, userID, docID, DatingDismissed, now, nil)
}

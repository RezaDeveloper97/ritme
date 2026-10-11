package insurance

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/insurance/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Sealer encrypts free text at rest (*loss.NoteBox, PRIVATE_NOTE_KEY).
type Sealer interface {
	SealWith(plain string, ad []byte) (string, error)
	OpenWith(stored string, ad []byte) (string, error)
	Disabled() bool
}

// Service is the insurance domain.
type Service struct {
	db   *sql.DB
	q    *store.Queries
	seal Sealer
}

// NewService wires the service.
func NewService(conn *sql.DB, seal Sealer) *Service {
	return &Service{db: conn, q: store.New(conn), seal: seal}
}

// Queries is the store handle (admin handlers).
func (s *Service) Queries() *store.Queries { return s.q }

func dbTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

func sealAD(kind string, userID uint64) []byte {
	return []byte("insurance:" + kind + ":" + strconv.FormatUint(userID, 10))
}

// sealText seals non-empty text for the owner (NULL for "").
func (s *Service) sealText(kind string, userID uint64, plain string) (sql.NullString, error) {
	if plain == "" {
		return sql.NullString{}, nil
	}
	if s.seal == nil || s.seal.Disabled() {
		return sql.NullString{}, ErrStorage
	}
	enc, err := s.seal.SealWith(plain, sealAD(kind, userID))
	if err != nil {
		return sql.NullString{}, fmt.Errorf("%w: %w", ErrStorage, err)
	}
	return sql.NullString{String: enc, Valid: true}, nil
}

// openText opens sealed text of the owner ("" when empty or unreadable).
func (s *Service) openText(kind string, userID uint64, enc sql.NullString) string {
	if !enc.Valid || enc.String == "" || s.seal == nil || s.seal.Disabled() {
		return ""
	}
	plain, err := s.seal.OpenWith(enc.String, sealAD(kind, userID))
	if err != nil {
		return ""
	}
	return plain
}

func (s *Service) tx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func nullInt(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func nullID(id uint64) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(id), Valid: true} //nolint:gosec // G115: DB ids
}

func nullStr(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }

// ---------------------------------------------------------------------------------------------------------------
// Catalog

// Insurers is the active insurer catalog.
func (s *Service) Insurers(ctx context.Context) ([]store.InsuranceInsurer, error) {
	return s.q.ListActiveInsurers(ctx)
}

// Centres are the active contracted centres for her insurer (her policy's catalog insurer; none = the centres open to
// every insurer). No location is taken: distances are computed on the device.
func (s *Service) Centres(ctx context.Context, userID uint64, typ, q string) ([]store.ListActiveCentresRow, error) {
	insurer := sql.NullInt64{Valid: true}
	if p, err := s.q.GetUserPolicy(ctx, userID); err == nil && p.InsurerID.Valid {
		insurer = p.InsurerID
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return s.q.ListActiveCentres(ctx, store.ListActiveCentresParams{Type: typ, InsurerID: insurer, Q: q})
}

// ---------------------------------------------------------------------------------------------------------------
// Policy

// PolicyView is her policy with its members and coverages.
type PolicyView struct {
	Policy    store.GetUserPolicyRow
	Members   []store.InsurancePolicyMember
	Coverages []store.InsuranceCoverage
}

// Policy loads her policy (ErrNoPolicy when she has none).
func (s *Service) Policy(ctx context.Context, userID uint64) (PolicyView, error) {
	p, err := s.q.GetUserPolicy(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return PolicyView{}, ErrNoPolicy
	}
	if err != nil {
		return PolicyView{}, err
	}
	v := PolicyView{Policy: p}
	if v.Members, err = s.q.ListPolicyMembers(ctx, p.ID); err != nil {
		return PolicyView{}, err
	}
	if v.Coverages, err = s.q.ListPolicyCoverages(ctx, p.ID); err != nil {
		return PolicyView{}, err
	}
	return v, nil
}

// MemberInput is one validated member row (ID 0 = new).
type MemberInput struct {
	ID      uint64
	Kind    string
	ChildID uint64
	Name    string
}

// CoverageInput is one validated coverage row.
type CoverageInput struct {
	Category    string
	Status      string
	Cap         *int64
	Deductible  *int64
	WaitingDays *int64
	Note        string
}

// PolicyInput is a validated policy form (full replace).
type PolicyInput struct {
	InsurerID     uint64
	InsurerName   string
	PlanName      string
	NumberLast4   string
	KeepNumber    bool // number not sent: keep the stored last 4
	ValidFrom     civildate.NullDate
	ValidTo       civildate.NullDate
	DeadlineDays  *int64
	IBANLast4     string
	KeepIBAN      bool
	NotCovered    string
	Members       []MemberInput
	Coverages     []CoverageInput
	CoveragesSent bool
}

// InsurerUsable reports whether id is an active catalog insurer.
func (s *Service) InsurerUsable(ctx context.Context, id uint64) (bool, error) {
	i, err := s.q.GetInsurer(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil && i.IsActive, err
}

// OwnsChild reports whether childID is one of her bloom children.
func (s *Service) OwnsChild(ctx context.Context, userID, childID uint64) (bool, error) {
	return s.q.UserOwnsChild(ctx, store.UserOwnsChildParams{ChildID: childID, UserID: userID})
}

// SavePolicy creates or replaces her policy: members are matched by id (unknown ids are treated as new, missing ones
// removed — their claims keep the row with no member), coverages are replaced when sent.
func (s *Service) SavePolicy(ctx context.Context, userID uint64, in PolicyInput, now time.Time) (bool, error) {
	ts := dbTime(now)
	created := false
	err := s.tx(ctx, func(q *store.Queries) error {
		cur, err := q.GetUserPolicy(ctx, userID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		number, iban := nullStr(in.NumberLast4), nullStr(in.IBANLast4)
		if in.KeepNumber && err == nil {
			number = cur.NumberLast4
		}
		if in.KeepIBAN && err == nil {
			iban = cur.IbanLast4
		}
		var pid uint64
		if errors.Is(err, sql.ErrNoRows) {
			id, err := q.CreatePolicy(ctx, store.CreatePolicyParams{UserID: userID, InsurerID: nullID(in.InsurerID),
				InsurerName: nullStr(in.InsurerName), PlanName: nullStr(in.PlanName), NumberLast4: number,
				ValidFrom: in.ValidFrom, ValidTo: in.ValidTo, ClaimDeadlineDays: nullInt16(in.DeadlineDays), IbanLast4: iban,
				NotCoveredNote: nullStr(in.NotCovered), CreatedAt: ts, UpdatedAt: ts})
			if err != nil {
				return err
			}
			pid, created = uint64(id), true //nolint:gosec // G115: DB id
		} else {
			pid = cur.ID
			if err := q.UpdatePolicy(ctx, store.UpdatePolicyParams{InsurerID: nullID(in.InsurerID),
				InsurerName: nullStr(in.InsurerName), PlanName: nullStr(in.PlanName), NumberLast4: number,
				ValidFrom: in.ValidFrom, ValidTo: in.ValidTo, ClaimDeadlineDays: nullInt16(in.DeadlineDays), IbanLast4: iban,
				NotCoveredNote: nullStr(in.NotCovered), UpdatedAt: ts, ID: pid, UserID: userID}); err != nil {
				return err
			}
		}
		existing, err := q.ListPolicyMembers(ctx, pid)
		if err != nil {
			return err
		}
		keep := map[uint64]bool{}
		known := map[uint64]bool{}
		for _, m := range existing {
			known[m.ID] = true
		}
		for i, m := range in.Members {
			order := int32(i) //nolint:gosec // G115: ≤ MaxMembers
			if m.ID != 0 && known[m.ID] {
				keep[m.ID] = true
				if err := q.UpdateMember(ctx, store.UpdateMemberParams{Kind: m.Kind, ChildID: nullID(m.ChildID), Name: m.Name,
					SortOrder: order, UpdatedAt: ts, ID: m.ID, PolicyID: pid, UserID: userID}); err != nil {
					return err
				}
				continue
			}
			if _, err := q.CreateMember(ctx, store.CreateMemberParams{PolicyID: pid, Kind: m.Kind, ChildID: nullID(m.ChildID),
				Name: m.Name, SortOrder: order, CreatedAt: ts, UpdatedAt: ts}); err != nil {
				return err
			}
		}
		for _, m := range existing {
			if !keep[m.ID] {
				if err := q.DeleteMember(ctx, store.DeleteMemberParams{ID: m.ID, PolicyID: pid, UserID: userID}); err != nil {
					return err
				}
			}
		}
		if !in.CoveragesSent && !created {
			return nil
		}
		if err := q.DeletePolicyCoverages(ctx, store.DeletePolicyCoveragesParams{PolicyID: pid, UserID: userID}); err != nil {
			return err
		}
		for i, c := range in.Coverages {
			if err := q.CreateCoverage(ctx, store.CreateCoverageParams{PolicyID: pid, Category: c.Category, Status: c.Status,
				CapAmount: nullInt(c.Cap), DeductiblePct: nullInt16(c.Deductible), WaitingDays: nullInt16(c.WaitingDays),
				Note: nullStr(c.Note), SortOrder: int32(i), CreatedAt: ts, UpdatedAt: ts}); err != nil { //nolint:gosec // G115: ≤ MaxCoverages
				return err
			}
		}
		return nil
	})
	return created, err
}

func nullInt16(v *int64) sql.NullInt16 {
	if v == nil {
		return sql.NullInt16{}
	}
	return sql.NullInt16{Int16: int16(*v), Valid: true} //nolint:gosec // G115: validated ≤ MaxWaitingDays
}

// DeletePolicy removes her policy and everything under it (members, coverages, claims, timelines, document links).
func (s *Service) DeletePolicy(ctx context.Context, userID uint64) error {
	n, err := s.q.DeleteUserPolicy(ctx, userID)
	if err == nil && n == 0 {
		return ErrNoPolicy
	}
	return err
}

// ---------------------------------------------------------------------------------------------------------------
// Claims

// Facts are the claim facts of her claims (coverage maths).
func Facts(rows []store.InsuranceClaim) []ClaimFact {
	out := make([]ClaimFact, 0, len(rows))
	for _, r := range rows {
		f := ClaimFact{ID: r.ID, CostType: r.CostType, ServiceDate: r.ServiceDate, Status: r.Status}
		if r.PaidAmount.Valid {
			f.Paid = r.PaidAmount.Int64
		}
		out = append(out, f)
	}
	return out
}

// Claims are all her claims, newest service day first.
func (s *Service) Claims(ctx context.Context, userID uint64) ([]store.InsuranceClaim, error) {
	return s.q.ListUserClaims(ctx, userID)
}

// Claim loads one of her claims.
func (s *Service) Claim(ctx context.Context, userID, id uint64) (store.InsuranceClaim, error) {
	c, err := s.q.GetUserClaim(ctx, store.GetUserClaimParams{ID: id, UserID: userID})
	return c, notFound(err)
}

// ClaimNote opens the claim's sealed note.
func (s *Service) ClaimNote(c store.InsuranceClaim) string {
	return s.openText("claim", c.UserID, c.NotesEnc)
}

// EventNote opens a timeline step's sealed note.
func (s *Service) EventNote(userID uint64, e store.InsuranceClaimEvent) string {
	return s.openText("claim_event", userID, e.NoteEnc)
}

// ClaimDetail is the claim's timeline and documents.
type ClaimDetail struct {
	Events    []store.InsuranceClaimEvent
	Documents []store.ListClaimDocumentsRow
}

// Detail loads the timeline and the attached record documents (still hers) of a claim already loaded for userID.
func (s *Service) Detail(ctx context.Context, userID uint64, c store.InsuranceClaim) (ClaimDetail, error) {
	var d ClaimDetail
	var err error
	if d.Events, err = s.q.ListClaimEvents(ctx, c.ID); err != nil {
		return d, err
	}
	d.Documents, err = s.q.ListClaimDocuments(ctx, store.ListClaimDocumentsParams{UserID: userID, ClaimID: c.ID})
	return d, err
}

// DocumentRef is one record document to attach.
type DocumentRef struct {
	ID    uint64
	Label string
}

// ClaimInput is a validated claim form.
type ClaimInput struct {
	MemberID   uint64
	CostType   string
	Title      string
	CentreID   uint64
	CentreName string
	Date       civildate.Date
	Amount     int64
	OverTariff int64
	Note       string
	Documents  []DocumentRef
	DocsSent   bool
}

// CheckClaimRefs verifies that the member is on her policy, the centre is active and every document is hers. It
// returns the offending field ("" when all is well).
func (s *Service) CheckClaimRefs(ctx context.Context, userID uint64, pv PolicyView, in ClaimInput) (string, error) {
	if in.MemberID != 0 {
		ok := false
		for _, m := range pv.Members {
			ok = ok || m.ID == in.MemberID
		}
		if !ok {
			return "member_id", nil
		}
	}
	if in.CentreID != 0 {
		if _, err := s.q.GetActiveCentre(ctx, in.CentreID); errors.Is(err, sql.ErrNoRows) {
			return "centre_id", nil
		} else if err != nil {
			return "", err
		}
	}
	for i, d := range in.Documents {
		ok, err := s.q.UserOwnsDocument(ctx, store.UserOwnsDocumentParams{DocumentID: d.ID, UserID: userID})
		if err != nil {
			return "", err
		}
		if !ok {
			return "documents." + strconv.Itoa(i) + ".id", nil
		}
	}
	return "", nil
}

func (s *Service) attachAll(ctx context.Context, q *store.Queries, userID, claimID uint64, docs []DocumentRef, ts sql.NullTime) error {
	for _, d := range docs {
		if err := q.AttachClaimDocument(ctx, store.AttachClaimDocumentParams{ClaimID: claimID, UserID: userID, DocumentID: d.ID,
			Label: nullStr(d.Label), CreatedAt: ts, UpdatedAt: ts}); err != nil {
			return err
		}
	}
	return nil
}

// CreateClaim files a claim on her policy with a first «ثبت شد» step on today.
func (s *Service) CreateClaim(ctx context.Context, userID, policyID uint64, in ClaimInput, now time.Time) (uint64, error) {
	note, err := s.sealText("claim", userID, in.Note)
	if err != nil {
		return 0, err
	}
	ts := dbTime(now)
	var id uint64
	err = s.tx(ctx, func(q *store.Queries) error {
		nid, err := q.CreateClaim(ctx, store.CreateClaimParams{UserID: userID, PolicyID: policyID, MemberID: nullID(in.MemberID),
			CostType: in.CostType, Title: nullStr(in.Title), CentreID: nullID(in.CentreID), CentreName: nullStr(in.CentreName),
			ServiceDate: in.Date, Amount: uint64(in.Amount), OverTariff: uint64(in.OverTariff), NotesEnc: note, //nolint:gosec // G115: validated ≥ 0
			CreatedAt: ts, UpdatedAt: ts})
		if err != nil {
			return err
		}
		id = uint64(nid) //nolint:gosec // G115: DB id
		if _, err := q.CreateClaimEvent(ctx, store.CreateClaimEventParams{ClaimID: id, Status: ClaimSubmitted,
			OccurredOn: civildate.InTehran(now), CreatedAt: ts, UpdatedAt: ts}); err != nil {
			return err
		}
		return s.attachAll(ctx, q, userID, id, in.Documents, ts)
	})
	return id, err
}

// UpdateClaim replaces the claim's fields (documents too when sent).
func (s *Service) UpdateClaim(ctx context.Context, userID, id uint64, in ClaimInput, now time.Time) error {
	note, err := s.sealText("claim", userID, in.Note)
	if err != nil {
		return err
	}
	ts := dbTime(now)
	return s.tx(ctx, func(q *store.Queries) error {
		// Ownership first: nothing below (document links) may touch a claim that isn't hers.
		if _, err := q.GetUserClaim(ctx, store.GetUserClaimParams{ID: id, UserID: userID}); err != nil {
			return notFound(err)
		}
		if err := q.UpdateClaim(ctx, store.UpdateClaimParams{MemberID: nullID(in.MemberID), CostType: in.CostType,
			Title: nullStr(in.Title), CentreID: nullID(in.CentreID), CentreName: nullStr(in.CentreName), ServiceDate: in.Date,
			Amount: uint64(in.Amount), OverTariff: uint64(in.OverTariff), NotesEnc: note, UpdatedAt: ts, //nolint:gosec // G115: validated ≥ 0
			ID: id, UserID: userID}); err != nil {
			return err
		}
		if !in.DocsSent {
			return nil
		}
		if err := q.DeleteClaimDocuments(ctx, store.DeleteClaimDocumentsParams{ClaimID: id, UserID: userID}); err != nil {
			return err
		}
		return s.attachAll(ctx, q, userID, id, in.Documents, ts)
	})
}

// DeleteClaim removes one of her claims.
func (s *Service) DeleteClaim(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteUserClaim(ctx, store.DeleteUserClaimParams{ID: id, UserID: userID})
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

// EventInput is a validated timeline step.
type EventInput struct {
	Status   string
	On       civildate.Date
	Deadline civildate.NullDate
	Amount   *int64
	Note     string
}

func (s *Service) syncState(ctx context.Context, q *store.Queries, userID, claimID uint64, ts sql.NullTime) error {
	evs, err := q.ListClaimEvents(ctx, claimID)
	if err != nil {
		return err
	}
	facts := make([]EventFact, 0, len(evs))
	for _, e := range evs {
		f := EventFact{ID: e.ID, Status: e.Status, OccurredOn: e.OccurredOn, Deadline: e.DeadlineOn}
		if e.Amount.Valid {
			v := e.Amount.Int64
			f.Amount = &v
		}
		facts = append(facts, f)
	}
	st := StateOf(facts)
	return q.SetClaimState(ctx, store.SetClaimStateParams{Status: st.Status, PaidAmount: nullInt(st.Paid),
		DeadlineOn: st.Deadline, UpdatedAt: ts, ID: claimID, UserID: userID})
}

// AddEvent appends a step she recorded and re-derives the claim status.
func (s *Service) AddEvent(ctx context.Context, userID, claimID uint64, in EventInput, now time.Time) error {
	note, err := s.sealText("claim_event", userID, in.Note)
	if err != nil {
		return err
	}
	ts := dbTime(now)
	return s.tx(ctx, func(q *store.Queries) error {
		if _, err := q.GetUserClaim(ctx, store.GetUserClaimParams{ID: claimID, UserID: userID}); err != nil {
			return notFound(err)
		}
		evs, err := q.ListClaimEvents(ctx, claimID)
		if err != nil {
			return err
		}
		if len(evs) >= MaxEventsPerClaim {
			return ErrTooMany
		}
		if _, err := q.CreateClaimEvent(ctx, store.CreateClaimEventParams{ClaimID: claimID, Status: in.Status, OccurredOn: in.On,
			DeadlineOn: in.Deadline, Amount: nullInt(in.Amount), NoteEnc: note, CreatedAt: ts, UpdatedAt: ts}); err != nil {
			return err
		}
		return s.syncState(ctx, q, userID, claimID, ts)
	})
}

// DeleteEvent removes a step (never the last remaining one) and re-derives the claim status.
func (s *Service) DeleteEvent(ctx context.Context, userID, claimID, eventID uint64, now time.Time) error {
	ts := dbTime(now)
	return s.tx(ctx, func(q *store.Queries) error {
		if _, err := q.GetUserClaim(ctx, store.GetUserClaimParams{ID: claimID, UserID: userID}); err != nil {
			return notFound(err)
		}
		evs, err := q.ListClaimEvents(ctx, claimID)
		if err != nil {
			return err
		}
		found := false
		for _, e := range evs {
			found = found || e.ID == eventID
		}
		if !found {
			return ErrNotFound
		}
		if len(evs) <= 1 {
			return ErrEventLast
		}
		if _, err := q.DeleteClaimEvent(ctx, store.DeleteClaimEventParams{ID: eventID, ClaimID: claimID, UserID: userID}); err != nil {
			return err
		}
		return s.syncState(ctx, q, userID, claimID, ts)
	})
}

// AttachDocument links one of her record documents to her claim.
func (s *Service) AttachDocument(ctx context.Context, userID, claimID uint64, d DocumentRef, now time.Time) error {
	if _, err := s.Claim(ctx, userID, claimID); err != nil {
		return err
	}
	ok, err := s.q.UserOwnsDocument(ctx, store.UserOwnsDocumentParams{DocumentID: d.ID, UserID: userID})
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	docs, err := s.q.ListClaimDocuments(ctx, store.ListClaimDocumentsParams{UserID: userID, ClaimID: claimID})
	if err != nil {
		return err
	}
	if len(docs) >= MaxDocsPerClaim {
		return ErrTooMany
	}
	ts := dbTime(now)
	return s.q.AttachClaimDocument(ctx, store.AttachClaimDocumentParams{ClaimID: claimID, UserID: userID, DocumentID: d.ID, Label: nullStr(d.Label),
		CreatedAt: ts, UpdatedAt: ts})
}

// DetachDocument unlinks a document from her claim (the record document itself stays).
func (s *Service) DetachDocument(ctx context.Context, userID, claimID, documentID uint64) error {
	if _, err := s.Claim(ctx, userID, claimID); err != nil {
		return err
	}
	n, err := s.q.DetachClaimDocument(ctx, store.DetachClaimDocumentParams{ClaimID: claimID, DocumentID: documentID, UserID: userID})
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

// ---------------------------------------------------------------------------------------------------------------
// Questionnaires

// Questionnaires are her questionnaires, newest first.
func (s *Service) Questionnaires(ctx context.Context, userID uint64) ([]store.InsuranceQuestionnaire, error) {
	return s.q.ListUserQuestionnaires(ctx, userID)
}

// Questionnaire loads one of hers.
func (s *Service) Questionnaire(ctx context.Context, userID, id uint64) (store.InsuranceQuestionnaire, error) {
	r, err := s.q.GetUserQuestionnaire(ctx, store.GetUserQuestionnaireParams{ID: id, UserID: userID})
	return r, notFound(err)
}

// Explanation opens the sealed explanation.
func (s *Service) Explanation(r store.InsuranceQuestionnaire) string {
	return s.openText("questionnaire", r.UserID, r.ExplanationEnc)
}

func decodeStrings(raw db.NullRawJSON) []string {
	if !raw.Valid {
		return nil
	}
	var items []any
	if json.Unmarshal(raw.V, &items) != nil || items == nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, v := range items {
		switch x := v.(type) {
		case string:
			out = append(out, x)
		case map[string]any:
			if c, ok := x["condition"].(string); ok {
				out = append(out, c)
			} else if t, ok := x["title"].(string); ok {
				out = append(out, t)
			}
		}
	}
	return out
}

// Facts reads the record items the prefill uses (owner only).
func (s *Service) Facts(ctx context.Context, userID uint64, today civildate.Date) (RecordFacts, error) {
	var f RecordFacts
	lp, err := s.q.PrefillLifeProfile(ctx, userID)
	switch {
	case err == nil:
		f.HasLifeProfile, f.LifeMode, f.IVFOrIUI = true, lp.LifeMode.String, lp.IvfIui
		f.Chronic, f.Medications = decodeStrings(lp.ChronicIllnesses), decodeStrings(lp.Medications)
		f.MenoSurgical = lp.MenopauseSurgical.Valid && lp.MenopauseSurgical.Bool
	case !errors.Is(err, sql.ErrNoRows):
		return f, err
	}
	hr, err := s.q.PrefillHealthRecord(ctx, userID)
	switch {
	case err == nil:
		if hr.Surgeries.Valid {
			f.Surgeries = []Surgery{}
			_ = json.Unmarshal(hr.Surgeries.V, &f.Surgeries)
		}
		f.FamilyHistory = decodeStrings(hr.FamilyHistory)
	case !errors.Is(err, sql.ErrNoRows):
		return f, err
	}
	pm, err := s.q.PrefillPregnancy(ctx, userID)
	switch {
	case err == nil:
		f.PregnancyMode = &pm
	case !errors.Is(err, sql.ErrNoRows):
		return f, err
	}
	ces, err := s.q.PrefillCesareans(ctx, userID)
	if err != nil {
		return f, err
	}
	for _, d := range ces {
		if d.Valid {
			f.Cesareans = append(f.Cesareans, d.Date)
		}
	}
	docs, err := s.q.PrefillHospitalDocuments(ctx, store.PrefillHospitalDocumentsParams{UserID: userID,
		Since: civildate.NullDate{Date: today.AddDays(-5 * 365), Valid: true}})
	if err != nil {
		return f, err
	}
	for _, d := range docs {
		f.HospitalDocs = append(f.HospitalDocs, d.Title.String)
	}
	n, err := s.q.PrefillOpenIVF(ctx, userID)
	f.OpenIVF = n > 0
	return f, err
}

// CreateQuestionnaire starts a draft prefilled from her record.
func (s *Service) CreateQuestionnaire(ctx context.Context, userID uint64, purpose, label string, now time.Time) (uint64, error) {
	n, err := s.q.CountUserQuestionnaires(ctx, userID)
	if err != nil {
		return 0, err
	}
	if n >= MaxQuestionnaires {
		return 0, ErrTooMany
	}
	facts, err := s.Facts(ctx, userID, civildate.InTehran(now))
	if err != nil {
		return 0, err
	}
	answers, err := json.Marshal(Prefill(facts, civildate.InTehran(now)))
	if err != nil {
		return 0, err
	}
	ts := dbTime(now)
	id, err := s.q.CreateQuestionnaire(ctx, store.CreateQuestionnaireParams{UserID: userID, Purpose: purpose, Label: nullStr(label),
		Answers: answers, CreatedAt: ts, UpdatedAt: ts})
	return uint64(id), err //nolint:gosec // G115: DB id
}

// QuestionnaireInput is a validated edit: answers by code (absent codes keep their answer).
type QuestionnaireInput struct {
	Answers         map[string]*string
	Explanation     string
	ExplanationSent bool
	Label           string
	LabelSent       bool
}

func (s *Service) saveQuestionnaire(ctx context.Context, r store.InsuranceQuestionnaire, answers []Answer, expl sql.NullString,
	status string, confirmed sql.NullTime, timeline []TimelineStep, now time.Time,
) error {
	a, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	tl := db.NullRawJSON{}
	if len(timeline) > 0 {
		b, err := json.Marshal(timeline)
		if err != nil {
			return err
		}
		tl = db.NullRawJSON{V: b, Valid: true}
	}
	return s.q.UpdateQuestionnaire(ctx, store.UpdateQuestionnaireParams{Answers: a, ExplanationEnc: expl, Label: r.Label,
		Status: status, ConfirmedAt: confirmed, Timeline: tl, UpdatedAt: dbTime(now), ID: r.ID, UserID: r.UserID})
}

// UpdateQuestionnaire edits a draft or a confirmed (not yet sent) questionnaire; a confirmed one goes back to draft.
func (s *Service) UpdateQuestionnaire(ctx context.Context, userID, id uint64, in QuestionnaireInput, now time.Time) error {
	r, err := s.Questionnaire(ctx, userID, id)
	if err != nil {
		return err
	}
	if r.Status != QDraft && r.Status != QConfirmed {
		return ErrLocked
	}
	answers := DecodeAnswers(r.Answers)
	for i := range answers {
		if v, ok := in.Answers[answers[i].Code]; ok {
			answers[i].Answer = v
		}
	}
	expl := r.ExplanationEnc
	if in.ExplanationSent {
		if expl, err = s.sealText("questionnaire", userID, in.Explanation); err != nil {
			return err
		}
	}
	if in.LabelSent {
		r.Label = nullStr(in.Label)
	}
	return s.saveQuestionnaire(ctx, r, answers, expl, QDraft, sql.NullTime{}, nil, now)
}

// ConfirmQuestionnaire marks a complete draft as checked by her (ready to export).
func (s *Service) ConfirmQuestionnaire(ctx context.Context, userID, id uint64, now time.Time) error {
	r, err := s.Questionnaire(ctx, userID, id)
	if err != nil {
		return err
	}
	if r.Status != QDraft && r.Status != QConfirmed {
		return ErrLocked
	}
	answers := DecodeAnswers(r.Answers)
	if !Complete(answers) {
		return ErrIncomplete
	}
	return s.saveQuestionnaire(ctx, r, answers, r.ExplanationEnc, QConfirmed, dbTime(now), nil, now)
}

// AdvanceQuestionnaire records a status step she was told by the insurer (only after confirming).
func (s *Service) AdvanceQuestionnaire(ctx context.Context, userID, id uint64, status string, on civildate.Date, now time.Time) error {
	r, err := s.Questionnaire(ctx, userID, id)
	if err != nil {
		return err
	}
	if r.Status == QDraft {
		return ErrNotConfirmed
	}
	tl := DecodeTimeline(r.Timeline.V)
	if len(tl) >= MaxEventsPerClaim {
		return ErrTooMany
	}
	tl = append(tl, TimelineStep{Status: status, On: on.String()})
	return s.saveQuestionnaire(ctx, r, DecodeAnswers(r.Answers), r.ExplanationEnc, status, r.ConfirmedAt, tl, now)
}

// DeleteQuestionnaire removes one of hers.
func (s *Service) DeleteQuestionnaire(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteUserQuestionnaire(ctx, store.DeleteUserQuestionnaireParams{ID: id, UserID: userID})
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

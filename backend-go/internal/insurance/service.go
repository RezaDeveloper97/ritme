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

// Centres are the active contracted centres for her insurer (her current policy's catalog insurer; none = the centres
// open to every insurer). No location is taken: distances are computed on the device.
func (s *Service) Centres(ctx context.Context, userID uint64, typ, q string, today civildate.Date) ([]store.ListActiveCentresRow, error) {
	insurer := sql.NullInt64{Valid: true}
	if pv, err := s.Policy(ctx, userID, today); err == nil && pv.Policy.InsurerID.Valid {
		insurer = pv.Policy.InsurerID
	} else if err != nil && !errors.Is(err, ErrNoPolicy) {
		return nil, err
	}
	return s.q.ListActiveCentres(ctx, store.ListActiveCentresParams{Type: typ, InsurerID: insurer, Q: q})
}

// ---------------------------------------------------------------------------------------------------------------
// Policy history

// PolicyRow is one of her policies (catalog insurer name joined).
type PolicyRow = store.ListUserPoliciesRow

// HistoryItem is one policy of her history with its place in it.
type HistoryItem struct {
	Policy PolicyRow
	Status string // PolicyCurrent / PolicyUpcoming / PolicyPast
}

// History is her policies, newest period first, with the current one and the one that can be renewed.
type History struct {
	Items     []HistoryItem
	CurrentID uint64
	NewestID  uint64
}

// ReadOnly reports whether the policy with id is past (it and its claims are read-only). Unknown ids are read-only.
func (h History) ReadOnly(id uint64) bool {
	for _, it := range h.Items {
		if it.Policy.ID == id {
			return it.Status == PolicyPast
		}
	}
	return true
}

func periodOfRow(p PolicyRow) Period { return Period{From: p.ValidFrom, To: p.ValidTo} }

func historyOf(rows []PolicyRow, today civildate.Date) History {
	facts := make([]PolicyFact, 0, len(rows))
	for _, r := range rows {
		facts = append(facts, PolicyFact{ID: r.ID, Period: periodOfRow(r)})
	}
	h := History{CurrentID: CurrentOf(facts, today), NewestID: NewestOf(facts)}
	for i, r := range rows {
		h.Items = append(h.Items, HistoryItem{Policy: r, Status: StatusOf(facts[i], h.CurrentID, today)})
	}
	return h
}

// History loads her policy history on `today`.
func (s *Service) History(ctx context.Context, userID uint64, today civildate.Date) (History, error) {
	return s.history(ctx, s.q, userID, today)
}

func (s *Service) history(ctx context.Context, q *store.Queries, userID uint64, today civildate.Date) (History, error) {
	rows, err := q.ListUserPolicies(ctx, userID)
	if err != nil {
		return History{}, err
	}
	return historyOf(rows, today), nil
}

// PolicyView is one of her policies with its members, coverages and place in the history.
type PolicyView struct {
	Policy    PolicyRow
	Status    string
	CanRenew  bool
	Members   []store.InsurancePolicyMember
	Coverages []store.InsuranceCoverage
}

// ReadOnly reports whether the policy is past (it and its claims cannot change).
func (v PolicyView) ReadOnly() bool { return v.Status == PolicyPast }

func (s *Service) view(ctx context.Context, h History, id uint64) (PolicyView, error) {
	for _, it := range h.Items {
		if it.Policy.ID != id {
			continue
		}
		v := PolicyView{Policy: it.Policy, Status: it.Status, CanRenew: id == h.NewestID}
		var err error
		if v.Members, err = s.q.ListPolicyMembers(ctx, id); err != nil {
			return PolicyView{}, err
		}
		if v.Coverages, err = s.q.ListPolicyCoverages(ctx, id); err != nil {
			return PolicyView{}, err
		}
		return v, nil
	}
	return PolicyView{}, ErrNotFound
}

// Policy loads her current policy (ErrNoPolicy when she has none).
func (s *Service) Policy(ctx context.Context, userID uint64, today civildate.Date) (PolicyView, error) {
	h, err := s.History(ctx, userID, today)
	if err != nil {
		return PolicyView{}, err
	}
	if h.CurrentID == 0 {
		return PolicyView{}, ErrNoPolicy
	}
	return s.view(ctx, h, h.CurrentID)
}

// PolicyByID loads one of her policies (ErrNotFound for a missing or another user's id).
func (s *Service) PolicyByID(ctx context.Context, userID, id uint64, today civildate.Date) (PolicyView, error) {
	h, err := s.History(ctx, userID, today)
	if err != nil {
		return PolicyView{}, err
	}
	return s.view(ctx, h, id)
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

// SavePolicy is PUT /policy: replaces her current policy, or creates her first one. It reports whether it created and
// the policy id.
func (s *Service) SavePolicy(ctx context.Context, userID uint64, in PolicyInput, now time.Time) (bool, uint64, error) {
	ts, today := dbTime(now), civildate.InTehran(now)
	created := false
	var pid uint64
	err := s.tx(ctx, func(q *store.Queries) error {
		h, err := s.history(ctx, q, userID, today)
		if err != nil {
			return err
		}
		if h.CurrentID == 0 {
			pid, err = s.insertPolicy(ctx, q, userID, in, ts)
			created = true
			return err
		}
		pid = h.CurrentID
		return s.writePolicy(ctx, q, userID, h, pid, in, ts)
	})
	return created, pid, err
}

// UpdatePolicy replaces one of her policies by id (ErrNotFound; ErrReadOnly for a past one).
func (s *Service) UpdatePolicy(ctx context.Context, userID, id uint64, in PolicyInput, now time.Time) error {
	ts, today := dbTime(now), civildate.InTehran(now)
	return s.tx(ctx, func(q *store.Queries) error {
		h, err := s.history(ctx, q, userID, today)
		if err != nil {
			return err
		}
		return s.writePolicy(ctx, q, userID, h, id, in, ts)
	})
}

// CreatePolicy adds a new policy to her history (another insurer, a policy she forgot to add, …).
func (s *Service) CreatePolicy(ctx context.Context, userID uint64, in PolicyInput, now time.Time) (uint64, error) {
	ts, today := dbTime(now), civildate.InTehran(now)
	var pid uint64
	err := s.tx(ctx, func(q *store.Queries) error {
		h, err := s.history(ctx, q, userID, today)
		if err != nil {
			return err
		}
		if len(h.Items) >= MaxPolicies {
			return ErrTooMany
		}
		pid, err = s.insertPolicy(ctx, q, userID, in, ts)
		return err
	})
	return pid, err
}

func (s *Service) insertPolicy(ctx context.Context, q *store.Queries, userID uint64, in PolicyInput, ts sql.NullTime) (uint64, error) {
	id, err := q.CreatePolicy(ctx, store.CreatePolicyParams{UserID: userID, InsurerID: nullID(in.InsurerID),
		InsurerName: nullStr(in.InsurerName), PlanName: nullStr(in.PlanName), NumberLast4: nullStr(in.NumberLast4),
		ValidFrom: in.ValidFrom, ValidTo: in.ValidTo, ClaimDeadlineDays: nullInt16(in.DeadlineDays), IbanLast4: nullStr(in.IBANLast4),
		NotCoveredNote: nullStr(in.NotCovered), CreatedAt: ts, UpdatedAt: ts})
	if err != nil {
		return 0, err
	}
	pid := uint64(id) //nolint:gosec // G115: DB id
	for i, m := range in.Members {
		if _, err := q.CreateMember(ctx, store.CreateMemberParams{PolicyID: pid, Kind: m.Kind, ChildID: nullID(m.ChildID),
			Name: m.Name, SortOrder: int32(i), CreatedAt: ts, UpdatedAt: ts}); err != nil { //nolint:gosec // G115: ≤ MaxMembers
			return 0, err
		}
	}
	return pid, s.replaceCoverages(ctx, q, userID, pid, in.Coverages, ts)
}

func (s *Service) replaceCoverages(ctx context.Context, q *store.Queries, userID, pid uint64, covs []CoverageInput, ts sql.NullTime) error {
	if err := q.DeletePolicyCoverages(ctx, store.DeletePolicyCoveragesParams{PolicyID: pid, UserID: userID}); err != nil {
		return err
	}
	for i, c := range covs {
		if err := q.CreateCoverage(ctx, store.CreateCoverageParams{PolicyID: pid, Category: c.Category, Status: c.Status,
			CapAmount: nullInt(c.Cap), DeductiblePct: nullInt16(c.Deductible), WaitingDays: nullInt16(c.WaitingDays),
			Note: nullStr(c.Note), SortOrder: int32(i), CreatedAt: ts, UpdatedAt: ts}); err != nil { //nolint:gosec // G115: ≤ MaxCoverages
			return err
		}
	}
	return nil
}

// writePolicy replaces an existing policy of hers: members are matched by id (unknown ids are treated as new, missing
// ones removed — their claims keep the row with no member), coverages are replaced when sent.
func (s *Service) writePolicy(ctx context.Context, q *store.Queries, userID uint64, h History, pid uint64, in PolicyInput, ts sql.NullTime) error {
	var cur *PolicyRow
	for i := range h.Items {
		if h.Items[i].Policy.ID == pid {
			cur = &h.Items[i].Policy
		}
	}
	if cur == nil {
		return ErrNotFound
	}
	if h.ReadOnly(pid) {
		return ErrReadOnly
	}
	number, iban := nullStr(in.NumberLast4), nullStr(in.IBANLast4)
	if in.KeepNumber {
		number = cur.NumberLast4
	}
	if in.KeepIBAN {
		iban = cur.IbanLast4
	}
	if err := q.UpdatePolicy(ctx, store.UpdatePolicyParams{InsurerID: nullID(in.InsurerID),
		InsurerName: nullStr(in.InsurerName), PlanName: nullStr(in.PlanName), NumberLast4: number,
		ValidFrom: in.ValidFrom, ValidTo: in.ValidTo, ClaimDeadlineDays: nullInt16(in.DeadlineDays), IbanLast4: iban,
		NotCoveredNote: nullStr(in.NotCovered), UpdatedAt: ts, ID: pid, UserID: userID}); err != nil {
		return err
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
	if !in.CoveragesSent {
		return nil
	}
	return s.replaceCoverages(ctx, q, userID, pid, in.Coverages, ts)
}

// RenewInput is the period of a renewal (zero = RenewalPeriod's default).
type RenewInput struct {
	ValidFrom civildate.NullDate
	ValidTo   civildate.NullDate
}

// Renew creates the next policy from her newest one: a new period, the same insurer, plan, masked number / IBAN,
// deadline, note, members and coverages as a starting point. The renewed policy keeps its claims and becomes past
// once the new period starts (ErrRenewed for an older policy, ErrNotFound for another user's).
func (s *Service) Renew(ctx context.Context, userID, id uint64, in RenewInput, now time.Time) (uint64, error) {
	ts, today := dbTime(now), civildate.InTehran(now)
	var pid uint64
	err := s.tx(ctx, func(q *store.Queries) error {
		h, err := s.history(ctx, q, userID, today)
		if err != nil {
			return err
		}
		var src *PolicyRow
		for i := range h.Items {
			if h.Items[i].Policy.ID == id {
				src = &h.Items[i].Policy
			}
		}
		switch {
		case src == nil:
			return ErrNotFound
		case id != h.NewestID:
			return ErrRenewed
		case len(h.Items) >= MaxPolicies:
			return ErrTooMany
		}
		period := RenewalPeriod(periodOfRow(*src), today)
		if in.ValidFrom.Valid {
			period.From = in.ValidFrom
		}
		if in.ValidTo.Valid {
			period.To = in.ValidTo
		}
		nid, err := q.CreatePolicy(ctx, store.CreatePolicyParams{UserID: userID, InsurerID: src.InsurerID,
			InsurerName: src.InsurerName, PlanName: src.PlanName, NumberLast4: src.NumberLast4, ValidFrom: period.From,
			ValidTo: period.To, ClaimDeadlineDays: src.ClaimDeadlineDays, IbanLast4: src.IbanLast4,
			NotCoveredNote: src.NotCoveredNote, CreatedAt: ts, UpdatedAt: ts})
		if err != nil {
			return err
		}
		pid = uint64(nid) //nolint:gosec // G115: DB id
		members, err := q.ListPolicyMembers(ctx, id)
		if err != nil {
			return err
		}
		for _, m := range members {
			if _, err := q.CreateMember(ctx, store.CreateMemberParams{PolicyID: pid, Kind: m.Kind, ChildID: m.ChildID,
				Name: m.Name, SortOrder: m.SortOrder, CreatedAt: ts, UpdatedAt: ts}); err != nil {
				return err
			}
		}
		covs, err := q.ListPolicyCoverages(ctx, id)
		if err != nil {
			return err
		}
		for _, c := range covs {
			if err := q.CreateCoverage(ctx, store.CreateCoverageParams{PolicyID: pid, Category: c.Category, Status: c.Status,
				CapAmount: c.CapAmount, DeductiblePct: c.DeductiblePct, WaitingDays: c.WaitingDays, Note: c.Note,
				SortOrder: c.SortOrder, CreatedAt: ts, UpdatedAt: ts}); err != nil {
				return err
			}
		}
		return nil
	})
	return pid, err
}

func nullInt16(v *int64) sql.NullInt16 {
	if v == nil {
		return sql.NullInt16{}
	}
	return sql.NullInt16{Int16: int16(*v), Valid: true} //nolint:gosec // G115: validated ≤ MaxWaitingDays
}

// DeletePolicy removes one of her policies (id 0 = the current one) and everything under it (members, coverages,
// claims, timelines, document links). Deleting is her right even for a past policy.
func (s *Service) DeletePolicy(ctx context.Context, userID, id uint64, today civildate.Date) error {
	if id == 0 {
		h, err := s.History(ctx, userID, today)
		if err != nil {
			return err
		}
		if h.CurrentID == 0 {
			return ErrNoPolicy
		}
		id = h.CurrentID
	}
	n, err := s.q.DeleteUserPolicy(ctx, store.DeleteUserPolicyParams{ID: id, UserID: userID})
	if err == nil && n == 0 {
		return ErrNotFound
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

// Claims are all her claims (every policy), newest service day first.
func (s *Service) Claims(ctx context.Context, userID uint64) ([]store.InsuranceClaim, error) {
	return s.q.ListUserClaims(ctx, userID)
}

// PolicyClaims are the claims of one of her policies, newest service day first.
func (s *Service) PolicyClaims(ctx context.Context, userID, policyID uint64) ([]store.InsuranceClaim, error) {
	return s.q.ListPolicyClaims(ctx, store.ListPolicyClaimsParams{PolicyID: policyID, UserID: userID})
}

// ClaimWithPolicy loads one of her claims with the policy it is attached to.
func (s *Service) ClaimWithPolicy(ctx context.Context, userID, id uint64, today civildate.Date) (store.InsuranceClaim, PolicyView, error) {
	c, err := s.Claim(ctx, userID, id)
	if err != nil {
		return c, PolicyView{}, err
	}
	pv, err := s.PolicyByID(ctx, userID, c.PolicyID, today)
	return c, pv, err
}

// WritableClaim is ClaimWithPolicy for a change: ErrReadOnly when the claim belongs to a past policy.
func (s *Service) WritableClaim(ctx context.Context, userID, id uint64, today civildate.Date) (store.InsuranceClaim, PolicyView, error) {
	c, pv, err := s.ClaimWithPolicy(ctx, userID, id, today)
	if err == nil && pv.ReadOnly() {
		err = ErrReadOnly
	}
	return c, pv, err
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

// Answers are the stored answers with their evidence: opened from evidence_enc (sealed, 00093), or the plain evidence
// of a row written before it. Without the key (or for an unreadable seal) the evidence is empty — never an error, the
// answers themselves stay usable.
func (s *Service) Answers(r store.InsuranceQuestionnaire) []Answer {
	answers := DecodeAnswers(r.Answers)
	if !r.EvidenceEnc.Valid {
		return answers
	}
	ev := map[string][]string{}
	if plain := s.openText("questionnaire", r.UserID, r.EvidenceEnc); plain != "" {
		_ = json.Unmarshal([]byte(plain), &ev)
	}
	for i := range answers {
		answers[i].Evidence = ev[answers[i].Code]
		if answers[i].Evidence == nil {
			answers[i].Evidence = []string{}
		}
	}
	return answers
}

// sealEvidence moves the answers' evidence into a sealed {code: [items]} JSON (NULL when there is none) and strips it
// from the answers. ErrStorage when there is evidence and no key: health-record items are never stored in clear.
func (s *Service) sealEvidence(userID uint64, answers []Answer) ([]Answer, sql.NullString, error) {
	ev := map[string][]string{}
	out := make([]Answer, len(answers))
	for i, a := range answers {
		if len(a.Evidence) > 0 {
			ev[a.Code] = a.Evidence
		}
		a.Evidence = nil
		out[i] = a
	}
	if len(ev) == 0 {
		return out, sql.NullString{}, nil
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return nil, sql.NullString{}, err
	}
	enc, err := s.sealText("questionnaire", userID, string(b))
	return out, enc, err
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
	stripped, evidence, err := s.sealEvidence(userID, Prefill(facts, civildate.InTehran(now)))
	if err != nil {
		return 0, err
	}
	answers, err := json.Marshal(stripped)
	if err != nil {
		return 0, err
	}
	ts := dbTime(now)
	id, err := s.q.CreateQuestionnaire(ctx, store.CreateQuestionnaireParams{UserID: userID, Purpose: purpose, Label: nullStr(label),
		Answers: answers, EvidenceEnc: evidence, CreatedAt: ts, UpdatedAt: ts})
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
	// The evidence never changes after the prefill: a sealed row keeps its seal untouched (so a missing key cannot
	// wipe it); a row from before 00093 gets its plain evidence sealed now (ErrStorage without a key — fail closed).
	var stripped []Answer
	evidence := r.EvidenceEnc
	if !r.EvidenceEnc.Valid {
		var err error
		if stripped, evidence, err = s.sealEvidence(r.UserID, answers); err != nil {
			return err
		}
	} else {
		stripped = make([]Answer, len(answers))
		for i, a := range answers {
			a.Evidence = nil
			stripped[i] = a
		}
	}
	a, err := json.Marshal(stripped)
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
	return s.q.UpdateQuestionnaire(ctx, store.UpdateQuestionnaireParams{Answers: a, EvidenceEnc: evidence, ExplanationEnc: expl, Label: r.Label,
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
	answers := s.Answers(r)
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
	answers := s.Answers(r)
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
	return s.saveQuestionnaire(ctx, r, s.Answers(r), r.ExplanationEnc, status, r.ConfirmedAt, tl, now)
}

// DeleteQuestionnaire removes one of hers.
func (s *Service) DeleteQuestionnaire(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteUserQuestionnaire(ctx, store.DeleteUserQuestionnaireParams{ID: id, UserID: userID})
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

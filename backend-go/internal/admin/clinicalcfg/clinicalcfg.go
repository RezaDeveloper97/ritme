// Package clinicalcfg is the admin API of the versioned clinical thresholds (bloom B-N9-11, D-88;
// docs/go-migration/admin-api.md «Clinical thresholds & config») — «آستانه‌ها و پیکربندی».
//
//   - Every key of internal/clinicalconfig with its version in force today, the next approved version (if any) and
//     the pending proposal. Version 1 is the code default and is never stored.
//   - A change is a proposal (value, effective date ≥ today, source, note) by an admin with the config capability
//     (manage). It becomes approved after two DISTINCT medical advisors approve it; the proposer can never approve.
//     One pending proposal per key. A proposal whose effective date passed before its second approval cannot be
//     approved (history would change) — it must be re-proposed.
//   - The engines read the approved version valid at their target date (clinicalconfig.At), so historic outputs are
//     reproducible. GET /clinical-config/resolve shows what the server reads for a key on a date.
//
// Every write is audited (clinical_config.*); values are thresholds, never user data.
package clinicalcfg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Statuses of a stored version.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// RequiredApprovals is how many distinct medical advisors approve a version.
const RequiredApprovals = 2

// Error codes.
const (
	CodeAdvisorRequired = "medical_advisor_required" // 403: only medical advisors approve / reject
	CodeSelfApproval    = "self_approval"            // 403: the proposer never approves their own proposal
	CodeAlreadyApproved = "already_approved"         // 409: this advisor already approved
	CodeNotPending      = "not_pending"              // 409
	CodePendingExists   = "pending_exists"           // 409: one pending proposal per key
	CodeEffectivePassed = "effective_date_passed"    // 422: re-propose with a future date
)

// Handlers serves the endpoints.
type Handlers struct {
	db     *sql.DB
	q      *store.Queries
	logger *slog.Logger
}

// NewHandlers wires the handlers.
func NewHandlers(db *sql.DB, logger *slog.Logger) *Handlers {
	return &Handlers{db: db, q: store.New(db), logger: logger}
}

// Loader reads the approved versions for clinicalconfig.Install.
func Loader(db *sql.DB) clinicalconfig.Loader {
	q := store.New(db)
	return func(ctx context.Context) ([]clinicalconfig.Version, error) {
		rows, err := q.ListApprovedClinicalConfigVersions(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]clinicalconfig.Version, 0, len(rows))
		for _, r := range rows {
			var v clinicalconfig.Value
			if json.Unmarshal([]byte(r.Value), &v) != nil {
				continue // never trust a broken row: the key keeps its previous version
			}
			out = append(out, clinicalconfig.Version{Key: r.ConfigKey, Version: int(r.Version), Value: v,
				EffectiveFrom: r.EffectiveFrom})
		}
		return out, nil
	}
}

// Routes registers the endpoints (paths under httpadmin.Prefix). The config capability gates them (view for GET,
// manage for writes); approve / reject additionally need the medical-advisor role.
func (h *Handlers) Routes(add func(method, path string, chain httpadmin.Chain), kit *httpadmin.Kit) {
	get, post := fiber.MethodGet, fiber.MethodPost
	add(get, "/clinical-config", kit.Admin(h.List))
	add(get, "/clinical-config/resolve", kit.Admin(h.Resolve))
	add(post, "/clinical-config/proposals/:id/approve", kit.Admin(h.Approve))
	add(post, "/clinical-config/proposals/:id/reject", kit.Admin(h.Reject))
	add(get, "/clinical-config/:key", kit.Admin(h.Show))
	add(post, "/clinical-config/:key/proposals", kit.Admin(h.Propose))
}

// ---------------------------------------------------------------------------
// JSON

type versionRow struct {
	store.ClinicalConfigVersion
	ProposerName sql.NullString
}

func approver(a store.ListClinicalConfigApprovalsRow) *jsonx.OrderedMap {
	return jsonx.Obj("id", a.AdminID, "name", httpadmin.NullString(a.AdminName), "at", httpadmin.Time(a.CreatedAt))
}

func valueJSON(def *clinicalconfig.Def, v clinicalconfig.Value) *jsonx.OrderedMap {
	out := jsonx.Obj()
	for _, f := range def.Fields {
		if x, ok := v[f.Name]; ok {
			out.Set(f.Name, x)
		}
	}
	return out
}

func defaultVersionJSON(def *clinicalconfig.Def) *jsonx.OrderedMap {
	return jsonx.Obj("id", nil, "version", 1, "value", valueJSON(def, def.DefaultValue()), "source", def.Source,
		"note", "", "effective_from", nil, "status", StatusApproved, "proposed_by", nil, "approvals", []any{},
		"required_approvals", 0, "decided_at", nil, "created_at", nil)
}

func storedVersionJSON(def *clinicalconfig.Def, r versionRow, approvals []store.ListClinicalConfigApprovalsRow) *jsonx.OrderedMap {
	var v clinicalconfig.Value
	_ = json.Unmarshal([]byte(r.Value), &v)
	aps := make([]any, 0, len(approvals))
	for _, a := range approvals {
		aps = append(aps, approver(a))
	}
	var proposer any
	if r.ProposedBy.Valid {
		proposer = jsonx.Obj("id", r.ProposedBy.Int64, "name", httpadmin.NullString(r.ProposerName))
	}
	return jsonx.Obj("id", r.ID, "version", r.Version, "value", valueJSON(def, v), "source", r.Source, "note", r.Note,
		"effective_from", r.EffectiveFrom.String(), "status", r.Status, "proposed_by", proposer, "approvals", aps,
		"required_approvals", RequiredApprovals, "decided_at", httpadmin.Time(r.DecidedAt),
		"created_at", httpadmin.Time(r.CreatedAt))
}

func defJSON(def *clinicalconfig.Def) *jsonx.OrderedMap {
	fields := make([]any, 0, len(def.Fields))
	for _, f := range def.Fields {
		fields = append(fields, jsonx.Obj("name", f.Name, "default", f.Default, "min", f.Min, "max", f.Max))
	}
	return jsonx.Obj("key", def.Key, "group", def.Group, "unit", def.Unit, "fields", fields)
}

// keyState splits a key's stored versions (newest version first) into the one in force today, the next approved
// one and the pending proposal.
func keyState(def *clinicalconfig.Def, rows []versionRow, byVersion map[uint64][]store.ListClinicalConfigApprovalsRow,
	today civildate.Date,
) (current, upcoming, pending any) {
	current = defaultVersionJSON(def)
	var cur, next *versionRow
	for i := range rows {
		r := &rows[i]
		switch {
		case r.Status == StatusPending && pending == nil:
			pending = storedVersionJSON(def, *r, byVersion[r.ID])
		case r.Status != StatusApproved:
		case !r.EffectiveFrom.After(today):
			if cur == nil || r.EffectiveFrom.After(cur.EffectiveFrom) ||
				(r.EffectiveFrom == cur.EffectiveFrom && r.Version > cur.Version) {
				cur = r
			}
		default:
			if next == nil || r.EffectiveFrom.Before(next.EffectiveFrom) {
				next = r
			}
		}
	}
	if cur != nil {
		current = storedVersionJSON(def, *cur, byVersion[cur.ID])
	}
	if next != nil {
		upcoming = storedVersionJSON(def, *next, byVersion[next.ID])
	}
	return current, upcoming, pending
}

func (h *Handlers) approvals(ctx context.Context) (map[uint64][]store.ListClinicalConfigApprovalsRow, error) {
	rows, err := h.q.ListClinicalConfigApprovals(ctx)
	if err != nil {
		return nil, err
	}
	out := map[uint64][]store.ListClinicalConfigApprovalsRow{}
	for _, r := range rows {
		out[r.VersionID] = append(out[r.VersionID], r)
	}
	return out, nil
}

func canApprove(c fiber.Ctx) bool {
	a := httpadmin.CurrentAdmin(c)
	return a != nil && a.Role == httpadmin.RoleMedicalAdvisor
}

func today(c fiber.Ctx) civildate.Date { return civildate.InTehran(httpadmin.Now(c)) }

// ---------------------------------------------------------------------------
// Reads

// List is GET /clinical-config?group=: {groups, keys:[{key, group, unit, fields, current, upcoming, pending}],
// can_approve, required_approvals, today}.
func (h *Handlers) List(c fiber.Ctx) error {
	group := c.Query("group")
	if group != "" && !slices.Contains(clinicalconfig.Groups, group) {
		return httpadmin.FieldError("group", "The selected group is invalid.")
	}
	rows, err := h.q.ListClinicalConfigVersions(c.Context())
	if err != nil {
		return err
	}
	byKey := map[string][]versionRow{}
	for _, r := range rows {
		byKey[r.ConfigKey] = append(byKey[r.ConfigKey], versionRow{store.ClinicalConfigVersion{
			ID: r.ID, ConfigKey: r.ConfigKey, Version: r.Version, Value: r.Value, Source: r.Source, Note: r.Note,
			EffectiveFrom: r.EffectiveFrom, Status: r.Status, ProposedBy: r.ProposedBy, DecidedBy: r.DecidedBy,
			DecidedAt: r.DecidedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}, r.ProposerName})
	}
	aps, err := h.approvals(c.Context())
	if err != nil {
		return err
	}
	day := today(c)
	keys := []any{}
	for i := range clinicalconfig.Defs {
		def := &clinicalconfig.Defs[i]
		if group != "" && def.Group != group {
			continue
		}
		cur, next, pend := keyState(def, byKey[def.Key], aps, day)
		keys = append(keys, defJSON(def).Set("current", cur).Set("upcoming", next).Set("pending", pend))
	}
	return httpadmin.OK(c, jsonx.Obj("groups", clinicalconfig.Groups, "keys", keys, "can_approve", canApprove(c),
		"required_approvals", RequiredApprovals, "today", day.String()))
}

func lookup(key string) (*clinicalconfig.Def, error) {
	def, ok := clinicalconfig.Lookup(key)
	if !ok {
		return nil, httpadmin.NotFound("Config key")
	}
	return def, nil
}

// Show is GET /clinical-config/:key: the key with every version (stored newest first, then version 1).
func (h *Handlers) Show(c fiber.Ctx) error {
	def, err := lookup(c.Params("key"))
	if err != nil {
		return err
	}
	rows, err := h.q.ListClinicalConfigVersionsOfKey(c.Context(), def.Key)
	if err != nil {
		return err
	}
	aps, err := h.approvals(c.Context())
	if err != nil {
		return err
	}
	vrs := make([]versionRow, 0, len(rows))
	versions := make([]any, 0, len(rows)+1)
	for _, r := range rows {
		vr := versionRow{store.ClinicalConfigVersion{
			ID: r.ID, ConfigKey: r.ConfigKey, Version: r.Version, Value: r.Value, Source: r.Source, Note: r.Note,
			EffectiveFrom: r.EffectiveFrom, Status: r.Status, ProposedBy: r.ProposedBy, DecidedBy: r.DecidedBy,
			DecidedAt: r.DecidedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}, r.ProposerName}
		vrs = append(vrs, vr)
		versions = append(versions, storedVersionJSON(def, vr, aps[r.ID]))
	}
	versions = append(versions, defaultVersionJSON(def))
	cur, next, pend := keyState(def, vrs, aps, today(c))
	return httpadmin.OK(c, jsonx.Obj("key", defJSON(def).Set("current", cur).Set("upcoming", next).Set("pending", pend),
		"versions", versions, "can_approve", canApprove(c), "required_approvals", RequiredApprovals))
}

// Resolve is GET /clinical-config/resolve?key=&date=: the version the server reads for key on date.
func (h *Handlers) Resolve(c fiber.Ctx) error {
	if _, err := httpadmin.Validate(c, validation.Rules{
		validation.F("key", "required|string|max:64"),
		validation.F("date", "required|date_format:Y-m-d"),
	}); err != nil {
		return err
	}
	def, err := lookup(c.Query("key"))
	if err != nil {
		return err
	}
	d, err := civildate.Parse(c.Query("date"))
	if err != nil {
		return httpadmin.FieldError("date", "The date does not match the format Y-m-d.")
	}
	v := clinicalconfig.At(def.Key, d)
	var eff any
	if !v.EffectiveFrom.IsZero() {
		eff = v.EffectiveFrom.String()
	}
	return httpadmin.OK(c, jsonx.Obj("key", def.Key, "date", d.String(), "version", v.Version,
		"value", valueJSON(def, v.Value), "effective_from", eff))
}

// ---------------------------------------------------------------------------
// Writes

// Propose is POST /clinical-config/:key/proposals {value:{field:number}, effective_from, source?, note?} → 201.
func (h *Handlers) Propose(c fiber.Ctx) error {
	def, err := lookup(c.Params("key"))
	if err != nil {
		return err
	}
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("value", "required|array"),
		validation.F("effective_from", "required|date_format:Y-m-d"),
		validation.F("source", "nullable|string|max:255"),
		validation.F("note", "nullable|string|max:500"),
	})
	if err != nil {
		return err
	}
	var body struct {
		Value clinicalconfig.Value `json:"value"`
	}
	if json.Unmarshal(c.Body(), &body) != nil {
		return httpadmin.FieldError("value", "The value must map every field to a number.")
	}
	if problem := def.Validate(body.Value); problem != "" {
		return httpadmin.FieldError("value", valueMessage(problem))
	}
	eff, err := civildate.Parse(httpadmin.String(data, "effective_from"))
	if err != nil {
		return httpadmin.FieldError("effective_from", "The effective from does not match the format Y-m-d.")
	}
	if eff.Before(today(c)) {
		return httpadmin.FieldError("effective_from", "The effective from must be today or a later date.")
	}
	source := strings.TrimSpace(httpadmin.String(data, "source"))
	if source == "" {
		source = def.Source
	}
	pending, err := h.q.CountPendingClinicalConfig(c.Context(), def.Key)
	if err != nil {
		return err
	}
	if pending > 0 {
		return httpadmin.Fail(fiber.StatusConflict, CodePendingExists, "This key already has a pending proposal.")
	}
	maxV, err := h.q.MaxClinicalConfigVersion(c.Context(), def.Key)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(body.Value)
	if err != nil {
		return err
	}
	a := httpadmin.CurrentAdmin(c)
	now := httpadmin.DBTime(httpadmin.Now(c))
	id, err := h.q.InsertClinicalConfigVersion(c.Context(), store.InsertClinicalConfigVersionParams{
		ConfigKey: def.Key, Version: uint32(maxV + 1), Value: string(raw), Source: source, //nolint:gosec // small
		Note: strings.TrimSpace(httpadmin.String(data, "note")), EffectiveFrom: eff,
		ProposedBy: sql.NullInt64{Int64: int64(a.ID), Valid: true}, Now: now, //nolint:gosec // ids fit int64
	})
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return httpadmin.Fail(fiber.StatusConflict, CodePendingExists, "This key already has a pending proposal.")
		}
		return err
	}
	httpadmin.Audit(c, h.logger, "clinical_config.propose", "clinical_config", uint64(id), //nolint:gosec // ids
		slog.String("key", def.Key), slog.Int64("version", maxV+1), slog.String("effective_from", eff.String()))
	out, err := h.versionJSON(c, def, uint64(id)) //nolint:gosec // ids
	if err != nil {
		return err
	}
	return httpadmin.Created(c, jsonx.Obj("proposal", out), "Proposal created.")
}

func valueMessage(problem string) string {
	switch {
	case strings.HasPrefix(problem, "range:"):
		return "The value of " + strings.TrimPrefix(problem, "range:") + " is out of the allowed range."
	case strings.HasPrefix(problem, "order:"):
		return "The value of " + strings.TrimPrefix(problem, "order:") + " must be lower than its upper bound."
	}
	return "The value must map every field to a number."
}

func (h *Handlers) versionJSON(c fiber.Ctx, def *clinicalconfig.Def, id uint64) (*jsonx.OrderedMap, error) {
	r, err := h.q.GetClinicalConfigVersion(c.Context(), id)
	if err != nil {
		return nil, err
	}
	aps, err := h.approvals(c.Context())
	if err != nil {
		return nil, err
	}
	var name sql.NullString
	if r.ProposedBy.Valid {
		if p, err := h.q.GetAdminByID(c.Context(), uint64(r.ProposedBy.Int64)); err == nil { //nolint:gosec // ids
			name = sql.NullString{String: p.Name, Valid: true}
		}
	}
	return storedVersionJSON(def, versionRow{r, name}, aps[id]), nil
}

// decision loads a pending version for a medical advisor who is not its proposer, inside tx.
func (h *Handlers) decision(c fiber.Ctx, q *store.Queries) (store.ClinicalConfigVersion, *clinicalconfig.Def, error) {
	var zero store.ClinicalConfigVersion
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return zero, nil, httpadmin.NotFound("Proposal")
	}
	r, err := q.GetClinicalConfigVersionForUpdate(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return zero, nil, httpadmin.NotFound("Proposal")
	}
	if err != nil {
		return zero, nil, err
	}
	def, ok := clinicalconfig.Lookup(r.ConfigKey)
	if !ok {
		return zero, nil, httpadmin.NotFound("Proposal")
	}
	if r.Status != StatusPending {
		return zero, nil, httpadmin.Fail(fiber.StatusConflict, CodeNotPending, "This proposal is no longer pending.")
	}
	return r, def, nil
}

func (h *Handlers) inTx(c fiber.Ctx, fn func(q *store.Queries) error) error {
	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(h.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// Approve is POST /clinical-config/proposals/:id/approve (medical advisors, never the proposer, once each). The
// second distinct approval makes the version approved and reloads the thresholds.
func (h *Handlers) Approve(c fiber.Ctx) error {
	if !canApprove(c) {
		return httpadmin.Fail(fiber.StatusForbidden, CodeAdvisorRequired, "Only a medical advisor can approve clinical thresholds.")
	}
	a := httpadmin.CurrentAdmin(c)
	var (
		r        store.ClinicalConfigVersion
		def      *clinicalconfig.Def
		approved bool
	)
	err := h.inTx(c, func(q *store.Queries) error {
		var err error
		if r, def, err = h.decision(c, q); err != nil {
			return err
		}
		if r.ProposedBy.Valid && uint64(r.ProposedBy.Int64) == a.ID { //nolint:gosec // ids
			return httpadmin.Fail(fiber.StatusForbidden, CodeSelfApproval, "The proposer cannot approve their own proposal.")
		}
		if r.EffectiveFrom.Before(today(c)) {
			return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeEffectivePassed,
				"The effective date has passed; propose the change again with a new date.")
		}
		all, err := q.ListClinicalConfigApprovals(c.Context())
		if err != nil {
			return err
		}
		n := 0
		for _, ap := range all {
			if ap.VersionID != r.ID {
				continue
			}
			if ap.AdminID == a.ID {
				return httpadmin.Fail(fiber.StatusConflict, CodeAlreadyApproved, "You already approved this proposal.")
			}
			n++
		}
		now := httpadmin.DBTime(httpadmin.Now(c))
		if err := q.InsertClinicalConfigApproval(c.Context(), store.InsertClinicalConfigApprovalParams{
			VersionID: r.ID, AdminID: a.ID, Now: now,
		}); err != nil {
			return err
		}
		if n+1 >= RequiredApprovals {
			approved = true
			_, err = q.DecideClinicalConfigVersion(c.Context(), store.DecideClinicalConfigVersionParams{
				Status: StatusApproved, DecidedBy: sql.NullInt64{Int64: int64(a.ID), Valid: true}, Now: now, ID: r.ID, //nolint:gosec // ids
			})
		}
		return err
	})
	if err != nil {
		return err
	}
	if approved {
		_ = clinicalconfig.Refresh(c.Context())
	}
	httpadmin.Audit(c, h.logger, "clinical_config.approve", "clinical_config", r.ID, slog.String("key", r.ConfigKey),
		slog.Int("version", int(r.Version)), slog.Bool("approved", approved))
	out, err := h.versionJSON(c, def, r.ID)
	if err != nil {
		return err
	}
	msg := "Approval recorded."
	if approved {
		msg = "Proposal approved."
	}
	return httpadmin.OK(c, jsonx.Obj("proposal", out), msg)
}

// Reject is POST /clinical-config/proposals/:id/reject: a medical advisor rejects, or the proposer withdraws.
func (h *Handlers) Reject(c fiber.Ctx) error {
	a := httpadmin.CurrentAdmin(c)
	var (
		r   store.ClinicalConfigVersion
		def *clinicalconfig.Def
	)
	err := h.inTx(c, func(q *store.Queries) error {
		var err error
		if r, def, err = h.decision(c, q); err != nil {
			return err
		}
		proposer := r.ProposedBy.Valid && uint64(r.ProposedBy.Int64) == a.ID //nolint:gosec // ids
		if !canApprove(c) && !proposer {
			return httpadmin.Fail(fiber.StatusForbidden, CodeAdvisorRequired, "Only a medical advisor can reject clinical thresholds.")
		}
		_, err = q.DecideClinicalConfigVersion(c.Context(), store.DecideClinicalConfigVersionParams{
			Status: StatusRejected, DecidedBy: sql.NullInt64{Int64: int64(a.ID), Valid: true}, //nolint:gosec // ids
			Now: httpadmin.DBTime(httpadmin.Now(c)), ID: r.ID,
		})
		return err
	})
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "clinical_config.reject", "clinical_config", r.ID, slog.String("key", r.ConfigKey),
		slog.Int("version", int(r.Version)))
	out, err := h.versionJSON(c, def, r.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("proposal", out), "Proposal rejected.")
}

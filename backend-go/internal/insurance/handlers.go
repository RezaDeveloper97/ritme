package insurance

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/insurance/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Handlers are the owner routes under /api/v1/insurance (locale, auth RequireUser, writes throttled). Every lookup
// is scoped by the caller's id: another user's row answers the same 404 as a missing one. Responses are no-store.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers.
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

func fail(status int, msgKey, code, locale string) error {
	return httpx.Fail(status, T("messages."+msgKey, locale), "error_code", code)
}

// mapErr turns a service error into its response.
func mapErr(err error, locale string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fail(fiber.StatusNotFound, "not_found", CodeNotFound, locale)
	case errors.Is(err, ErrNoPolicy):
		return fail(fiber.StatusNotFound, "no_policy", CodeNoPolicy, locale)
	case errors.Is(err, ErrStorage):
		return fail(fiber.StatusServiceUnavailable, "storage_unavailable", CodeStorage, locale)
	case errors.Is(err, ErrLocked):
		return fail(fiber.StatusConflict, "locked", CodeLocked, locale)
	case errors.Is(err, ErrIncomplete):
		return fail(fiber.StatusUnprocessableEntity, "incomplete", CodeIncomplete, locale)
	case errors.Is(err, ErrNotConfirmed):
		return fail(fiber.StatusConflict, "not_confirmed", CodeNotConfirmed, locale)
	case errors.Is(err, ErrTooMany):
		return fail(fiber.StatusConflict, "too_many", CodeTooMany, locale)
	case errors.Is(err, ErrEventLast):
		return fail(fiber.StatusConflict, "event_last", CodeEventLast, locale)
	case errors.Is(err, ErrReadOnly):
		return fail(fiber.StatusConflict, "read_only", CodeReadOnly, locale)
	case errors.Is(err, ErrRenewed):
		return fail(fiber.StatusConflict, "renewed", CodeRenewed, locale)
	}
	return err
}

func idParam(c fiber.Ctx, name, locale string) (uint64, error) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	if err != nil || id == 0 {
		return 0, mapErr(ErrNotFound, locale)
	}
	return id, nil
}

func noStore(c fiber.Ctx) { c.Set(fiber.HeaderCacheControl, "no-store") }

// ---------------------------------------------------------------------------------------------------------------
// Input helpers

func pick(body phpval.Map, keys ...string) phpval.Map {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	return data
}

func has(data any, key string) bool {
	v, ok := phpval.Get(data, key)
	return ok && v != nil
}

func text(data any, key string) string {
	v, ok := phpval.Get(data, key)
	if !ok || v == nil {
		return ""
	}
	return strings.Join(strings.Fields(phpval.ToString(v)), " ")
}

// longText keeps line breaks (notes) but trims the ends.
func longText(data any, key string) string {
	v, ok := phpval.Get(data, key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

func num(data any, key string) *int64 {
	if !has(data, key) {
		return nil
	}
	v, _ := phpval.Get(data, key)
	n := int64(phpval.ToFloat(v))
	return &n
}

func uid(data any, key string) uint64 {
	if n := num(data, key); n != nil && *n > 0 {
		return uint64(*n)
	}
	return 0
}

func date(data any, key string) civildate.NullDate {
	d, err := civildate.Parse(text(data, key))
	if err != nil {
		return civildate.NullDate{}
	}
	return civildate.NullDate{Date: d, Valid: true}
}

func elems(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case phpval.Map:
		out := make([]any, 0, x.Len())
		for _, k := range x.Keys() {
			e, _ := x.Get(k)
			out = append(out, e)
		}
		return out
	}
	return nil
}

func validate(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

func in(values []string) validation.Rule { return validation.In(values...) }

func maxAmount() string { return "max:" + strconv.FormatInt(MaxAmount, 10) }

// ---------------------------------------------------------------------------------------------------------------
// JSON

func isoOrNil(t time.Time, ok bool) any {
	if !ok {
		return nil
	}
	return jsonx.ISO8601(t)
}

func dateOrNil(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date.String()
}

// strOrNil is the value of a nullable column (nil for NULL).
func strOrNil(s driver.Valuer) any {
	v, _ := s.Value()
	return v
}

func ptrOrNil(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func coverageOf(c store.InsuranceCoverage) Coverage {
	cv := Coverage{Category: c.Category, Status: c.Status}
	if c.CapAmount.Valid {
		v := c.CapAmount.Int64
		cv.Cap = &v
	}
	if c.DeductiblePct.Valid {
		cv.DeductiblePct = int64(c.DeductiblePct.Int16)
	}
	if c.WaitingDays.Valid {
		cv.WaitingDays = int64(c.WaitingDays.Int16)
	}
	return cv
}

func (h *Handlers) policyJSON(pv PolicyView, claims []ClaimFact, today civildate.Date) *jsonx.OrderedMap {
	p := pv.Policy
	var insurer any
	name := p.InsurerName.String
	if p.InsurerID.Valid {
		insurer = jsonx.Obj("id", p.InsurerID.Int64, "name", p.CatalogInsurerName.String)
		name = p.CatalogInsurerName.String
	}
	members := make([]*jsonx.OrderedMap, 0, len(pv.Members))
	for _, m := range pv.Members {
		var child any
		if m.ChildID.Valid {
			child = m.ChildID.Int64
		}
		members = append(members, jsonx.Obj("id", m.ID, "kind", m.Kind, "child_id", child, "name", m.Name))
	}
	covs := make([]*jsonx.OrderedMap, 0, len(pv.Coverages))
	for _, c := range pv.Coverages {
		u := UsageOf(coverageOf(c), periodOfRow(p), claims, 0, today)
		covs = append(covs, jsonx.Obj("category", c.Category, "status", c.Status, "cap_amount", strOrNil(c.CapAmount),
			"deductible_pct", strOrNil(c.DeductiblePct), "waiting_days", strOrNil(c.WaitingDays), "note", strOrNil(c.Note),
			"used", u.Used, "remaining", ptrOrNil(u.Remaining), "waiting_ends_on", dateOrNil(u.WaitingEndsOn),
			"waiting_over", u.WaitingOver))
	}
	return jsonx.Obj("id", p.ID, "insurer", insurer, "insurer_name", name, "plan_name", strOrNil(p.PlanName),
		"number_last4", strOrNil(p.NumberLast4), "valid_from", dateOrNil(p.ValidFrom), "valid_to", dateOrNil(p.ValidTo),
		"claim_deadline_days", strOrNil(p.ClaimDeadlineDays), "iban_last4", strOrNil(p.IbanLast4),
		"not_covered_note", strOrNil(p.NotCoveredNote), "members", members, "coverages", covs,
		"updated_at", isoOrNil(p.UpdatedAt.Time, p.UpdatedAt.Valid), "status", pv.Status, "read_only", pv.ReadOnly(),
		"can_renew", pv.CanRenew)
}

// insurerNameOf is the display name: the catalog insurer's, else the one she typed.
func insurerNameOf(p PolicyRow) string {
	if p.InsurerID.Valid {
		return p.CatalogInsurerName.String
	}
	return p.InsurerName.String
}

// historyItemJSON is one row of the policy switcher.
func historyItemJSON(it HistoryItem, h History, claims int) *jsonx.OrderedMap {
	p := it.Policy
	return jsonx.Obj("id", p.ID, "insurer_name", insurerNameOf(p), "plan_name", strOrNil(p.PlanName),
		"number_last4", strOrNil(p.NumberLast4), "valid_from", dateOrNil(p.ValidFrom), "valid_to", dateOrNil(p.ValidTo),
		"status", it.Status, "read_only", it.Status == PolicyPast, "can_renew", p.ID == h.NewestID, "claims_count", claims)
}

func memberJSON(id int64, members []store.InsurancePolicyMember) any {
	for _, m := range members {
		if int64(m.ID) == id { //nolint:gosec // G115: DB ids
			return jsonx.Obj("id", m.ID, "kind", m.Kind, "name", m.Name)
		}
	}
	return nil
}

func submitBy(c store.InsuranceClaim, p PolicyRow) any {
	if !p.ClaimDeadlineDays.Valid {
		return nil
	}
	return c.ServiceDate.AddDays(int(p.ClaimDeadlineDays.Int16)).String()
}

func claimJSON(c store.InsuranceClaim, pv PolicyView) *jsonx.OrderedMap {
	var member, centre any
	if c.MemberID.Valid {
		member = memberJSON(c.MemberID.Int64, pv.Members)
	}
	if c.CentreID.Valid {
		centre = c.CentreID.Int64
	}
	return jsonx.Obj("id", c.ID, "policy_id", c.PolicyID, "read_only", pv.ReadOnly(), "title", strOrNil(c.Title),
		"cost_type", c.CostType, "category", CategoryOf(c.CostType),
		"member", member, "centre_id", centre, "centre_name", strOrNil(c.CentreName), "service_date", c.ServiceDate.String(),
		"amount", c.Amount, "over_tariff", c.OverTariff, "paid_amount", strOrNil(c.PaidAmount), "status", c.Status,
		"needs_action", c.Status == ClaimDocsRequested, "deadline_on", dateOrNil(c.DeadlineOn), "submit_by", submitBy(c, pv.Policy),
		"created_at", isoOrNil(c.CreatedAt.Time, c.CreatedAt.Valid))
}

// ---------------------------------------------------------------------------------------------------------------
// Catalog

// Insurers is GET /insurance/insurers: the active insurer catalog (names only).
func (h *Handlers) Insurers(c fiber.Ctx) error {
	if _, err := user(c); err != nil {
		return err
	}
	rows, err := h.svc.Insurers(c)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, i := range rows {
		items = append(items, jsonx.Obj("id", i.ID, "code", i.Code, "name", i.Name, "website", strOrNil(i.Website),
			"phone", strOrNil(i.Phone)))
	}
	return httpx.OK(c, jsonx.Obj("insurers", items))
}

// Centres is GET /insurance/centres?type=&q=: the contracted centres for her insurer. No location is sent; the
// device computes distances from lat/lng.
func (h *Handlers) Centres(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	typ := c.Query("type")
	if !contains(CentreTypes, typ) {
		typ = ""
	}
	q := strings.TrimSpace(c.Query("q"))
	if len([]rune(q)) > 80 {
		q = string([]rune(q)[:80])
	}
	rows, err := h.svc.Centres(c, userID, typ, q, civildate.InTehran(h.now(c)))
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonx.Obj("id", r.ID, "name", r.Name, "type", r.Type, "services", strOrNil(r.Services),
			"city", strOrNil(r.City), "address", strOrNil(r.Address), "phone", strOrNil(r.Phone), "lat", strOrNil(r.Lat),
			"lng", strOrNil(r.Lng), "is_direct", r.IsDirect, "insurer_name", strOrNil(r.InsurerName)))
	}
	return httpx.OK(c, jsonx.Obj("centres", items, "types", jsonx.List(CentreTypes)))
}

// ---------------------------------------------------------------------------------------------------------------
// Policy

// policyResponse answers the policy state of one of her policies (id 0 = the current one; none → policy null).
func (h *Handlers) policyResponse(c fiber.Ctx, userID, id uint64, msg string, created bool) error {
	today := civildate.InTehran(h.now(c))
	var pv PolicyView
	var err error
	if id == 0 {
		pv, err = h.svc.Policy(c, userID, today)
	} else {
		pv, err = h.svc.PolicyByID(c, userID, id, today)
	}
	if errors.Is(err, ErrNoPolicy) {
		return h.emptyPolicy(c, userID)
	}
	if err != nil {
		return mapErr(err, i18n.Locale(c))
	}
	claims, err := h.svc.PolicyClaims(c, userID, pv.Policy.ID)
	if err != nil {
		return err
	}
	children, err := h.svc.q.ListUserChildren(c, userID)
	if err != nil {
		return err
	}
	needs := 0
	for _, cl := range claims {
		if cl.Status == ClaimDocsRequested {
			needs++
		}
	}
	body := jsonx.Obj("policy", h.policyJSON(pv, Facts(claims), today), "children", childrenJSON(children),
		"needs_action", needs, "options", options())
	noStore(c)
	if created {
		return httpx.Created(c, body, msg)
	}
	if msg != "" {
		return httpx.OK(c, body, msg)
	}
	return httpx.OK(c, body)
}

func childrenJSON(rows []store.ListUserChildrenRow) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, ch := range rows {
		out = append(out, jsonx.Obj("id", ch.ID, "name", ch.Name))
	}
	return out
}

func options() *jsonx.OrderedMap {
	return jsonx.Obj("categories", jsonx.List(Categories), "coverage_statuses", jsonx.List(CoverageStatuses),
		"member_kinds", jsonx.List(MemberKinds), "cost_types", jsonx.List(CostTypes), "claim_statuses", jsonx.List(ClaimStatuses),
		"document_labels", jsonx.List(DocumentLabels))
}

func (h *Handlers) emptyPolicy(c fiber.Ctx, userID uint64) error {
	children, err := h.svc.q.ListUserChildren(c, userID)
	if err != nil {
		return err
	}
	noStore(c)
	return httpx.OK(c, jsonx.Obj("policy", nil, "children", childrenJSON(children), "needs_action", 0, "options", options()))
}

// Policy is GET /insurance/policy: her current policy with members, coverage usage of its period («پوشش باقی‌مانده»)
// and the needs-action count of its claims, or policy null (the client shows the setup form and the «how it works»
// link). The home reads this.
func (h *Handlers) Policy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	return h.policyResponse(c, userID, 0, "", false)
}

// Policies is GET /insurance/policies: her policy history (newest period first) for the switcher, with the current
// policy's id.
func (h *Handlers) Policies(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	hist, err := h.svc.History(c, userID, civildate.InTehran(h.now(c)))
	if err != nil {
		return err
	}
	claims, err := h.svc.Claims(c, userID)
	if err != nil {
		return err
	}
	counts := map[uint64]int{}
	for _, cl := range claims {
		counts[cl.PolicyID]++
	}
	items := make([]*jsonx.OrderedMap, 0, len(hist.Items))
	for _, it := range hist.Items {
		items = append(items, historyItemJSON(it, hist, counts[it.Policy.ID]))
	}
	var current any
	if hist.CurrentID != 0 {
		current = hist.CurrentID
	}
	noStore(c)
	return httpx.OK(c, jsonx.Obj("policies", items, "current_id", current))
}

// ShowPolicy is GET /insurance/policies/{id}: one of her policies in the GET /policy shape (a past one is read_only).
func (h *Handlers) ShowPolicy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	id, err := idParam(c, "id", i18n.Locale(c))
	if err != nil {
		return err
	}
	return h.policyResponse(c, userID, id, "", false)
}

func policyRules() validation.Rules {
	return validation.Rules{
		validation.F("insurer_id", "nullable", "integer", "min:1"),
		validation.F("insurer_name", "nullable", "required_without:insurer_id", "string", "max:120"),
		validation.F("plan_name", "nullable", "string", "max:120"),
		validation.F("policy_number", "nullable", "string", "max:40"),
		validation.F("iban", "nullable", "string", "max:40"),
		validation.F("valid_from", "nullable", "date_format:Y-m-d"),
		validation.F("valid_to", "nullable", "date_format:Y-m-d"),
		validation.F("claim_deadline_days", "nullable", "integer", "min:0", "max:"+strconv.Itoa(MaxDeadlineDays)),
		validation.F("not_covered_note", "nullable", "string", "max:"+strconv.Itoa(MaxNote)),
		validation.F("members", "nullable", "array", "max:"+strconv.Itoa(MaxMembers)),
		validation.F("members.*", "required", "array"),
		validation.F("members.*.id", "nullable", "integer", "min:1"),
		validation.F("members.*.kind", "required", in(MemberKinds)),
		validation.F("members.*.child_id", "nullable", "integer", "min:1"),
		validation.F("members.*.name", "required", "string", "max:64"),
		validation.F("coverages", "nullable", "array", "max:"+strconv.Itoa(MaxCoverages)),
		validation.F("coverages.*", "required", "array"),
		validation.F("coverages.*.category", "required", in(Categories)),
		validation.F("coverages.*.status", "nullable", in(CoverageStatuses)),
		validation.F("coverages.*.cap_amount", "nullable", "integer", "min:0", maxAmount()),
		validation.F("coverages.*.deductible_pct", "nullable", "integer", "min:0", "max:100"),
		validation.F("coverages.*.waiting_days", "nullable", "integer", "min:0", "max:"+strconv.Itoa(MaxWaitingDays)),
		validation.F("coverages.*.note", "nullable", "string", "max:160"),
	}
}

func (h *Handlers) parsePolicy(c fiber.Ctx, userID uint64, locale string, now time.Time) (PolicyInput, error) {
	body := validation.Input(c)
	data := pick(body, "insurer_id", "insurer_name", "plan_name", "policy_number", "iban", "valid_from", "valid_to",
		"claim_deadline_days", "not_covered_note", "members", "coverages")
	if err := validate(data, policyRules(), locale, now); err != nil {
		return PolicyInput{}, err
	}
	in := PolicyInput{InsurerID: uid(data, "insurer_id"), InsurerName: text(data, "insurer_name"), PlanName: text(data, "plan_name"),
		ValidFrom: date(data, "valid_from"), ValidTo: date(data, "valid_to"), DeadlineDays: num(data, "claim_deadline_days"),
		NotCovered: longText(data, "not_covered_note")}
	if in.ValidFrom.Valid && in.ValidTo.Valid && in.ValidTo.Date.Before(in.ValidFrom.Date) {
		return in, fieldFail(locale, "valid_to", "valid_to")
	}
	if _, ok := data.Get("policy_number"); ok {
		in.NumberLast4 = Last4(text(data, "policy_number"))
	} else {
		in.KeepNumber = true
	}
	if _, ok := data.Get("iban"); ok {
		in.IBANLast4 = Last4(text(data, "iban"))
	} else {
		in.KeepIBAN = true
	}
	if in.InsurerID != 0 {
		ok, err := h.svc.InsurerUsable(c, in.InsurerID)
		if err != nil {
			return in, err
		}
		if !ok {
			return in, fieldFail(locale, "insurer_id", "insurer")
		}
		in.InsurerName = ""
	}
	members, _ := data.Get("members")
	selves := 0
	for i, m := range elems(members) {
		mi := MemberInput{ID: uid(m, "id"), Kind: text(m, "kind"), ChildID: uid(m, "child_id"), Name: text(m, "name")}
		if mi.Kind == MemberSelf {
			selves++
			if selves > 1 {
				return in, fieldFail(locale, "members."+strconv.Itoa(i)+".kind", "one_self")
			}
		}
		if mi.Kind != MemberChild {
			mi.ChildID = 0
		}
		if mi.ChildID != 0 {
			ok, err := h.svc.OwnsChild(c, userID, mi.ChildID)
			if err != nil {
				return in, err
			}
			if !ok {
				return in, fieldFail(locale, "members."+strconv.Itoa(i)+".child_id", "child")
			}
		}
		in.Members = append(in.Members, mi)
	}
	covs, sent := data.Get("coverages")
	in.CoveragesSent = sent
	seen := map[string]bool{}
	for i, cv := range elems(covs) {
		ci := CoverageInput{Category: text(cv, "category"), Status: text(cv, "status"), Cap: num(cv, "cap_amount"),
			Deductible: num(cv, "deductible_pct"), WaitingDays: num(cv, "waiting_days"), Note: text(cv, "note")}
		if seen[ci.Category] {
			return in, fieldFail(locale, "coverages."+strconv.Itoa(i)+".category", "distinct")
		}
		seen[ci.Category] = true
		if ci.Status == "" {
			ci.Status = CoverageCovered
		}
		in.Coverages = append(in.Coverages, ci)
	}
	return in, nil
}

// SavePolicy is PUT /insurance/policy: creates (201) or replaces (200) her policy. The policy number and IBAN keep
// only their last four characters; omitting them keeps the stored ones, null clears them.
func (h *Handlers) SavePolicy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := h.parsePolicy(c, userID, locale, now)
	if err != nil {
		return err
	}
	created, id, err := h.svc.SavePolicy(c, userID, in, now)
	if err != nil {
		return mapErr(err, locale)
	}
	return h.policyResponse(c, userID, id, T("messages.policy_saved", locale), created)
}

// StorePolicy is POST /insurance/policies → 201: adds a policy to her history (same body as PUT /policy).
func (h *Handlers) StorePolicy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := h.parsePolicy(c, userID, locale, now)
	if err != nil {
		return err
	}
	id, err := h.svc.CreatePolicy(c, userID, in, now)
	if err != nil {
		return mapErr(err, locale)
	}
	return h.policyResponse(c, userID, id, T("messages.policy_saved", locale), true)
}

// UpdatePolicy is PUT /insurance/policies/{id}: replaces one of her policies (409 insurance_policy_read_only for a
// past one).
func (h *Handlers) UpdatePolicy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	pv, err := h.svc.PolicyByID(c, userID, id, civildate.InTehran(now))
	if err != nil {
		return mapErr(err, locale)
	}
	if pv.ReadOnly() {
		return mapErr(ErrReadOnly, locale)
	}
	in, err := h.parsePolicy(c, userID, locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UpdatePolicy(c, userID, id, in, now); err != nil {
		return mapErr(err, locale)
	}
	return h.policyResponse(c, userID, id, T("messages.policy_saved", locale), false)
}

// RenewPolicy is POST /insurance/policies/{id}/renew {valid_from?, valid_to?} → 201: the next policy from her newest
// one (members and coverages copied as a starting point; default period = the day after the old one ends, same
// length). The renewed policy keeps its claims and turns past (read-only) once the new period starts. 409
// insurance_policy_renewed for an older policy.
func (h *Handlers) RenewPolicy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if _, err := h.svc.PolicyByID(c, userID, id, civildate.InTehran(now)); err != nil {
		return mapErr(err, locale)
	}
	data := pick(validation.Input(c), "valid_from", "valid_to")
	if err := validate(data, validation.Rules{
		validation.F("valid_from", "nullable", "date_format:Y-m-d"),
		validation.F("valid_to", "nullable", "date_format:Y-m-d"),
	}, locale, now); err != nil {
		return err
	}
	in := RenewInput{ValidFrom: date(data, "valid_from"), ValidTo: date(data, "valid_to")}
	if in.ValidFrom.Valid && in.ValidTo.Valid && in.ValidTo.Date.Before(in.ValidFrom.Date) {
		return fieldFail(locale, "valid_to", "valid_to")
	}
	nid, err := h.svc.Renew(c, userID, id, in, now)
	if err != nil {
		return mapErr(err, locale)
	}
	return h.policyResponse(c, userID, nid, T("messages.policy_renewed", locale), true)
}

// DestroyPolicy is DELETE /insurance/policies/{id}: removes one of her policies (past ones too) with its claims.
func (h *Handlers) DestroyPolicy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if err := h.svc.DeletePolicy(c, userID, id, civildate.InTehran(h.now(c))); err != nil {
		return mapErr(err, locale)
	}
	return httpx.OK(c, nil, T("messages.policy_deleted", locale))
}

// DeletePolicy is DELETE /insurance/policy: removes her current policy and all its claims.
func (h *Handlers) DeletePolicy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.DeletePolicy(c, userID, 0, civildate.InTehran(h.now(c))); err != nil {
		return mapErr(err, locale)
	}
	return httpx.OK(c, nil, T("messages.policy_deleted", locale))
}

// ---------------------------------------------------------------------------------------------------------------
// Claims

// Claims is GET /insurance/claims?policy_id=&status=&q=&member_id=&from=&to=: the claims of one of her policies
// (default the current one; newest day first) with the totals of the window (default: that policy's period) —
// requested, paid, count and needs-action.
func (h *Handlers) Claims(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, today := i18n.Locale(c), civildate.InTehran(h.now(c))
	var pv PolicyView
	if raw := c.Query("policy_id"); raw != "" {
		id, perr := strconv.ParseUint(raw, 10, 64)
		if perr != nil || id == 0 {
			return mapErr(ErrNotFound, locale)
		}
		pv, err = h.svc.PolicyByID(c, userID, id, today)
	} else {
		pv, err = h.svc.Policy(c, userID, today)
	}
	if err != nil {
		return mapErr(err, locale)
	}
	rows, err := h.svc.PolicyClaims(c, userID, pv.Policy.ID)
	if err != nil {
		return err
	}
	window := periodOfRow(pv.Policy)
	if d, err := civildate.Parse(c.Query("from")); err == nil {
		window.From = civildate.NullDate{Date: d, Valid: true}
	}
	if d, err := civildate.Parse(c.Query("to")); err == nil {
		window.To = civildate.NullDate{Date: d, Valid: true}
	}
	filter := c.Query("status")
	if !contains(ClaimFilters, filter) {
		filter = ""
	}
	member, _ := strconv.ParseInt(c.Query("member_id"), 10, 64)
	q := c.Query("q")
	var requested, paid int64
	count, needs := 0, 0
	items := []*jsonx.OrderedMap{}
	for _, r := range rows {
		if !window.Contains(r.ServiceDate) {
			continue
		}
		count++
		requested += int64(r.Amount) //nolint:gosec // G115: ≤ MaxAmount
		if r.PaidAmount.Valid {
			paid += r.PaidAmount.Int64
		}
		if r.Status == ClaimDocsRequested {
			needs++
		}
		if !MatchesFilter(r.Status, filter) || (member > 0 && r.MemberID.Int64 != member) ||
			!MatchesQuery(q, r.Title.String, r.CentreName.String) {
			continue
		}
		items = append(items, claimJSON(r, pv))
	}
	noStore(c)
	return httpx.OK(c, jsonx.Obj("policy", jsonx.Obj("id", pv.Policy.ID, "status", pv.Status, "read_only", pv.ReadOnly()),
		"claims", items,
		"totals", jsonx.Obj("requested", requested, "paid", paid, "count", count, "needs_action", needs),
		"window", jsonx.Obj("from", dateOrNil(window.From), "to", dateOrNil(window.To)),
		"filters", jsonx.Obj("status", filter, "q", q, "member_id", member)))
}

func claimRules() validation.Rules {
	return validation.Rules{
		validation.F("member_id", "nullable", "integer", "min:1"),
		validation.F("cost_type", "required", in(CostTypes)),
		validation.F("title", "nullable", "string", "max:120"),
		validation.F("centre_id", "nullable", "integer", "min:1"),
		validation.F("centre_name", "nullable", "string", "max:160"),
		validation.F("service_date", "required", "date_format:Y-m-d"),
		validation.F("amount", "required", "integer", "min:1", maxAmount()),
		validation.F("over_tariff", "nullable", "integer", "min:0", maxAmount()),
		validation.F("note", "nullable", "string", "max:"+strconv.Itoa(MaxNote)),
		validation.F("documents", "nullable", "array", "max:"+strconv.Itoa(MaxDocsPerClaim)),
		validation.F("documents.*", "required", "array"),
		validation.F("documents.*.id", "required", "integer", "min:1"),
		validation.F("documents.*.label", "nullable", in(DocumentLabels)),
	}
}

func (h *Handlers) parseClaim(c fiber.Ctx, userID uint64, pv PolicyView, locale string, now time.Time) (ClaimInput, error) {
	data := pick(validation.Input(c), "member_id", "cost_type", "title", "centre_id", "centre_name", "service_date", "amount",
		"over_tariff", "note", "documents")
	if err := validate(data, claimRules(), locale, now); err != nil {
		return ClaimInput{}, err
	}
	in := ClaimInput{MemberID: uid(data, "member_id"), CostType: text(data, "cost_type"), Title: text(data, "title"),
		CentreID: uid(data, "centre_id"), CentreName: text(data, "centre_name"), Note: longText(data, "note")}
	in.Amount = *num(data, "amount")
	if v := num(data, "over_tariff"); v != nil {
		in.OverTariff = min(*v, in.Amount)
	}
	d := date(data, "service_date")
	if !d.Valid || d.Date.After(civildate.InTehran(now)) {
		return in, fieldFail(locale, "service_date", "not_future")
	}
	in.Date = d.Date
	docs, sent := data.Get("documents")
	in.DocsSent = sent
	seen := map[uint64]bool{}
	for _, x := range elems(docs) {
		id := uid(x, "id")
		if !seen[id] {
			seen[id] = true
			in.Documents = append(in.Documents, DocumentRef{ID: id, Label: text(x, "label")})
		}
	}
	field, err := h.svc.CheckClaimRefs(c, userID, pv, in)
	if err != nil {
		return in, err
	}
	if field != "" {
		return in, fieldFail(locale, field, "reference")
	}
	if in.CentreID != 0 { // the catalog name is the display name of a catalog centre
		ct, err := h.svc.q.GetActiveCentre(c, in.CentreID)
		if err != nil {
			return in, err
		}
		in.CentreName = ct.Name
	}
	return in, nil
}

func (h *Handlers) claimResponse(c fiber.Ctx, userID, id uint64, status int, msg string) error {
	locale, today := i18n.Locale(c), civildate.InTehran(h.now(c))
	cl, pv, err := h.svc.ClaimWithPolicy(c, userID, id, today)
	if err != nil {
		return mapErr(err, locale)
	}
	all, err := h.svc.PolicyClaims(c, userID, pv.Policy.ID)
	if err != nil {
		return err
	}
	d, err := h.svc.Detail(c, userID, cl)
	if err != nil {
		return err
	}
	o := claimJSON(cl, pv)
	o.Set("note", h.svc.ClaimNote(cl))

	var cov *Coverage
	for _, x := range pv.Coverages {
		if x.Category == CategoryOf(cl.CostType) {
			cv := coverageOf(x)
			cov = &cv
		}
	}
	var remaining *int64
	if cov != nil {
		remaining = UsageOf(*cov, periodOfRow(pv.Policy), Facts(all), cl.ID, today).Remaining
	}
	p, known := ExpectedPayable(int64(cl.Amount), int64(cl.OverTariff), cov, remaining) //nolint:gosec // G115: ≤ MaxAmount
	o.Set("payable", jsonx.Obj("known", known, "invoice", p.Invoice, "over_tariff", p.OverTariff, "deductible_pct", p.Pct,
		"deductible", p.Deductible, "payable", p.Payable, "user_share", p.UserShare, "capped", p.Capped))

	events := make([]*jsonx.OrderedMap, 0, len(d.Events))
	for _, e := range d.Events {
		events = append(events, jsonx.Obj("id", e.ID, "status", e.Status, "occurred_on", e.OccurredOn.String(),
			"deadline_on", dateOrNil(e.DeadlineOn), "amount", strOrNil(e.Amount), "note", h.svc.EventNote(userID, e)))
	}
	o.Set("events", events)
	docs := make([]*jsonx.OrderedMap, 0, len(d.Documents))
	for _, x := range d.Documents {
		docs = append(docs, jsonx.Obj("document_id", x.DocumentID, "label", strOrNil(x.Label), "kind", x.DocumentKind,
			"title", strOrNil(x.DocumentTitle), "document_date", dateOrNil(x.DocumentDate)))
	}
	o.Set("documents", docs)
	noStore(c)
	switch {
	case status == fiber.StatusCreated:
		return httpx.Created(c, o, msg)
	case msg != "":
		return httpx.OK(c, o, msg)
	}
	return httpx.OK(c, o)
}

// StoreClaim is POST /insurance/claims → 201: a claim on her current policy (404 insurance_no_policy without one).
func (h *Handlers) StoreClaim(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	pv, err := h.svc.Policy(c, userID, civildate.InTehran(now))
	if err != nil {
		return mapErr(err, locale)
	}
	in, err := h.parseClaim(c, userID, pv, locale, now)
	if err != nil {
		return err
	}
	id, err := h.svc.CreateClaim(c, userID, pv.Policy.ID, in, now)
	if err != nil {
		return mapErr(err, locale)
	}
	return h.claimResponse(c, userID, id, fiber.StatusCreated, T("messages.claim_saved", locale))
}

// ShowClaim is GET /insurance/claims/{id}: the claim with its payable breakdown, timeline and documents.
func (h *Handlers) ShowClaim(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	id, err := idParam(c, "id", i18n.Locale(c))
	if err != nil {
		return err
	}
	return h.claimResponse(c, userID, id, fiber.StatusOK, "")
}

// UpdateClaim is PUT /insurance/claims/{id} (full replace; documents replaced when sent).
func (h *Handlers) UpdateClaim(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	_, pv, err := h.svc.WritableClaim(c, userID, id, civildate.InTehran(now))
	if err != nil {
		return mapErr(err, locale)
	}
	in, err := h.parseClaim(c, userID, pv, locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UpdateClaim(c, userID, id, in, now); err != nil {
		return mapErr(err, locale)
	}
	return h.claimResponse(c, userID, id, fiber.StatusOK, T("messages.claim_saved", locale))
}

// DestroyClaim is DELETE /insurance/claims/{id}.
func (h *Handlers) DestroyClaim(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if _, _, err := h.svc.WritableClaim(c, userID, id, civildate.InTehran(h.now(c))); err != nil {
		return mapErr(err, locale)
	}
	if err := h.svc.DeleteClaim(c, userID, id); err != nil {
		return mapErr(err, locale)
	}
	return httpx.OK(c, nil, T("messages.claim_deleted", locale))
}

// StoreEvent is POST /insurance/claims/{id}/events {status, occurred_on, deadline_on?, amount?, note?}: a step of the
// timeline she maintains («درخواست مدرک» with its deadline, «پرداخت» with the paid amount, …).
func (h *Handlers) StoreEvent(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if _, _, err := h.svc.WritableClaim(c, userID, id, civildate.InTehran(now)); err != nil {
		return mapErr(err, locale)
	}
	data := pick(validation.Input(c), "status", "occurred_on", "deadline_on", "amount", "note")
	if err := validate(data, validation.Rules{
		validation.F("status", "required", in(ClaimStatuses)),
		validation.F("occurred_on", "required", "date_format:Y-m-d"),
		validation.F("deadline_on", "nullable", "date_format:Y-m-d"),
		validation.F("amount", "nullable", "required_if:status,paid", "integer", "min:0", maxAmount()),
		validation.F("note", "nullable", "string", "max:"+strconv.Itoa(MaxNote)),
	}, locale, now); err != nil {
		return err
	}
	ev := EventInput{Status: text(data, "status"), On: date(data, "occurred_on").Date, Note: longText(data, "note")}
	if ev.On.After(civildate.InTehran(now)) {
		return fieldFail(locale, "occurred_on", "not_future")
	}
	if ev.Status == ClaimDocsRequested {
		ev.Deadline = date(data, "deadline_on")
		if ev.Deadline.Valid && ev.Deadline.Date.Before(ev.On) {
			return fieldFail(locale, "deadline_on", "deadline")
		}
	}
	if ev.Status == ClaimPaid {
		ev.Amount = num(data, "amount")
	}
	if err := h.svc.AddEvent(c, userID, id, ev, now); err != nil {
		return mapErr(err, locale)
	}
	return h.claimResponse(c, userID, id, fiber.StatusCreated, T("messages.event_saved", locale))
}

// DestroyEvent is DELETE /insurance/claims/{id}/events/{eventId} (409 for the only step left).
func (h *Handlers) DestroyEvent(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	eid, err := idParam(c, "eventId", locale)
	if err != nil {
		return err
	}
	if _, _, err := h.svc.WritableClaim(c, userID, id, civildate.InTehran(now)); err != nil {
		return mapErr(err, locale)
	}
	if err := h.svc.DeleteEvent(c, userID, id, eid, now); err != nil {
		return mapErr(err, locale)
	}
	return h.claimResponse(c, userID, id, fiber.StatusOK, T("messages.event_deleted", locale))
}

// AttachDocument is POST /insurance/claims/{id}/documents {document_id, label?}: links one of her record documents
// (the «انتخاب از پرونده سلامت» of a missing document). Another user's document answers 404.
func (h *Handlers) AttachDocument(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if _, _, err := h.svc.WritableClaim(c, userID, id, civildate.InTehran(now)); err != nil {
		return mapErr(err, locale)
	}
	data := pick(validation.Input(c), "document_id", "label")
	if err := validate(data, validation.Rules{
		validation.F("document_id", "required", "integer", "min:1"),
		validation.F("label", "nullable", in(DocumentLabels)),
	}, locale, now); err != nil {
		return err
	}
	if err := h.svc.AttachDocument(c, userID, id, DocumentRef{ID: uid(data, "document_id"), Label: text(data, "label")}, now); err != nil {
		return mapErr(err, locale)
	}
	return h.claimResponse(c, userID, id, fiber.StatusOK, T("messages.document_attached", locale))
}

// DetachDocument is DELETE /insurance/claims/{id}/documents/{documentId} (the record document stays).
func (h *Handlers) DetachDocument(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	did, err := idParam(c, "documentId", locale)
	if err != nil {
		return err
	}
	if _, _, err := h.svc.WritableClaim(c, userID, id, civildate.InTehran(h.now(c))); err != nil {
		return mapErr(err, locale)
	}
	if err := h.svc.DetachDocument(c, userID, id, did); err != nil {
		return mapErr(err, locale)
	}
	return h.claimResponse(c, userID, id, fiber.StatusOK, T("messages.document_detached", locale))
}

// ---------------------------------------------------------------------------------------------------------------
// Questionnaires

func (h *Handlers) questionnaireJSON(r store.InsuranceQuestionnaire, details bool) *jsonx.OrderedMap {
	answers := h.svc.Answers(r)
	prefilled, answered := 0, 0
	qs := make([]*jsonx.OrderedMap, 0, len(answers))
	for _, a := range answers {
		if a.FromRecord() {
			prefilled++
		}
		if a.Answer != nil {
			answered++
		}
		var ans, rec any
		if a.Answer != nil {
			ans = *a.Answer
		}
		if a.RecordAnswer != nil {
			rec = *a.RecordAnswer
		}
		qs = append(qs, jsonx.Obj("code", a.Code, "section", SectionOf(a.Code), "answer", ans, "record_answer", rec,
			"from_record", a.FromRecord(), "evidence", jsonx.List(a.Evidence)))
	}
	o := jsonx.Obj("id", r.ID, "purpose", r.Purpose, "label", strOrNil(r.Label), "status", r.Status,
		"prefilled_count", prefilled, "answered_count", answered, "question_count", len(Questions),
		"complete", Complete(answers), "confirmed_at", isoOrNil(r.ConfirmedAt.Time, r.ConfirmedAt.Valid),
		"timeline", DecodeTimeline(r.Timeline.V), "created_at", isoOrNil(r.CreatedAt.Time, r.CreatedAt.Valid),
		"updated_at", isoOrNil(r.UpdatedAt.Time, r.UpdatedAt.Valid))
	if details {
		o.Set("sections", jsonx.List(Sections))
		o.Set("questions", qs)
		o.Set("explanation", h.svc.Explanation(r))
	}
	return o
}

// Questionnaires is GET /insurance/questionnaires: her questionnaires (status page), newest first.
func (h *Handlers) Questionnaires(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	rows, err := h.svc.Questionnaires(c, userID)
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, h.questionnaireJSON(r, false))
	}
	noStore(c)
	return httpx.OK(c, jsonx.Obj("questionnaires", items, "steps", jsonx.List(QuestionnaireSteps)))
}

func (h *Handlers) questionnaireResponse(c fiber.Ctx, userID, id uint64, status int, msg string) error {
	locale := i18n.Locale(c)
	r, err := h.svc.Questionnaire(c, userID, id)
	if err != nil {
		return mapErr(err, locale)
	}
	noStore(c)
	switch {
	case status == fiber.StatusCreated:
		return httpx.Created(c, h.questionnaireJSON(r, true), msg)
	case msg != "":
		return httpx.OK(c, h.questionnaireJSON(r, true), msg)
	}
	return httpx.OK(c, h.questionnaireJSON(r, true))
}

// StoreQuestionnaire is POST /insurance/questionnaires {purpose, label?} → 201: a draft prefilled from her record;
// she reviews every answer before confirming. Nothing is sent anywhere.
func (h *Handlers) StoreQuestionnaire(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	data := pick(validation.Input(c), "purpose", "label")
	if err := validate(data, validation.Rules{
		validation.F("purpose", "required", in(Purposes)),
		validation.F("label", "nullable", "string", "max:80"),
	}, locale, now); err != nil {
		return err
	}
	id, err := h.svc.CreateQuestionnaire(c, userID, text(data, "purpose"), text(data, "label"), now)
	if err != nil {
		return mapErr(err, locale)
	}
	return h.questionnaireResponse(c, userID, id, fiber.StatusCreated, T("messages.questionnaire_created", locale))
}

// ShowQuestionnaire is GET /insurance/questionnaires/{id} (also the PDF source drawn on the device).
func (h *Handlers) ShowQuestionnaire(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	id, err := idParam(c, "id", i18n.Locale(c))
	if err != nil {
		return err
	}
	return h.questionnaireResponse(c, userID, id, fiber.StatusOK, "")
}

// UpdateQuestionnaire is PUT /insurance/questionnaires/{id} {answers: {code: yes|no|null}, explanation?, label?}: edits
// a draft (a confirmed one goes back to draft; 409 once she recorded it as sent).
func (h *Handlers) UpdateQuestionnaire(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if _, err := h.svc.Questionnaire(c, userID, id); err != nil {
		return mapErr(err, locale)
	}
	data := pick(validation.Input(c), "answers", "explanation", "label")
	rules := validation.Rules{
		validation.F("answers", "nullable", "array"),
		validation.F("explanation", "nullable", "string", "max:"+strconv.Itoa(MaxExplanation)),
		validation.F("label", "nullable", "string", "max:80"),
	}
	for _, q := range Questions {
		rules = append(rules, validation.F("answers."+q.Code, "nullable", in([]string{AnswerYes, AnswerNo})))
	}
	if err := validate(data, rules, locale, now); err != nil {
		return err
	}
	qi := QuestionnaireInput{Answers: map[string]*string{}, Explanation: longText(data, "explanation"), Label: text(data, "label")}
	_, qi.ExplanationSent = data.Get("explanation")
	_, qi.LabelSent = data.Get("label")
	if raw, ok := data.Get("answers"); ok {
		for _, q := range Questions {
			if v, ok := phpval.Get(raw, q.Code); ok {
				if v == nil {
					qi.Answers[q.Code] = nil
				} else {
					s := phpval.ToString(v)
					qi.Answers[q.Code] = &s
				}
			}
		}
	}
	if err := h.svc.UpdateQuestionnaire(c, userID, id, qi, now); err != nil {
		return mapErr(err, locale)
	}
	return h.questionnaireResponse(c, userID, id, fiber.StatusOK, T("messages.questionnaire_saved", locale))
}

// ConfirmQuestionnaire is POST /insurance/questionnaires/{id}/confirm: she checked every answer (422 while some are
// unanswered); the client then exports the PDF.
func (h *Handlers) ConfirmQuestionnaire(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if err := h.svc.ConfirmQuestionnaire(c, userID, id, now); err != nil {
		return mapErr(err, locale)
	}
	return h.questionnaireResponse(c, userID, id, fiber.StatusOK, T("messages.questionnaire_confirmed", locale))
}

// AdvanceQuestionnaire is POST /insurance/questionnaires/{id}/status {status, occurred_on}: a status step she was told
// by the insurer (after confirming; 409 for a draft).
func (h *Handlers) AdvanceQuestionnaire(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if _, err := h.svc.Questionnaire(c, userID, id); err != nil {
		return mapErr(err, locale)
	}
	data := pick(validation.Input(c), "status", "occurred_on")
	if err := validate(data, validation.Rules{
		validation.F("status", "required", in(QuestionnaireSteps)),
		validation.F("occurred_on", "required", "date_format:Y-m-d"),
	}, locale, now); err != nil {
		return err
	}
	on := date(data, "occurred_on").Date
	if on.After(civildate.InTehran(now)) {
		return fieldFail(locale, "occurred_on", "not_future")
	}
	if err := h.svc.AdvanceQuestionnaire(c, userID, id, text(data, "status"), on, now); err != nil {
		return mapErr(err, locale)
	}
	return h.questionnaireResponse(c, userID, id, fiber.StatusOK, T("messages.questionnaire_saved", locale))
}

// DestroyQuestionnaire is DELETE /insurance/questionnaires/{id}.
func (h *Handlers) DestroyQuestionnaire(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteQuestionnaire(c, userID, id); err != nil {
		return mapErr(err, locale)
	}
	return httpx.OK(c, nil, T("messages.questionnaire_deleted", locale))
}

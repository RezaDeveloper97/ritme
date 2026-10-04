package companions

import (
	"database/sql"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Filter values of /companions/links (besides the companion statuses and types).
const filterAll = "all"

var statuses = []string{string(companion.StatusInvited), string(companion.StatusActive), string(companion.StatusRevoked)}

func types() []string {
	out := make([]string, 0, len(companion.Types))
	for _, t := range companion.Types {
		out = append(out, string(t))
	}
	return out
}

// filter maps a ?status= / ?type= value to (echoed value, LIKE pattern); unknown → all.
func filter(v string, allowed []string) (string, string) {
	for _, a := range allowed {
		if v == a {
			return v, v
		}
	}
	return filterAll, "%"
}

// MaskMobile keeps only the last 2 digits of a mobile number: 09123456789 → •••••••••89 (CMP-L6: the invited number
// is one the owner typed and may belong to someone who never signed up); short numbers fully.
func MaskMobile(m sql.NullString) any {
	if !m.Valid || m.String == "" {
		return nil
	}
	r := []rune(m.String)
	if len(r) < 9 {
		return "•••"
	}
	return strings.Repeat("•", len(r)-2) + string(r[len(r)-2:])
}

// MaskName keeps the first letter of a name: «سارا رضایی» → «س•••». Empty → null.
func MaskName(n sql.NullString) any {
	if !n.Valid {
		return nil
	}
	r := []rune(n.String)
	for len(r) > 0 && (r[0] == ' ' || r[0] == '\t') {
		r = r[1:]
	}
	if len(r) == 0 {
		return nil
	}
	return string(r[:1]) + "•••"
}

// personJSON is one party of a link: masked name and mobile, no user id (CMP-L6).
func personJSON(name, mobile sql.NullString) *jsonx.OrderedMap {
	return jsonx.Obj("name", MaskName(name), "mobile", MaskMobile(mobile))
}

func (h *Handlers) linkJSON(c fiber.Ctx, r *store.ListAdminCompanionLinksRow) *jsonx.OrderedMap {
	var comp any
	if r.CompanionUserID.Valid {
		comp = personJSON(r.CompanionName, r.CompanionMobile)
	}
	var invite any
	if r.Status == string(companion.StatusInvited) {
		expired := !r.InviteExpiresAt.Valid || !httpadmin.Now(c).Before(r.InviteExpiresAt.Time)
		invite = jsonx.Obj("phone", MaskMobile(r.InvitePhone), "expires_at", httpadmin.Time(r.InviteExpiresAt), "expired", expired)
	}
	return jsonx.Obj(
		"id", r.ID,
		"type", r.Type,
		"status", r.Status,
		"label", MaskName(r.DisplayName),
		"owner", personJSON(r.OwnerName, r.OwnerMobile),
		"companion", comp,
		"invite", invite,
		"grants_count", r.GrantsCount,
		"invited_at", httpadmin.Time(r.InvitedAt),
		"accepted_at", httpadmin.Time(r.AcceptedAt),
		"revoked_at", httpadmin.Time(r.RevokedAt),
		"revoked_by", httpadmin.NullString(r.RevokedBy),
		"created_at", httpadmin.Time(r.CreatedAt),
	)
}

// ListLinks is GET /companions/links?status=all|invited|active|revoked&type=all|partner|spouse&page=&per_page=,
// newest first, plus counts over every link (by status, and by type × status). Read-only, masked.
func (h *Handlers) ListLinks(c fiber.Ctx) error {
	status, statusPattern := filter(c.Query("status"), statuses)
	typ, typePattern := filter(c.Query("type"), types())
	total, err := h.q.CountAdminCompanionLinks(c.Context(), store.CountAdminCompanionLinksParams{
		Status: statusPattern, Type: typePattern,
	})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminCompanionLinks(c.Context(), store.ListAdminCompanionLinksParams{
		Status: statusPattern, Type: typePattern,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	counted, err := h.q.CountCompanionLinksByTypeStatus(c.Context())
	if err != nil {
		return err
	}
	byStatus := map[string]int64{}
	byType := map[string]map[string]int64{}
	var all int64
	for _, r := range counted {
		byStatus[r.Status] += r.Total
		if byType[r.Type] == nil {
			byType[r.Type] = map[string]int64{}
		}
		byType[r.Type][r.Status] += r.Total
		all += r.Total
	}
	statusCounts := jsonx.Obj("all", all)
	for _, s := range statuses {
		statusCounts.Set(s, byStatus[s])
	}
	typeCounts := jsonx.NewObject()
	for _, t := range types() {
		o := jsonx.NewObject()
		var sum int64
		for _, s := range statuses {
			o.Set(s, byType[t][s])
			sum += byType[t][s]
		}
		o.Set("all", sum)
		typeCounts.Set(t, o)
	}

	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, h.linkJSON(c, &rows[i]))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("status", status, "type", typ))
	page.Set("counts", jsonx.Obj("by_status", statusCounts, "by_type", typeCounts))
	page.Set("statuses", statuses)
	page.Set("types", types())
	return httpadmin.OK(c, page)
}

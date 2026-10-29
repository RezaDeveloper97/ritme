package messages

import (
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// FactList is GET /messages/facts (T-M10-03, D-161): the fact registry for the rule editor (T-M10-12) — every fact a
// condition may test with its type, enum values, unit, producer group, descriptions by language code, the member-safe
// flag and, for a fact not filled yet, the task that fills it. Sorted by name (stable). Text variables such as name
// are not facts and are listed by the rules meta (variables).
func (h *Handlers) FactList(c fiber.Ctx) error {
	defs := facts.Defs()
	items := make([]*jsonx.OrderedMap, 0, len(defs))
	for _, d := range defs {
		if d.Variable {
			continue
		}
		var enum any
		if d.Enum != nil {
			enum = jsonx.List(d.Enum)
		}
		var unit, pending any
		if d.Unit != "" {
			unit = d.Unit
		}
		if d.Pending != "" {
			pending = d.Pending
		}
		description := jsonx.Obj()
		for _, k := range sortedLangs(d.Description) {
			description.Set(k, d.Description[k])
		}
		items = append(items, jsonx.Obj("name", d.Name, "type", string(d.Type), "enum", enum, "unit", unit,
			"group", d.Group, "description", description, "member_safe", d.MemberSafe, "pending", pending))
	}
	return httpadmin.OK(c, jsonx.Obj("facts", items, "groups", jsonx.List(facts.Groups()),
		"fertility_facts", jsonx.List(facts.FertilityFacts)))
}

func sortedLangs(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

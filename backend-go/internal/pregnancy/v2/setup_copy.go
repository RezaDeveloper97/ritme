package v2

import (
	"context"
	"fmt"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// SetupGroup is the message_contents group of the Setup screen's admin-edited copy.
const SetupGroup = "pregnancy_setup"

// setupItem is one pregnancy_setup item the Setup screens show, with its text keys (= the
// registry schema in internal/admin/messages/registry) and its string-list keys.
type setupItem struct {
	key   string
	texts []string
	lists []string
}

// setupItems are the items of GET /pregnancy/v2/setup-copy, in screen order. calendar_note and
// due_disclaimer belong to other screens and are not listed.
var setupItems = []setupItem{
	{key: "welcome", texts: []string{"title", "body", "primary", "secondary"}, lists: []string{"benefits"}},
	{key: "dating", texts: []string{"title", "body"}},
	{key: "source_lmp", texts: []string{"label", "hint"}},
	{key: "source_ultrasound", texts: []string{"label", "hint"}},
	{key: "source_manual", texts: []string{"label", "hint"}},
	{key: "history", texts: []string{"title", "body", "disclaimer", "skip"}},
	{key: "result", texts: resultTemplateKeys},
}

// resultTemplateKeys are every text of pregnancy_setup/result; setup-copy returns them unfilled
// (dating-preview fills range / basis and returns the rest as `copy`).
var resultTemplateKeys = []string{
	"lead", "suffix", "due_label", "confidence", "range", "basis_lmp", "basis_ultrasound", "basis_manual",
	"primary", "secondary",
}

// SetupCopy is GET /pregnancy/v2/setup-copy: the admin-edited pregnancy_setup texts in the request
// locale (per item: the request-locale row, else the default-language row). Every item is always
// present; a text is null (a list []) when no live row has it, so the app falls back to its bundle.
// One query.
func (s *Service) SetupCopy(ctx context.Context, l Lang) (*jsonx.OrderedMap, error) {
	rows, err := s.q.ListV2MessageGroupPayloads(ctx, store.ListV2MessageGroupPayloadsParams{
		MessageGroup: SetupGroup, Locales: l.codes(),
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy v2: setup copy: %w", err)
	}
	byItem := map[string][]store.ListV2MessagePayloadsRow{}
	for _, r := range rows {
		byItem[r.ItemKey] = append(byItem[r.ItemKey], store.ListV2MessagePayloadsRow{Locale: r.Locale, Payload: r.Payload})
	}
	out := jsonx.Obj()
	for _, it := range setupItems {
		p := payload(byItem[it.key], l)
		item := jsonx.Obj()
		for _, k := range it.texts {
			v := str(p, k)
			item.Set(k, nullStr(v, v != ""))
		}
		for _, k := range it.lists {
			item.Set(k, strList(p, k))
		}
		out.Set(it.key, item)
	}
	return out, nil
}

// strList is a payload's non-empty strings under k; [] when missing.
func strList(m map[string]any, k string) []string {
	out := []string{}
	raw, _ := m[k].([]any)
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// CalendarNoteItem is the pregnancy_setup item holding the calendar's source note: `plan_note`
// (the care-plan caveat) and `basis_<source>` (the dating-basis sentence), design audit E2.
const CalendarNoteItem = "calendar_note"

// MessagePayload is one live message item in the request locale (else the default language's
// row); nil when neither exists.
func MessagePayload(ctx context.Context, q store.Querier, group, item string, l Lang) (map[string]any, error) {
	rows, err := q.ListV2MessagePayloads(ctx, store.ListV2MessagePayloadsParams{
		MessageGroup: group, ItemKey: item, Locales: l.codes(),
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy v2: message %s/%s: %w", group, item, err)
	}
	return payload(rows, l), nil
}

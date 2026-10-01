package analysis

import "github.com/ritme/backend-go/internal/platform/jsonx"

// Section is one card / block of a report: {plus, locked, ready, data}.
//
//   - plus: the section needs plus.deep_analysis (patterns of mood, correlations, labs trend);
//   - locked: plus and the user is not entitled — ready is false and data is null (no health data is
//     computed or sent for a locked section);
//   - ready: enough data for the section's main content (the min-data rules); data may still carry
//     counts («۲ سیکل دیگر لازم است») when not ready;
//   - data: the section body, or null.
type Section struct {
	Plus, Locked, Ready bool
	Data                any
}

// freeSection is an ungated section.
func freeSection(ready bool, data any) Section { return Section{Ready: ready, Data: data} }

// plusSection gates build behind the entitlement: a locked section never calls it.
func plusSection(entitled bool, build func() (bool, any)) Section {
	if !entitled {
		return Section{Plus: true, Locked: true}
	}
	ready, data := build()
	return Section{Plus: true, Ready: ready, Data: data}
}

// JSON is the section object.
func (s Section) JSON() *jsonx.OrderedMap {
	return jsonx.Obj("plus", s.Plus, "locked", s.Locked, "ready", s.Ready, "data", s.Data)
}

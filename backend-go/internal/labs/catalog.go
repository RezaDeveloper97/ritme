package labs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/ritme/backend-go/internal/catalog"
)

// CatalogGroup is the catalog_items group of the marker catalog (admin-editable, seeded by migration 00034).
const CatalogGroup = "lab_markers"

// Marker is one catalog marker. Title is the display name, Body «X چیست؟»; the rest lives in Meta.
type Marker struct {
	Code        string
	Title, Body json.RawMessage
	Meta        MarkerMeta
	NeedsReview bool
}

// MarkerMeta is the meta JSON of a catalog marker (translatable values are {lang: text} objects).
type MarkerMeta struct {
	Category string          `json:"category"`
	Subtitle json.RawMessage `json:"subtitle"`
	Aliases  []string        `json:"aliases"`
	Unit     string          `json:"unit"`
	Typical  *struct {
		Low  *float64 `json:"low"`
		High *float64 `json:"high"`
		Text string   `json:"text"`
	} `json:"typical"`
	// Critical are the red-flag thresholds in Unit: urgent = contact a doctor today, soon = see one soon.
	Critical struct {
		UrgentLow  *float64 `json:"urgent_low"`
		UrgentHigh *float64 `json:"urgent_high"`
		SoonLow    *float64 `json:"soon_low"`
		SoonHigh   *float64 `json:"soon_high"`
	} `json:"critical"`
	// RangeVaries: the range depends on the cycle phase or age (hormones, AMH) — the typical range is shown as text
	// only and never used to classify.
	RangeVaries bool    `json:"range_varies"`
	Low         *Advice `json:"low"`
	High        *Advice `json:"high"`
}

// Advice is what the catalog says about a low or high value.
type Advice struct {
	Factors   []json.RawMessage `json:"factors"`
	Questions []json.RawMessage `json:"questions"`
	SeeDoctor json.RawMessage   `json:"see_doctor"`
}

// TypicalRange is the catalog's typical range (Source RangeTypical); empty when the range varies or is unset.
func (m Marker) TypicalRange() Range {
	t := m.Meta.Typical
	if t == nil || m.Meta.RangeVaries {
		return Range{}
	}
	return Range{Low: t.Low, High: t.High, Text: t.Text, Source: RangeTypical}
}

// TypicalText is the typical range as printed in the catalog ("" when none).
func (m Marker) TypicalText() string {
	if m.Meta.Typical == nil {
		return ""
	}
	return m.Meta.Typical.Text
}

// AdviceFor is the low / high advice for a status (nil for normal / unknown).
func (m Marker) AdviceFor(state string) *Advice {
	switch {
	case lowSide(state):
		return m.Meta.Low
	case Attention(state):
		return m.Meta.High
	}
	return nil
}

// Catalog is the active marker catalog with an alias index.
type Catalog struct {
	items   []Marker
	byCode  map[string]int
	aliases []aliasEntry // longest first
}

type aliasEntry struct {
	alias string
	index int
}

// minPrefixAlias: aliases at least this long also match names that start with them («vitamind25oh» → vitamin_d).
const minPrefixAlias = 4

// NewCatalog indexes catalog items (rows whose meta does not decode are skipped and logged by the caller).
func NewCatalog(items []catalog.Item) (*Catalog, []string) {
	c := &Catalog{byCode: map[string]int{}}
	var bad []string
	for _, it := range items {
		m := Marker{Code: it.Code, Title: it.Title, Body: it.Body, NeedsReview: it.NeedsReview}
		if len(it.Meta) > 0 {
			if err := json.Unmarshal(it.Meta, &m.Meta); err != nil {
				bad = append(bad, it.Code)
				continue
			}
		}
		c.byCode[m.Code] = len(c.items)
		c.items = append(c.items, m)
		seen := map[string]bool{}
		for _, a := range append([]string{m.Code}, m.Meta.Aliases...) {
			if n := normalizeName(a); n != "" && !seen[n] {
				seen[n] = true
				c.aliases = append(c.aliases, aliasEntry{alias: n, index: len(c.items) - 1})
			}
		}
	}
	sort.SliceStable(c.aliases, func(i, j int) bool { return len(c.aliases[i].alias) > len(c.aliases[j].alias) })
	return c, bad
}

// Items are the markers in catalog order.
func (c *Catalog) Items() []Marker {
	if c == nil {
		return nil
	}
	return c.items
}

// ByCode is the marker with code.
func (c *Catalog) ByCode(code string) (Marker, bool) {
	if c == nil {
		return Marker{}, false
	}
	i, ok := c.byCode[code]
	if !ok {
		return Marker{}, false
	}
	return c.items[i], true
}

// Match finds the catalog marker a printed name means: an exact alias, else the longest alias (≥ 4 characters)
// the name starts with. ok=false for an uncatalogued marker.
func (c *Catalog) Match(name string) (Marker, bool) {
	if c == nil {
		return Marker{}, false
	}
	n := normalizeName(name)
	if n == "" {
		return Marker{}, false
	}
	for _, a := range c.aliases {
		if a.alias == n {
			return c.items[a.index], true
		}
	}
	for _, a := range c.aliases {
		if len(a.alias) >= minPrefixAlias && len(n) > len(a.alias) && n[:len(a.alias)] == a.alias {
			return c.items[a.index], true
		}
	}
	return Marker{}, false
}

// catalogLoader reads the catalog group through the cached catalog reader.
type catalogLoader struct {
	reader *catalog.Reader
	logger *slog.Logger
}

func (l catalogLoader) load(ctx context.Context) (*Catalog, error) {
	if l.reader == nil {
		return NewCatalogEmpty(), nil
	}
	items, err := l.reader.Items(ctx, CatalogGroup)
	if err != nil {
		return nil, fmt.Errorf("labs: marker catalog: %w", err)
	}
	c, bad := NewCatalog(items)
	if len(bad) > 0 && l.logger != nil {
		l.logger.WarnContext(ctx, "labs: catalog markers with invalid meta skipped", slog.Any("codes", bad))
	}
	return c, nil
}

// NewCatalogEmpty is a catalog without markers.
func NewCatalogEmpty() *Catalog { return &Catalog{byCode: map[string]int{}} }

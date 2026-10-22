package clinicalconfig

import (
	"bytes"
	"encoding/json"
	"math"
	"slices"
	"strconv"
)

// Kind is the shape of a key's value (T-M10-16, DECISIONS.md H-12).
type Kind string

// Kinds. A Def with an empty Kind is a numbers key.
const (
	KindNumbers Kind = "numbers" // named numbers {"systolic":180,"diastolic":120}
	KindRows    Kind = "rows"    // a table {"rows":[{…},{…}]} validated by the key's Table
)

// ColumnType is the type of one column of a rows key.
type ColumnType string

// Column types.
const (
	ColNumber ColumnType = "number"
	ColString ColumnType = "string"
	ColEnum   ColumnType = "enum"
)

// Column is one column of a rows key.
type Column struct {
	Name     string
	Type     ColumnType
	Enum     []string // ColEnum: the allowed values
	Min, Max float64  // ColNumber: accepted range
	Integer  bool     // ColNumber: whole numbers only
	Nullable bool     // null allowed (an open bound, "any"); a missing nullable column reads null
	MaxLen   int      // ColString: longest value (0 = 64)
}

// Range declares that each row is a band [From, To) (or [From, To] with ToInclusive) and that bands of rows with the
// same GroupBy values never overlap; Contiguous additionally needs every band to start where the previous one ends
// (no gaps). A null From is open below, a null To open above. A null GroupBy value means "any" and so collides with
// every value.
type Range struct {
	From, To    string
	ToInclusive bool
	Contiguous  bool
	GroupBy     []string
}

// Table is the row schema of a rows key.
type Table struct {
	Columns          []Column
	MinRows, MaxRows int
	// SortBy, when set, is a column the rows must be ascending in (StrictSort: strictly); a null value is an open
	// upper end and only the last row may have it.
	SortBy     string
	StrictSort bool
	Range      *Range
	Unique     []string    // no two rows share the values of these columns
	RowOrder   [][2]string // within a row, a < b when both are set
	// Default is version 1, one []any per row in column order (numbers as int or float64, strings, nil).
	Default [][]any
}

// Row is one row of a rows value: column → float64 | string | nil.
type Row map[string]any

// Num is a number column of the row (false when null or not a number).
func (r Row) Num(col string) (float64, bool) {
	v, ok := r[col].(float64)
	return v, ok
}

// Str is a string or enum column of the row ("" when null).
func (r Row) Str(col string) string {
	s, _ := r[col].(string)
	return s
}

// IsRows reports whether the key is table-valued.
func (d *Def) IsRows() bool { return d.Kind == KindRows }

// KindName is the key's kind for the admin API ("numbers" when unset).
func (d *Def) KindName() Kind {
	if d.Kind == "" {
		return KindNumbers
	}
	return d.Kind
}

// Column returns the column of a rows key by name.
func (t *Table) Column(name string) (*Column, bool) {
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			return &t.Columns[i], true
		}
	}
	return nil, false
}

// DefaultRows is version 1 of a rows key as a fresh copy (nil for a numbers key).
func (d *Def) DefaultRows() []Row {
	if !d.IsRows() || d.Table == nil {
		return nil
	}
	out := make([]Row, 0, len(d.Table.Default))
	for _, vals := range d.Table.Default {
		r := make(Row, len(d.Table.Columns))
		for i, c := range d.Table.Columns {
			var v any
			if i < len(vals) {
				v = vals[i]
			}
			switch x := v.(type) {
			case int:
				v = float64(x)
			case float32:
				v = float64(x)
			}
			r[c.Name] = v
		}
		out = append(out, r)
	}
	return out
}

// ParseRows reads a stored or proposed rows value ({"rows":[{…}]}) and validates it. It returns the normalised rows
// (every column present, a missing nullable column as nil) or the first problem ("" = valid).
func (d *Def) ParseRows(raw []byte) ([]Row, string) {
	if !d.IsRows() || d.Table == nil {
		return nil, "rows"
	}
	var body map[string]json.RawMessage
	if json.Unmarshal(raw, &body) != nil || len(body) != 1 || body["rows"] == nil {
		return nil, "rows"
	}
	var items []json.RawMessage
	if json.Unmarshal(body["rows"], &items) != nil {
		return nil, "rows"
	}
	rows := make([]Row, 0, len(items))
	for i, it := range items {
		dec := json.NewDecoder(bytes.NewReader(it))
		var m map[string]any
		if dec.Decode(&m) != nil || m == nil {
			return nil, "row:" + strconv.Itoa(i)
		}
		for k := range m {
			if _, ok := d.Table.Column(k); !ok {
				return nil, "column:" + strconv.Itoa(i) + ":" + k
			}
		}
		r := make(Row, len(d.Table.Columns))
		for _, c := range d.Table.Columns {
			r[c.Name] = m[c.Name] // missing → nil
		}
		rows = append(rows, r)
	}
	if p := d.ValidateRows(rows); p != "" {
		return nil, p
	}
	return rows, ""
}

// ValidateRows reports the first problem of a rows value ("" = valid). Problem codes carry the 0-based row index and
// the column: count, type:i:col, null:i:col, range:i:col, integer:i:col, enum:i:col, length:i:col, order:i:col
// (row pair), sort:i:col, open:i:col (a null sort / To value before the last row), overlap:i, gap:i, unique:i.
func (d *Def) ValidateRows(rows []Row) string {
	t := d.Table
	if t == nil {
		return "rows"
	}
	if len(rows) < t.MinRows || (t.MaxRows > 0 && len(rows) > t.MaxRows) {
		return "count"
	}
	for i, r := range rows {
		if p := t.validateRow(i, r); p != "" {
			return p
		}
	}
	if p := t.validateSort(rows); p != "" {
		return p
	}
	if p := t.validateRange(rows); p != "" {
		return p
	}
	return t.validateUnique(rows)
}

func at(i int, col string) string { return strconv.Itoa(i) + ":" + col }

func (t *Table) validateRow(i int, r Row) string {
	if len(r) != len(t.Columns) {
		return "row:" + strconv.Itoa(i)
	}
	for _, c := range t.Columns {
		v, ok := r[c.Name]
		if !ok {
			return "row:" + strconv.Itoa(i)
		}
		if v == nil {
			if !c.Nullable {
				return "null:" + at(i, c.Name)
			}
			continue
		}
		switch c.Type {
		case ColNumber:
			x, ok := v.(float64)
			if !ok || math.IsNaN(x) || math.IsInf(x, 0) {
				return "type:" + at(i, c.Name)
			}
			if c.Integer && x != math.Trunc(x) {
				return "integer:" + at(i, c.Name)
			}
			if x < c.Min || x > c.Max {
				return "range:" + at(i, c.Name)
			}
		case ColString:
			s, ok := v.(string)
			if !ok || s == "" {
				return "type:" + at(i, c.Name)
			}
			limit := c.MaxLen
			if limit == 0 {
				limit = 64
			}
			if len(s) > limit {
				return "length:" + at(i, c.Name)
			}
		case ColEnum:
			s, ok := v.(string)
			if !ok {
				return "type:" + at(i, c.Name)
			}
			if !slices.Contains(c.Enum, s) {
				return "enum:" + at(i, c.Name)
			}
		}
	}
	for _, p := range t.RowOrder {
		a, okA := r.Num(p[0])
		b, okB := r.Num(p[1])
		if okA && okB && a >= b {
			return "order:" + at(i, p[0])
		}
	}
	return ""
}

func (t *Table) validateSort(rows []Row) string {
	if t.SortBy == "" {
		return ""
	}
	for i := range rows {
		cur, ok := rows[i].Num(t.SortBy)
		if !ok {
			if i != len(rows)-1 {
				return "open:" + at(i, t.SortBy)
			}
			continue
		}
		if i == 0 {
			continue
		}
		prev, _ := rows[i-1].Num(t.SortBy)
		if cur < prev || (t.StrictSort && cur == prev) {
			return "sort:" + at(i, t.SortBy)
		}
	}
	return ""
}

// bounds of row r's band: from (−∞ when null), to (+∞ when null).
func (t *Table) bounds(r Row) (from, to float64) {
	from, to = math.Inf(-1), math.Inf(1)
	if v, ok := r.Num(t.Range.From); ok {
		from = v
	}
	if v, ok := r.Num(t.Range.To); ok {
		to = v
	}
	return from, to
}

func (t *Table) sameGroup(a, b Row) bool {
	for _, g := range t.Range.GroupBy {
		if a[g] != nil && b[g] != nil && a[g] != b[g] {
			return false
		}
	}
	return true
}

func (t *Table) validateRange(rows []Row) string {
	if t.Range == nil {
		return ""
	}
	for i, r := range rows {
		from, to := t.bounds(r)
		if from > to || (!t.Range.ToInclusive && from == to) {
			return "order:" + at(i, t.Range.From)
		}
		if t.Range.Contiguous && r[t.Range.To] == nil && i != len(rows)-1 {
			return "open:" + at(i, t.Range.To)
		}
	}
	for i := range rows {
		for j := range i {
			if !t.sameGroup(rows[i], rows[j]) {
				continue
			}
			fi, ti := t.bounds(rows[i])
			fj, tj := t.bounds(rows[j])
			overlap := fi < tj && fj < ti
			if t.Range.ToInclusive {
				overlap = fi <= tj && fj <= ti
			}
			if overlap {
				return "overlap:" + strconv.Itoa(i)
			}
		}
	}
	if t.Range.Contiguous {
		for i := 1; i < len(rows); i++ {
			_, prevTo := t.bounds(rows[i-1])
			from, _ := t.bounds(rows[i])
			if from != prevTo {
				return "gap:" + strconv.Itoa(i)
			}
		}
	}
	return ""
}

func (t *Table) validateUnique(rows []Row) string {
	if len(t.Unique) == 0 {
		return ""
	}
	seen := map[string]bool{}
	for i, r := range rows {
		parts := make([]any, 0, len(t.Unique))
		for _, c := range t.Unique {
			parts = append(parts, r[c])
		}
		k, _ := json.Marshal(parts)
		if seen[string(k)] {
			return "unique:" + strconv.Itoa(i)
		}
		seen[string(k)] = true
	}
	return ""
}

// MarshalRows is the stored form of a rows value: {"rows":[{column: value} in column order]}.
func (d *Def) MarshalRows(rows []Row) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"rows":[`)
	for i, r := range rows {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		for j, c := range d.Table.Columns {
			if j > 0 {
				buf.WriteByte(',')
			}
			k, _ := json.Marshal(c.Name)
			v, err := json.Marshal(r[c.Name])
			if err != nil {
				return nil, err
			}
			buf.Write(k)
			buf.WriteByte(':')
			buf.Write(v)
		}
		buf.WriteByte('}')
	}
	buf.WriteString(`]}`)
	return buf.Bytes(), nil
}

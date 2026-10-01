package taxonomy

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Legacy ↔ v2 mapping. Every data column of daily_health_logs maps to exactly one v2 slot, and every
// legacy value to exactly one v2 code (renames are injective), so the backfill (migration 00020) loses
// nothing: unknown legacy values are copied verbatim, booleans keep false as "no", JSON arrays keep every
// element (objects: every member value; nested values: their raw JSON), medications keep every key.
//
// Two pairs share an item: vaginal_burning(+_intensity) and vaginal_itching(+_intensity). The intensity
// column is projected first and wins; the boolean fills the item only when the intensity is empty (an
// intensity already implies "yes"). The only combination that does not round-trip is the contradictory
// false + intensity, which becomes the intensity.

// Kind is how a legacy column is projected.
type Kind uint8

// Legacy column kinds.
const (
	KindEnum     Kind = iota // varchar enum → single param (value_code)
	KindBool                 // tinyint → bool param (yes/no)
	KindLevel                // pain-intensity varchar → items param, one item (value_code = level)
	KindBoolItem             // tinyint → items param, one item (yes/no)
	KindNumber               // decimal / smallint → number / integer param (value_num)
	KindText                 // text → text param (value_text)
	KindArray                // JSON list → multi param (one row per element)
	KindMeds                 // medications JSON object → text_items param (one row per key)
)

// LegacyColumn maps one daily_health_logs column onto a v2 slot.
type LegacyColumn struct {
	Column   string
	Kind     Kind
	Category string
	Param    string
	Item     string            // KindLevel / KindBoolItem
	Rename   map[string]string // legacy value → v2 code (absent = same code)
	// LegacySet are the values the legacy column accepts (reverse projection: a v2-only code with no
	// ReverseExtra mapping is dropped instead of being written into the legacy column).
	LegacySet    []string
	ReverseExtra map[string]string // v2-only code → nearest legacy value
	Scale        int               // KindNumber: legacy decimal places (0 = integer column)
}

// RawItem is the item of medications values that were not a JSON object (kept whole in value_text).
const RawItem = "_raw"

// maxItemLen is the item column width (varchar(191)); longer legacy array values are cut to it.
const maxItemLen = 191

// SourceLegacy marks rows projected from daily_health_logs (backfill, old-endpoint writes).
const SourceLegacy = "legacy"

var levelRename = map[string]string{"low": "mild", "medium": "moderate", "high": "severe"}
var painSet = []string{"low", "medium", "high"}
var smellSet = []string{"normal", "slightly_unusual", "strong_unpleasant"}

func level(col, cat, param, item string) LegacyColumn {
	return LegacyColumn{Column: col, Kind: KindLevel, Category: cat, Param: param, Item: item, Rename: levelRename, LegacySet: painSet}
}

func boolItem(col, cat, param, item string) LegacyColumn {
	return LegacyColumn{Column: col, Kind: KindBoolItem, Category: cat, Param: param, Item: item}
}

func enum(col, cat, param string, set []string, rename, extra map[string]string) LegacyColumn {
	return LegacyColumn{Column: col, Kind: KindEnum, Category: cat, Param: param, LegacySet: set, Rename: rename, ReverseExtra: extra}
}

// legacyColumns are in projection order (= the backfill statement order; the first row of a slot wins).
var legacyColumns = []LegacyColumn{
	enum("bleeding_intensity", "bleeding", "flow", []string{"low", "medium", "high", "very_high"},
		map[string]string{"low": "light", "high": "heavy", "very_high": "very_heavy"}, nil),
	enum("blood_color", "bleeding", "color", []string{"bright_red", "red", "dark_red", "brown"}, nil, nil),
	{Column: "has_clots", Kind: KindBool, Category: "bleeding", Param: "clots"},
	enum("clots_amount", "bleeding", "clot_size", []string{"none", "low", "medium", "high"},
		map[string]string{"low": "small", "high": "large"}, nil),
	{Column: "spotting", Kind: KindBool, Category: "bleeding", Param: "spotting"},
	enum("bleeding_smell", "bleeding", "odor", smellSet, nil, map[string]string{"changed": "slightly_unusual"}),
	level("headache_intensity", "pain", "location", "head"),
	level("stomach_ache_intensity", "pain", "location", "abdomen"),
	level("pelvic_pain_intensity", "pain", "location", "pelvis"),
	level("breast_pain_intensity", "pain", "location", "breast"),
	level("back_pain_intensity", "pain", "location", "back"),
	level("ovarian_pain_intensity", "pain", "location", "ovary"),
	level("nausea_intensity", "symptoms", "digestive", "nausea"),
	level("bloating_intensity", "symptoms", "digestive", "bloating"),
	boolItem("diarrhea", "symptoms", "digestive", "diarrhea"),
	boolItem("constipation", "symptoms", "digestive", "constipation"),
	enum("appetite_change", "appetite_energy", "appetite", []string{"loss", "gain", "normal"},
		map[string]string{"loss": "decreased", "gain": "increased"}, nil),
	boolItem("food_craving", "appetite_energy", "cravings", "any"),
	level("breast_sensitivity_intensity", "symptoms", "general", "breast_tenderness"),
	boolItem("vaginal_dryness", "urogenital", "symptoms", "vaginal_dryness"),
	level("vaginal_burning_intensity", "urogenital", "symptoms", "vaginal_burning"),
	boolItem("vaginal_burning", "urogenital", "symptoms", "vaginal_burning"),
	level("vaginal_itching_intensity", "urogenital", "symptoms", "vaginal_itching"),
	boolItem("vaginal_itching", "urogenital", "symptoms", "vaginal_itching"),
	boolItem("vaginal_smell_change", "urogenital", "symptoms", "odor_change"),
	enum("urination_change", "urogenital", "urination", []string{"increase", "decrease", "normal"},
		map[string]string{"increase": "increased", "decrease": "decreased"}, nil),
	level("urination_burning_intensity", "urogenital", "symptoms", "urination_burning"),
	boolItem("acne", "skin_hair", "symptoms", "acne"),
	boolItem("oily_skin", "skin_hair", "symptoms", "oily_skin"),
	boolItem("hair_loss", "skin_hair", "symptoms", "hair_loss"),
	boolItem("swelling", "symptoms", "general", "swelling"),
	boolItem("fatigue", "symptoms", "general", "fatigue"),
	boolItem("dizziness", "symptoms", "general", "dizziness"),
	boolItem("hot_flashes", "symptoms", "general", "hot_flashes"),
	boolItem("chills", "symptoms", "general", "chills"),
	{Column: "moods", Kind: KindArray, Category: "mood", Param: "moods",
		LegacySet: []string{"happy", "calm", "angry", "anxious", "sad", "frustrated", "sensitive", "bored"}},
	enum("sleep_duration", "sleep", "duration", []string{"0_3", "3_6", "6_9", "9_plus"}, nil, nil),
	enum("sleep_quality", "sleep", "quality", []string{"good", "medium", "bad"},
		map[string]string{"medium": "fair", "bad": "poor"}, map[string]string{"great": "good"}),
	{Column: "exercise_type", Kind: KindArray, Category: "activity", Param: "types",
		LegacySet:    []string{"walking", "running", "cycling", "gym", "yoga", "swimming", "dance", "team_sport", "other"},
		ReverseExtra: map[string]string{"pilates": "other", "stretching": "other", "pelvic_floor": "other"}},
	{Column: "exercise_duration", Kind: KindNumber, Category: "activity", Param: "duration"},
	enum("exercise_intensity", "activity", "intensity", []string{"low", "medium", "high"}, nil, nil),
	{Column: "sexual_activities", Kind: KindArray, Category: "sex", Param: "symptoms", LegacySet: []string{
		"high_desire", "protected_intercourse", "unprotected_intercourse", "no_desire", "dryness", "burning",
		"pain_during_intercourse", "bleeding_after_intercourse", "lubricant_use"}},
	enum("sexual_desire", "sex", "desire", []string{"lower", "normal", "higher"}, nil, nil),
	enum("intercourse_type", "sex", "intercourse", []string{"protected", "unprotected"}, nil, nil),
	{Column: "weight", Kind: KindNumber, Category: "measurements", Param: "weight", Scale: 2},
	{Column: "basal_body_temperature", Kind: KindNumber, Category: "measurements", Param: "bbt", Scale: 2},
	{Column: "heart_rate", Kind: KindNumber, Category: "measurements", Param: "heart_rate"},
	{Column: "systolic_pressure", Kind: KindNumber, Category: "measurements", Param: "bp_systolic"},
	{Column: "diastolic_pressure", Kind: KindNumber, Category: "measurements", Param: "bp_diastolic"},
	{Column: "blood_sugar", Kind: KindNumber, Category: "measurements", Param: "blood_sugar", Scale: 1},
	enum("energy_level", "appetite_energy", "energy", []string{"very_low", "low", "medium", "high", "very_high"}, nil, nil),
	enum("discharge_color", "discharge", "color", []string{"clear", "white", "yellow", "green", "gray", "pink_bloody"}, nil, nil),
	enum("discharge_texture", "discharge", "consistency", []string{"watery", "creamy", "egg_white", "thick"},
		map[string]string{"thick": "sticky"}, nil),
	enum("discharge_amount", "discharge", "amount", []string{"low", "medium", "high"},
		map[string]string{"low": "light", "high": "heavy"}, nil),
	enum("discharge_smell", "discharge", "odor", smellSet, nil, map[string]string{"changed": "slightly_unusual"}),
	boolItem("discharge_itching", "discharge", "symptoms", "itching"),
	boolItem("discharge_burning", "discharge", "symptoms", "burning"),
	boolItem("frequent_urination", "urogenital", "symptoms", "frequent_urination"),
	{Column: "medications", Kind: KindMeds, Category: "meds", Param: "other"},
	{Column: "notes", Kind: KindText, Category: "note", Param: "text"},
}

// LegacyColumns returns the mapping (do not modify).
func LegacyColumns() []LegacyColumn { return legacyColumns }

// LegacyColumnByName finds the mapping of a column.
func LegacyColumnByName(name string) (*LegacyColumn, bool) {
	for i := range legacyColumns {
		if legacyColumns[i].Column == name {
			return &legacyColumns[i], true
		}
	}
	return nil, false
}

// WholeParam reports whether the column owns every item of its param (arrays, medications) rather than
// one slot.
func (lc *LegacyColumn) WholeParam() bool { return lc.Kind == KindArray || lc.Kind == KindMeds }

// Slot is the slot key the column writes ("cat.param.item"; for whole-param columns "cat.param").
func (lc *LegacyColumn) Slot() string {
	if lc.WholeParam() {
		return lc.Category + "." + lc.Param
	}
	return lc.Category + "." + lc.Param + "." + lc.Item
}

func (lc *LegacyColumn) rename(v string) string {
	if r, ok := lc.Rename[v]; ok {
		return r
	}
	return v
}

// Entry is one health_log_entries value.
type Entry struct {
	Category, Param, Item string
	Code, Num, Text       sql.NullString
}

// Slot is "cat.param.item".
func (e Entry) Slot() string { return e.Category + "." + e.Param + "." + e.Item }

// ParamKey is "cat.param".
func (e Entry) ParamKey() string { return e.Category + "." + e.Param }

func str(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

// LegacyValue is one raw legacy column value as scanned: nil, string (varchar / text / decimal text),
// bool, int64 (smallint) or json.RawMessage (JSON columns).
type LegacyValue = any

// Project maps one legacy column value onto its v2 entries (empty for NULL).
func (lc *LegacyColumn) Project(v LegacyValue) []Entry {
	if v == nil {
		return nil
	}
	base := Entry{Category: lc.Category, Param: lc.Param}
	switch lc.Kind {
	case KindEnum:
		base.Code = str(lc.rename(fmt.Sprint(v)))
	case KindBool, KindBoolItem:
		base.Item = lc.Item
		base.Code = str(No)
		if b, _ := v.(bool); b {
			base.Code = str(Yes)
		}
	case KindLevel:
		base.Item = lc.Item
		base.Code = str(lc.rename(fmt.Sprint(v)))
	case KindNumber:
		n, err := decimal2(v)
		if err != nil {
			return nil
		}
		base.Num = str(n)
	case KindText:
		base.Text = str(fmt.Sprint(v))
	case KindArray:
		return projectArray(base, v)
	case KindMeds:
		return projectMeds(base, v)
	}
	return []Entry{base}
}

// decimal2 formats a legacy number the way decimal(8,2) stores it.
func decimal2(v any) (string, error) {
	switch x := v.(type) {
	case int64:
		return strconv.FormatInt(x, 10) + ".00", nil
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(f, 'f', 2, 64), nil
	}
	return "", fmt.Errorf("taxonomy: not a number: %T", v)
}

func rawJSON(v any) ([]byte, bool) {
	switch x := v.(type) {
	case json.RawMessage:
		return bytes.TrimSpace(x), true
	case []byte:
		return bytes.TrimSpace(x), true
	case string:
		return bytes.TrimSpace([]byte(x)), true
	}
	return nil, false
}

// jsonMember is one element of a JSON array / one member of an object, as MariaDB's JSON_TABLE yields
// it: scalars as their text (strings unquoted), nested values as their raw JSON, null skipped.
type jsonMember struct {
	Key  string
	Text string
}

// jsonContainer splits raw JSON: kind "array" | "object" | "scalar" | "null".
func jsonContainer(raw []byte) (string, []jsonMember, string) {
	if len(raw) == 0 {
		return "null", nil, ""
	}
	switch raw[0] {
	case '[', '{':
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if _, err := dec.Token(); err != nil {
			return "null", nil, ""
		}
		isObj := raw[0] == '{'
		var out []jsonMember
		for dec.More() {
			var key string
			if isObj {
				t, err := dec.Token()
				if err != nil {
					return "null", nil, ""
				}
				key, _ = t.(string)
			}
			var m json.RawMessage
			if err := dec.Decode(&m); err != nil {
				return "null", nil, ""
			}
			if text, ok := scalarText(m); ok {
				out = append(out, jsonMember{Key: key, Text: text})
			}
		}
		if isObj {
			return "object", out, ""
		}
		return "array", out, ""
	case 'n':
		return "null", nil, ""
	}
	text, _ := scalarText(raw)
	return "scalar", nil, text
}

// scalarText is JSON_TABLE's `COALESCE(v LONGTEXT PATH '$' NULL ON ERROR, j JSON PATH '$')`: strings
// unquoted, other scalars as written, nested values as raw JSON; ok=false for null.
func scalarText(m json.RawMessage) (string, bool) {
	m = bytes.TrimSpace(m)
	if len(m) == 0 || string(m) == "null" {
		return "", false
	}
	if m[0] == '"' {
		var s string
		if err := json.Unmarshal(m, &s); err == nil {
			return s, true
		}
	}
	return string(m), true
}

func cut(s string) string {
	if utf8.RuneCountInString(s) <= maxItemLen {
		return s
	}
	return string([]rune(s)[:maxItemLen])
}

func projectArray(base Entry, v any) []Entry {
	raw, ok := rawJSON(v)
	if !ok {
		return nil
	}
	kind, members, scalar := jsonContainer(raw)
	var items []string
	switch kind {
	case "array", "object":
		for _, m := range members {
			items = append(items, m.Text)
		}
	case "scalar":
		items = []string{scalar}
	}
	var out []Entry
	seen := map[string]bool{}
	for _, it := range items {
		it = cut(it)
		if seen[it] { // item is utf8mb4_bin: exact duplicates only
			continue
		}
		seen[it] = true
		e := base
		e.Item, e.Code = it, str(Yes)
		out = append(out, e)
	}
	return out
}

func projectMeds(base Entry, v any) []Entry {
	raw, ok := rawJSON(v)
	if !ok {
		return nil
	}
	kind, members, _ := jsonContainer(raw)
	switch kind {
	case "null":
		return nil
	case "object":
		var out []Entry
		seen := map[string]bool{}
		for _, m := range members {
			k := cut(m.Key)
			if seen[k] {
				continue
			}
			seen[k] = true
			e := base
			e.Item, e.Text = k, str(m.Text)
			out = append(out, e)
		}
		return out
	}
	e := base
	e.Item, e.Text = RawItem, str(string(raw))
	return []Entry{e}
}

// ProjectRow projects a whole legacy row (column name → raw value) in mapping order; the first entry of
// a slot wins, exactly as the backfill's INSERT IGNORE does.
func ProjectRow(get func(column string) LegacyValue) []Entry {
	return ProjectColumns(get, nil)
}

// ProjectColumns is ProjectRow limited to the slots the given columns write (nil = all columns). A slot
// shared by two columns is always recomputed from both.
func ProjectColumns(get func(column string) LegacyValue, columns []string) []Entry {
	slots := map[string]bool{}
	for _, c := range columns {
		if lc, ok := LegacyColumnByName(c); ok {
			slots[lc.Slot()] = true
		}
	}
	var out []Entry
	seen := map[string]bool{}
	for i := range legacyColumns {
		lc := &legacyColumns[i]
		if columns != nil && !slots[lc.Slot()] {
			continue
		}
		for _, e := range lc.Project(get(lc.Column)) {
			if seen[e.Slot()] {
				continue
			}
			seen[e.Slot()] = true
			out = append(out, e)
		}
	}
	return out
}

// SlotsOf are the distinct slots the columns write (nil = all), split into whole params ("cat.param") and
// single slots ("cat.param.item").
func SlotsOf(columns []string) (params, items []string) {
	add := func(lc *LegacyColumn) {
		if lc.WholeParam() {
			if !slices.Contains(params, lc.Slot()) {
				params = append(params, lc.Slot())
			}
		} else if !slices.Contains(items, lc.Slot()) {
			items = append(items, lc.Slot())
		}
	}
	for i := range legacyColumns {
		if columns == nil || slices.Contains(columns, legacyColumns[i].Column) {
			add(&legacyColumns[i])
		}
	}
	return params, items
}

// Reverse is the legacy column value of a day's v2 entries (nil = NULL): what the v2 write puts back into
// daily_health_logs so the engines that still read it (cycle, fertility, messages, export) keep working.
// v2-only codes without a legacy equivalent become NULL / are dropped; verbatim legacy values go back as
// they came.
func (lc *LegacyColumn) Reverse(entries []Entry) any {
	var mine []Entry
	for _, e := range entries {
		if e.Category != lc.Category || e.Param != lc.Param {
			continue
		}
		if (lc.Kind == KindLevel || lc.Kind == KindBoolItem) && e.Item != lc.Item {
			continue
		}
		mine = append(mine, e)
	}
	if len(mine) == 0 {
		return nil
	}
	first := mine[0]
	switch lc.Kind {
	case KindEnum:
		return lc.reverseCode(first.Code.String)
	case KindBool:
		return first.Code.String != No
	case KindBoolItem:
		return first.Code.String != No
	case KindLevel:
		for legacyV, v2 := range lc.Rename {
			if v2 == first.Code.String {
				return legacyV
			}
		}
		if slices.Contains([]string{Yes, No}, first.Code.String) {
			return nil
		}
		return first.Code.String // verbatim legacy value
	case KindNumber:
		f, err := strconv.ParseFloat(first.Num.String, 64)
		if err != nil {
			return nil
		}
		if lc.Scale == 0 {
			return int64(f)
		}
		return strconv.FormatFloat(f, 'f', lc.Scale, 64)
	case KindText:
		return first.Text.String
	case KindArray:
		out := []any{}
		for _, e := range mine {
			if code := lc.reverseCode(e.Item); code != nil && !slices.Contains(out, code) {
				out = append(out, code)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case KindMeds:
		if len(mine) == 1 && first.Item == RawItem {
			if v, err := phpval.Decode([]byte(first.Text.String)); err == nil {
				return v
			}
			return nil
		}
		obj := phpval.NewMap()
		for _, e := range mine {
			obj.Set(e.Item, e.Text.String)
		}
		return obj
	}
	return nil
}

// reverseCode maps a v2 code back to the legacy value (nil = no legacy equivalent).
func (lc *LegacyColumn) reverseCode(code string) any {
	for legacyV, v2 := range lc.Rename {
		if v2 == code {
			return legacyV
		}
	}
	if slices.Contains(lc.LegacySet, code) {
		return code
	}
	if r, ok := lc.ReverseExtra[code]; ok {
		return r
	}
	if cat, ok := CategoryByCode(lc.Category); ok {
		if p, ok := cat.Param(lc.Param); ok {
			if _, isOption := p.Option(code); isOption {
				return nil // a v2-only value
			}
		}
	}
	return code // a verbatim legacy value
}

// ---------------------------------------------------------------------------------------------------
// Backfill SQL (migration 00020 and its Laravel twin carry this text verbatim; TestBackfillSQL_InMigrations
// keeps them in step).

const insertHead = "INSERT IGNORE INTO `health_log_entries` (`user_id`, `log_date`, `category`, `param`, `item`, " +
	"`value_code`, `value_num`, `value_text`, `source`, `created_at`, `updated_at`)\n"

func q(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func (lc *LegacyColumn) codeExpr(col string) string {
	if len(lc.Rename) == 0 {
		return col
	}
	keys := make([]string, 0, len(lc.Rename))
	for k := range lc.Rename {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("CASE " + col)
	for _, k := range keys {
		b.WriteString(" WHEN " + q(k) + " THEN " + q(lc.Rename[k]))
	}
	b.WriteString(" ELSE " + col + " END")
	return b.String()
}

// BackfillStatements are the idempotent INSERT IGNORE … SELECT statements of the backfill, in projection
// order.
func BackfillStatements() []string {
	var out []string
	for i := range legacyColumns {
		out = append(out, legacyColumns[i].backfill()...)
	}
	return out
}

// BackfillSQL is BackfillStatements joined into one script (each statement ends with ";\n").
func BackfillSQL() string {
	var b strings.Builder
	for _, s := range BackfillStatements() {
		b.WriteString(s)
		b.WriteString(";\n")
	}
	return b.String()
}

func (lc *LegacyColumn) backfill() []string {
	col := "l.`" + lc.Column + "`"
	cat, param := q(lc.Category), q(lc.Param)
	tail := ", 'legacy', l.`created_at`, l.`updated_at`\nFROM `daily_health_logs` l"
	sel := func(item, code, num, text, from, where string) string {
		return insertHead + "SELECT l.`user_id`, l.`log_date`, " + cat + ", " + param + ", " + item + ", " + code + ", " +
			num + ", " + text + tail + from + "\nWHERE " + where
	}
	notNull := col + " IS NOT NULL"
	switch lc.Kind {
	case KindEnum:
		return []string{sel("''", lc.codeExpr(col), "NULL", "NULL", "", notNull)}
	case KindBool:
		return []string{sel("''", "IF("+col+", 'yes', 'no')", "NULL", "NULL", "", notNull)}
	case KindLevel:
		return []string{sel(q(lc.Item), lc.codeExpr(col), "NULL", "NULL", "", notNull)}
	case KindBoolItem:
		return []string{sel(q(lc.Item), "IF("+col+", 'yes', 'no')", "NULL", "NULL", "", notNull)}
	case KindNumber:
		return []string{sel("''", "NULL", col, "NULL", "", notNull)}
	case KindText:
		return []string{sel("''", "NULL", "NULL", col, "", notNull)}
	case KindArray:
		member := func(path string) string {
			return ",\n  JSON_TABLE(" + col + ", '" + path + "' COLUMNS (`v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t"
		}
		item := "LEFT(COALESCE(t.`v`, t.`j`), 191)"
		return []string{
			sel(item, "'yes'", "NULL", "NULL", member("$[*]"), "JSON_TYPE("+col+") = 'ARRAY' AND JSON_TYPE(t.`j`) <> 'NULL'"),
			sel(item, "'yes'", "NULL", "NULL", member("$.*"), "JSON_TYPE("+col+") = 'OBJECT' AND JSON_TYPE(t.`j`) <> 'NULL'"),
			sel("LEFT(JSON_UNQUOTE("+col+"), 191)", "'yes'", "NULL", "NULL", "",
				"JSON_TYPE("+col+") NOT IN ('ARRAY', 'OBJECT', 'NULL')"),
		}
	case KindMeds:
		from := ",\n  JSON_TABLE(JSON_KEYS(" + col + "), '$[*]' COLUMNS (`n` FOR ORDINALITY, `k` LONGTEXT PATH '$')) k,\n" +
			"  JSON_TABLE(" + col + ", '$.*' COLUMNS (`n` FOR ORDINALITY, `v` LONGTEXT PATH '$' NULL ON ERROR, `j` JSON PATH '$')) t"
		return []string{
			sel("LEFT(k.`k`, 191)", "NULL", "NULL", "COALESCE(t.`v`, t.`j`)", from,
				"JSON_TYPE("+col+") = 'OBJECT' AND k.`n` = t.`n` AND JSON_TYPE(t.`j`) <> 'NULL'"),
			sel(q(RawItem), "NULL", "NULL", col, "", "JSON_TYPE("+col+") NOT IN ('OBJECT', 'NULL')"),
		}
	}
	return nil
}

// BackfillPrepared is the goose spelling of BackfillSQL: each statement wrapped in PREPARE … FROM '…' /
// EXECUTE, because sqlc parses every migration and its MySQL parser has no JSON_TABLE (the statement text
// inside a string literal is executed by MariaDB but never parsed by sqlc).
func BackfillPrepared() string {
	var b strings.Builder
	for _, s := range BackfillStatements() {
		b.WriteString("PREPARE `backfill` FROM " + q(strings.ReplaceAll(s, `\`, `\\`)) + ";\nEXECUTE `backfill`;\n")
	}
	b.WriteString("DEALLOCATE PREPARE `backfill`;\n")
	return b.String()
}

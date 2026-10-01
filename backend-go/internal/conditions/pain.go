package conditions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/conditions/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Pain diary (nbl_Cond_Endo). The taxonomy half (category `pain`):
//
//	score 0        pain.none = yes, no location
//	score 1–10     pain.location items = the chosen locations, each {level: LevelFor(score), score}
//	relief         pain.relief (multi)
//
// associated symptoms with a taxonomy slot go to that slot; the rest of the diary is condition_pain_entries.

// Analgesic effects («کمک کرد / کمی / نه»).
var AnalgesicEffects = []string{"no", "a_little", "helped"}

// MaxScore is the top of the 0–10 pain scale.
const MaxScore = 10

// MaxAnalgesicLen bounds the analgesic name.
const MaxAnalgesicLen = 100

// LevelFor maps the 1–10 score onto the taxonomy pain level (1–3 mild, 4–6 moderate, 7–10 severe;
// [needs clinical review]). The level is what the legacy daily_health_logs pain columns receive.
func LevelFor(score int) string {
	switch {
	case score >= 7:
		return "severe"
	case score >= 4:
		return "moderate"
	default:
		return "mild"
	}
}

// FieldError is a 422 on one field with a conditions validation line (lang key "validation.<Key>").
type FieldError struct{ Field, Key string }

func (e *FieldError) Error() string { return "conditions: " + e.Field + ": " + e.Key }

// PainDay is one pain-diary day (zero values when nothing was logged).
type PainDay struct {
	Date            civildate.Date
	Score           *int
	Locations       []string
	Relief          []string
	Types           []string
	Associated      []string
	MissedActivity  *bool
	Analgesic       *string
	AnalgesicTime   *string
	AnalgesicEffect *string
}

// PainChoices are the values a pain-diary day accepts for the user (her mode's taxonomy options plus what the day
// already stores; the active catalog codes).
type PainChoices struct {
	Locations, Relief, Types, Associated []string
}

// logSlot is a pain_associated item's taxonomy slot (meta.log "category.param.item").
type logSlot struct {
	cat   *taxonomy.Category
	par   *taxonomy.Param
	item  string
	valid bool
}

func slotOf(it catalog.Item) logSlot {
	var meta struct {
		Log string `json:"log"`
	}
	if len(it.Meta) == 0 || json.Unmarshal(it.Meta, &meta) != nil {
		return logSlot{}
	}
	parts := strings.Split(meta.Log, ".")
	if len(parts) != 3 {
		return logSlot{}
	}
	cat, ok := taxonomy.CategoryByCode(parts[0])
	if !ok {
		return logSlot{}
	}
	par, ok := cat.Param(parts[1])
	if !ok {
		return logSlot{}
	}
	if _, ok := par.Option(parts[2]); !ok {
		return logSlot{}
	}
	if par.Type != taxonomy.Multi && (par.Type != taxonomy.Items || !slices.Contains(par.Levels, taxonomy.Yes)) {
		return logSlot{}
	}
	return logSlot{cat: cat, par: par, item: parts[2], valid: true}
}

// present: the slot holds a "yes"-like value (a multi option, or an items level other than "no").
func (l logSlot) present(day []taxonomy.Entry) bool {
	for _, e := range day {
		if e.Category == l.cat.Code && e.Param == l.par.Code && e.Item == l.item {
			return !e.Code.Valid || e.Code.String != taxonomy.No
		}
	}
	return false
}

// writable: the slot can take a new value in mode (or already holds one).
func (l logSlot) writable(day []taxonomy.Entry, mode string) bool {
	if l.present(day) {
		return true
	}
	o, _ := l.par.Option(l.item)
	return l.cat.Available(l.par, mode) && o.OptionAvailable(mode)
}

func painParam(code string) *taxonomy.Param {
	cat, _ := taxonomy.CategoryByCode("pain")
	p, _ := cat.Param(code)
	return p
}

// options of a pain param available in mode, plus items the day already stores.
func painOptions(param, mode string, day []taxonomy.Entry) []string {
	p := painParam(param)
	var out []string
	for i := range p.Options {
		if p.Options[i].OptionAvailable(mode) {
			out = append(out, p.Options[i].Code)
		}
	}
	for _, e := range day {
		if e.Category == "pain" && e.Param == param && !slices.Contains(out, e.Item) {
			out = append(out, e.Item)
		}
	}
	return out
}

// PainChoices loads what a pain-diary day of the user accepts.
func (s *Service) PainChoices(ctx context.Context, userID uint64, date civildate.Date) (PainChoices, error) {
	logs := healthlog.NewService(s.conn)
	mode, err := logs.LifeMode(ctx, userID)
	if err != nil {
		return PainChoices{}, err
	}
	day, err := logs.Day(ctx, userID, date)
	if err != nil {
		return PainChoices{}, err
	}
	_, types, err := s.codes(ctx, GroupPainTypes)
	if err != nil {
		return PainChoices{}, err
	}
	_, assoc, err := s.codes(ctx, GroupPainAssociated)
	if err != nil {
		return PainChoices{}, err
	}
	return PainChoices{
		Locations: painOptions("location", mode, day), Relief: painOptions("relief", mode, day),
		Types: types, Associated: assoc,
	}, nil
}

// Pain loads one pain-diary day.
func (s *Service) Pain(ctx context.Context, userID uint64, date civildate.Date) (PainDay, error) {
	return s.painDay(ctx, store.New(s.conn), healthlog.NewService(s.conn), userID, date)
}

func (s *Service) painDay(ctx context.Context, q *store.Queries, logs *healthlog.Service, userID uint64, date civildate.Date) (PainDay, error) {
	day, err := logs.Day(ctx, userID, date)
	if err != nil {
		return PainDay{}, err
	}
	assocItems, err := s.catalog.Items(ctx, GroupPainAssociated)
	if err != nil {
		return PainDay{}, fmt.Errorf("conditions: load catalog: %w", err)
	}
	d := painFromLog(date, day)
	row, err := q.GetPainEntry(ctx, store.GetPainEntryParams{UserID: userID, EntryDate: date})
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return PainDay{}, fmt.Errorf("conditions: load pain entry: %w", err)
	default:
		applyPainRow(&d, row)
	}
	// associated = catalog order: slot-backed items present in the log, then row codes
	stored := d.Associated
	d.Associated = []string{}
	for _, it := range assocItems {
		sl := slotOf(it)
		if (sl.valid && sl.present(day)) || slices.Contains(stored, it.Code) {
			d.Associated = append(d.Associated, it.Code)
		}
	}
	for _, c := range stored { // codes of items an admin deactivated stay visible
		if !slices.Contains(d.Associated, c) {
			d.Associated = append(d.Associated, c)
		}
	}
	return d, nil
}

// painFromLog reads the taxonomy half of a day.
func painFromLog(date civildate.Date, day []taxonomy.Entry) PainDay {
	d := PainDay{Date: date, Locations: []string{}, Relief: []string{}, Types: []string{}, Associated: []string{}}
	none := false
	best := -1
	for _, e := range day {
		if e.Category != "pain" {
			continue
		}
		switch e.Param {
		case "none":
			none = e.Code.Valid && e.Code.String == taxonomy.Yes
		case "location":
			if e.Code.Valid && e.Code.String == taxonomy.No {
				continue
			}
			d.Locations = append(d.Locations, e.Item)
			if e.Num.Valid {
				if f, err := strconv.ParseFloat(e.Num.String, 64); err == nil && int(f) > best {
					best = int(f)
				}
			}
		case "relief":
			d.Relief = append(d.Relief, e.Item)
		}
	}
	switch {
	case len(d.Locations) > 0 && best >= 0:
		d.Score = &best
	case len(d.Locations) == 0 && none:
		zero := 0
		d.Score = &zero
	}
	return d
}

func decodeCodes(raw db.NullRawJSON) []string {
	out := []string{}
	if raw.Valid {
		var list []string
		if json.Unmarshal(raw.V, &list) == nil && list != nil {
			out = list
		}
	}
	return out
}

func applyPainRow(d *PainDay, r store.ConditionPainEntry) {
	d.Types = decodeCodes(r.PainTypes)
	d.Associated = decodeCodes(r.Associated)
	if r.MissedActivity.Valid {
		b := r.MissedActivity.Bool
		d.MissedActivity = &b
	}
	d.Analgesic = nullStr(r.Analgesic)
	d.AnalgesicTime = nullStr(r.AnalgesicTime)
	d.AnalgesicEffect = nullStr(r.AnalgesicEffect)
}

func nullStr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func toNullStr(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func encodeCodes(list []string) (db.NullRawJSON, error) {
	if len(list) == 0 {
		return db.NullRawJSON{}, nil
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return db.NullRawJSON{}, fmt.Errorf("conditions: encode codes: %w", err)
	}
	return db.NullRawJSON{V: raw, Valid: true}, nil
}

func yes(cat, param, item string) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Item: item, Code: sql.NullString{String: taxonomy.Yes, Valid: true}}
}

// dayEdit collects whole-param replacements of a day's log.
type dayEdit struct {
	day   []taxonomy.Entry
	order []string
	repl  map[string]taxonomy.Change
}

func newDayEdit(day []taxonomy.Entry) *dayEdit {
	return &dayEdit{day: day, repl: map[string]taxonomy.Change{}}
}

// entries are the param's current entries (replaced or stored).
func (d *dayEdit) entries(cat, param string) []taxonomy.Entry {
	if ch, ok := d.repl[cat+"."+param]; ok {
		return ch.Entries
	}
	var out []taxonomy.Entry
	for _, e := range d.day {
		if e.Category == cat && e.Param == param {
			out = append(out, e)
		}
	}
	return out
}

func (d *dayEdit) set(cat, param string, entries []taxonomy.Entry) {
	k := cat + "." + param
	if _, ok := d.repl[k]; !ok {
		d.order = append(d.order, k)
	}
	d.repl[k] = taxonomy.Change{Category: cat, Param: param, Entries: entries}
}

func (d *dayEdit) changes() []taxonomy.Change {
	out := make([]taxonomy.Change, 0, len(d.order))
	for _, k := range d.order {
		out = append(out, d.repl[k])
	}
	return out
}

// setSlot adds (on) or removes (off) one item of a multi / items param.
func (d *dayEdit) setSlot(sl logSlot, on bool) {
	cur := d.entries(sl.cat.Code, sl.par.Code)
	if on == sl.present(cur) { // already as wanted (a stored level like "severe" is kept)
		return
	}
	next := make([]taxonomy.Entry, 0, len(cur)+1)
	for _, e := range cur {
		if e.Item != sl.item {
			next = append(next, e)
		}
	}
	if on { // an explicit "no" becomes yes
		next = append(next, yes(sl.cat.Code, sl.par.Code, sl.item))
	}
	d.set(sl.cat.Code, sl.par.Code, next)
}

// PainInput is a validated PUT /conditions/pain/{date}: Set says which keys the body sent (a sent null clears).
type PainInput struct {
	Date            civildate.Date
	Set             map[string]bool
	Score           *int
	Locations       []string
	Relief          []string
	Types           []string
	Associated      []string
	MissedActivity  *bool
	Analgesic       *string
	AnalgesicTime   *string
	AnalgesicEffect *string
}

// SavePain merges the sent keys over the day (log half through healthlog, the rest in condition_pain_entries) and
// returns the saved day. ErrNotEnrolled without the endometriosis program; *FieldError when score and locations
// don't fit together (a score above 0 needs a location; a location needs a score).
func (s *Service) SavePain(ctx context.Context, userID uint64, in PainInput, locale string, now time.Time) (PainDay, error) {
	assocItems, err := s.catalog.Items(ctx, GroupPainAssociated)
	if err != nil {
		return PainDay{}, fmt.Errorf("conditions: load catalog: %w", err)
	}
	var saved PainDay
	err = s.inTx(ctx, func(q *store.Queries, logs *healthlog.Service) error {
		if err := requireEnrolled(ctx, q, userID, ProgramEndo); err != nil {
			return err
		}
		cur, err := s.painDay(ctx, q, logs, userID, in.Date)
		if err != nil {
			return err
		}
		mode, err := logs.LifeMode(ctx, userID)
		if err != nil {
			return err
		}
		day, err := logs.Day(ctx, userID, in.Date)
		if err != nil {
			return err
		}
		edit := newDayEdit(day)
		if in.Set["score"] || in.Set["locations"] {
			if err := painScoreChanges(edit, &cur, in); err != nil {
				return err
			}
		}
		if in.Set["relief"] {
			entries := make([]taxonomy.Entry, len(in.Relief))
			for i, r := range in.Relief {
				entries[i] = yes("pain", "relief", r)
			}
			edit.set("pain", "relief", entries)
		}
		var rowAssoc []string
		if in.Set["associated"] {
			rowAssoc = associatedChanges(edit, assocItems, day, mode, cur.Associated, in.Associated)
		} else {
			rowAssoc = rowOnly(assocItems, day, cur.Associated)
		}
		if ch := edit.changes(); len(ch) > 0 {
			if _, err := logs.SaveDay(ctx, userID, in.Date, ch, locale, now); err != nil {
				return err
			}
		}
		if in.Set["types"] {
			cur.Types = in.Types
		}
		if in.Set["missed_activity"] {
			cur.MissedActivity = in.MissedActivity
		}
		if in.Set["analgesic"] {
			cur.Analgesic = in.Analgesic
		}
		if in.Set["analgesic_time"] {
			cur.AnalgesicTime = in.AnalgesicTime
		}
		if in.Set["analgesic_effect"] {
			cur.AnalgesicEffect = in.AnalgesicEffect
		}
		if err := savePainRow(ctx, q, userID, in.Date, cur, rowAssoc, now); err != nil {
			return err
		}
		saved, err = s.painDay(ctx, q, logs, userID, in.Date)
		return err
	})
	if err != nil {
		return PainDay{}, err
	}
	return saved, nil
}

// painScoreChanges rewrites pain.none and pain.location from the merged score and locations.
func painScoreChanges(edit *dayEdit, cur *PainDay, in PainInput) error {
	score, locations := cur.Score, cur.Locations
	if in.Set["score"] {
		score = in.Score
	}
	if in.Set["locations"] {
		locations = in.Locations
	}
	if score != nil && *score == 0 {
		locations = nil // 0 = no pain («بدون درد»)
	}
	switch {
	case score != nil && *score > 0 && len(locations) == 0:
		return &FieldError{Field: "locations", Key: "locations_required"}
	case score == nil && len(locations) > 0:
		return &FieldError{Field: "score", Key: "score_required"}
	}
	var none, locs []taxonomy.Entry
	if score != nil && *score == 0 {
		none = []taxonomy.Entry{yes("pain", "none", "")}
	}
	for _, l := range locations {
		locs = append(locs, taxonomy.Entry{
			Category: "pain", Param: "location", Item: l,
			Code: sql.NullString{String: LevelFor(*score), Valid: true},
			Num:  sql.NullString{String: strconv.Itoa(*score) + ".00", Valid: true},
		})
	}
	edit.set("pain", "none", none)
	edit.set("pain", "location", locs)
	return nil
}

// associatedChanges writes the slot-backed choices into the log and returns the codes the row keeps (no slot, or a
// slot the user's mode can't take).
func associatedChanges(edit *dayEdit, items []catalog.Item, day []taxonomy.Entry, mode string, before, want []string) []string {
	var row []string
	for _, it := range items {
		sl := slotOf(it)
		on := slices.Contains(want, it.Code)
		switch {
		case sl.valid && sl.writable(day, mode):
			if on || slices.Contains(before, it.Code) {
				edit.setSlot(sl, on)
			}
		case on:
			row = append(row, it.Code)
		}
	}
	for _, c := range want { // codes validated against a catalog that changed meanwhile
		if !slices.ContainsFunc(items, func(it catalog.Item) bool { return it.Code == c }) && !slices.Contains(row, c) {
			row = append(row, c)
		}
	}
	return row
}

// rowOnly are the associated codes that live in the row (not backed by a present log slot).
func rowOnly(items []catalog.Item, day []taxonomy.Entry, assoc []string) []string {
	var row []string
	for _, c := range assoc {
		i := slices.IndexFunc(items, func(it catalog.Item) bool { return it.Code == c })
		if i >= 0 {
			if sl := slotOf(items[i]); sl.valid && sl.present(day) {
				continue
			}
		}
		row = append(row, c)
	}
	return row
}

func savePainRow(ctx context.Context, q *store.Queries, userID uint64, date civildate.Date, d PainDay, assoc []string, now time.Time) error {
	if len(d.Types) == 0 && len(assoc) == 0 && d.MissedActivity == nil && d.Analgesic == nil && d.AnalgesicTime == nil && d.AnalgesicEffect == nil {
		if err := q.DeletePainEntry(ctx, store.DeletePainEntryParams{UserID: userID, EntryDate: date}); err != nil {
			return fmt.Errorf("conditions: delete pain entry: %w", err)
		}
		return nil
	}
	types, err := encodeCodes(d.Types)
	if err != nil {
		return err
	}
	as, err := encodeCodes(assoc)
	if err != nil {
		return err
	}
	p := store.UpsertPainEntryParams{
		UserID: userID, EntryDate: date, PainTypes: types, Associated: as,
		Analgesic: toNullStr(d.Analgesic), AnalgesicTime: toNullStr(d.AnalgesicTime), AnalgesicEffect: toNullStr(d.AnalgesicEffect),
		Now: tehranNow(now),
	}
	if d.MissedActivity != nil {
		p.MissedActivity = sql.NullBool{Bool: *d.MissedActivity, Valid: true}
	}
	if err := q.UpsertPainEntry(ctx, p); err != nil {
		return fmt.Errorf("conditions: save pain entry: %w", err)
	}
	return nil
}

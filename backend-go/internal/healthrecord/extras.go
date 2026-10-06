package healthrecord

// Record extras (canvas-build CB-REC-01, D-70; nbl_Rec_Home): what B-N6-03's record lacks — whether the allergies go
// on the emergency card, surgeries / hospital stays and family history — kept on the same health_records row (00041).
// Owner-only, never logged.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Relatives are the family_history relative codes (labels live in the clients' record namespace).
var Relatives = []string{
	"mother", "father", "sister", "brother", "grandmother", "grandfather", "aunt", "uncle", "child", "other",
}

// Extras limits.
const (
	MaxSurgeries     = 20
	MaxFamilyHistory = 20
)

// Surgery is one «بستری و جراحی» entry.
type Surgery struct {
	Title string `json:"title"`
	Date  string `json:"date"` // Y-m-d, "" = unknown
}

// FamilyItem is one «سابقه خانوادگی» entry.
type FamilyItem struct {
	Condition string `json:"condition"`
	Relative  string `json:"relative"` // a Relatives code, "" = not told
}

// Extras are the record extras of a user. A nil list was never answered; an empty one is «ندارم».
type Extras struct {
	Allergies       []string
	AllergiesOnCard bool
	Surgeries       []Surgery
	FamilyHistory   []FamilyItem
}

func decodeList[T any](raw db.NullRawJSON) []T {
	if !raw.Valid {
		return nil
	}
	out := []T{}
	_ = json.Unmarshal(raw.V, &out)
	if out == nil {
		out = []T{}
	}
	return out
}

func encodeList[T any](xs []T) db.NullRawJSON {
	if xs == nil {
		return db.NullRawJSON{}
	}
	b, _ := json.Marshal(xs)
	return db.NullRawJSON{V: b, Valid: true}
}

// GetExtras reads the user's extras (defaults when she has no record row yet: allergies go on the card).
func (s *Documents) GetExtras(ctx context.Context, userID uint64) (Extras, error) {
	row, err := optional(s.q.GetRecordExtras(ctx, userID))
	if err != nil {
		return Extras{}, fmt.Errorf("healthrecord: extras: %w", err)
	}
	if row == nil {
		return Extras{AllergiesOnCard: true}, nil
	}
	return Extras{
		Allergies: codes(row.Allergies), AllergiesOnCard: row.AllergiesOnEmergencyCard,
		Surgeries: decodeList[Surgery](row.Surgeries), FamilyHistory: decodeList[FamilyItem](row.FamilyHistory),
	}, nil
}

// ExtrasInput is a validated PUT /health-record/extras: only the Set* fields change.
type ExtrasInput struct {
	SetAllergiesOnCard bool
	AllergiesOnCard    bool
	SetSurgeries       bool
	Surgeries          []Surgery // nil = clear (never answered), [] = «ندارم»
	SetFamilyHistory   bool
	FamilyHistory      []FamilyItem
}

// SaveExtras applies in to the user's health_records row (created on first save; blood type and allergies untouched).
func (s *Documents) SaveExtras(ctx context.Context, userID uint64, in ExtrasInput, now time.Time) error {
	cur, err := s.GetExtras(ctx, userID)
	if err != nil {
		return err
	}
	if in.SetAllergiesOnCard {
		cur.AllergiesOnCard = in.AllergiesOnCard
	}
	if in.SetSurgeries {
		cur.Surgeries = in.Surgeries
	}
	if in.SetFamilyHistory {
		cur.FamilyHistory = in.FamilyHistory
	}
	if err := s.q.UpsertRecordExtras(ctx, store.UpsertRecordExtrasParams{
		UserID: userID, AllergiesOnEmergencyCard: cur.AllergiesOnCard, Surgeries: encodeList(cur.Surgeries),
		FamilyHistory: encodeList(cur.FamilyHistory), Now: nullTime(now),
	}); err != nil {
		return fmt.Errorf("healthrecord: save extras: %w", err)
	}
	return nil
}

func surgeriesJSON(xs []Surgery) any {
	if xs == nil {
		return nil
	}
	out := make([]*jsonx.OrderedMap, 0, len(xs))
	for _, x := range xs {
		out = append(out, jsonx.Obj("title", x.Title, "date", strOrNil(x.Date)))
	}
	return out
}

func familyJSON(xs []FamilyItem) any {
	if xs == nil {
		return nil
	}
	out := make([]*jsonx.OrderedMap, 0, len(xs))
	for _, x := range xs {
		out = append(out, jsonx.Obj("condition", x.Condition, "relative", strOrNil(x.Relative)))
	}
	return out
}

// JSON is the GET /health-record/extras body. allergies mirrors B-N6-03's list (edited through PUT /basics).
func (e Extras) JSON() *jsonx.OrderedMap {
	return jsonx.Obj(
		"allergies", listOrNil(e.Allergies),
		"allergies_on_emergency_card", e.AllergiesOnCard,
		"surgeries", surgeriesJSON(e.Surgeries),
		"family_history", familyJSON(e.FamilyHistory),
	)
}

// surgeryDate normalises a validated date ("" stays unknown).
func surgeryDate(s string, now time.Time) string {
	if s == "" {
		return ""
	}
	t, err := civildate.ParseLenient(s, now, civildate.Tehran)
	if err != nil {
		return ""
	}
	return civildate.InTehran(t).String()
}

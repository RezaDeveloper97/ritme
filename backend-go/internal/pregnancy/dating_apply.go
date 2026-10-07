package pregnancy

import (
	"context"
	"errors"
	"time"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Ultrasound dating from a record document (canvas-build CB-REC-02): a confirmed imaging document with a gestational
// age offers «سن بارداری را از این سند به‌روز کنم؟»; only the user's explicit confirm calls ApplyUltrasoundDating —
// never a silent update. It writes what PUT /pregnancy/profile writes for age_source=ultrasound (confidence, ±1 day,
// then the recomputed due date).

// ErrNotPregnant: the user has no pregnancy profile in pregnancy mode.
var ErrNotPregnant = errors.New("pregnancy: not in pregnancy mode")

// UltrasoundDating is a scan's gestational age on its date.
type UltrasoundDating struct {
	ScanDate civildate.Date
	Weeks    int // completed weeks at the scan
	Days     int // 0..6
}

// ActiveProfile is the user's pregnancy profile when it is in pregnancy mode (ErrNotPregnant otherwise).
func ActiveProfile(ctx context.Context, q store.Querier, userID uint64) (*store.PregnancyProfile, error) {
	p, err := LoadProfile(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.PregnancyMode {
		return nil, ErrNotPregnant
	}
	return p, nil
}

// CurrentDue is the profile's due date as the calculator resolves it on today (ok false without dating data).
func CurrentDue(p *store.PregnancyProfile, today civildate.Date) (civildate.Date, bool) {
	return calc.New(p, "en", today).EDDDate()
}

// ApplyUltrasoundDating dates the user's active pregnancy from a scan and returns the updated profile.
func ApplyUltrasoundDating(ctx context.Context, q store.Querier, userID uint64, d UltrasoundDating, now time.Time,
) (*store.PregnancyProfile, error) {
	p, err := ActiveProfile(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	conf, unc, _ := confidenceFor(string(enums.PregnancyAgeSourceUltrasound))
	attrs := jsonx.Obj(
		"age_source", string(enums.PregnancyAgeSourceUltrasound),
		"ultrasound_date", d.ScanDate,
		"ultrasound_weeks", int64(d.Weeks),
		"ultrasound_days", int64(d.Days),
		"confidence_level", string(conf),
		"uncertainty_days", int64(unc),
	)
	if err := saveProfile(ctx, q, userID, p, attrs, now); err != nil {
		return nil, err
	}
	p, err = LoadProfile(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	if due, ok := CurrentDue(p, civildate.InTehran(now)); ok {
		if err := saveProfile(ctx, q, userID, p, jsonx.Obj("estimated_due_date", due), now); err != nil {
			return nil, err
		}
	}
	return LoadProfile(ctx, q, userID)
}

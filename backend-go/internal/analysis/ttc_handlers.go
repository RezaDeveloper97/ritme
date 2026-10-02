package analysis

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// TTCHandlers serve GET /analysis/ttc and /analysis/fertility (B-N3-11). Same rules as Handlers: only the
// authenticated user's rows, nothing logged.
type TTCHandlers struct {
	*Handlers
	fertility *fertility.Service
}

// NewTTCHandlers wraps the analysis handlers with the fertility_logs reader.
func NewTTCHandlers(h *Handlers, fert *fertility.Service) *TTCHandlers {
	return &TTCHandlers{Handlers: h, fertility: fert}
}

// loadTTC reads the shared Input (no day logs: the TTC report reads its own signals), the first TTC log
// day and the TTC logs of the analysed cycles (taxonomy v2 rows merged with fertility_logs).
func (h *TTCHandlers) loadTTC(c fiber.Ctx) (*TTCInput, error) {
	// plusOnly = true and a logs window that starts after today: load never reads health_log_entries
	// for the shared Days map, which this report does not use.
	req, err := h.load(c, func(today civildate.Date) Range { return NewRange(RangeAll, today) },
		func(r Range) civildate.Date { return r.To.AddDays(1) }, true)
	if err != nil {
		return nil, err
	}
	in := &TTCInput{Input: req.in, Signals: NewTTCSignals()}
	ctx := c.Context()
	if in.FirstSignal, err = h.fertility.FirstSignal(ctx, req.userID); err != nil {
		return nil, err
	}
	if from := in.AnalysedFrom(); !from.IsZero() {
		if err := h.signals(ctx, req.userID, in, from); err != nil {
			return nil, err
		}
	}
	return in, nil
}

func (h *TTCHandlers) signals(ctx context.Context, uid uint64, in *TTCInput, from civildate.Date) error {
	rows, err := h.logs.Range(ctx, uid, from, in.Today)
	if err != nil {
		return err
	}
	entries := make([]DayEntries, len(rows))
	for i, r := range rows {
		entries[i] = DayEntries{Date: r.Date, Entries: r.Entries}
	}
	in.Signals.AddEntries(entries)
	legacy, err := h.fertility.Signals(ctx, uid, from, in.Today)
	if err != nil {
		return err
	}
	for _, s := range legacy {
		in.Signals.AddFertilityLog(s.Date, s.LH, s.Mucus)
	}
	return nil
}

// TTC is GET /analysis/ttc: the TTC hub (An_Hub_TTC). Plus cards (timing, mucus, luteal) arrive locked
// without data for a free user.
func (h *TTCHandlers) TTC(c fiber.Ctx) error {
	in, err := h.loadTTC(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, BuildTTC(in).JSON())
}

// Fertility is GET /analysis/fertility[?cycle=YYYY-MM-DD]: one analysed cycle's BBT chart and ovulation
// (An_Fertility); cycle is the start of one of the hub's cycles (default: the newest). 404 when the user
// has no cycle at all.
func (h *TTCHandlers) Fertility(c fiber.Ctx) error {
	q := validation.Query(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), q, validation.Rules{
		validation.F("cycle", "nullable", "date_format:Y-m-d"),
	})
	if v.Fails() {
		return v.Errors()
	}
	var start civildate.Date
	if raw, ok := q.Get("cycle"); ok {
		if s, isStr := raw.(string); isStr && s != "" {
			d, err := civildate.Parse(s)
			if err != nil {
				return fieldFail(c, "cycle", "validation.date_format", map[string]string{"format": "Y-m-d"})
			}
			start = d
		}
	}
	in, err := h.loadTTC(c)
	if err != nil {
		return err
	}
	r := BuildTTC(in)
	if len(r.Cycles) == 0 {
		return httpx.NotFound()
	}
	i, ok := r.FindCycle(start)
	if !ok {
		return fieldFail(c, "cycle", "validation.in", nil)
	}
	return httpx.OK(c, r.FertilityJSON(i))
}

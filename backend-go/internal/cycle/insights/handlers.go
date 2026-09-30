package insights

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	hlmodel "github.com/ritme/backend-go/internal/healthlog/model"
	hlstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Handlers serve GET /cycle/history and GET /cycle/symptom-pattern. Both read only the current
// user's own rows (no id parameter), so there is nothing another user can address.
type Handlers struct {
	cycle *cycleservice.Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (clock.Middleware's request clock wins).
func NewHandlers(cycle *cycleservice.Service, base clock.Clock) *Handlers {
	return &Handlers{cycle: cycle, clock: base}
}

func (h *Handlers) today(c fiber.Ctx) civildate.Date {
	return civildate.InTehran(clock.FromContext(c, h.clock).Now())
}

func userID(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// History is GET /cycle/history.
func (h *Handlers) History(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	today := h.today(c)
	// No daily logs are needed: an empty one-day range keeps the snapshot cheap.
	sn, err := h.cycle.Load(c.Context(), uid, today, today, today)
	if err != nil {
		return err
	}
	return httpx.OK(c, BuildHistory(sn.Histories, sn.EngineProfile(), today).JSON())
}

// SymptomPattern is GET /cycle/symptom-pattern.
func (h *Handlers) SymptomPattern(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	today := h.today(c)
	sn, err := h.cycle.Load(c.Context(), uid, today.AddDays(-PatternLookback), today, today)
	if err != nil {
		return err
	}
	logs := make(map[civildate.Date]DayLog, len(sn.LogRows))
	for _, r := range sn.LogRows {
		if w := Weigh(hlmodel.FromRow(hlstore.DailyHealthLog(r))); len(w) > 0 {
			logs[r.LogDate] = w
		}
	}
	return httpx.OK(c, BuildPattern(sn.Histories, sn.EngineProfile(), logs, today).JSON())
}

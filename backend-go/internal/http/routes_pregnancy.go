package http

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Pregnancy mode (backend/routes/api.php, prefix pregnancy, auth:api). Static paths are
// registered before their parameter siblings, in the Laravel order.
func init() {
	Register("pregnancy", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := pregnancy.NewHandlers(store.New(d.DB))
		h.SetFamilyGrants(d.DB)                                      // reset stale pregnancy grants on activate/onboarding/deactivate (T-M9-02)
		h.SetCycleLength(effectiveCycleLength(cyclestore.New(d.DB))) // optional EDD cycle correction (D-169, T-M10-18c)
		const p = "/api/v1/pregnancy"

		r.Post(p+"/activate", locale, guard, h.Activate)
		r.Post(p+"/deactivate", locale, guard, h.Deactivate)
		r.Post(p+"/onboarding", locale, guard, h.Onboarding)
		r.Get(p+"/profile", locale, guard, h.Show)
		r.Put(p+"/profile", locale, guard, h.Update)
		r.Post(p+"/confirm", locale, guard, h.Confirm)
		r.Get(p+"/status", locale, guard, h.Status)
		r.Get(p+"/enums", locale, guard, h.Enums)

		r.Get(p+"/symptoms/enums", locale, guard, h.SymptomEnums)
		r.Get(p+"/symptoms", locale, guard, h.SymptomIndex)
		r.Post(p+"/symptoms", locale, guard, h.SymptomStore)
		r.Get(p+"/symptoms/:date", locale, guard, h.SymptomShow)
		r.Delete(p+"/symptoms/:date", locale, guard, h.SymptomDestroy)

		r.Get(p+"/weekly/enums", locale, guard, h.WeeklyEnums)
		r.Get(p+"/weekly", locale, guard, h.WeeklyIndex)
		r.Post(p+"/weekly", locale, guard, h.WeeklyStore)
		r.Get(p+"/weekly/:week", locale, guard, h.WeeklyShow)

		r.Get(p+"/fetal-movement", locale, guard, h.FetalIndex)
		r.Post(p+"/fetal-movement", locale, guard, h.FetalStore)

		r.Get(p+"/content/:week", locale, guard, h.Content)

		r.Get(p+"/alerts/summary", locale, guard, h.AlertSummary)
		r.Get(p+"/alerts", locale, guard, h.AlertIndex)
		r.Get(p+"/alerts/:id", locale, guard, h.AlertShow)
		r.Post(p+"/alerts/:id/read", locale, guard, h.AlertRead)
		r.Post(p+"/alerts/read-all", locale, guard, h.AlertReadAll)
		r.Post(p+"/alerts/:id/dismiss", locale, guard, h.AlertDismiss)
	})
}

// effectiveCycleLength is the pregnancy handlers' cycle-length source (T-M10-18c, D-169): the cycle engine's
// effective cycle length (confirmed history → profile, «خودکار از داده‌ها» honoured), 0 when it only has the system
// default. It moves the due date only while pregnancy.edd.cycle_correction is enabled (calc.EDDFromLMP).
func effectiveCycleLength(cq *cyclestore.Queries) pregnancy.CycleLength {
	return func(ctx context.Context, userID uint64) (int, error) {
		var profile *cyclestore.UserProfile
		manual := false
		p, err := cq.GetEngineProfileByUserID(ctx, userID)
		switch {
		case err == nil:
			profile = &p.UserProfile
			manual = !p.LengthsAuto
		case !errors.Is(err, sql.ErrNoRows):
			return 0, fmt.Errorf("pregnancy: cycle profile: %w", err)
		}
		rows, err := cq.ListCycleHistoriesNewestFirst(ctx, userID)
		if err != nil {
			return 0, fmt.Errorf("pregnancy: cycle histories: %w", err)
		}
		engineProfile := cycleservice.ProfileFromRow(profile)
		if engineProfile != nil {
			engineProfile.LengthsManual = manual
		}
		m := metrics.Calculate(cycleservice.HistoriesFromRows(rows), engineProfile)
		if m.CycleLengthSource == enums.EffectiveSourceDefault {
			return 0, nil
		}
		return m.EffectiveCycleLength, nil
	}
}

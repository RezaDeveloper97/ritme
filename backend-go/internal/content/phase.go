package content

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// PhaseContent is GET /cycle/phase-content/{phase} (PhaseContentController::show).
func (h *Handlers) PhaseContent(c fiber.Ctx) error {
	// The controller clamps to its hard-coded fa/en pair (not the language registry).
	locale := i18n.Clamp(i18n.ResolveLocale(c, ""), "fa", i18n.LegacyPair...)
	fa := locale == "fa"

	sub, ok := enums.CycleSubphaseFrom(c.Params("phase"))
	if !ok {
		return httpx.Fail(fiber.StatusUnprocessableEntity, pick(fa, "فاز نامعتبر است", "Invalid phase key"))
	}
	row, err := h.q.GetPhaseContent(c, string(sub.Canonical()))
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.Fail(fiber.StatusNotFound, pick(fa, "محتوای این فاز یافت نشد", "Content not found for this phase"))
	}
	if err != nil {
		return fmt.Errorf("content: phase content: %w", err)
	}

	// PhaseContent::SECTIONS in display order; only sections with copy for the locale.
	fields := []struct {
		name string
		col  db.NullRawJSON
	}{
		{"symptom_prediction", row.SymptomPrediction},
		{"vaginal_discharge", row.VaginalDischarge},
		{"fertility", row.Fertility},
		{"hormonal_changes", row.HormonalChanges},
		{"sex_tips", row.SexTips},
		{"nutrition", row.Nutrition},
		{"exercise", row.Exercise},
		{"skin_care", row.SkinCare},
		{"sleep", row.Sleep},
	}
	sections := jsonx.NewArray() // [] when empty, like a PHP array
	for _, f := range fields {
		if !f.col.Valid {
			continue
		}
		// getLocalizedContent: $content[$locale] ?? $content['fa'] ?? $content['en'] ?? null.
		v, isStr := phpValue(i18n.PickChain(f.col.V, locale, "fa", "en")).(string)
		if isStr && strings.Trim(v, " \t\n\r\x00\x0B") != "" {
			sections.Set(f.name, v)
		}
	}
	return httpx.OK(c, jsonx.Obj("phase", string(sub), "phase_label", sub.Label(locale), "sections", sections))
}

func pick(fa bool, faText, enText string) string {
	if fa {
		return faText
	}
	return enText
}

func nullJSON(v db.NullRawJSON) []byte {
	if !v.Valid {
		return nil
	}
	return v.V
}

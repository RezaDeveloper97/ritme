package dashboard

// Alert rules of the dashboard (T-M10-31, D-180): GET /dashboard/alert-rules lists the rules, PUT
// /dashboard/alert-rules/:key changes a rule's threshold and enabled flag. Capability «dashboard» by path (GET view,
// PUT manage); every change is audited. The comparator, window and minimum volume are not editable.

import (
	"database/sql"
	"errors"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/metrics/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Route is the registrar AlertRoutes takes (the admin core routes file's `h`).
type Route func(method, path string, chain httpadmin.Chain)

// AlertHandlers serve the alert rules.
type AlertHandlers struct {
	q      *store.Queries
	logger *slog.Logger
}

// NewAlertHandlers wires the handlers over db.
func NewAlertHandlers(db *sql.DB, logger *slog.Logger) *AlertHandlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &AlertHandlers{q: store.New(db), logger: logger}
}

// Routes mounts the rule endpoints.
func (h *AlertHandlers) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/dashboard/alert-rules", kit.Admin(h.List))
	route(fiber.MethodPut, "/dashboard/alert-rules/:key", kit.Admin(h.Update))
}

func ruleJSON(r store.MetricAlertRule) *jsonx.OrderedMap {
	th, _ := strconv.ParseFloat(r.Threshold, 64)
	var minVol any
	if r.MinVolume.Valid {
		minVol = r.MinVolume.Int32
	}
	return jsonx.Obj("key", r.MetricKey, "comparator", r.Comparator, "threshold", th, "window_days", r.WindowDays,
		"min_volume", minVol, "enabled", r.Enabled)
}

// List is GET /dashboard/alert-rules.
func (h *AlertHandlers) List(c fiber.Ctx) error {
	rules, err := h.q.ListMetricAlertRules(c.Context())
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleJSON(r))
	}
	return httpadmin.OK(c, out)
}

// Update is PUT /dashboard/alert-rules/:key {threshold, enabled}.
func (h *AlertHandlers) Update(c fiber.Ctx) error {
	key := c.Params("key")
	rule, err := h.q.GetMetricAlertRuleByKey(c.Context(), key)
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Alert rule")
	}
	if err != nil {
		return err
	}
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("threshold", "required|numeric|min:0|max:100000000"),
		validation.F("enabled", "required|boolean"),
	})
	if err != nil {
		return err
	}
	th, err := strconv.ParseFloat(httpadmin.String(data, "threshold"), 64)
	if err != nil {
		return httpadmin.FieldError("threshold", "The threshold must be a number.")
	}
	enabled := httpadmin.Bool(data, "enabled")
	if _, err := h.q.UpdateMetricAlertRuleSettings(c.Context(), store.UpdateMetricAlertRuleSettingsParams{
		Threshold: strconv.FormatFloat(th, 'f', 4, 64), Enabled: enabled, MetricKey: key, Now: httpadmin.DBTime(httpadmin.Now(c)),
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "dashboard.alert_rule.update", "metric_alert_rule", rule.ID,
		slog.String("rule", key), slog.Float64("threshold", th), slog.Bool("enabled", enabled))
	rule, err = h.q.GetMetricAlertRuleByKey(c.Context(), key)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, ruleJSON(rule), "Alert rule updated.")
}

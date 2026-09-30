// Package support is the admin inbox of «گزارش مشکل» reports (B-N1-12b): list (newest first, status filter,
// a short preview only), detail (the full message, device info and whether a screenshot exists), the screenshot
// streamed from the private storage volume to an authenticated admin, and resolve / reopen.
//
// Reports are written by POST /api/v1/support/reports (internal/profile). Screenshots live under
// STORAGE_PATH/app/private/support-reports and are never reachable through the public /storage disk: the only
// way to read one is GET /support-reports/:id/screenshot behind the admin session chain.
package support

import (
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/profile"
)

// Report statuses (support_reports.status) and the ?status= filter values.
const (
	StatusOpen     = "open"
	StatusResolved = "resolved"
	StatusAll      = "all"
)

// screenshotMaxBytes caps what the stream will send (the upload is re-encoded to ≤ 1600 px WebP, far below it).
const screenshotMaxBytes = 8 << 20

// Handlers serve /support-reports.
type Handlers struct {
	q           *store.Queries
	logger      *slog.Logger
	storagePath string
}

// NewHandlers wires the handlers; storagePath is STORAGE_PATH (where the private screenshots live).
func NewHandlers(db *sql.DB, storagePath string, logger *slog.Logger) *Handlers {
	return &Handlers{q: store.New(db), logger: logger, storagePath: storagePath}
}

func statusFilter(c fiber.Ctx) (status, pattern string) {
	switch s := c.Query("status"); s {
	case StatusOpen, StatusResolved:
		return s, s
	case StatusAll:
		return StatusAll, "%"
	default:
		return StatusOpen, StatusOpen // the inbox opens on what still needs an answer
	}
}

func userJSON(id uint64, name, mobile sql.NullString) *jsonx.OrderedMap {
	return jsonx.Obj("id", id, "name", httpadmin.NullString(name), "mobile", httpadmin.NullString(mobile))
}

// List is GET /support-reports?status=open|resolved|all&page=&per_page= (default open), newest first.
// Rows carry a preview of the message (first 160 characters), never the full text or anything else.
func (h *Handlers) List(c fiber.Ctx) error {
	status, pattern := statusFilter(c)
	total, err := h.q.CountSupportReports(c.Context(), pattern)
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListSupportReports(c.Context(), store.ListSupportReportsParams{
		Status: pattern, Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	counts, err := h.q.CountSupportReportsByStatus(c.Context())
	if err != nil {
		return err
	}
	byStatus := map[string]int64{}
	for _, r := range counts {
		byStatus[r.Status] = r.Total
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonx.Obj(
			"id", r.ID,
			"user", userJSON(r.UserID, r.UserName, r.UserMobile),
			"preview", r.Preview,
			"has_screenshot", r.HasScreenshot,
			"app_version", httpadmin.NullString(r.AppVersion),
			"status", r.Status,
			"created_at", httpadmin.Time(r.CreatedAt),
			"updated_at", httpadmin.Time(r.UpdatedAt),
		))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("status", status))
	page.Set("counts", jsonx.Obj("open", byStatus[StatusOpen], "resolved", byStatus[StatusResolved]))
	return httpadmin.OK(c, page)
}

func (h *Handlers) find(c fiber.Ctx) (store.GetSupportReportRow, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.GetSupportReportRow{}, httpadmin.NotFound("Support report")
	}
	r, err := h.q.GetSupportReport(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return r, httpadmin.NotFound("Support report")
	}
	return r, err
}

func detailJSON(r *store.GetSupportReportRow) *jsonx.OrderedMap {
	return jsonx.Obj("support_report", jsonx.Obj(
		"id", r.ID,
		"user", userJSON(r.UserID, r.UserName, r.UserMobile),
		"message", r.Message,
		"has_screenshot", r.ScreenshotPath.Valid,
		"app_version", httpadmin.NullString(r.AppVersion),
		"user_agent", httpadmin.NullString(r.UserAgent),
		"status", r.Status,
		"created_at", httpadmin.Time(r.CreatedAt),
		"updated_at", httpadmin.Time(r.UpdatedAt),
	))
}

// Show is GET /support-reports/:id.
func (h *Handlers) Show(c fiber.Ctx) error {
	r, err := h.find(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, detailJSON(&r))
}

// Screenshot is GET /support-reports/:id/screenshot: the WebP from private storage, never cached by a shared
// cache or sniffed as anything else. 404 when the report has no screenshot or the file is gone.
func (h *Handlers) Screenshot(c fiber.Ctx) error {
	r, err := h.find(c)
	if err != nil {
		return err
	}
	abs, ok := profile.SupportFilePath(h.storagePath, r.ScreenshotPath)
	if !ok {
		return httpadmin.NotFound("Screenshot")
	}
	info, err := os.Stat(abs)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && (!info.Mode().IsRegular() || info.Size() > screenshotMaxBytes)) {
		return httpadmin.NotFound("Screenshot")
	}
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs) //nolint:gosec // G304: path comes from SupportFilePath (private support dir only)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "support_report.screenshot", "support_report", r.ID)
	c.Set(fiber.HeaderContentType, "image/webp")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderContentDisposition, "inline")
	return c.Send(data)
}

// Resolve is POST /support-reports/:id/resolve.
func (h *Handlers) Resolve(c fiber.Ctx) error {
	return h.setStatus(c, StatusResolved, "Report resolved.")
}

// Reopen is POST /support-reports/:id/reopen.
func (h *Handlers) Reopen(c fiber.Ctx) error { return h.setStatus(c, StatusOpen, "Report reopened.") }

func (h *Handlers) setStatus(c fiber.Ctx, status, msg string) error {
	r, err := h.find(c)
	if err != nil {
		return err
	}
	if r.Status != status {
		if _, err := h.q.SetSupportReportStatus(c.Context(), store.SetSupportReportStatusParams{
			Status: status, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: r.ID,
		}); err != nil {
			return err
		}
		httpadmin.Audit(c, h.logger, "support_report."+map[string]string{StatusResolved: "resolve", StatusOpen: "reopen"}[status],
			"support_report", r.ID)
	}
	r, err = h.q.GetSupportReport(c.Context(), r.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, detailJSON(&r), msg)
}

// Route registers one admin route (httpadmin.Handle under httpadmin.Prefix).
type Route func(method, path string, chain httpadmin.Chain)

// Routes mounts /support-reports. Any active admin (the same role as /users: reports carry the user's mobile and
// free text); resolve / reopen go through the CSRF check of the admin chain.
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	get, post := fiber.MethodGet, fiber.MethodPost
	route(get, "/support-reports", kit.Admin(h.List))
	route(get, "/support-reports/:id", kit.Admin(h.Show))
	route(get, "/support-reports/:id/screenshot", kit.Admin(h.Screenshot))
	route(post, "/support-reports/:id/resolve", kit.Admin(h.Resolve))
	route(post, "/support-reports/:id/reopen", kit.Admin(h.Reopen))
}

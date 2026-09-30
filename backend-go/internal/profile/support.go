package profile

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/profile/store"
)

const (
	// SupportScreenshotMaxKB caps the «گزارش مشکل» screenshot upload.
	SupportScreenshotMaxKB = 5120
	// SupportReportsPerHour is the per-user flood guard of POST /support/reports (on top of the write throttle).
	SupportReportsPerHour = 5
	// supportReportDir is where screenshots live, relative to STORAGE_PATH: the private part of the storage
	// volume (never app/public, which /storage/* serves). Files are 0600, directories 0700.
	supportReportDir = "app/private/support-reports"
	userAgentMax     = 255
)

// supportMaxPixels bounds the decoded size of a screenshot (decompression-bomb guard): a phone screenshot is
// ~3–4 Mpx, 12 Mpx leaves room for tablets. Larger images are refused with 422 before anything is decoded.
const supportMaxPixels = 12_000_000

// supportOptimizer re-encodes screenshots (fit 1600×1600, WebP): drops EXIF / metadata and anything appended
// after the image data. Phone screenshots are ~1080–1290 px wide, so they keep their size.
var supportOptimizer = media.Optimizer{MaxWidth: 1600, MaxHeight: 1600, Quality: 80, MaxPixels: supportMaxPixels}

// optimizeSlots caps concurrent screenshot re-encodes process-wide (each can hold ~50 MB of pixels); a request
// that finds both slots busy gets a localized 503 instead of queueing.
var optimizeSlots = make(chan struct{}, 2)

// errUndecodable: the screenshot passed the header checks but could not be decoded / re-encoded → 422.
var errUndecodable = errors.New("profile: support screenshot cannot be decoded")

// errBusy: every optimizer slot is taken → 503.
var errBusy = errors.New("profile: support screenshot optimizer busy")

// SupportReportLimiter is the atomic per-user hourly limit of POST /support/reports (Redis counter, before the
// body is decoded). The DB count in the handler stays as a backup (e.g. Redis flushed). nil cache → no-op.
func SupportReportLimiter(c *cache.Client, base clock.Clock) fiber.Handler {
	return SupportReportLimiterWith(c, base, auth.ThrottleIdentity)
}

// SupportReportLimiterWith is SupportReportLimiter keyed by identity (tests).
func SupportReportLimiterWith(c *cache.Client, base clock.Clock, identity ratelimit.Identity) fiber.Handler {
	if c == nil {
		return func(ctx fiber.Ctx) error { return ctx.Next() }
	}
	return ratelimit.New(c, base).NamedWith("support-reports", SupportReportsPerHour, time.Hour, identity,
		func(ctx fiber.Ctx, r ratelimit.Rejection) error {
			e := httpx.Fail(fiber.StatusTooManyRequests, PT("messages.too_many_reports", i18n.Locale(ctx)), "retry_after", r.RetryAfter)
			for k, v := range r.Headers() {
				e = e.WithHeader(k, v)
			}
			return e
		})
}

func supportRules() validation.Rules {
	return validation.Rules{
		validation.F("message", "required", "string", "min:10", "max:2000"),
		validation.F("app_version", "nullable", "string", "max:32"),
	}
}

// screenshotError validates the optional `screenshot` — a multipart file, or in a JSON body a base64 data URL
// (`data:image/<type>;base64,…`, what the web client sends) — as jpeg/png/webp judged by its bytes, ≤ 5 MB. It
// returns the localized message, or "" and the inspected image (nil when none was sent).
func screenshotError(c fiber.Ctx, locale string, attrs map[string]string, sent any) (string, *media.Image) {
	name := attrs["screenshot"]
	if name == "" {
		name = "screenshot"
	}
	msg := func(key string, extra ...string) string {
		p := map[string]string{"attribute": name}
		for i := 0; i+1 < len(extra); i += 2 {
			p[extra[i]] = extra[i+1]
		}
		return lang.Default().Trans(key, p, locale)
	}
	var data []byte
	fh, err := c.FormFile("screenshot")
	switch {
	case err == nil && fh != nil:
		data, err = media.ReadUpload(fh, SupportScreenshotMaxKB*1024)
	case sent == nil:
		return "", nil
	default:
		str, isStr := sent.(string)
		if data, err = decodeDataURL(str); !isStr || err != nil {
			return msg("validation.image"), nil // a text value where an image belongs
		}
		if len(data) > SupportScreenshotMaxKB*1024 {
			err = media.ErrTooLarge
		}
	}
	if errors.Is(err, media.ErrTooLarge) {
		return msg("validation.max.file", "max", strconv.Itoa(SupportScreenshotMaxKB)), nil
	}
	if err != nil {
		return msg("validation.uploaded"), nil
	}
	img, err := media.Inspect(data)
	switch {
	case errors.Is(err, media.ErrWrongType):
		return msg("validation.mimes", "values", "jpeg, jpg, png, webp"), nil
	case err != nil, img.Width*img.Height > supportMaxPixels:
		return msg("validation.image"), nil
	}
	return "", &img
}

// storeScreenshot writes the re-encoded screenshot under STORAGE_PATH/app/private/support-reports with a random
// name and returns its path relative to STORAGE_PATH. An image that cannot be re-encoded is not stored raw.
func (h *PrivacyHandlers) storeScreenshot(img *media.Image) (string, error) {
	if h.storagePath == "" {
		return "", errors.New("profile: support screenshot: STORAGE_PATH not configured")
	}
	select {
	case optimizeSlots <- struct{}{}:
		defer func() { <-optimizeSlots }()
	default:
		return "", errBusy
	}
	out, err := supportOptimizer.Optimize(*img)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errUndecodable, err)
	}
	rel := media.NewPath(supportReportDir, "webp")
	abs := filepath.Join(h.storagePath, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return "", fmt.Errorf("profile: support screenshot dir: %w", err)
	}
	if err := os.WriteFile(abs, out, 0o600); err != nil {
		return "", fmt.Errorf("profile: support screenshot write: %w", err)
	}
	return rel, nil
}

// decodeDataURL returns the bytes of a `data:image/…;base64,` URL (the declared type is ignored: the bytes decide).
func decodeDataURL(s string) ([]byte, error) {
	head, payload, ok := strings.Cut(s, ",")
	if !ok || !strings.HasPrefix(head, "data:image/") || !strings.HasSuffix(head, ";base64") {
		return nil, errors.New("profile: not an image data URL")
	}
	if len(payload) > base64.StdEncoding.EncodedLen(SupportScreenshotMaxKB*1024+1) {
		return nil, media.ErrTooLarge
	}
	b, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("profile: data URL: %w", err)
	}
	return b, nil
}

// SupportFilePath resolves a stored support_reports.screenshot_path (relative to storagePath) to its absolute
// path. ok=false for an empty path, an unset storagePath, or anything outside the private support-report
// directory (so a tampered row can never point the admin screenshot stream or a delete elsewhere).
func SupportFilePath(storagePath string, rel sql.NullString) (string, bool) {
	if storagePath == "" || !rel.Valid || !strings.HasPrefix(rel.String, supportReportDir+"/") ||
		strings.Contains(rel.String, "..") || strings.ContainsRune(rel.String, '\\') {
		return "", false
	}
	return filepath.Join(storagePath, filepath.FromSlash(rel.String)), true
}

// RemoveSupportFiles deletes support-report screenshots (paths relative to storagePath, as stored in
// support_reports.screenshot_path) after their rows are gone — account deletion and the admin user delete.
// Only paths under the support-report directory are touched; a missing file is fine, any other failure is
// logged with the relative path only.
func RemoveSupportFiles(storagePath string, paths []sql.NullString, logger *slog.Logger) {
	if storagePath == "" {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}
	for _, p := range paths {
		abs, ok := SupportFilePath(storagePath, p)
		if !ok {
			continue
		}
		if err := os.Remove(abs); err != nil && !errors.Is(err, fs.ErrNotExist) {
			reason := err.Error()
			if pe := (*fs.PathError)(nil); errors.As(err, &pe) {
				reason = pe.Err.Error() // PathError carries the absolute path; log the relative one only
			}
			logger.Error("support screenshot not removed", slog.String("path", p.String), slog.String("error", reason))
		}
	}
}

func attrName(names map[string]string, field string) string {
	if n := names[field]; n != "" {
		return n
	}
	return field
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// CreateSupportReport is POST /support/reports (multipart or JSON): {message, app_version?, screenshot?}. The
// report is stored for the support team (read in the admin, not sent to any third party); 201 {id, status,
// created_at}. More than SupportReportsPerHour reports in an hour → localized 429.
func (h *PrivacyHandlers) CreateSupportReport(c fiber.Ctx) error {
	id, err := privacyUserID(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	now := h.now(c)
	body := validation.Input(c)
	attrs := privacyAttributes(locale)
	v := validation.Make(lang.Default(), locale, body, supportRules(),
		validation.Now(now), validation.Attributes(attrs...))
	names := make(map[string]string, len(attrs)/2)
	for i := 0; i+1 < len(attrs); i += 2 {
		names[attrs[i]] = attrs[i+1]
	}
	sent, _ := body.Get("screenshot")
	shotMsg, img := screenshotError(c, locale, names, sent)
	if v.Fails() || shotMsg != "" {
		bag := v.ErrorBag()
		if shotMsg != "" {
			bag.Set("screenshot", []string{shotMsg})
		}
		return httpx.Fail(fiber.StatusUnprocessableEntity, PT("messages.validation_failed", locale), "errors", bag)
	}

	recent, err := h.q.CountRecentSupportReports(c, store.CountRecentSupportReportsParams{
		UserID: id, CreatedAt: sql.NullTime{Time: now.Add(-time.Hour), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("profile: support reports count: %w", err)
	}
	if recent >= SupportReportsPerHour {
		return httpx.Fail(fiber.StatusTooManyRequests, PT("messages.too_many_reports", locale))
	}

	var shot string
	if img != nil {
		shot, err = h.storeScreenshot(img)
		switch {
		case errors.Is(err, errUndecodable):
			bag := jsonx.NewArray()
			bag.Set("screenshot", []string{lang.Default().Trans("validation.image",
				map[string]string{"attribute": attrName(names, "screenshot")}, locale)})
			return httpx.Fail(fiber.StatusUnprocessableEntity, PT("messages.validation_failed", locale), "errors", bag)
		case errors.Is(err, errBusy):
			return httpx.Fail(fiber.StatusServiceUnavailable, PT("messages.busy", locale))
		case err != nil:
			return err
		}
	}
	message, _ := body.Get("message")
	version, _ := body.Get("app_version")
	versionStr, _ := version.(string)
	msgStr, _ := message.(string)
	reportID, err := h.q.CreateSupportReport(c, store.CreateSupportReportParams{
		UserID:         id,
		Message:        strings.TrimSpace(msgStr),
		ScreenshotPath: nullStr(shot),
		AppVersion:     nullStr(versionStr),
		UserAgent:      nullStr(truncateRunes(string(c.Request().Header.UserAgent()), userAgentMax)),
		Now:            sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		if shot != "" {
			_ = os.Remove(filepath.Join(h.storagePath, filepath.FromSlash(shot)))
		}
		return fmt.Errorf("profile: support report: %w", err)
	}
	h.logger.Info("support report", "id", reportID, "user_id", id, "screenshot", shot != "")
	return httpx.Created(c, jsonx.Obj(
		"id", reportID,
		"status", "open",
		"created_at", jsonx.ISO8601(now),
	), PT("messages.report_sent", locale))
}

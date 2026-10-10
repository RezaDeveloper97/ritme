package httpadmin

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/store"
)

// AuditWriter is the slice of store.Queries that persists audit rows (admin_audit_logs, B-N9-02).
type AuditWriter interface {
	InsertAdminAuditLog(ctx context.Context, arg store.InsertAdminAuditLogParams) error
}

// maxDetailString caps string attrs copied into the DB detail: ids, codes and flags fit; free text does not.
const maxDetailString = 64

// Audit writes one audit record for an admin action: who (admin id), what (action, e.g. "user.block"), on what
// (target type + id) and from where — as the slog "admin audit" line and as an admin_audit_logs row. Never pass
// secrets or health payloads; only scalar extras (numbers, flags, short strings) reach the DB detail.
func Audit(c fiber.Ctx, logger *slog.Logger, action, targetType string, targetID uint64, extra ...slog.Attr) {
	var adminID uint64
	if a := CurrentAdmin(c); a != nil {
		adminID = a.ID
	}
	AuditAs(c, logger, adminID, action, targetType, targetID, extra...)
}

// AuditAs is Audit for an explicit admin (login, before the session exists).
func AuditAs(c fiber.Ctx, logger *slog.Logger, adminID uint64, action, targetType string, targetID uint64,
	extra ...slog.Attr) {
	attrs := []slog.Attr{
		slog.String("audit", action),
		slog.String("target_type", targetType),
		slog.Uint64("target_id", targetID),
		slog.String("ip", c.IP()),
	}
	if adminID > 0 {
		attrs = append(attrs, slog.Uint64("admin_id", adminID))
	}
	logger.LogAttrs(c.Context(), slog.LevelInfo, "admin audit", append(attrs, extra...)...)

	k, _ := c.Locals(kitKey).(*Kit)
	if k == nil || k.audit == nil {
		return
	}
	row := store.InsertAdminAuditLogParams{
		AdminID:    sql.NullInt64{Int64: int64(adminID), Valid: adminID > 0}, //nolint:gosec // ids fit int64
		Action:     truncate(action, 64),
		TargetType: truncate(targetType, 64),
		TargetID:   targetID,
		Ip:         truncate(c.IP(), 45),
		Detail:     auditDetail(extra),
		CreatedAt:  DBTime(Now(c)),
	}
	// A cancelled request must not lose the row; the insert is one small statement.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Context()), 3*time.Second)
	defer cancel()
	if err := k.audit.InsertAdminAuditLog(ctx, row); err != nil {
		logger.ErrorContext(ctx, "admin audit row not written", slog.String("audit", action), slog.Any("err", err))
	}
}

// auditDetail keeps the scalar extras as a JSON object (NULL when none).
func auditDetail(extra []slog.Attr) sql.NullString {
	m := map[string]any{}
	for _, a := range extra {
		v := a.Value.Resolve()
		switch v.Kind() {
		case slog.KindInt64:
			m[a.Key] = v.Int64()
		case slog.KindUint64:
			m[a.Key] = v.Uint64()
		case slog.KindFloat64:
			m[a.Key] = v.Float64()
		case slog.KindBool:
			m[a.Key] = v.Bool()
		case slog.KindString:
			if s := v.String(); utf8.RuneCountInString(s) <= maxDetailString {
				m[a.Key] = s
			}
		default: // groups, durations, times, any: not copied
		}
	}
	if len(m) == 0 {
		return sql.NullString{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

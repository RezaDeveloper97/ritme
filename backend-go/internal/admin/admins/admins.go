// Package admins is admin-account management (Admin\AdminController), super admins only:
// list, show, create, update, delete. An admin cannot demote, deactivate or delete
// themselves; new password hashes are bcrypt cost 12. Changing an admin's password,
// deactivating or deleting them ends their sessions.
package admins

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	adminauth "github.com/ritme/backend-go/internal/admin/auth"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Handlers serve /admins.
type Handlers struct {
	q        *store.Queries
	sessions *httpadmin.Sessions
	logger   *slog.Logger
}

// NewHandlers wires the handlers.
func NewHandlers(db *sql.DB, sessions *httpadmin.Sessions, logger *slog.Logger) *Handlers {
	return &Handlers{q: store.New(db), sessions: sessions, logger: logger}
}

// List is GET /admins?page=&per_page= (newest first).
func (h *Handlers) List(c fiber.Ctx) error {
	total, err := h.q.CountAdmins(c.Context())
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdmins(c.Context(), store.ListAdminsParams{
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, httpadmin.AdminJSON(&rows[i]))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

func (h *Handlers) find(c fiber.Ctx) (*store.Admin, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return nil, httpadmin.NotFound("Admin")
	}
	a, err := h.q.GetAdminByID(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpadmin.NotFound("Admin")
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Show is GET /admins/:id.
func (h *Handlers) Show(c fiber.Ctx) error {
	a, err := h.find(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("admin", httpadmin.AdminJSON(a)))
}

func rules(passwordRequired bool) validation.Rules {
	pw := "nullable|string|min:8|max:72"
	if passwordRequired {
		pw = "required|string|min:8|max:72"
	}
	return validation.Rules{
		validation.F("name", "required|string|max:255"),
		validation.F("email", "required|string|email|max:255"),
		validation.F("password", pw),
		validation.F("password_confirmation", "nullable|string"),
		validation.F("role", "required", validation.In(httpadmin.Roles...)),
		validation.F("is_active", "sometimes|boolean"),
	}
}

// checkEmailAndConfirmation adds the unique:admins,email and confirmed checks the
// rule engine does not have.
func (h *Handlers) checkEmailAndConfirmation(c fiber.Ctx, data phpval.Map, exceptID uint64) error {
	taken, err := h.q.AdminEmailTaken(c.Context(), store.AdminEmailTakenParams{
		Email: httpadmin.String(data, "email"), ExceptID: exceptID,
	})
	if err != nil {
		return err
	}
	if taken {
		return httpadmin.FieldError("email", httpadmin.Trans(c, "validation.unique",
			map[string]string{"attribute": httpadmin.AttributeName(c, "email")}))
	}
	if httpadmin.String(data, "password") != "" {
		if msg, bad := adminauth.Unconfirmed(c, data.Get); bad {
			return httpadmin.FieldError("password", msg)
		}
	}
	return nil
}

// Store is POST /admins {name, email, password, password_confirmation, role, is_active?}
// (is_active defaults to true).
func (h *Handlers) Store(c fiber.Ctx) error {
	data, err := httpadmin.Validate(c, rules(true))
	if err != nil {
		return err
	}
	if err := h.checkEmailAndConfirmation(c, data, 0); err != nil {
		return err
	}
	hash, err := adminauth.HashPassword(httpadmin.String(data, "password"))
	if err != nil {
		return err
	}
	active := true
	if _, ok := data.Get("is_active"); ok {
		active = httpadmin.Bool(data, "is_active")
	}
	res, err := h.q.CreateAdmin(c.Context(), store.CreateAdminParams{
		Name: httpadmin.String(data, "name"), Email: httpadmin.String(data, "email"), Password: hash,
		Role: httpadmin.String(data, "role"), IsActive: active, Now: httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	a, err := h.q.GetAdminByID(c.Context(), uint64(id)) //nolint:gosec // G115: auto-increment ids are positive
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "admin.create", "admin", a.ID,
		slog.String("role", a.Role), slog.Bool("is_active", a.IsActive))
	return httpadmin.Created(c, jsonx.Obj("admin", httpadmin.AdminJSON(&a)), "Admin created.")
}

// Update is PUT /admins/:id {name, email, password?, password_confirmation?, role,
// is_active?} (password empty/absent = unchanged, is_active absent = unchanged). An
// admin cannot change their own role or deactivate themselves (422 cannot_modify_self).
func (h *Handlers) Update(c fiber.Ctx) error {
	target, err := h.find(c)
	if err != nil {
		return err
	}
	data, err := httpadmin.Validate(c, rules(false))
	if err != nil {
		return err
	}
	if err := h.checkEmailAndConfirmation(c, data, target.ID); err != nil {
		return err
	}
	role := httpadmin.String(data, "role")
	active := target.IsActive
	if _, ok := data.Get("is_active"); ok {
		active = httpadmin.Bool(data, "is_active")
	}
	me := httpadmin.CurrentAdmin(c)
	self := me.ID == target.ID
	if self && (role != target.Role || !active) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, httpadmin.CodeSelf,
			"You cannot change your own role or deactivate your own account.")
	}
	now := httpadmin.DBTime(httpadmin.Now(c))
	if err := h.q.UpdateAdmin(c.Context(), store.UpdateAdminParams{
		Name: httpadmin.String(data, "name"), Email: httpadmin.String(data, "email"),
		Role: role, IsActive: active, Now: now, ID: target.ID,
	}); err != nil {
		return err
	}
	passwordChanged := false
	if pw := httpadmin.String(data, "password"); pw != "" {
		hash, err := adminauth.HashPassword(pw)
		if err != nil {
			return err
		}
		if err := h.q.UpdateAdminPassword(c.Context(), store.UpdateAdminPasswordParams{
			Password: hash, Now: now, ID: target.ID,
		}); err != nil {
			return err
		}
		passwordChanged = true
	}
	if passwordChanged || !active {
		var keep *httpadmin.Session
		if self {
			keep = httpadmin.CurrentSession(c)
		}
		if err := h.endSessions(c.Context(), target.ID, keep); err != nil {
			return err
		}
	}
	a, err := h.q.GetAdminByID(c.Context(), target.ID)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "admin.update", "admin", a.ID, slog.String("role", a.Role),
		slog.Bool("is_active", a.IsActive), slog.Bool("password_changed", passwordChanged))
	return httpadmin.OK(c, jsonx.Obj("admin", httpadmin.AdminJSON(&a)), "Admin updated.")
}

// Destroy is DELETE /admins/:id (not yourself: 422 cannot_modify_self).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	target, err := h.find(c)
	if err != nil {
		return err
	}
	if httpadmin.CurrentAdmin(c).ID == target.ID {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, httpadmin.CodeSelf, "You cannot delete your own account.")
	}
	res, err := h.q.DeleteAdmin(c.Context(), target.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpadmin.NotFound("Admin")
	}
	if err := h.endSessions(c.Context(), target.ID, nil); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "admin.delete", "admin", target.ID)
	return httpadmin.OK(c, jsonx.Obj("id", target.ID), "Admin deleted.")
}

func (h *Handlers) endSessions(ctx context.Context, adminID uint64, keep *httpadmin.Session) error {
	if h.sessions == nil {
		return nil
	}
	return h.sessions.DestroyAll(ctx, adminID, keep)
}

package content

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// resource bundles what the generic show/toggle/delete endpoints need for one table.
type resource[T any] struct {
	name   string // "Affirmation" (404 message)
	key    string // "affirmation" (response key, audit target)
	get    func(context.Context, uint64) (T, error)
	json   func(*T) *jsonx.OrderedMap
	id     func(*T) uint64
	toggle func(ctx context.Context, now sql.NullTime, id uint64) error
	del    func(context.Context, uint64) (sql.Result, error)
}

func (r resource[T]) show(c fiber.Ctx) error {
	row, err := find(c, r.name, r.get)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj(r.key, r.json(&row)))
}

// respond reloads the row and answers {key: row} with status 200 or 201.
func (r resource[T]) respond(c fiber.Ctx, id uint64, created bool, msg string) error {
	row, err := r.get(c.Context(), id)
	if err != nil {
		return err
	}
	body := jsonx.Obj(r.key, r.json(&row))
	if created {
		return httpadmin.Created(c, body, msg)
	}
	return httpadmin.OK(c, body, msg)
}

func (r resource[T]) doToggle(c fiber.Ctx, logger *slog.Logger, msg string) error {
	row, err := find(c, r.name, r.get)
	if err != nil {
		return err
	}
	id := r.id(&row)
	if err := r.toggle(c.Context(), httpadmin.DBTime(httpadmin.Now(c)), id); err != nil {
		return err
	}
	httpadmin.Audit(c, logger, r.key+".toggle", r.key, id)
	return r.respond(c, id, false, msg)
}

func (r resource[T]) destroy(c fiber.Ctx, logger *slog.Logger, msg string) error {
	row, err := find(c, r.name, r.get)
	if err != nil {
		return err
	}
	id := r.id(&row)
	res, err := r.del(c.Context(), id)
	if err := deleted(res, err, r.name); err != nil {
		return err
	}
	httpadmin.Audit(c, logger, r.key+".delete", r.key, id)
	return httpadmin.OK(c, jsonx.Obj("id", id), msg)
}

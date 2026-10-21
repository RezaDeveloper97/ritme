package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/httpx"
)

// RouteUnmatched labels requests no route matched (404 probes), so random paths never become label values.
const RouteUnmatched = "unmatched"

// HTTP is the request middleware: it carries a QueryCounter in the request context (and Locals) for the DB driver
// wrapper, then records http_request_duration_seconds and db_queries_per_request under the route template. Mount
// it before anything that queries the DB (module gate, auth) so their queries are counted too.
func HTTP() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		ctx, q := WithQueryCounter(c.Context())
		c.SetContext(ctx)
		c.Locals(QueryCounterKey, q)

		err := c.Next()

		status := c.Response().StatusCode()
		if err != nil {
			status = httpx.StatusOf(err)
		}
		route := RouteLabel(c, status)
		ObserveHTTP(route, c.Method(), strconv.Itoa(status), time.Since(start))
		ObserveQueriesPerRequest(route, q.Load())
		return err
	}
}

// RouteLabel is the Fiber route template of the request (`/api/v1/cycle/date/:date`), never the raw path. Call it
// after c.Next() with the final status. A 404 no route matched is RouteUnmatched; a request a group middleware
// stopped before any route matched (auth 401, module gate 403) is labelled with that middleware's mount prefix —
// both come from code, so the label set stays bounded.
func RouteLabel(c fiber.Ctx, status int) string {
	if !c.Matched() && status == fiber.StatusNotFound {
		return RouteUnmatched
	}
	return c.Route().Path
}

// RequestQueries is the number of DB queries the request made so far (0 outside the HTTP middleware).
func RequestQueries(c fiber.Ctx) int64 {
	q, _ := c.Locals(QueryCounterKey).(*QueryCounter)
	return q.Load()
}

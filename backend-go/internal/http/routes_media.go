package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/media"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// MediaChunkThrottleMax bounds the upload chunks one user sends per minute (a 2 GB video at 8 MB chunks is 256).
const MediaChunkThrottleMax = 240

// Lesson media pipeline (bloom B-N8-02, internal/media, D-81), Go only — no Laravel route.
// Uploads: /api/instructor/v1/media (auth:api + approved instructor, owner-scoped; tus-style resumable chunks).
// Playback URLs: /api/v1/learning/lessons/{id}/playback (student, the lesson's own access rules) and
// /api/instructor/v1/media/{id}/playback (owner). The stream /api/v1/media/{id}/stream is unauthenticated — the
// short-lived HMAC link is the credential — and IP-throttled. MEDIA_URL_KEY; a public development key only for
// APP_ENV local / testing / contract; anywhere else without it playback answers 503 (uploads keep working).
// Files: MEDIA_STORAGE_PATH (its own volume), default STORAGE_PATH/app/private/media.
func init() {
	Register("media", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		writes := writeThrottle(d)
		chunks, stream := func(c fiber.Ctx) error { return c.Next() }, func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			rl := ratelimit.New(d.Cache, clock.Real{})
			chunks = rl.NamedWith("media-chunks", MediaChunkThrottleMax, time.Minute, auth.ThrottleIdentity, writeThrottled)
			stream = rl.Named("media-stream", 600, time.Minute, auth.ThrottleIdentity)
		}

		key, dev, missing := d.Config.Media.ResolveKey(d.Config.App)
		if dev {
			key = media.DevKey()
		}
		if missing {
			d.Logger.Error("MEDIA_URL_KEY is not set: lesson media playback is disabled (503)")
		}
		root := d.Config.Media.Root(d.Config.StoragePath)
		if root == "" {
			d.Logger.Warn("MEDIA_STORAGE_PATH / STORAGE_PATH is not set: lesson media uploads are disabled (503)")
		}
		lsvc := learningService(d, nil)
		svc := media.NewService(media.Options{
			DB: d.DB, Disk: media.NewDisk(root), Signer: media.NewSigner(key), BaseURL: d.Config.App.URL, Logger: d.Logger,
		})
		h := media.NewHandlers(svc, lsvc, clock.Real{})
		ins := learning.NewHandlers(lsvc, clock.Real{}).RequireInstructor
		if root != "" && d.DB != nil {
			ctx, cancel := context.WithCancel(context.Background())
			now := func() time.Time { return time.Now().In(civildate.Tehran) }
			auth.OnLifecycle(r, func() error { go svc.SweepLoop(ctx, now); return nil }, cancel)
		}

		r.Get("/api/v1/learning/lessons/:id/playback", locale, guard, h.LessonPlayback)
		r.Get("/api/v1/media/:id/stream", stream, locale, h.Stream)

		p := "/api/instructor/v1/media"
		r.Post(p, locale, guard, ins, writes, h.Create)
		r.Get(p+"/:id", locale, guard, ins, h.Show)
		r.Patch(p+"/:id", locale, guard, ins, chunks, h.Patch)
		r.Delete(p+"/:id", locale, guard, ins, writes, h.Destroy)
		r.Get(p+"/:id/playback", locale, guard, ins, h.InstructorPlayback)
	})
}

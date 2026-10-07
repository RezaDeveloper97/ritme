package http

import (
	"context"
	stdhttp "net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/learning"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// learningService wires the courses service (sms nil = no SMS: the signup claim only writes grants and inbox rows).
func learningService(d *Deps, sms learning.Sender) *learning.Service {
	langs := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
	return learning.NewService(learning.Options{
		DB: d.DB, SMS: sms, Logger: d.Logger,
		Languages: func(ctx context.Context) []string { return langs.All(ctx).Codes() },
	})
}

// learningSignupHook is the auth signup callback (routes_auth.go): a new number's pending course grants become active.
func learningSignupHook(d *Deps) auth.SignupHook {
	if d.DB == nil {
		return nil
	}
	return learningService(d, nil).SignupHook()
}

// Courses (bloom B-N8-01, D-68), Go only. Students: /api/v1/learning/* (auth:api). Instructors: /api/instructor/v1/*
// (auth:api; everything but /me and /apply also needs an admin-approved instructor — 403 otherwise, never 401). Every
// row is scoped by the caller (a foreign id is a uniform 404); writes are per-user throttled. Phone numbers are never
// logged in full. The «دوره برایت باز شد» SMS outbox is drained by an in-process loop (LEARNING_SMS_PROVIDER, fake by
// default outside production).
func init() {
	Register("learning", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		sender, err := learning.NewSender(d.Config, &stdhttp.Client{Timeout: 30 * time.Second}, d.Logger)
		if err != nil {
			auth.FailStartup(r, d.Logger, err)
			return
		}
		svc := learningService(d, sender)
		h := learning.NewHandlers(svc, clock.Real{})
		writes := writeThrottle(d)
		if d.DB != nil {
			ctx, cancel := context.WithCancel(context.Background())
			now := func() time.Time { return time.Now().In(civildate.Tehran) }
			auth.OnLifecycle(r, func() error { go svc.DispatchLoop(ctx, now); return nil }, cancel)
		}

		s := "/api/v1/learning"
		r.Get(s+"/courses", locale, guard, h.MyCourses)
		r.Get(s+"/courses/:id", locale, guard, h.StudentCourse)
		r.Get(s+"/lessons/:id", locale, guard, h.ShowLesson)
		r.Put(s+"/lessons/:id/progress", locale, guard, writes, h.SaveProgress)
		r.Get(s+"/unlocked/:grant", locale, guard, h.Unlocked)

		p := "/api/instructor/v1"
		ins := h.RequireInstructor
		r.Get(p+"/me", locale, guard, h.Me)
		r.Post(p+"/apply", locale, guard, writes, h.Apply)
		r.Get(p+"/courses", locale, guard, ins, h.Courses)
		r.Post(p+"/courses", locale, guard, ins, writes, h.StoreCourse)
		r.Get(p+"/courses/:id", locale, guard, ins, h.ShowCourse)
		r.Put(p+"/courses/:id", locale, guard, ins, writes, h.UpdateCourse)
		r.Delete(p+"/courses/:id", locale, guard, ins, writes, h.DestroyCourse)
		r.Post(p+"/courses/:id/chapters", locale, guard, ins, writes, h.StoreChapter)
		r.Put(p+"/courses/:id/chapters/:chapter", locale, guard, ins, writes, h.UpdateChapter)
		r.Delete(p+"/courses/:id/chapters/:chapter", locale, guard, ins, writes, h.DestroyChapter)
		r.Post(p+"/courses/:id/lessons", locale, guard, ins, writes, h.StoreLesson)
		r.Put(p+"/courses/:id/lessons/:lesson", locale, guard, ins, writes, h.UpdateLesson)
		r.Delete(p+"/courses/:id/lessons/:lesson", locale, guard, ins, writes, h.DestroyLesson)
		r.Get(p+"/groups", locale, guard, ins, h.Groups)
		r.Post(p+"/groups", locale, guard, ins, writes, h.StoreGroup)
		r.Get(p+"/groups/:id", locale, guard, ins, h.ShowGroup)
		r.Put(p+"/groups/:id", locale, guard, ins, writes, h.UpdateGroup)
		r.Delete(p+"/groups/:id", locale, guard, ins, writes, h.DestroyGroup)
		r.Get(p+"/students", locale, guard, ins, h.Students)
		r.Post(p+"/grants", locale, guard, ins, writes, h.StoreGrants)
		r.Delete(p+"/grants/:id", locale, guard, ins, writes, h.DestroyGrant)
	})
}

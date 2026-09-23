package httpadmin

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

// Cookie base names. With Options.CookieSecure they get the __Host- prefix, which makes
// the browser insist on Secure, Path=/ and no Domain (host-only: the admin host alone).
const (
	sessionCookie = "ritme_admin_session" // HttpOnly: the opaque session id
	csrfCookie    = "ritme_admin_csrf"    // readable: the CSRF token to echo in X-CSRF-Token
)

func (k *Kit) cookieName(base string) string {
	if k.opts.CookieSecure {
		return "__Host-" + base
	}
	return base
}

// SessionCookieName is the session cookie's name.
func (k *Kit) SessionCookieName() string { return k.cookieName(sessionCookie) }

// CSRFCookieName is the CSRF cookie's name.
func (k *Kit) CSRFCookieName() string { return k.cookieName(csrfCookie) }

// SetCookies writes the session and CSRF cookies for sess. Remember-me sessions get a
// 30-day cookie; others are browser-session cookies (the server side still expires
// after IdleTTL of inactivity).
func (k *Kit) SetCookies(c fiber.Ctx, sess *Session) {
	maxAge := 0
	if sess.Remember {
		maxAge = int(RememberTTL / time.Second)
	}
	c.Cookie(&fiber.Cookie{
		Name: k.SessionCookieName(), Value: sess.ID(), Path: "/", MaxAge: maxAge,
		Secure: k.opts.CookieSecure, HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode,
	})
	c.Cookie(&fiber.Cookie{
		Name: k.CSRFCookieName(), Value: sess.CSRF, Path: "/", MaxAge: maxAge,
		Secure: k.opts.CookieSecure, HTTPOnly: false, SameSite: fiber.CookieSameSiteLaxMode,
	})
}

// ClearCookies expires both cookies.
func (k *Kit) ClearCookies(c fiber.Ctx) {
	for _, name := range []string{k.SessionCookieName(), k.CSRFCookieName()} {
		c.Cookie(&fiber.Cookie{
			Name: name, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0),
			Secure: k.opts.CookieSecure, HTTPOnly: name == k.SessionCookieName(),
			SameSite: fiber.CookieSameSiteLaxMode,
		})
	}
}

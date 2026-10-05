# Seo / Redirects (L7-03)

Redirect manager + 404 monitor. `ApplyRedirects` (global middleware wrapping `CanonicalizeUrl`) reads the cached
`RedirectMap` only for non-canonical / legacy URLs (admin rows win over `LegacyUrlMap`) and for 404 responses — live
pages never pay for it. `SaveRedirect` normalises, collapses chains (A→B→C ⇒ A→C) and rejects loops; hits and 404s
go through `Support/HitBuffer` (cache) and `seo:flush-redirect-stats` (every 5 min); `seo:purge-404` (daily, 90 d).
Auto 301s on taxonomy slug changes: `Observers/SlugRedirectObserver` (posts/products/places use their slug history).

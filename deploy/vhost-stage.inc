## stage.ritmeapp.ir — the whole staging stack behind ONE hostname.
##
## Production splits api/adpanell/web across three subdomains, but only
## `stage.ritmeapp.ir` has a DNS record pointing here, so staging routes by
## path instead:
##
##   /up, /api, /storage, /docs          -> stage backend-go (Go)
##   /api/admin/                         -> stage backend-go (Go admin API)
##   /panel                              -> stage admin-web (new Next.js admin)
##   /admin, /admin/*                    -> 301 /panel/ (Blade panel retired)
##   /oauth/*                            -> 404 (Passport's routes; no client uses them)
##   /api/session/flag                   -> stage frontend (Next route handler)
##   everything else                     -> stage frontend (Next.js)
##
## Staging runs Go ONLY since T-M2-28: no Laravel container is needed to serve
## any path. The strangler route groups (deploy/go-routes.inc) still go through
## the `upstream stage_route_<group>` blocks in stage-ssl.conf, all on Go.
##
## Same-origin is a feature here: the browser never makes a cross-origin call,
## so backend/config/cors.php needs no staging entry.
##
## ── How the password gate works ──────────────────────────────────────────────
##
## Staging is private, but Basic auth cannot be the gate *by itself*: there is
## only one Authorization header per request and the app already spends it on
## `Bearer <token>`, so Chrome refuses to overwrite it with the cached Basic
## credentials. Every /api call therefore arrived un-credentialed and nginx
## answered 401 — which the frontend's 401 interceptor read as "session
## expired", wiped the token and bounced back to /signup. Profile saves never
## reached the backend at all. Requests the browser sends with `credentials: omit`
## (the web manifest) failed the same way and re-opened the password dialog on
## every page.
##
## So the gate is a COOKIE, which rides alongside the bearer token:
##
##   1. A plain page load carries no Authorization header, so Basic auth still
##      works there — nginx asks for the password exactly once.
##   2. Any authenticated response (2xx/3xx) hands out `ritme_stage`.
##   3. `$stage_auth_realm` (deploy/stage-gate.sh -> server-only
##      stage-gate.conf) evaluates to `off` once that cookie is present, so
##      every later request — XHR with a bearer token, service-worker fetch,
##      RSC prefetch — passes without any Authorization header of its own.
##
## The same map turns the gate off for container-to-container traffic, which is
## how the Next.js server reaches /api/v1/languages during SSR.
##
## Credentials still live in /opt/ritme/stage.htpasswd (server-only).

client_max_body_size 25m;

auth_basic           $stage_auth_realm;
auth_basic_user_file /etc/nginx/stage.htpasswd;

# Deliberately NOT `always`: nginx only adds this on 2xx/3xx, so a 401 never
# hands out the very cookie that opens the gate. Locations that switch the
# password off define their own add_header, which cancels this inheritance —
# an exempt public file must not mint a session either.
add_header Set-Cookie "ritme_stage=$stage_gate_token; Path=/; Max-Age=31536000; HttpOnly; SameSite=Lax";

# Staging is private: keep it out of every index. It is also the add_header
# the password-exempt locations below repeat, so they still cancel the
# Set-Cookie inheritance WITHOUT stacking a second Cache-Control on top of the
# one Next.js already sends (a duplicate, sometimes contradictory header —
# pwa-audit H-5). Cache policy for app files lives in frontend/next.config.ts.
add_header X-Robots-Tag "noindex, nofollow" always;

# HTTPS only (no includeSubDomains, no preload). Ignored by browsers when this
# vhost is served over plain HTTP (stage-http.conf), so it is safe in both.
add_header Strict-Transport-Security "max-age=31536000" always;

# ── Stage backend (Go) — resolved at REQUEST time ───────────────────────────
# Variable proxy_pass + Docker's embedded DNS, never an upstream{} block: this
# file is loaded by PRODUCTION's proxy, so a missing stage container must 502
# its own paths rather than fail `nginx -t` for the whole shared proxy — and a
# recreated container's new IP is picked up without a reload. (`set` is a
# server-level rewrite directive: it runs for every request whatever its
# position relative to the locations below.)
resolver 127.0.0.11 valid=10s ipv6=off;
set $stage_admin_web_target  http://stage-admin-web:3000;
set $stage_backend_go_target http://stage-backend-go:80;

# Health check must stay reachable for uptime probes and deploy verification.
# Go's /up (X-Backend: go) — the same probe the container HEALTHCHECK uses.
location = /up {
    auth_basic off;
    add_header Cache-Control "no-store";
    proxy_pass $stage_backend_go_target;
    include /etc/nginx/conf.d/proxy-params.inc;
}

# Fetched with `credentials: omit` per spec, so neither the password nor the
# cookie can ever be attached: gated, it 401s forever and Chrome re-opens the
# password dialog on every navigation. It is public app metadata — a name, a
# theme colour and icon paths — so it is served openly.
location = /manifest.webmanifest {
    auth_basic off;
    add_header X-Robots-Tag "noindex, nofollow" always;
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

# The worker script, its precache manifest and the icons it caches. All public
# static assets; keeping them gated breaks service-worker registration, which
# is one of the things staging exists to test.
location ~ ^/(sw\.js|version\.json|offline\.html|favicon\.ico)$ {
    auth_basic off;
    add_header X-Robots-Tag "noindex, nofollow" always;
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

location ^~ /icons/ {
    auth_basic off;
    add_header X-Robots-Tag "noindex, nofollow" always;
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

# The one /api path that is NOT the backend: a Next.js route handler (session
# flag, T-M1-03). An exact match beats the `^~ /api/` prefix below. It stays
# behind the password gate like every other app request. Prod needs nothing:
# web.ritme.app sends everything, /api/ included, to Next (vhost-web.inc).
location = /api/session/flag {
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

# ── Strangler route groups (T-M2-09) ──────────────────────────────────────
# Environment selector and client-IP policy for deploy/go-routes.inc. Stage is
# the edge (stage.ritmeapp.ir resolves straight to this server), so the real
# client is $remote_addr and a forged X-Forwarded-For is DROPPED rather than
# forwarded — the per-IP OTP limits key on its leftmost value.
set $ritme_env        stage;
set $ritme_client_xff $remote_addr;
include /etc/nginx/conf.d/go-routes.inc;

# ── New admin (T-M2-25): admin-web at /panel, its API on Go ─────────────────
# Production gives the admin its own host (adpanell.ritme.app, vhost-admin.inc)
# with admin-web at `/`; staging has one origin, so admin-web is built with
# basePath `/panel` (docker-compose.stage.yml); /admin redirects here. Everything here is same-origin, which is
# what the __Host- session cookie and the CSRF/Origin checks need; backend-go
# answers the admin API only for ADMIN_HOSTS=stage.ritmeapp.ir (Host is $host).
# admin-web's other API call, GET /api/v1/languages, is the `content` group in
# go-routes.inc. Both stay behind the password/cookie gate.
# Both targets are the request-time variables set at the top of this file.

# `^~`, and longer than `/api/`: wins over the plain `/api/` prefix AND skips
# the go-routes.inc regexes (none of which match /api/admin/ anyway). Headers
# are spelled out like go-routes.inc so X-Forwarded-For is the edge value
# (the admin login rate limit keys on it) instead of a forgeable chain.
location ^~ /api/admin/ {
    proxy_pass $stage_backend_go_target;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $ritme_client_xff;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
    proxy_set_header Upgrade           $http_upgrade;
    proxy_set_header Connection        "upgrade";
    proxy_read_timeout 120s;
    proxy_send_timeout 120s;
}

# `/panel` itself is admin-web's root (Next basePath, no trailing slash);
# `/panel/…` is every page and its /panel/_next assets. Not `/panel` as a bare
# prefix, which would also swallow e.g. `/panelists`.
location = /panel  { proxy_pass $stage_admin_web_target; include /etc/nginx/conf.d/proxy-params.inc; }
location ^~ /panel/ { proxy_pass $stage_admin_web_target; include /etc/nginx/conf.d/proxy-params.inc; }

# ── Everything else the backend answers: Go (T-M2-28) ──────────────────────
# Plain prefixes, NOT `^~`: a `^~` match would stop nginx from evaluating the
# route-group regexes above. /api/ catches every /api path no group claims
# (unported ones get Go's JSON 404); /storage/ is normally claimed by the
# `content` group already. Headers spelled out as for /api/admin/ (edge XFF).
location /api/ {
    proxy_pass $stage_backend_go_target;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $ritme_client_xff;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
    proxy_set_header Upgrade           $http_upgrade;
    proxy_set_header Connection        "upgrade";
    proxy_read_timeout 120s;
    proxy_send_timeout 120s;
}
# Doctor share links (B-N6-04): the path token is the bearer secret and the
# report's decryption key, so these requests are never written to the access log.
location ^~ /api/v1/shared-reports/ {
    access_log off;
    proxy_pass $stage_backend_go_target;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $ritme_client_xff;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
}
# Public emergency card (CB-REC-03): the path token is the bearer secret, never logged.
location ^~ /api/v1/emergency-cards/ {
    access_log off;
    proxy_pass $stage_backend_go_target;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $ritme_client_xff;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
}
# Lesson media (B-N8-02). Upload chunks (≤ 16 MiB, under client_max_body_size 25m): streamed to Go without an nginx
# temp file, longer timeouts for slow uplinks.
location ^~ /api/instructor/v1/media {
    proxy_pass $stage_backend_go_target;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $ritme_client_xff;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
    proxy_request_buffering off;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}
# Signed playback stream: the HMAC token is in the query string, so no access log; Range / If-Range pass through and
# the response is not buffered to disk (2 GB files).
location ^~ /api/v1/media/ {
    access_log off;
    proxy_pass $stage_backend_go_target;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $ritme_client_xff;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
    proxy_set_header Range             $http_range;
    proxy_set_header If-Range          $http_if_range;
    proxy_buffering off;
    proxy_read_timeout 300s;
}
location /storage/   { proxy_pass $stage_backend_go_target; include /etc/nginx/conf.d/proxy-params.inc; }
location ^~ /docs    { proxy_pass $stage_backend_go_target; include /etc/nginx/conf.d/proxy-params.inc; }

# Passport's own /oauth/* routes: no client uses them (the app authenticates
# through /api/v1/auth), and Go does not implement them.
#
# Both blocks below answer from the rewrite phase, i.e. BEFORE auth_basic runs,
# so they also answer without credentials. That is harmless for a 404 and a
# redirect, but a 301 is a 3xx and would inherit the server-level Set-Cookie —
# handing the gate cookie to anyone. Each block therefore sets its own
# add_header, which cancels that inheritance (as the exempt locations do).
location ^~ /oauth/ {
    add_header X-Robots-Tag "noindex, nofollow" always;
    return 404;
}

# The Blade panel is retired on stage: its replacement lives at /panel.
# `= /admin` + `^~ /admin/` rather than a bare `/admin` prefix, which would
# also swallow e.g. /administrators (a frontend path, if one ever exists).
location = /admin {
    add_header X-Robots-Tag "noindex, nofollow" always;
    return 301 /panel/;
}
location ^~ /admin/ {
    add_header X-Robots-Tag "noindex, nofollow" always;
    return 301 /panel/;
}

location ~ ^/[A-Za-z-]+/shared/report/ {
    access_log off;
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}
# Public emergency card page (CB-REC-03): bearer token in the path, never logged.
location ~ ^/[A-Za-z-]+/emergency/ {
    access_log off;
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}
location / {
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

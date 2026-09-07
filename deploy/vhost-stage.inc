## stage.ritmeapp.ir — the whole staging stack behind ONE hostname.
##
## Production splits api/adpanell/web across three subdomains, but only
## `stage.ritmeapp.ir` has a DNS record pointing here, so staging routes by
## path instead:
##
##   /api, /oauth, /storage, /up, /docs  -> stage backend (Laravel)
##   /admin                              -> stage backend (Blade panel)
##   everything else                     -> stage frontend (Next.js)
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
## reached Laravel at all. Requests the browser sends with `credentials: omit`
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

# Health check must stay reachable for uptime probes and deploy verification.
location = /up {
    auth_basic off;
    add_header Cache-Control "no-store";
    proxy_pass http://stage_backend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

# Fetched with `credentials: omit` per spec, so neither the password nor the
# cookie can ever be attached: gated, it 401s forever and Chrome re-opens the
# password dialog on every navigation. It is public app metadata — a name, a
# theme colour and icon paths — so it is served openly.
location = /manifest.webmanifest {
    auth_basic off;
    add_header Cache-Control "no-store";
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

# The worker script, its precache manifest and the icons it caches. All public
# static assets; keeping them gated breaks service-worker registration, which
# is one of the things staging exists to test.
location ~ ^/(sw\.js|version\.json|offline\.html|favicon\.ico)$ {
    auth_basic off;
    add_header Cache-Control "no-store";
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

location ^~ /icons/ {
    auth_basic off;
    add_header Cache-Control "public, max-age=3600";
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

location ^~ /api/    { proxy_pass http://stage_backend_upstream; include /etc/nginx/conf.d/proxy-params.inc; }
location ^~ /oauth/  { proxy_pass http://stage_backend_upstream; include /etc/nginx/conf.d/proxy-params.inc; }
location ^~ /storage/{ proxy_pass http://stage_backend_upstream; include /etc/nginx/conf.d/proxy-params.inc; }
location ^~ /admin   { proxy_pass http://stage_backend_upstream; include /etc/nginx/conf.d/proxy-params.inc; }
location ^~ /docs    { proxy_pass http://stage_backend_upstream; include /etc/nginx/conf.d/proxy-params.inc; }

location / {
    proxy_pass http://stage_frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

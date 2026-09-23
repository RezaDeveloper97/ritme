## Strangler route groups — Laravel vs Go, one location per group (T-M2-09).
##
## Included (server context) by BOTH vhost-api.inc (prod, api.ritme.app) and
## vhost-stage.inc (stage.ritmeapp.ir). This file never names a backend: each
## location proxies to the upstream `<env>_route_<group>`, where <env> is the
## `$ritme_env` the including vhost sets (`prod` / `stage`). Those upstream
## blocks live in the environment's own top-level config —
##
##   prod   deploy/proxy-ssl.conf   upstream prod_route_<group>  { server ritme-backend-1:80; }
##   stage  deploy/stage-ssl.conf   upstream stage_route_<group> { server stage-backend:80; }
##
## — and deploy/switch-go-route.sh flips ONE of those lines between the
## Laravel and the Go target, runs `nginx -t` and reloads. So a group moves in
## one environment without the other ever being touched, and the route map
## itself (below) is shared and cannot drift between stage and prod.
##
## Because the upstream name is resolved at request time from a variable,
## nginx looks it up among the defined upstream{} groups (no resolver, no DNS
## at request time); a name with no upstream block would 502 — which is why
## every group has a block in both environments, even while on Laravel.
##
## Matching: these are REGEX locations, so they beat the plain `location /`
## (api vhost) and `location /api/` (stage vhost) prefixes, but lose to exact
## matches such as `= /up` and `= /api/session/flag` and to `^~` prefixes such
## as `/admin`. The first matching regex wins, so the order below matters only
## where two groups overlap (content's /cycle/phase-content vs cycle — cycle
## also excludes it explicitly). Anything under /api/ that no group claims
## (and /oauth, /docs) stays on Laravel through the vhost's own locations.
##
## X-Forwarded-For: the per-IP rate limits (OTP send/verify) key on the
## leftmost XFF value, which a client can forge (T-M2-08 note). The including
## vhost therefore sets `$ritme_client_xff`: stage is the edge, so it is
## `$remote_addr`; prod sits behind a CDN (api.ritme.app resolves to CDN
## edges, not to the server), where $remote_addr is the CDN node, so it stays
## `$proxy_add_x_forwarded_for` until T-M2-26 wires real_ip for the CDN ranges
## (see cutover.md).
##
## Why the headers are spelled out instead of `include proxy-params.inc`:
## nginx SENDS BOTH when the same header is set twice at one level ("a, b"),
## so overriding X-Forwarded-For after the include would still forward the
## client's forged value first. Keep these lines in sync with proxy-params.inc
## (only X-Forwarded-For differs).
##
## The group list must match ROUTE_GROUPS in deploy/switch-go-route.sh.

# content — languages, info, privacy, banners, articles, phase content and the
# public uploads (/storage stays with whichever backend serves content).
location ~ ^/(api/v1/(languages|info|privacy|banners|articles)(/|$)|api/v1/cycle/phase-content(/|$)|storage/) {
    proxy_pass http://${ritme_env}_route_content;
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

# reminders
location ~ ^/api/v1/reminders(/|$) {
    proxy_pass http://${ritme_env}_route_reminders;
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

# healthlog — daily health logs
location ~ ^/api/v1/health-logs(/|$) {
    proxy_pass http://${ritme_env}_route_healthlog;
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

# profile — profile, export, account deletion
location ~ ^/api/v1/(profile|account)(/|$) {
    proxy_pass http://${ritme_env}_route_profile;
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

# pregnancy
location ~ ^/api/v1/pregnancy(/|$) {
    proxy_pass http://${ritme_env}_route_pregnancy;
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

# cycle — status/today/date/month/recalculate and period logs
# (phase-content belongs to `content`).
location ~ ^/api/v1/cycle/(?!phase-content(/|$)) {
    proxy_pass http://${ritme_env}_route_cycle;
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

# messages — daily / mode smart messages
location ~ ^/api/v1/messages(/|$) {
    proxy_pass http://${ritme_env}_route_messages;
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

# home
location ~ ^/api/v1/home(/|$) {
    proxy_pass http://${ritme_env}_route_home;
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

# auth — OTP, user, logout, refresh-session. Moves LAST, after the Laravel
# queue is drained (cutover.md). /oauth/* stays on Laravel.
location ~ ^/api/v1/auth(/|$) {
    proxy_pass http://${ritme_env}_route_auth;
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

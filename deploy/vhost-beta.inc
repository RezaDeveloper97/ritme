## beta.ritme.app — the whole beta stack behind ONE hostname, PUBLIC (real users, no password gate).
##
## Same path routing as staging (vhost-stage.inc), all on Go:
##
##   /up, /api, /storage, /docs          -> beta backend-go
##   /api/admin/                         -> 404 (the admin lives on beta-admin.ritme.app)
##   /panel, /admin                      -> 301 https://beta-admin.ritme.app/
##   /oauth/*                            -> 404
##   /api/session/flag                   -> beta frontend (Next route handler)
##   everything else                     -> beta frontend (Next.js)
##
## Same-origin, so no CORS. Every target is a request-time variable (Docker DNS): this file is loaded by
## PRODUCTION's proxy, so a missing beta container must 502 its own paths, never fail `nginx -t` for everyone.
##
## Forwarded headers are spelled out everywhere (no proxy-params.inc): the client IP is $beta_client_ip
## (beta-ssl.conf — the rightmost X-Forwarded-For entry, the one ArvanCloud appended) and the scheme is always
## https, because ArvanCloud decides the public scheme whatever it used to reach this origin.

client_max_body_size 25m;

# Beta is a pre-release of the product, not the canonical site (web.ritme.app is).
add_header X-Robots-Tag "noindex, nofollow" always;
add_header Strict-Transport-Security "max-age=31536000" always;

resolver 127.0.0.11 valid=10s ipv6=off;
set $beta_backend_go_target http://beta-backend-go:80;
set $beta_frontend_target   http://beta-frontend:3000;

proxy_http_version 1.1;
proxy_set_header Host              $host;
proxy_set_header X-Real-IP         $beta_client_ip;
proxy_set_header X-Forwarded-For   $beta_client_ip;
proxy_set_header X-Forwarded-Proto https;
proxy_set_header X-Forwarded-Host  $host;
proxy_set_header X-Forwarded-Port  443;
proxy_set_header Upgrade           $http_upgrade;
proxy_set_header Connection        "upgrade";
proxy_read_timeout 120s;
proxy_send_timeout 120s;

location = /up {
    # add_header here cancels the server-level ones, so they are repeated.
    add_header Cache-Control "no-store";
    add_header X-Robots-Tag "noindex, nofollow" always;
    add_header Strict-Transport-Security "max-age=31536000" always;
    proxy_pass $beta_backend_go_target;
}

# A Next.js route handler under /api (session flag, T-M1-03); the exact match beats `/api/`.
location = /api/session/flag {
    proxy_pass $beta_frontend_target;
}

# Admin API only on beta-admin.ritme.app (backend-go also refuses it here: ADMIN_HOSTS).
location ^~ /api/admin/ { return 404; }

location /api/ {
    proxy_pass $beta_backend_go_target;
}

# Bearer secrets in the path / query string: never written to the access log.
location ^~ /api/v1/shared-reports/  { access_log off; proxy_pass $beta_backend_go_target; }
location ^~ /api/v1/emergency-cards/ { access_log off; proxy_pass $beta_backend_go_target; }
location ~ ^/api/v1/files/[0-9]+/download$ { access_log off; proxy_pass $beta_backend_go_target; }

# Lesson media: upload chunks streamed without an nginx temp file; signed playback streamed, Range passed through.
location ^~ /api/instructor/v1/media {
    proxy_request_buffering off;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
    proxy_pass $beta_backend_go_target;
}
location ^~ /api/v1/media/ {
    access_log off;
    # Range / If-Range reach Go as the client sent them (no proxy_set_header here: one would cancel the
    # server-level forwarded headers above).
    proxy_buffering off;
    proxy_read_timeout 300s;
    proxy_pass $beta_backend_go_target;
}

location /storage/ { proxy_pass $beta_backend_go_target; }
location ^~ /docs  { proxy_pass $beta_backend_go_target; }

location ^~ /oauth/ { return 404; }
location = /panel   { return 301 https://beta-admin.ritme.app/; }
location ^~ /panel/ { return 301 https://beta-admin.ritme.app/; }
location = /admin   { return 301 https://beta-admin.ritme.app/; }
location ^~ /admin/ { return 301 https://beta-admin.ritme.app/; }

# Public record pages with a bearer token in the path (doctor share, 24h doctor page, emergency card): never logged.
location ~ ^/(?:[A-Za-z-]+/)?(?:shared/report|d|emergency)/ {
    access_log off;
    proxy_pass $beta_frontend_target;
}

location / {
    proxy_pass $beta_frontend_target;
}

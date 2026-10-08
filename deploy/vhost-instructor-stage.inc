## instructor.ritmeapp.ir — the instructor panel on STAGE (bloom B-N8-05).
##
##   /api/      -> stage backend-go (auth OTP /api/v1/auth/*, /api/instructor/v1/*)
##   /api/admin -> 404 (the admin API belongs to stage.ritmeapp.ir/panel only)
##   /up        -> stage backend-go, password-exempt (probes)
##   else       -> stage instructor-web (Next.js)
##
## Same password/cookie gate as stage.ritmeapp.ir (vhost-stage.inc explains why a
## cookie: the app spends the Authorization header on `Bearer <token>`). The
## `ritme_stage` cookie is host-only, so this host asks for the password once
## itself and then mints its own cookie from the same server-only token
## (stage-gate.conf). Upstreams are resolved at REQUEST time (variable + Docker
## DNS): this file is loaded by PRODUCTION's proxy, so a missing stage container
## must 502 here instead of failing `nginx -t` for everyone.

client_max_body_size 25m;

auth_basic           $stage_auth_realm;
auth_basic_user_file /etc/nginx/stage.htpasswd;
add_header Set-Cookie "ritme_stage=$stage_gate_token; Path=/; Max-Age=31536000; HttpOnly; SameSite=Lax";
add_header X-Robots-Tag "noindex, nofollow" always;
add_header Strict-Transport-Security "max-age=31536000" always;

resolver 127.0.0.11 valid=10s ipv6=off;
set $stage_instructor_web_target http://stage-instructor-web:3000;
set $stage_instructor_api_target http://stage-backend-go:80;

location = /up {
    auth_basic off;
    add_header Cache-Control "no-store";
    proxy_pass $stage_instructor_api_target;
    include /etc/nginx/conf.d/proxy-params.inc;
}

location ^~ /api/admin/ {
    add_header X-Robots-Tag "noindex, nofollow" always;
    return 404;
}

# Edge XFF (stage resolves straight to this server): the per-IP OTP limits key on it.
location /api/ {
    proxy_pass $stage_instructor_api_target;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $remote_addr;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
    proxy_read_timeout 120s;
    proxy_send_timeout 120s;
}

location / {
    proxy_pass $stage_instructor_web_target;
    include /etc/nginx/conf.d/proxy-params.inc;
}

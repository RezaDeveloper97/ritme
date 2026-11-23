## beta-admin.ritme.app — the beta admin panel on its own host (like production's adpanell.ritme.app).
##
##   /api/admin/          -> beta backend-go (admin API; ADMIN_HOSTS=beta-admin.ritme.app)
##   /api/v1/languages    -> beta backend-go (admin-web's one public-API call)
##   /storage/            -> beta backend-go (uploaded images the panel previews)
##   other /api/          -> 404 (the app API belongs to beta.ritme.app)
##   everything else      -> beta admin-web (Next.js, served at `/`)
##
## Same-origin for the __Host- session cookie and the CSRF/Origin checks. Targets are request-time variables so a
## down beta stack never fails the shared production proxy's `nginx -t`. Client IP/scheme: see vhost-beta.inc.

# Client IP: $remote_addr is the real client only after cdn-real-ip.inc (ArvanCloud edges trusted, nothing else).
include /etc/nginx/conf.d/cdn-real-ip.inc;

client_max_body_size 25m;

add_header X-Robots-Tag "noindex, nofollow" always;
add_header Strict-Transport-Security "max-age=31536000" always;

resolver 127.0.0.11 valid=10s ipv6=off;
set $beta_backend_go_target http://beta-backend-go:80;
set $beta_admin_web_target  http://beta-admin-web:3000;

proxy_http_version 1.1;
proxy_set_header Host              $host;
proxy_set_header X-Real-IP         $remote_addr;
proxy_set_header X-Forwarded-For   $remote_addr;
proxy_set_header X-Forwarded-Proto https;
proxy_set_header X-Forwarded-Host  $host;
proxy_set_header X-Forwarded-Port  443;
proxy_set_header Upgrade           $http_upgrade;
proxy_set_header Connection        "upgrade";
proxy_read_timeout 120s;
proxy_send_timeout 120s;

location ^~ /api/admin/        { proxy_pass $beta_backend_go_target; }
location ^~ /api/v1/languages  { proxy_pass $beta_backend_go_target; }
location ^~ /api/              { return 404; }
location ^~ /storage/          { proxy_pass $beta_backend_go_target; }

location / {
    proxy_pass $beta_admin_web_target;
}

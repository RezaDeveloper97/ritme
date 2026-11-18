## api.ritme.app — the Laravel API.
## Included by both the HTTP (pre-certificate) and HTTPS server blocks.

client_max_body_size 25m;

# The admin panel lives on its own hostname; don't expose a second door to it.
location ^~ /admin {
    return 404;
}

# ── Strangler route groups (T-M2-09) ──────────────────────────────────────
# deploy/go-routes.inc proxies each group to `upstream prod_route_<group>` in
# proxy-ssl.conf. Every one of those points at Laravel (ritme-backend-1) until
# T-M2-26, so this block is behaviour-neutral today — including XFF, which
# stays `$proxy_add_x_forwarded_for` exactly like proxy-params.inc: prod sits
# behind a CDN, so $remote_addr is the CDN edge, not the client. T-M2-26 must
# configure real_ip for the CDN ranges before switching this to $remote_addr
# (docs/go-migration/cutover.md).
set $ritme_env        prod;
set $ritme_client_xff $proxy_add_x_forwarded_for;
include /etc/nginx/conf.d/go-routes.inc;

# Lesson media (B-N8-02 / B-N8-02b L2) — prefixes (^~) so they win over the go-routes.inc regexes; the same
# blocks stage has in vhost-stage.inc (they are not in go-routes.inc because stage defines its own, and two
# identical locations in one server fail `nginx -t`). The signed stream carries its HMAC token in the query
# string, so it is never written to an access log. Laravel has no such routes (404) — T-M2-26 points both at Go
# (backend-go upstream) at cutover.
location ^~ /api/instructor/v1/media {
    proxy_pass http://backend_upstream;
    # proxy-params.inc inlined: it sets 120s timeouts, and repeating a directive in one location fails `nginx -t`.
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
    proxy_request_buffering off;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}
location ^~ /api/v1/media/ {
    access_log off;
    proxy_pass http://backend_upstream;
    # proxy-params.inc inlined (its 120s read timeout would duplicate the 300s below).
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;
    proxy_send_timeout 120s;
    proxy_set_header Range    $http_range;
    proxy_set_header If-Range $http_if_range;
    proxy_buffering off;
    proxy_read_timeout 300s;
}

# Everything no route group claims (/oauth, /docs, /up, unported /api paths).
location / {
    proxy_pass http://backend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

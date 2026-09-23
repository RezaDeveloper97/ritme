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

# Everything no route group claims (/oauth, /docs, /up, unported /api paths).
location / {
    proxy_pass http://backend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

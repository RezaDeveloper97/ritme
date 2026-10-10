## web.ritme.app — the Next.js frontend.
##
## It talks to https://api.ritme.app directly (baked in at build time via
## NEXT_PUBLIC_API_BASE_URL), so nothing here proxies to the backend.

client_max_body_size 25m;

# Public record pages with a bearer token in the path — doctor share links (B-N6-04), the 24h doctor page `/d/<token>`
# (CB-REC-05) and the emergency card (CB-REC-03), with or without the locale prefix: never written to the access log
# (CB-REC-06 L1).
location ~ ^/(?:[A-Za-z-]+/)?(?:shared/report|d|emergency)/ {
    access_log off;
    proxy_pass http://frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

location / {
    proxy_pass http://frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

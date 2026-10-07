## web.ritme.app — the Next.js frontend.
##
## It talks to https://api.ritme.app directly (baked in at build time via
## NEXT_PUBLIC_API_BASE_URL), so nothing here proxies to the backend.

client_max_body_size 25m;

# Doctor share links (B-N6-04): the token in the path is a bearer secret — no access log.
location ~ ^/[A-Za-z-]+/shared/report/ {
    access_log off;
    proxy_pass http://frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

# Public emergency card page (CB-REC-03): the token in the path is a bearer secret — no access log.
location ~ ^/[A-Za-z-]+/emergency/ {
    access_log off;
    proxy_pass http://frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

location / {
    proxy_pass http://frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

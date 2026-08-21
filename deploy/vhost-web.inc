## web.ritme.app — the Next.js frontend.
##
## It talks to https://api.ritme.app directly (baked in at build time via
## NEXT_PUBLIC_API_BASE_URL), so nothing here proxies to the backend.

client_max_body_size 25m;

location / {
    proxy_pass http://frontend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

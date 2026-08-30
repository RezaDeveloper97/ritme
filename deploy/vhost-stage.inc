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
## The entire vhost is gated by HTTP Basic auth (htpasswd lives at
## /opt/ritme/stage.htpasswd, server-only, never rsynced). Because the API is
## on the same origin as the page, the browser replays the cached credentials
## on every fetch/XHR automatically — no client change is needed.

client_max_body_size 25m;

auth_basic           "Ritme staging";
auth_basic_user_file /etc/nginx/stage.htpasswd;

# Health check must stay reachable for uptime probes and deploy verification.
location = /up {
    auth_basic off;
    proxy_pass http://stage_backend_upstream;
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

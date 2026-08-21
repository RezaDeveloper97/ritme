## adpanell.ritme.app — the Blade admin panel (same backend image, mounted
## because ADMIN_PANEL_ENABLED=true).

# Admin uploads images (banners, article covers), so allow a real payload.
client_max_body_size 25m;

# The API belongs to api.ritme.app only — keep this hostname panel-only.
location ^~ /api/ {
    return 404;
}

location = / {
    return 302 /admin;
}

location / {
    proxy_pass http://backend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

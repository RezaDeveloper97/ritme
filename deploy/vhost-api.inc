## api.ritme.app — the Laravel API.
## Included by both the HTTP (pre-certificate) and HTTPS server blocks.

client_max_body_size 25m;

# The admin panel lives on its own hostname; don't expose a second door to it.
location ^~ /admin {
    return 404;
}

location / {
    proxy_pass http://backend_upstream;
    include /etc/nginx/conf.d/proxy-params.inc;
}

## Shared proxy body of the laravel-site vhosts (included by site-ssl.conf; mounted as conf.d/zz-site-app.inc).

# Admin media uploads go up to 15 MB; join-request forms carry several photos.
client_max_body_size 64m;

resolver 127.0.0.11 valid=10s ipv6=off;
set $site_upstream http://ritme-site-app-1:80;

location / {
    proxy_pass $site_upstream;
    proxy_http_version 1.1;
    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    # Always https: the public scheme is decided by ArvanCloud, whatever it used to reach this origin.
    proxy_set_header X-Forwarded-Proto https;
    proxy_set_header X-Forwarded-Host  $host;
    proxy_set_header X-Forwarded-Port  443;
    proxy_read_timeout 120s;
    proxy_send_timeout 120s;
}

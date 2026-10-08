## instructor.ritme.app — the instructor panel in PRODUCTION (bloom B-N8-05).
##
##   /api/      -> backend-go (ritme-backend-go-1): /api/v1/auth/*, /api/instructor/v1/*
##   /api/admin -> 404 (admin API is not served on this host)
##   else       -> instructor-web (ritme-instructor-web-1)
##
## Both targets are pinned container names resolved at REQUEST time (variable +
## Docker DNS), never upstream{} blocks: until T-M2-26 (COMPOSE_PROFILES=go) neither
## container exists, and an upstream{} would make `nginx -t` fail for the whole
## production proxy. Until then this host simply answers 502.
## XFF: prod sits behind a CDN, so keep the chain like proxy-params.inc
## (docs/go-migration/cutover.md: real_ip before trusting $remote_addr).

client_max_body_size 25m;

add_header X-Robots-Tag "noindex, nofollow" always;

resolver 127.0.0.11 valid=10s ipv6=off;
set $instructor_web_target http://ritme-instructor-web-1:3000;
set $instructor_api_target http://ritme-backend-go-1:80;

location ^~ /api/admin/ {
    return 404;
}

location /api/ {
    proxy_pass $instructor_api_target;
    include /etc/nginx/conf.d/proxy-params.inc;
}

location / {
    proxy_pass $instructor_web_target;
    include /etc/nginx/conf.d/proxy-params.inc;
}

## Shared proxy headers for every location in proxy.conf / proxy-ssl.conf.
## X-Forwarded-Proto is what makes Laravel generate correct absolute URLs
## (and keeps redirects on https once TLS is terminated here).
proxy_http_version 1.1;
proxy_set_header Host              $host;
proxy_set_header X-Real-IP         $remote_addr;
proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Forwarded-Host  $host;
proxy_set_header Upgrade           $http_upgrade;
proxy_set_header Connection        "upgrade";
proxy_read_timeout 120s;
proxy_send_timeout 120s;

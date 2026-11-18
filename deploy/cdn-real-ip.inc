## The real client IP behind ArvanCloud (T-M12-02, findings S-03). Included INSIDE a server block (vhost-beta.inc,
## vhost-beta-admin.inc, vhost-stage.inc), never at http level, so production's own hosts are untouched (T-M12-09).
##
## How it works: nginx rewrites $remote_addr from X-Forwarded-For ONLY when the TCP peer is one of the ArvanCloud
## edge ranges below. ArvanCloud APPENDS the address it saw to the X-Forwarded-For it received, and
## real_ip_recursive walks that list from the right, skipping trusted edges — so $remote_addr becomes the rightmost
## address a client could not have written. A request straight to the origin (:443 is public) comes from a peer that
## is not an edge: its X-Forwarded-For is ignored and $remote_addr stays the peer. The vhosts then forward
## $remote_addr ALONE as X-Real-IP / X-Forwarded-For (never $proxy_add_x_forwarded_for), and backend-go trusts that
## header only from the nginx container (cmd/api TRUSTED_PROXIES).
##
## Fail-safe: an edge missing from this list is not trusted, so its visitors are keyed on the edge address (shared,
## stricter rate limits) — never on a forgeable header. Refresh the list when ArvanCloud announces new ranges.
##
## Source: https://www.arvancloud.ir/en/ips.txt  (fetched 2026-11-18; 14 IPv4 ranges, no IPv6 published)
## Mounted into the proxy as /etc/nginx/conf.d/cdn-real-ip.inc (docker-compose.prod.yml, proxy volumes).

set_real_ip_from 185.143.232.0/22;
set_real_ip_from 188.229.116.16/30;
set_real_ip_from 94.101.182.0/27;
set_real_ip_from 2.144.3.128/28;
set_real_ip_from 37.32.16.0/27;
set_real_ip_from 37.32.17.0/27;
set_real_ip_from 37.32.18.0/27;
set_real_ip_from 37.32.19.0/27;
set_real_ip_from 185.215.232.0/22;
set_real_ip_from 178.131.120.48/28;
set_real_ip_from 94.101.183.0/28;
set_real_ip_from 78.157.36.112/28;
set_real_ip_from 95.38.61.80/28;
set_real_ip_from 193.24.119.0/29;

real_ip_header    X-Forwarded-For;
real_ip_recursive on;

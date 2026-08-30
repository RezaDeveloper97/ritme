## Let's Encrypt HTTP-01 webroot. Every :80 server block includes this so
## issuance and renewal keep working regardless of which config is active.
location ^~ /.well-known/acme-challenge/ {
    # Server-level auth_basic (the staging vhost sets one) is inherited by
    # locations, which would make the ACME challenge unfetchable — and an
    # unfetchable challenge means no certificate. Opt out explicitly. No-op for
    # the production vhosts, which have no auth_basic to inherit.
    auth_basic off;
    root /var/www/certbot;
    default_type "text/plain";
}

## Let's Encrypt HTTP-01 webroot. Every :80 server block includes this so
## issuance and renewal keep working regardless of which config is active.
location ^~ /.well-known/acme-challenge/ {
    root /var/www/certbot;
    default_type "text/plain";
}

# Ritme site — betatest image (deploy-betatest.sh).
# Apache + mod_php 8.2, the same runtime shape as the cPanel target, so `public/.htaccess` is exercised for real.
# The build context is an extracted release package from deploy/build-cpanel.sh (vendor + built assets included),
# so the image needs neither Composer nor Node.
FROM php:8.2-apache-bookworm

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        libicu-dev libzip-dev libpng-dev libjpeg62-turbo-dev libwebp-dev libavif-dev libfreetype6-dev \
        mariadb-client unzip; \
    docker-php-ext-configure gd --with-jpeg --with-webp --with-avif --with-freetype; \
    docker-php-ext-install -j"$(nproc)" gd intl zip exif bcmath pdo_mysql opcache pcntl; \
    a2enmod rewrite headers expires deflate; \
    apt-get purge -y --auto-remove libicu-dev libzip-dev libpng-dev libjpeg62-turbo-dev libwebp-dev libavif-dev libfreetype6-dev; \
    apt-get install -y --no-install-recommends libicu72 libzip4 libpng16-16 libjpeg62-turbo libwebp7 libwebpmux3 libwebpdemux2 libavif15 libfreetype6; \
    rm -rf /var/lib/apt/lists/*

COPY deploy/betatest/php.ini /usr/local/etc/php/conf.d/zz-ritme.ini
COPY deploy/betatest/apache.conf /etc/apache2/sites-available/000-default.conf
RUN sed -ri 's/^ServerTokens .*/ServerTokens Prod/; s/^ServerSignature .*/ServerSignature Off/' /etc/apache2/conf-available/security.conf

WORKDIR /var/www/html
COPY --chown=www-data:www-data release/ /var/www/html/

COPY deploy/betatest/entrypoint.sh /usr/local/bin/ritme-entrypoint
RUN chmod +x /usr/local/bin/ritme-entrypoint \
    && mkdir -p storage/app/private storage/app/pending storage/framework/cache storage/framework/sessions \
        storage/framework/views storage/logs bootstrap/cache public/media \
    && chown -R www-data:www-data storage bootstrap/cache public/media

ENTRYPOINT ["ritme-entrypoint"]
CMD ["apache2-foreground"]


# ─────────────────────────────────────────────────────────────────────────────
# Ritme backend (Laravel 12) — production image.
# Stage 1 installs composer dependencies; stage 2 is Apache + PHP 8.4 with the
# extensions Laravel/Passport/Redis need, serving public/ as the docroot.
# ─────────────────────────────────────────────────────────────────────────────

# ---- Vendor -----------------------------------------------------------------
FROM composer:2 AS vendor
WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-interaction --no-progress --prefer-dist \
    --no-scripts --no-autoloader
COPY . .
RUN composer dump-autoload --optimize

# ---- Runtime ----------------------------------------------------------------
FROM php:8.4-apache AS runtime

# Core extensions via docker-php-ext-install; phpredis built from the GitHub
# source tarball because pecl.php.net sits behind a CDN unreachable from here.
ARG PHPREDIS_VERSION=6.3.0
RUN apt-get update \
 && apt-get install -y --no-install-recommends libicu-dev libzip-dev \
      libjpeg62-turbo-dev libpng-dev libwebp-dev libfreetype6-dev \
 && docker-php-ext-configure gd --with-jpeg --with-webp --with-freetype \
 && docker-php-ext-install -j"$(nproc)" pdo_mysql bcmath opcache pcntl intl zip gd \
 && curl -fsSL "https://github.com/phpredis/phpredis/archive/refs/tags/${PHPREDIS_VERSION}.tar.gz" -o /tmp/phpredis.tar.gz \
 && mkdir -p /usr/src/phpredis \
 && tar xzf /tmp/phpredis.tar.gz -C /usr/src/phpredis --strip-components=1 \
 && cd /usr/src/phpredis && phpize && ./configure && make -j"$(nproc)" && make install \
 && docker-php-ext-enable redis \
 && rm -rf /usr/src/phpredis /tmp/phpredis.tar.gz /var/lib/apt/lists/*

# Serve Laravel's public/ and let its .htaccess handle routing.
ENV APACHE_DOCUMENT_ROOT=/var/www/html/public
RUN a2enmod rewrite \
 && sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf \
 && printf '<Directory ${APACHE_DOCUMENT_ROOT}>\n    AllowOverride All\n    Require all granted\n</Directory>\n' \
      > /etc/apache2/conf-available/laravel.conf \
 && a2enconf laravel

RUN { \
      echo 'opcache.enable=1'; \
      echo 'opcache.memory_consumption=192'; \
      echo 'opcache.max_accelerated_files=20000'; \
      echo 'opcache.validate_timestamps=0'; \
    } > /usr/local/etc/php/conf.d/opcache-prod.ini

# Admin image uploads: PHP's 2M default would reject a normal phone photo
# before validation ever runs, and GD needs headroom to resize one.
RUN { \
      echo 'upload_max_filesize=10M'; \
      echo 'post_max_size=12M'; \
      echo 'memory_limit=256M'; \
    } > /usr/local/etc/php/conf.d/uploads.ini

WORKDIR /var/www/html
COPY --from=vendor --chown=www-data:www-data /app /var/www/html

COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 80
ENTRYPOINT ["entrypoint.sh"]
CMD ["apache2-foreground"]

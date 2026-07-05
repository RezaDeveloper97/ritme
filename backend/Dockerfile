
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

# install-php-extensions handles system deps and phpredis (pecl.php.net is EOL).
ADD https://github.com/mlocati/docker-php-extension-installer/releases/latest/download/install-php-extensions /usr/local/bin/
RUN chmod +x /usr/local/bin/install-php-extensions \
 && install-php-extensions pdo_mysql bcmath opcache pcntl intl zip redis

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

WORKDIR /var/www/html
COPY --from=vendor --chown=www-data:www-data /app /var/www/html

COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 80
ENTRYPOINT ["entrypoint.sh"]
CMD ["apache2-foreground"]

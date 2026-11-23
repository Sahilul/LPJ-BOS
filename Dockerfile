FROM php:8.3-apache

RUN apt-get update \
    && apt-get install -y --no-install-recommends libzip-dev libpng-dev libjpeg62-turbo-dev libfreetype6-dev libsqlite3-dev unzip \
    && docker-php-ext-configure gd --with-freetype --with-jpeg \
    && docker-php-ext-install -j"$(nproc)" pdo_sqlite zip gd \
    && a2enmod rewrite \
    && rm -rf /var/lib/apt/lists/*

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

ENV APACHE_DOCUMENT_ROOT=/var/www/html/public
RUN sed -ri 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf \
    && sed -ri 's!AllowOverride None!AllowOverride All!g' /etc/apache2/apache2.conf

COPY docker/php.ini /usr/local/etc/php/conf.d/lpjbos.ini
COPY docker/entrypoint.sh /usr/local/bin/lpjbos-entrypoint
RUN chmod +x /usr/local/bin/lpjbos-entrypoint

WORKDIR /var/www/html
COPY src/ /var/www/html/
RUN composer install --no-dev --no-interaction --prefer-dist --optimize-autoloader

ENTRYPOINT ["lpjbos-entrypoint"]
CMD ["apache2-foreground"]

#!/bin/sh
set -e

# Saat mode dev, folder src di-mount sehingga vendor dari image tertutup.
if [ ! -f /var/www/html/vendor/autoload.php ]; then
    echo "[lpjbos] vendor belum ada, menjalankan composer install..."
    composer install --no-interaction --prefer-dist --working-dir=/var/www/html
fi

mkdir -p /var/www/data/uploads /var/www/data/tmp
chown -R www-data:www-data /var/www/data

exec docker-php-entrypoint "$@"

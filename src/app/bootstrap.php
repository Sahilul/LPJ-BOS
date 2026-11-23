<?php

declare(strict_types=1);

require __DIR__ . '/../vendor/autoload.php';

date_default_timezone_set('Asia/Jakarta');

foreach (['data_dir', 'upload_dir', 'tmp_dir'] as $dir) {
    if (!is_dir(config($dir))) {
        @mkdir(config($dir), 0775, true);
    }
}

// Endpoint MCP memakai token, bukan sesi/cookie.
$isApi = (parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?: '/') === '/mcp';
if (!$isApi) {
    session_name('lpjbos');
    session_set_cookie_params(['httponly' => true, 'samesite' => 'Lax']);
    session_start();
}

App\Database::migrate();

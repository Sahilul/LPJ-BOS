<?php

$dataDir = rtrim(getenv('DATA_DIR') ?: dirname(__DIR__, 2) . '/data', '/');

return [
    'app_name' => 'LPJ BOS',
    'data_dir' => $dataDir,
    'db_path' => getenv('DB_PATH') ?: $dataDir . '/lpjbos.sqlite',
    'upload_dir' => $dataDir . '/uploads',
    'tmp_dir' => $dataDir . '/tmp',
    'migrations_dir' => dirname(__DIR__) . '/database/migrations',
    'max_image_bytes' => 2 * 1024 * 1024,
];

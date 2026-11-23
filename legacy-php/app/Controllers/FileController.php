<?php

declare(strict_types=1);

namespace App\Controllers;

final class FileController
{
    public function show(string $name): void
    {
        $name = basename($name);
        $path = upload_path($name);
        if (!preg_match('/^[a-z0-9_.-]+$/i', $name) || !is_file($path)) {
            abort(404, 'Berkas tidak ditemukan.');
        }
        $mime = mime_content_type($path) ?: 'application/octet-stream';
        if (!str_starts_with($mime, 'image/') && $mime !== 'application/pdf') {
            $mime = 'application/octet-stream';
        }
        header('Content-Type: ' . $mime);
        header('Content-Length: ' . filesize($path));
        header('Cache-Control: private, max-age=31536000');
        header('X-Content-Type-Options: nosniff');
        readfile($path);
    }
}

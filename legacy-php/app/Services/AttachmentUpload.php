<?php

declare(strict_types=1);

namespace App\Services;

use RuntimeException;

final class AttachmentUpload
{
    private const MAX_BYTES = 2 * 1024 * 1024;
    private const TYPES = [
        'application/pdf' => 'pdf',
        'image/jpeg' => 'jpg',
        'image/png' => 'png',
    ];

    public static function store(array $file): array
    {
        if (($file['error'] ?? UPLOAD_ERR_NO_FILE) === UPLOAD_ERR_NO_FILE) {
            throw new RuntimeException('Pilih berkas nota atau dokumen lebih dulu.');
        }
        if (($file['error'] ?? UPLOAD_ERR_OK) !== UPLOAD_ERR_OK) {
            throw new RuntimeException('Unggahan gagal (kode ' . (int) $file['error'] . ').');
        }
        if ((int) ($file['size'] ?? 0) > self::MAX_BYTES) {
            throw new RuntimeException('Ukuran berkas maksimal 2 MB.');
        }
        if (!is_uploaded_file((string) ($file['tmp_name'] ?? ''))) {
            throw new RuntimeException('Berkas unggahan tidak valid.');
        }

        $mime = (new \finfo(FILEINFO_MIME_TYPE))->file($file['tmp_name']);
        if (!isset(self::TYPES[$mime])) {
            throw new RuntimeException('Gunakan berkas PDF, JPG, atau PNG.');
        }

        $original = preg_replace('/[\x00-\x1F\x7F]/u', '', basename((string) ($file['name'] ?? 'dokumen')));
        $original = substr($original ?: 'dokumen', 0, 180);
        $name = 'lampiran_' . bin2hex(random_bytes(12)) . '.' . self::TYPES[$mime];
        if (!move_uploaded_file($file['tmp_name'], upload_path($name))) {
            throw new RuntimeException('Gagal menyimpan berkas unggahan.');
        }

        return ['path' => $name, 'nama_asli' => $original];
    }
}

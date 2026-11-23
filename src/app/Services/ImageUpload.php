<?php

declare(strict_types=1);

namespace App\Services;

use RuntimeException;

final class ImageUpload
{
    /**
     * Simpan gambar unggahan (PNG/JPG), ubah ukuran bila terlalu lebar,
     * dan kembalikan nama berkas di folder uploads.
     */
    public static function store(array $file, string $prefix, int $maxWidth): string
    {
        if (($file['error'] ?? UPLOAD_ERR_NO_FILE) === UPLOAD_ERR_NO_FILE) {
            throw new RuntimeException('Pilih berkas gambar lebih dulu.');
        }
        if ($file['error'] !== UPLOAD_ERR_OK) {
            throw new RuntimeException('Unggahan gagal (kode ' . $file['error'] . ').');
        }
        if ($file['size'] > config('max_image_bytes')) {
            throw new RuntimeException('Ukuran gambar maksimal 2 MB.');
        }
        $info = @getimagesize($file['tmp_name']);
        $type = $info[2] ?? null;
        if (!in_array($type, [IMAGETYPE_PNG, IMAGETYPE_JPEG], true)) {
            throw new RuntimeException('Gunakan gambar PNG atau JPG.');
        }

        $ext = $type === IMAGETYPE_PNG ? 'png' : 'jpg';
        $name = $prefix . '_' . bin2hex(random_bytes(6)) . '.' . $ext;
        $dest = upload_path($name);

        [$w, $h] = $info;
        if ($w <= $maxWidth) {
            if (!move_uploaded_file($file['tmp_name'], $dest)) {
                throw new RuntimeException('Gagal menyimpan gambar.');
            }
            return $name;
        }

        $src = $type === IMAGETYPE_PNG ? imagecreatefrompng($file['tmp_name']) : imagecreatefromjpeg($file['tmp_name']);
        $nh = (int) round($h * $maxWidth / $w);
        $dst = imagecreatetruecolor($maxWidth, $nh);
        if ($type === IMAGETYPE_PNG) {
            imagealphablending($dst, false);
            imagesavealpha($dst, true);
            imagefill($dst, 0, 0, imagecolorallocatealpha($dst, 0, 0, 0, 127));
        }
        imagecopyresampled($dst, $src, 0, 0, 0, 0, $maxWidth, $nh, $w, $h);
        $type === IMAGETYPE_PNG ? imagepng($dst, $dest, 6) : imagejpeg($dst, $dest, 88);
        imagedestroy($src);
        imagedestroy($dst);
        return $name;
    }

    public static function delete(?string $name): void
    {
        if ($name && is_file(upload_path($name))) {
            @unlink(upload_path($name));
        }
        if ($name) {
            foreach (glob(dirname(upload_path($name)) . '/pdf-cache/' . pathinfo($name, PATHINFO_FILENAME) . '-*') ?: [] as $cache) {
                @unlink($cache);
            }
        }
    }
}

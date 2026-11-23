<?php

declare(strict_types=1);

namespace App\Services;

use Dompdf\Dompdf;
use Dompdf\Options;
use RuntimeException;

/** Bantuan bersama untuk dokumen PDF: gambar kop/TTD/stempel sebagai data URI dan render Dompdf. */
final class Pdf
{
    /** @var array<string, string> data URI per berkas dalam satu request */
    private static array $memo = [];

    /**
     * Gambar untuk PDF sebagai data URI, memakai versi ringan yang di-cache di uploads/pdf-cache.
     * Kop diratakan ke latar putih dan disimpan JPEG (PNG transparan sangat lambat di Dompdf);
     * TTD/stempel tetap PNG transparan tetapi diperkecil.
     */
    public static function imageData(?string $name): ?string
    {
        if (!$name) {
            return null;
        }
        $path = upload_path($name);
        if (!is_file($path)) {
            throw new RuntimeException('Gambar dokumen tidak ditemukan: ' . basename($name));
        }
        $mime = mime_content_type($path);
        if (!in_array($mime, ['image/png', 'image/jpeg'], true)) {
            throw new RuntimeException('Format gambar dokumen harus PNG atau JPEG.');
        }
        $key = $name . '@' . filemtime($path);
        if (isset(self::$memo[$key])) {
            return self::$memo[$key];
        }
        [$file, $mime] = self::optimized($path, $mime, str_starts_with(basename($name), 'kop'));
        $content = file_get_contents($file);
        if ($content === false) {
            throw new RuntimeException('Gambar dokumen tidak dapat dibaca.');
        }
        return self::$memo[$key] = 'data:' . $mime . ';base64,' . base64_encode($content);
    }

    /** @return array{0: string, 1: string} [berkas, mime] */
    private static function optimized(string $path, string $mime, bool $flatten): array
    {
        if (!function_exists('imagecreatefrompng')) {
            return [$path, $mime];
        }
        $maxWidth = $flatten ? 1600 : 600;
        $outMime = $flatten ? 'image/jpeg' : 'image/png';
        $dir = dirname($path) . '/pdf-cache';
        $cache = $dir . '/' . pathinfo($path, PATHINFO_FILENAME) . '-' . filemtime($path) . ($flatten ? '.jpg' : '.png');
        if (is_file($cache)) {
            return [$cache, $outMime];
        }
        [$w, $h] = @getimagesize($path) ?: [0, 0];
        if (!$w || !$h || (!$flatten && $w <= $maxWidth)) {
            return [$path, $mime];
        }
        $src = $mime === 'image/png' ? @imagecreatefrompng($path) : @imagecreatefromjpeg($path);
        if (!$src) {
            return [$path, $mime];
        }
        $nw = min($w, $maxWidth);
        $nh = max(1, (int) round($h * $nw / $w));
        $dst = imagecreatetruecolor($nw, $nh);
        if ($flatten) {
            imagefill($dst, 0, 0, imagecolorallocate($dst, 255, 255, 255));
        } else {
            imagealphablending($dst, false);
            imagesavealpha($dst, true);
            imagefill($dst, 0, 0, imagecolorallocatealpha($dst, 0, 0, 0, 127));
        }
        imagecopyresampled($dst, $src, 0, 0, 0, 0, $nw, $nh, $w, $h);
        if (!is_dir($dir)) {
            @mkdir($dir, 0775, true);
        }
        $ok = $flatten ? @imagejpeg($dst, $cache, 85) : @imagepng($dst, $cache, 6);
        imagedestroy($src);
        imagedestroy($dst);
        return $ok ? [$cache, $outMime] : [$path, $mime];
    }

    public static function official(?array $row): array
    {
        return [
            'nama' => $row['nama'] ?? '',
            'nip' => $row['nip'] ?? '',
            'ttd' => self::imageData($row['ttd_path'] ?? null),
        ];
    }

    /** Kop, stempel, dan pejabat yang menjabat pada $tgl. */
    public static function identity(string $tgl): array
    {
        $s = sekolah();
        return [
            'sekolah' => $s,
            'kop' => self::imageData($s['kop_path'] ?? null),
            'stempel' => self::imageData($s['stempel_path'] ?? null),
            'kepala' => self::official(pejabat_aktif('kepala', $tgl)),
            'bendahara' => self::official(pejabat_aktif('bendahara', $tgl)),
            'lokasiTanggal' => trim(($s['kabupaten'] ?? '') . ', ' . tanggal_indo($tgl), ', '),
        ];
    }

    public static function render(string $html, string $filename, bool $download, string $orientation = 'portrait'): never
    {
        // Lepas kunci sesi agar halaman lain tidak ikut menunggu selama PDF dirender.
        if (session_status() === PHP_SESSION_ACTIVE) {
            session_write_close();
        }
        $filename = preg_replace('/[^A-Za-z0-9._-]/', '-', $filename) ?: 'dokumen.pdf';
        $options = new Options();
        $options->setDefaultFont('DejaVu Sans');
        $options->setIsRemoteEnabled(false);
        $options->setIsPhpEnabled(false);
        $pdf = new Dompdf($options);
        $pdf->loadHtml($html, 'UTF-8');
        $pdf->setPaper('A4', $orientation);
        $pdf->render();
        $pdf->stream($filename, ['Attachment' => $download]);
        exit;
    }
}

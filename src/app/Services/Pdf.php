<?php

declare(strict_types=1);

namespace App\Services;

use Dompdf\Dompdf;
use Dompdf\Options;
use RuntimeException;

/** Bantuan bersama untuk dokumen PDF: gambar kop/TTD/stempel sebagai data URI dan render Dompdf. */
final class Pdf
{
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
        $content = file_get_contents($path);
        if ($content === false) {
            throw new RuntimeException('Gambar dokumen tidak dapat dibaca.');
        }
        return 'data:' . $mime . ';base64,' . base64_encode($content);
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

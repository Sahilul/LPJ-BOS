<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\ImageUpload;
use RuntimeException;

final class SekolahController
{
    public const FIELDS = [
        'nama', 'npsn', 'jenjang', 'status', 'alamat', 'desa', 'kecamatan', 'kabupaten', 'provinsi',
        'kode_pos', 'telepon', 'email', 'npwp', 'bank_nama', 'bank_rekening', 'bank_atas_nama',
    ];

    public const IMAGES = [
        'kop' => ['col' => 'kop_path', 'label' => 'Kop surat', 'width' => 2400],
        'logo' => ['col' => 'logo_path', 'label' => 'Logo', 'width' => 800],
        'stempel' => ['col' => 'stempel_path', 'label' => 'Stempel', 'width' => 800],
    ];

    public function edit(): void
    {
        view('sekolah/edit', ['title' => 'Profil sekolah', 's' => sekolah()]);
    }

    public function update(): void
    {
        $in = [];
        foreach (self::FIELDS as $f) {
            $in[$f] = trim((string) ($_POST[$f] ?? ''));
        }
        $errors = [];
        if ($in['nama'] === '') {
            $errors['nama'] = 'Isi nama sekolah.';
        }
        if ($in['npsn'] !== '' && !preg_match('/^\d{8}$/', $in['npsn'])) {
            $errors['npsn'] = 'NPSN terdiri dari 8 angka.';
        }
        if ($in['email'] !== '' && !filter_var($in['email'], FILTER_VALIDATE_EMAIL)) {
            $errors['email'] = 'Alamat surel tidak valid.';
        }
        if ($errors) {
            with_old($in, $errors);
            redirect('/sekolah');
        }
        $in['updated_at'] = date('Y-m-d H:i:s');
        Database::update('sekolah', $in, 1);
        flash('success', 'Profil sekolah disimpan.');
        redirect('/sekolah');
    }

    public function upload(string $jenis): void
    {
        $cfg = self::IMAGES[$jenis] ?? abort(404);
        try {
            $name = ImageUpload::store($_FILES['gambar'] ?? [], $jenis, $cfg['width']);
        } catch (RuntimeException $e) {
            flash('danger', $cfg['label'] . ': ' . $e->getMessage());
            redirect('/sekolah#gambar');
        }
        ImageUpload::delete(sekolah()[$cfg['col']] ?? null);
        Database::run("UPDATE sekolah SET {$cfg['col']} = ? WHERE id = 1", [$name]);
        flash('success', $cfg['label'] . ' diperbarui.');
        redirect('/sekolah#gambar');
    }

    public function removeImage(string $jenis): void
    {
        $cfg = self::IMAGES[$jenis] ?? abort(404);
        ImageUpload::delete(sekolah()[$cfg['col']] ?? null);
        Database::run("UPDATE sekolah SET {$cfg['col']} = NULL WHERE id = 1");
        flash('success', $cfg['label'] . ' dihapus.');
        redirect('/sekolah#gambar');
    }
}

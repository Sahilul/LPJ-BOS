<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\AttachmentUpload;
use RuntimeException;

final class LampiranController
{
    private const JENIS = ['nota', 'bast', 'lainnya'];

    public function store(string $id): void
    {
        $belanja = $this->belanja($id);
        $jenis = (string) ($_POST['jenis'] ?? 'lainnya');
        if (!in_array($jenis, self::JENIS, true)) {
            flash('danger', 'Pilih jenis lampiran yang tersedia.');
            redirect('/belanja/' . (int) $belanja['id'] . '/edit#lampiran');
        }

        try {
            $file = AttachmentUpload::store($_FILES['berkas'] ?? []);
        } catch (RuntimeException $e) {
            flash('danger', $e->getMessage());
            redirect('/belanja/' . (int) $belanja['id'] . '/edit#lampiran');
        }

        Database::insert('lampiran', $file + ['belanja_id' => (int) $belanja['id'], 'jenis' => $jenis]);
        flash('success', 'Lampiran berhasil diunggah.');
        redirect('/belanja/' . (int) $belanja['id'] . '/edit#lampiran');
    }

    public function destroy(string $id, string $lampiranId): void
    {
        $belanja = $this->belanja($id);
        $file = Database::one('SELECT * FROM lampiran WHERE id = ? AND belanja_id = ?', [(int) $lampiranId, (int) $belanja['id']]);
        if (!$file) {
            abort(404, 'Lampiran tidak ditemukan.');
        }

        $path = upload_path($file['path']);
        if (is_file($path) && !unlink($path)) {
            throw new RuntimeException('Lampiran tidak dapat dihapus dari penyimpanan.');
        }
        Database::run('DELETE FROM lampiran WHERE id = ?', [(int) $lampiranId]);
        flash('success', 'Lampiran dihapus.');
        redirect('/belanja/' . (int) $belanja['id'] . '/edit#lampiran');
    }

    private function belanja(string $id): array
    {
        $tahun = tahun_aktif();
        $row = Database::one(
            'SELECT id, tgl FROM belanja WHERE id = ? AND tgl BETWEEN ? AND ?',
            [(int) $id, "$tahun-01-01", "$tahun-12-31"]
        );
        if (!$row) {
            abort(404, 'Transaksi tidak ditemukan di tahun anggaran yang sedang dibuka.');
        }
        return $row;
    }
}

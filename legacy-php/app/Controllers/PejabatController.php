<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\ImageUpload;
use RuntimeException;

final class PejabatController
{
    public const PERAN = ['kepala' => 'Kepala Sekolah', 'bendahara' => 'Bendahara BOS'];

    public function index(): void
    {
        $rows = Database::all('SELECT * FROM pejabat ORDER BY peran, mulai DESC');
        $aktif = ['kepala' => pejabat_aktif('kepala'), 'bendahara' => pejabat_aktif('bendahara')];
        view('pejabat/index', ['title' => 'Pejabat & tanda tangan', 'rows' => $rows, 'aktif' => $aktif]);
    }

    public function create(): void
    {
        $peran = $_GET['peran'] ?? 'kepala';
        view('pejabat/form', ['title' => 'Tambah pejabat', 'p' => ['peran' => $peran, 'mulai' => date('Y-01-01')]]);
    }

    public function store(): void
    {
        $data = $this->validated('/pejabat/baru');
        $data['ttd_path'] = $this->storeTtd(null, '/pejabat/baru');
        Database::insert('pejabat', $data);
        flash('success', self::PERAN[$data['peran']] . ' ditambahkan.');
        redirect('/pejabat');
    }

    public function edit(string $id): void
    {
        $p = $this->find($id);
        view('pejabat/form', ['title' => 'Ubah pejabat', 'p' => $p]);
    }

    public function update(string $id): void
    {
        $p = $this->find($id);
        $data = $this->validated("/pejabat/{$id}/edit");
        if (!empty($_POST['hapus_ttd'])) {
            ImageUpload::delete($p['ttd_path']);
            $data['ttd_path'] = null;
            $p['ttd_path'] = null;
        }
        $new = $this->storeTtd($p['ttd_path'], "/pejabat/{$id}/edit");
        if ($new !== $p['ttd_path']) {
            ImageUpload::delete($p['ttd_path']);
            $data['ttd_path'] = $new;
        }
        Database::update('pejabat', $data, (int) $id);
        flash('success', 'Data pejabat disimpan.');
        redirect('/pejabat');
    }

    public function destroy(string $id): void
    {
        $p = $this->find($id);
        ImageUpload::delete($p['ttd_path']);
        Database::run('DELETE FROM pejabat WHERE id = ?', [$id]);
        flash('success', 'Pejabat dihapus.');
        redirect('/pejabat');
    }

    private function find(string $id): array
    {
        return Database::one('SELECT * FROM pejabat WHERE id = ?', [(int) $id]) ?: abort(404, 'Pejabat tidak ditemukan.');
    }

    private function validated(string $back): array
    {
        $in = [
            'peran' => $_POST['peran'] ?? '',
            'nama' => trim($_POST['nama'] ?? ''),
            'nip' => digits($_POST['nip'] ?? ''),
            'pangkat' => trim($_POST['pangkat'] ?? ''),
            'mulai' => $_POST['mulai'] ?? '',
            'selesai' => ($_POST['selesai'] ?? '') ?: null,
        ];
        $errors = [];
        if (!isset(self::PERAN[$in['peran']])) {
            $errors['peran'] = 'Pilih peran.';
        }
        if ($in['nama'] === '') {
            $errors['nama'] = 'Isi nama lengkap beserta gelar.';
        }
        if ($in['nip'] !== '' && strlen($in['nip']) !== 18) {
            $errors['nip'] = 'NIP terdiri dari 18 angka.';
        }
        if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $in['mulai'])) {
            $errors['mulai'] = 'Isi tanggal mulai menjabat.';
        }
        if ($in['selesai'] && $in['selesai'] < $in['mulai']) {
            $errors['selesai'] = 'Tanggal selesai tidak boleh sebelum tanggal mulai.';
        }
        if ($errors) {
            with_old($in, $errors);
            redirect($back);
        }
        return $in;
    }

    private function storeTtd(?string $current, string $back): ?string
    {
        if (($_FILES['ttd']['error'] ?? UPLOAD_ERR_NO_FILE) === UPLOAD_ERR_NO_FILE) {
            return $current;
        }
        try {
            return ImageUpload::store($_FILES['ttd'], 'ttd', 900);
        } catch (RuntimeException $e) {
            with_old($_POST, ['ttd' => $e->getMessage()]);
            redirect($back);
        }
    }
}

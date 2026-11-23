<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\MutasiService;
use App\ValidationException;

final class MutasiController
{
    public const JENIS = MutasiService::JENIS;

    public const URAIAN = MutasiService::URAIAN;

    public function index(): void
    {
        $tahun = tahun_aktif();
        $rows = Database::all("SELECT * FROM mutasi_kas WHERE strftime('%Y', tgl) = ? ORDER BY tgl, id", [(string) $tahun]);
        view('mutasi/index', ['title' => 'Mutasi kas', 'rows' => $rows, 'tahun' => $tahun]);
    }

    public function create(): void
    {
        $jenis = isset(self::JENIS[$_GET['jenis'] ?? '']) ? $_GET['jenis'] : 'tarik';
        $tgl = tgl_default();
        $this->form(['tgl' => $tgl, 'no_bukti' => no_bukti_berikut($tgl, 'mut'), 'jenis' => $jenis, 'uraian' => self::URAIAN[$jenis]]);
    }

    public function edit(string $id): void
    {
        $this->form($this->find($id));
    }

    public function store(): void
    {
        $data = $this->validated(null);
        $id = MutasiService::create($data);
        flash('success', self::JENIS[$data['jenis']] . ' ' . rupiah($data['jumlah']) . ' dicatat.');
        redirect('/mutasi#row-' . $id);
    }

    public function update(string $id): void
    {
        $this->find($id);
        $data = $this->validated((int) $id);
        MutasiService::update((int) $id, $data);
        flash('success', 'Mutasi kas diperbarui.');
        redirect('/mutasi#row-' . $id);
    }

    public function destroy(string $id): void
    {
        $row = $this->find($id);
        MutasiService::delete($row);
        flash('success', 'Mutasi kas dihapus.');
        back('/mutasi');
    }

    private function form(array $row): void
    {
        view('mutasi/form', ['title' => !empty($row['id']) ? 'Ubah mutasi kas' : 'Catat mutasi kas', 'row' => $row]);
    }

    private function find(string $id): array
    {
        $row = Database::one('SELECT * FROM mutasi_kas WHERE id = ?', [(int) $id]);
        if (!$row) {
            abort(404, 'Mutasi kas tidak ditemukan.');
        }
        return $row;
    }

    private function validated(?int $id): array
    {
        try {
            return MutasiService::validate($_POST, $id, tahun_aktif());
        } catch (ValidationException $e) {
            with_old($_POST, $e->errors);
            redirect($id ? "/mutasi/$id/edit" : '/mutasi/baru');
        }
    }
}

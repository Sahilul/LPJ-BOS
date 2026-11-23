<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;

final class RencanaPendapatanController
{
    public const TAHAP = PenerimaanController::TAHAP;

    public function index(): void
    {
        $tahun = tahun_aktif();
        $rows = Database::all(
            'SELECT r.*, COALESCE(p.realisasi, 0) AS realisasi, COALESCE(p.n, 0) AS n_penerimaan
             FROM rencana_pendapatan r
             LEFT JOIN (
                SELECT rencana_pendapatan_id, SUM(jumlah) AS realisasi, COUNT(*) AS n
                FROM penerimaan
                WHERE rencana_pendapatan_id IS NOT NULL
                GROUP BY rencana_pendapatan_id
             ) p ON p.rencana_pendapatan_id = r.id
             WHERE r.tahun = ?
             ORDER BY COALESCE(r.target_bulan, 13), r.id',
            [$tahun]
        );
        $totalTarget = array_sum(array_column($rows, 'target_jumlah'));
        $totalRealisasi = array_sum(array_column($rows, 'realisasi'));
        $belumDipetakan = (float) Database::value(
            "SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan
             WHERE strftime('%Y', tgl) = ? AND rencana_pendapatan_id IS NULL",
            [(string) $tahun]
        );

        view('rencana/pendapatan/index', [
            'title' => 'Rencana pendapatan',
            'rows' => $rows,
            'tahun' => $tahun,
            'totalTarget' => $totalTarget,
            'totalRealisasi' => $totalRealisasi,
            'belumDipetakan' => $belumDipetakan,
        ]);
    }

    public function create(): void
    {
        $this->form(['tahun' => tahun_aktif(), 'tahap' => 'Tahap 1']);
    }

    public function edit(string $id): void
    {
        $this->form($this->find($id));
    }

    public function store(): void
    {
        $data = $this->validated(null);
        $id = Database::insert('rencana_pendapatan', $data);
        flash('success', 'Rencana pendapatan “' . $data['uraian'] . '” ditambahkan.');
        redirect('/rencana/pendapatan#row-' . $id);
    }

    public function update(string $id): void
    {
        $existing = $this->find($id);
        $data = $this->validated((int) $id);
        $jumlahAktual = (int) Database::value(
            'SELECT COUNT(*) FROM penerimaan WHERE rencana_pendapatan_id = ?',
            [(int) $id]
        );
        if ($jumlahAktual && $existing['tahap'] !== $data['tahap']) {
            with_old($_POST, ['tahap' => 'Tahap tidak bisa diubah karena sudah dipasangkan dengan penerimaan aktual.']);
            redirect("/rencana/pendapatan/$id/edit");
        }
        Database::update('rencana_pendapatan', $data, (int) $id);
        flash('success', 'Rencana pendapatan diperbarui.');
        redirect('/rencana/pendapatan#row-' . $id);
    }

    public function destroy(string $id): void
    {
        $row = $this->find($id);
        $count = (int) Database::value(
            'SELECT COUNT(*) FROM penerimaan WHERE rencana_pendapatan_id = ?',
            [(int) $id]
        );
        if ($count) {
            flash('danger', 'Rencana ini sudah dipasangkan dengan penerimaan aktual. Lepaskan pasangannya sebelum menghapus.');
        } else {
            Database::run('DELETE FROM rencana_pendapatan WHERE id = ?', [(int) $id]);
            flash('success', 'Rencana pendapatan “' . $row['uraian'] . '” dihapus.');
        }
        redirect('/rencana/pendapatan');
    }

    private function form(array $row): void
    {
        view('rencana/pendapatan/form', [
            'title' => !empty($row['id']) ? 'Ubah rencana pendapatan' : 'Tambah rencana pendapatan',
            'row' => $row,
        ]);
    }

    private function find(string $id): array
    {
        $row = Database::one('SELECT * FROM rencana_pendapatan WHERE id = ?', [(int) $id]);
        if (!$row) {
            abort(404, 'Rencana pendapatan tidak ditemukan.');
        }
        return $row;
    }

    private function validated(?int $id): array
    {
        $p = $_POST;
        $bulanRaw = trim((string) ($p['target_bulan'] ?? ''));
        $data = [
            'tahun' => $id
                ? (int) Database::value('SELECT tahun FROM rencana_pendapatan WHERE id = ?', [$id])
                : tahun_aktif(),
            'tahap' => (string) ($p['tahap'] ?? ''),
            'uraian' => trim((string) ($p['uraian'] ?? '')),
            'target_bulan' => $bulanRaw === '' ? null : (int) $bulanRaw,
            'target_jumlah' => uang($p['target_jumlah'] ?? 0),
        ];
        $errors = [];
        if (!isset(self::TAHAP[$data['tahap']])) {
            $errors['tahap'] = 'Pilih tahap pendapatan.';
        }
        if ($data['uraian'] === '') {
            $errors['uraian'] = 'Nama rencana pendapatan wajib diisi.';
        }
        if ($bulanRaw !== '' && ($data['target_bulan'] < 1 || $data['target_bulan'] > 12)) {
            $errors['target_bulan'] = 'Pilih bulan target yang valid.';
        }
        if ($data['target_jumlah'] <= 0) {
            $errors['target_jumlah'] = 'Target pendapatan harus lebih dari nol.';
        }
        if ($errors) {
            with_old($p, $errors);
            redirect($id ? "/rencana/pendapatan/$id/edit" : '/rencana/pendapatan/baru');
        }
        return $data;
    }
}

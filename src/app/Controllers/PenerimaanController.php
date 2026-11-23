<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\PenerimaanService;
use App\ValidationException;

final class PenerimaanController
{
    public const TAHAP = PenerimaanService::TAHAP;

    public function index(): void
    {
        $tahun = tahun_aktif();
        $rows = Database::all(
            "SELECT p.*, rp.uraian AS rencana_uraian
             FROM penerimaan p
             LEFT JOIN rencana_pendapatan rp ON rp.id = p.rencana_pendapatan_id
             WHERE strftime('%Y', p.tgl) = ?
             ORDER BY p.tgl, p.id",
            [(string) $tahun]
        );
        view('penerimaan/index', ['title' => 'Penerimaan dana', 'rows' => $rows, 'tahun' => $tahun]);
    }

    public function create(): void
    {
        $tahun = tahun_aktif();
        $n = (int) Database::value("SELECT COUNT(*) FROM penerimaan WHERE strftime('%Y', tgl) = ?", [(string) $tahun]);
        $row = [
            'tgl' => tgl_default(),
            'no_bukti' => no_bukti_berikut(tgl_default()),
            'uraian' => 'Penerimaan Dana BOSP Reguler ' . ($n === 0 ? 'Tahap 1' : 'Tahap 2') . " Tahun $tahun",
            'tahap' => $n === 0 ? 'Tahap 1' : 'Tahap 2',
            'rencana_pendapatan_id' => '',
        ];
        $planId = (int) ($_GET['rencana_pendapatan'] ?? 0);
        $plan = $planId ? Database::one('SELECT * FROM rencana_pendapatan WHERE id = ? AND tahun = ?', [$planId, $tahun]) : null;
        if ($plan) {
            $row['rencana_pendapatan_id'] = (int) $plan['id'];
            $row['uraian'] = $plan['uraian'];
            $row['tahap'] = $plan['tahap'];
            $row['jumlah'] = $plan['target_jumlah'];
        }
        $this->form($row);
    }

    public function edit(string $id): void
    {
        $this->form($this->find($id));
    }

    public function store(): void
    {
        $data = $this->validated(null);
        $id = PenerimaanService::create($data);
        flash('success', 'Penerimaan ' . rupiah($data['jumlah']) . ' dicatat.');
        redirect('/bku?bulan=' . (int) substr($data['tgl'], 5, 2) . '#trm-' . $id);
    }

    public function update(string $id): void
    {
        $this->find($id);
        $data = $this->validated((int) $id);
        PenerimaanService::update((int) $id, $data);
        flash('success', 'Penerimaan diperbarui.');
        redirect('/penerimaan#row-' . $id);
    }

    public function destroy(string $id): void
    {
        $row = $this->find($id);
        PenerimaanService::delete($row);
        flash('success', 'Penerimaan “' . $row['uraian'] . '” dihapus.');
        back('/penerimaan');
    }

    private function form(array $row): void
    {
        $tahun = (int) substr((string) ($row['tgl'] ?? tgl_default()), 0, 4);
        view('penerimaan/form', [
            'title' => !empty($row['id']) ? 'Ubah penerimaan' : 'Catat penerimaan',
            'row' => $row,
            'rencanaPendapatan' => Database::all(
                'SELECT r.*, COALESCE(p.realisasi, 0) AS realisasi
                 FROM rencana_pendapatan r
                 LEFT JOIN (
                    SELECT rencana_pendapatan_id, SUM(jumlah) AS realisasi
                    FROM penerimaan
                    WHERE rencana_pendapatan_id IS NOT NULL
                    GROUP BY rencana_pendapatan_id
                 ) p ON p.rencana_pendapatan_id = r.id
                 WHERE r.tahun = ?
                 ORDER BY COALESCE(r.target_bulan, 13), r.id',
                [$tahun]
            ),
        ]);
    }

    private function find(string $id): array
    {
        $row = Database::one('SELECT * FROM penerimaan WHERE id = ?', [(int) $id]);
        if (!$row) {
            abort(404, 'Penerimaan tidak ditemukan.');
        }
        return $row;
    }

    private function validated(?int $id): array
    {
        try {
            return PenerimaanService::validate($_POST, $id, tahun_aktif());
        } catch (ValidationException $e) {
            with_old($_POST, $e->errors);
            redirect($id ? "/penerimaan/$id/edit" : '/penerimaan/baru');
        }
    }
}

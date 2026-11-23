<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\BelanjaService;
use App\ValidationException;

final class BelanjaController
{
    public const METODE = BelanjaService::METODE;
    private const BATAS_PAJAK_BARANG = 2000000;

    public function create(): void
    {
        $rencanaId = (int) ($_GET['rencana'] ?? 0);
        $row = ['tgl' => tgl_default(), 'rencana_id' => $rencanaId ?: '', 'jumlah_periode' => 1, 'metode' => 'tunai', 'items' => [], 'penerima' => [], 'pajak' => []];
        if ($rencanaId && ($r = Database::one('SELECT uraian, satuan, jenis, toko_id FROM rencana_belanja WHERE id = ?', [$rencanaId]))) {
            $row['uraian'] = $r['uraian'];
            $row['satuan_default'] = $r['satuan'];
            $row['toko_id'] = $r['toko_id'];
            if ($r['jenis'] === 'barang') {
                $row['items'] = array_map(fn ($item) => [
                    'nama' => $item['nama'],
                    'volume' => $item['volume'],
                    'satuan' => $item['satuan'],
                    'harga' => $item['harga_satuan'],
                    'package_volume' => $item['volume'],
                ], Database::all('SELECT * FROM rencana_item WHERE rencana_id = ? ORDER BY id', [$rencanaId]));
            } else {
                $row['penerima'] = array_map(fn ($person) => [
                    'penerima_id' => $person['penerima_id'],
                    'volume' => $person['volume'],
                    'satuan' => $person['satuan'],
                    'tarif' => $person['tarif'],
                    'package_volume' => $person['volume'],
                ], Database::all('SELECT * FROM rencana_penerima WHERE rencana_id = ? ORDER BY id', [$rencanaId]));
            }
        }
        $this->form($row);
    }

    public function edit(string $id): void
    {
        $row = $this->find($id);
        $row['jumlah_periode'] = (int) ($row['jumlah_periode'] ?? 1);
        $row['items'] = Database::all('SELECT * FROM item_belanja WHERE belanja_id = ? ORDER BY id', [(int) $id]);
        $row['penerima'] = Database::all('SELECT * FROM penerima_belanja WHERE belanja_id = ? ORDER BY id', [(int) $id]);
        $row['pajak'] = array_column(Database::all('SELECT * FROM pajak WHERE belanja_id = ?', [(int) $id]), null, 'jenis');
        $this->form($row);
    }

    public function store(): void
    {
        [$data, $children] = $this->validated(null);
        $id = BelanjaService::create($data, $children);
        flash('success', 'Belanja ' . rupiah($data['total']) . ' dicatat. Sisa pagu sekarang ' . rupiah(sisa_rencana($data['rencana_id'])) . '.');
        if (!empty($_POST['_lagi'])) {
            redirect('/belanja/baru?rencana=' . $data['rencana_id']);
        }
        redirect('/bku?bulan=' . (int) substr($data['tgl'], 5, 2) . '#bel-' . $id);
    }

    public function update(string $id): void
    {
        $this->find($id);
        [$data, $children] = $this->validated((int) $id);
        BelanjaService::update((int) $id, $data, $children);
        flash('success', 'Belanja diperbarui. Sisa pagu ' . rupiah(sisa_rencana($data['rencana_id'])) . '.');
        redirect('/bku?bulan=' . (int) substr($data['tgl'], 5, 2) . '#bel-' . $id);
    }

    public function destroy(string $id): void
    {
        $row = $this->find($id);
        BelanjaService::delete($row);
        flash('success', 'Belanja ' . $row['no_bukti'] . ' dihapus. Pagu ' . rupiah((float) $row['total']) . ' kembali ke rencana.');
        redirect('/bku?bulan=' . (int) substr($row['tgl'], 5, 2));
    }

    private function form(array $row): void
    {
        $editId = !empty($row['id']) ? (int) $row['id'] : null;
        $tahun = tahun_aktif();
        $rencana = Database::all(
            'SELECT r.id, r.tahun, r.uraian, r.jenis, r.satuan, r.pagu, r.komponen_kode, r.harga_satuan,
                    r.frekuensi_bulan, r.toko_id,
                    r.pagu - COALESCE((SELECT SUM(total) FROM belanja b WHERE b.rencana_id = r.id AND b.id != ?), 0) AS sisa,
                    r.frekuensi_bulan - COALESCE((SELECT SUM(jumlah_periode) FROM belanja b WHERE b.rencana_id = r.id AND b.id != ?), 0) AS periode_sisa
             FROM rencana_belanja r
             WHERE r.tahun = ? OR r.id = ?
             ORDER BY r.komponen_kode, r.id',
            [$editId ?? 0, $editId ?? 0, $tahun, (int) ($row['rencana_id'] ?? 0)]
        );
        $paket = [];
        foreach (Database::all('SELECT * FROM rencana_item ORDER BY id') as $item) {
            $paket[$item['rencana_id']]['items'][] = [
                'nama' => $item['nama'], 'volume' => (float) $item['volume'],
                'satuan' => $item['satuan'], 'harga' => (float) $item['harga_satuan'],
            ];
        }
        foreach (Database::all('SELECT * FROM rencana_penerima ORDER BY id') as $person) {
            $paket[$person['rencana_id']]['penerima'][] = [
                'penerima_id' => (int) $person['penerima_id'], 'volume' => (float) $person['volume'],
                'satuan' => $person['satuan'], 'tarif' => (float) $person['tarif'],
            ];
        }
        foreach ($rencana as $plan) {
            $entries = $plan['jenis'] === 'barang'
                ? ($paket[$plan['id']]['items'] ?? [])
                : ($paket[$plan['id']]['penerima'] ?? []);
            $paket[$plan['id']]['satuan'] = $entries[0]['satuan'] ?? '';
        }

        $packagePeople = Database::all('SELECT penerima_id FROM rencana_penerima WHERE rencana_id = ?', [(int) ($row['rencana_id'] ?? 0)]);
        $ids = array_filter(array_merge(array_column($row['penerima'] ?? [], 'penerima_id'), array_column($packagePeople, 'penerima_id')));
        $in = $ids ? ' OR id IN (' . implode(',', array_map('intval', $ids)) . ')' : '';
        $penerima = [];
        foreach (Database::all("SELECT id, nama, nip, npwp, kategori, golongan, jabatan, tarif_default FROM penerima WHERE aktif = 1$in ORDER BY nama") as $p) {
            $penerima[] = [
                'id' => (int) $p['id'],
                'nama' => $p['nama'],
                'ket' => trim($p['kategori'] . ($p['golongan'] !== '' ? ' · Gol ' . $p['golongan'] : '') . ($p['jabatan'] !== '' ? ' · ' . $p['jabatan'] : '')),
                'tarif' => (float) $p['tarif_default'],
                'pph' => tarif_pph21($p),
            ];
        }

        $toko = Database::all('SELECT id, nama, pkp FROM toko WHERE aktif = 1 OR id = ? ORDER BY nama', [(int) ($row['toko_id'] ?? 0)]);

        view('belanja/form', [
            'title' => $editId ? 'Ubah belanja ' . $row['no_bukti'] : 'Catat belanja',
            'row' => $row,
            'rencana' => $rencana,
            'paket' => $paket,
            'penerimaList' => $penerima,
            'toko' => $toko,
            'lampiran' => $editId ? Database::all('SELECT * FROM lampiran WHERE belanja_id = ? ORDER BY id DESC', [$editId]) : [],
            'saldo' => saldo_kas($row['tgl'] ?? tgl_default(), null, $editId),
            'tarif' => ['PPN' => tarif_pajak('PPN'), 'PPH22' => tarif_pajak('PPH22'), 'PPH23' => tarif_pajak('PPH23')],
            'batas' => self::BATAS_PAJAK_BARANG,
        ]);
    }

    private function find(string $id): array
    {
        $row = Database::one('SELECT * FROM belanja WHERE id = ?', [(int) $id]);
        if (!$row) {
            abort(404, 'Transaksi belanja tidak ditemukan.');
        }
        return $row;
    }

    /** @return array{0: array, 1: array} */
    private function validated(?int $id): array
    {
        try {
            return BelanjaService::validate($_POST, $id, tahun_aktif());
        } catch (ValidationException $e) {
            with_old($_POST, $e->errors);
            flash('danger', $e->getMessage());
            $rencanaId = (int) ($_POST['rencana_id'] ?? 0);
            $rencana = $rencanaId && Database::value('SELECT COUNT(*) FROM rencana_belanja WHERE id = ?', [$rencanaId]);
            redirect($id ? "/belanja/$id/edit" : '/belanja/baru' . ($rencana ? '?rencana=' . $rencanaId : ''));
        }
    }
}

<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\RencanaService;
use App\ValidationException;

final class RencanaController
{
    public const JENIS = RencanaService::JENIS;

    public function index(): void
    {
        $tahun = tahun_aktif();
        $q = trim((string) ($_GET['q'] ?? ''));
        $komp = (string) ($_GET['komponen'] ?? '');

        $where = ['r.tahun = ?'];
        $args = [$tahun];
        if ($komp !== '') {
            $where[] = 'r.komponen_kode = ?';
            $args[] = $komp;
        }
        if ($q !== '') {
            $where[] = '(r.uraian LIKE ? OR r.kegiatan LIKE ? OR r.rekening_kode LIKE ?)';
            array_push($args, "%$q%", "%$q%", "%$q%");
        }
        $rows = Database::all(
            'SELECT r.*, k.nama AS komponen_nama, t.nama AS toko_nama,
                    COALESCE(b.pakai, 0) AS realisasi, COALESCE(b.n, 0) AS n_belanja,
                    COALESCE(p.periode, 0) AS periode_terpakai
             FROM rencana_belanja r
             LEFT JOIN komponen k ON k.kode = r.komponen_kode
             LEFT JOIN toko t ON t.id = r.toko_id
             LEFT JOIN (SELECT rencana_id, SUM(total) AS pakai, COUNT(*) AS n FROM belanja GROUP BY rencana_id) b ON b.rencana_id = r.id
             LEFT JOIN (SELECT rencana_id, SUM(jumlah_periode) AS periode FROM belanja GROUP BY rencana_id) p ON p.rencana_id = r.id
             WHERE ' . implode(' AND ', $where) . '
             ORDER BY r.komponen_kode, r.id',
            $args
        );

        $paket = [];
        foreach (Database::all('SELECT * FROM rencana_item ORDER BY id') as $item) {
            $paket[$item['rencana_id']]['items'][] = $item;
        }
        foreach (Database::all(
            'SELECT rp.*, p.nama FROM rencana_penerima rp LEFT JOIN penerima p ON p.id = rp.penerima_id ORDER BY rp.id'
        ) as $person) {
            $paket[$person['rencana_id']]['penerima'][] = $person;
        }

        view('rencana/index', [
            'title' => 'Rencana belanja',
            'rows' => $rows,
            'tahun' => $tahun,
            'q' => $q,
            'komp' => $komp,
            'komponen' => self::komponenOptions(),
            'paket' => $paket,
        ]);
    }

    public function create(): void
    {
        $this->form(['tahun' => tahun_aktif(), 'jenis' => 'barang', 'frekuensi_bulan' => 1, 'items' => [], 'penerima' => []]);
    }

    public function edit(string $id): void
    {
        $this->form($this->find($id));
    }

    public function store(): void
    {
        [$data, $items, $recipients] = $this->validated(null);
        $id = RencanaService::create($data, $items, $recipients);
        flash('success', 'Paket “' . $data['uraian'] . '” ditambahkan dengan pagu ' . rupiah($data['pagu']) . '.');
        if (!empty($_POST['_lagi'])) {
            redirect('/rencana/baru');
        }
        redirect('/rencana#row-' . $id);
    }

    public function update(string $id): void
    {
        $this->find($id);
        [$data, $items, $recipients] = $this->validated((int) $id);
        RencanaService::update((int) $id, $data, $items, $recipients);
        flash('success', 'Paket “' . $data['uraian'] . '” diperbarui.');
        redirect('/rencana#row-' . $id);
    }

    public function destroy(string $id): void
    {
        $row = $this->find($id);
        try {
            RencanaService::delete((int) $id);
            flash('success', 'Paket “' . $row['uraian'] . '” dihapus.');
        } catch (\RuntimeException $e) {
            flash('danger', $e->getMessage());
        }
        redirect('/rencana');
    }

    public static function komponenOptions(): array
    {
        $out = [];
        foreach (Database::all('SELECT kode, nama FROM komponen ORDER BY urutan, kode') as $k) {
            $out[$k['kode']] = $k['kode'] . ' · ' . $k['nama'];
        }
        return $out;
    }

    private function form(array $row): void
    {
        $id = (int) ($row['id'] ?? 0);
        $row['items'] ??= $id ? Database::all('SELECT * FROM rencana_item WHERE rencana_id = ? ORDER BY id', [$id]) : [];
        $row['penerima'] ??= $id ? Database::all('SELECT * FROM rencana_penerima WHERE rencana_id = ? ORDER BY id', [$id]) : [];
        $ids = array_filter(array_map('intval', array_column($row['penerima'], 'penerima_id')));
        $in = $ids ? ' OR id IN (' . implode(',', $ids) . ')' : '';

        view('rencana/form', [
            'title' => $id ? 'Ubah paket rencana' : 'Tambah paket rencana',
            'row' => $row,
            'komponen' => self::komponenOptions(),
            'rekening' => Database::all('SELECT kode, nama, jenis FROM kode_rekening ORDER BY kode'),
            'toko' => Database::all('SELECT id, nama FROM toko WHERE aktif = 1 OR id = ? ORDER BY nama', [(int) ($row['toko_id'] ?? 0)]),
            'penerimaList' => Database::all("SELECT id, nama, nip, kategori, golongan, jabatan, tarif_default, aktif FROM penerima WHERE aktif = 1$in ORDER BY nama"),
            'terpakai' => (float) Database::value('SELECT COALESCE(SUM(total), 0) FROM belanja WHERE rencana_id = ?', [$id]),
            'periodeTerpakai' => (int) Database::value('SELECT COALESCE(SUM(jumlah_periode), 0) FROM belanja WHERE rencana_id = ?', [$id]),
        ]);
    }

    private function find(string $id): array
    {
        $row = Database::one('SELECT * FROM rencana_belanja WHERE id = ?', [(int) $id]);
        if (!$row) {
            abort(404, 'Paket rencana belanja tidak ditemukan.');
        }
        return $row;
    }

    /** @return array{0: array, 1: array, 2: array} */
    private function validated(?int $id): array
    {
        try {
            return RencanaService::validate($_POST, $id, tahun_aktif());
        } catch (ValidationException $e) {
            with_old($_POST, $e->errors);
            redirect($id ? "/rencana/$id/edit" : '/rencana/baru');
        }
    }
}

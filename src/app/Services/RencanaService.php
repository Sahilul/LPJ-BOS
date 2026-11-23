<?php

declare(strict_types=1);

namespace App\Services;

use App\Database;
use App\ValidationException;

/**
 * Paket rencana belanja (RKAS): validasi, simpan, ubah, hapus.
 * Dipakai bersama oleh form web dan tool MCP supaya aturannya satu.
 */
final class RencanaService
{
    public const JENIS = ['barang' => 'Barang / jasa', 'honor' => 'Honor', 'transport' => 'Transport'];

    public static function find(int $id): ?array
    {
        return Database::one('SELECT * FROM rencana_belanja WHERE id = ?', [$id]) ?: null;
    }

    /** Paket beserta barang/penerima dan sisa pagu & periode. */
    public static function detail(int $id): ?array
    {
        $row = self::find($id);
        if (!$row) {
            return null;
        }
        $row['items'] = Database::all('SELECT nama, volume, satuan, harga_satuan FROM rencana_item WHERE rencana_id = ? ORDER BY id', [$id]);
        $row['penerima'] = Database::all(
            'SELECT rp.penerima_id, p.nama, rp.volume, rp.satuan, rp.tarif
             FROM rencana_penerima rp LEFT JOIN penerima p ON p.id = rp.penerima_id
             WHERE rp.rencana_id = ? ORDER BY rp.id',
            [$id]
        );
        $row['terpakai'] = (float) Database::value('SELECT COALESCE(SUM(total), 0) FROM belanja WHERE rencana_id = ?', [$id]);
        $row['periode_terpakai'] = (int) Database::value('SELECT COALESCE(SUM(jumlah_periode), 0) FROM belanja WHERE rencana_id = ?', [$id]);
        $row['sisa'] = (float) $row['pagu'] - $row['terpakai'];
        $row['periode_sisa'] = (int) $row['frekuensi_bulan'] - $row['periode_terpakai'];
        return $row;
    }

    /**
     * @return array{0: array, 1: array, 2: array} data paket, baris barang, baris penerima
     * @throws ValidationException
     */
    public static function validate(array $p, ?int $id, int $tahun): array
    {
        $frekuensi = (int) ($p['frekuensi_bulan'] ?? 0);
        $jenis = (string) ($p['jenis'] ?? 'barang');
        $bulan = array_values(array_filter(array_map('intval', (array) ($p['jadwal'] ?? [])), fn ($m) => $m >= 1 && $m <= 12));
        $bulan = array_values(array_unique($bulan));
        sort($bulan);
        $tokoId = (int) ($p['toko_id'] ?? 0);
        $toko = $tokoId ? Database::one('SELECT id FROM toko WHERE id = ?', [$tokoId]) : null;
        $data = [
            'tahun' => $id ? (int) Database::value('SELECT tahun FROM rencana_belanja WHERE id = ?', [$id]) : $tahun,
            'komponen_kode' => trim((string) ($p['komponen_kode'] ?? '')),
            'kegiatan' => trim((string) ($p['kegiatan'] ?? '')),
            'rekening_kode' => trim((string) ($p['rekening_kode'] ?? '')),
            'uraian' => trim((string) ($p['uraian'] ?? '')),
            'jenis' => $jenis,
            'frekuensi_bulan' => $frekuensi,
            'toko_id' => $jenis === 'barang' && $toko ? (int) $toko['id'] : null,
            'volume' => $frekuensi,
            'satuan' => 'bulan',
            'harga_satuan' => 0.0,
            'pagu' => 0.0,
            'jadwal_bulan' => implode(',', $bulan),
        ];
        $items = [];
        $recipients = [];
        $perPeriode = 0.0;
        $err = [];

        if ($data['tahun'] < 2000 || $data['tahun'] > 2100) {
            $err['tahun'] = 'Tahun tidak valid.';
        }
        if (!Database::value('SELECT COUNT(*) FROM komponen WHERE kode = ?', [$data['komponen_kode']])) {
            $err['komponen_kode'] = 'Pilih komponen BOSP.';
        }
        if ($data['uraian'] === '') {
            $err['uraian'] = 'Nama paket wajib diisi, misalnya “Belanja ATK”.';
        }
        if (!isset(self::JENIS[$jenis])) {
            $err['jenis'] = 'Pilih jenis paket.';
        }
        if ($frekuensi < 1 || $frekuensi > 12) {
            $err['frekuensi_bulan'] = 'Frekuensi harus antara 1 sampai 12 bulan.';
        }
        if ($jenis === 'barang') {
            foreach ((array) ($p['items'] ?? []) as $item) {
                $nama = trim((string) ($item['nama'] ?? ''));
                $volume = desimal($item['volume'] ?? 0);
                $satuan = trim((string) ($item['satuan'] ?? ''));
                $harga = uang($item['harga'] ?? 0);
                if ($nama === '' && $harga <= 0) {
                    continue;
                }
                if ($nama === '' || $volume <= 0 || $harga <= 0) {
                    $err['items'] = 'Setiap barang perlu nama, jumlah per bulan, dan harga satuan.';
                    continue;
                }
                $items[] = ['nama' => $nama, 'volume' => $volume, 'satuan' => $satuan, 'harga_satuan' => $harga];
                $perPeriode += round($volume * $harga);
            }
            if (!$items && !isset($err['items'])) {
                $err['items'] = 'Tambahkan minimal satu barang atau jasa ke paket.';
            }
            if ($tokoId && !$toko) {
                $err['toko_id'] = 'Pilih toko/penyedia yang valid.';
            }
        } elseif (isset(self::JENIS[$jenis])) {
            $seen = [];
            foreach ((array) ($p['penerima'] ?? []) as $person) {
                $penerimaId = (int) ($person['penerima_id'] ?? 0);
                $volume = desimal($person['volume'] ?? 0);
                $satuan = trim((string) ($person['satuan'] ?? ''));
                $tarif = uang($person['tarif'] ?? 0);
                if (!$penerimaId && $tarif <= 0) {
                    continue;
                }
                $master = $penerimaId ? Database::one('SELECT id FROM penerima WHERE id = ?', [$penerimaId]) : null;
                if (!$master || $volume <= 0 || $tarif <= 0) {
                    $err['penerima'] = 'Setiap baris perlu penerima, volume, dan tarif.';
                    continue;
                }
                if (isset($seen[$penerimaId])) {
                    $err['penerima'] = 'Penerima yang sama tidak boleh ditambahkan dua kali.';
                    continue;
                }
                $seen[$penerimaId] = true;
                $recipients[] = ['penerima_id' => $penerimaId, 'volume' => $volume, 'satuan' => $satuan, 'tarif' => $tarif];
                $perPeriode += round($volume * $tarif);
            }
            if (!$recipients && !isset($err['penerima'])) {
                $err['penerima'] = 'Tambahkan minimal satu penerima ke paket.';
            }
        }

        $data['harga_satuan'] = round($perPeriode);
        $data['pagu'] = round($perPeriode * $frekuensi);
        if ($perPeriode <= 0) {
            $key = $jenis === 'barang' ? 'items' : 'penerima';
            $err[$key] ??= 'Nilai paket harus lebih dari nol.';
        }
        if ($id && !$err) {
            $terpakai = (float) Database::value('SELECT COALESCE(SUM(total), 0) FROM belanja WHERE rencana_id = ?', [$id]);
            if ($data['pagu'] + 0.001 < $terpakai) {
                $err['frekuensi_bulan'] = 'Pagu baru ' . rupiah($data['pagu']) . ' lebih kecil dari yang sudah dibelanjakan (' . rupiah($terpakai) . ').';
            }
            $periodeTerpakai = (int) Database::value('SELECT COALESCE(SUM(jumlah_periode), 0) FROM belanja WHERE rencana_id = ?', [$id]);
            if ($frekuensi < $periodeTerpakai) {
                $err['frekuensi_bulan'] = 'Frekuensi tidak boleh lebih kecil dari periode yang sudah dicatat (' . $periodeTerpakai . ' bulan).';
            }
        }
        if ($err) {
            throw new ValidationException($err, 'Paket rencana belum valid.');
        }
        return [$data, $items, $recipients];
    }

    public static function create(array $data, array $items, array $recipients): int
    {
        return Database::transaction(function () use ($data, $items, $recipients) {
            $id = Database::insert('rencana_belanja', $data);
            self::savePackage($id, $items, $recipients);
            return $id;
        });
    }

    public static function update(int $id, array $data, array $items, array $recipients): void
    {
        Database::transaction(function () use ($id, $data, $items, $recipients) {
            Database::update('rencana_belanja', $data, $id);
            Database::run('DELETE FROM rencana_item WHERE rencana_id = ?', [$id]);
            Database::run('DELETE FROM rencana_penerima WHERE rencana_id = ?', [$id]);
            self::savePackage($id, $items, $recipients);
        });
    }

    /** @throws \RuntimeException bila paket sudah dipakai di BKU */
    public static function delete(int $id): void
    {
        if (Database::value('SELECT COUNT(*) FROM belanja WHERE rencana_id = ?', [$id])) {
            throw new \RuntimeException('Rencana ini sudah dipakai di BKU. Hapus transaksinya dulu bila memang ingin menghapus.');
        }
        Database::run('DELETE FROM rencana_belanja WHERE id = ?', [$id]);
    }

    private static function savePackage(int $id, array $items, array $recipients): void
    {
        foreach ($items as $item) {
            Database::insert('rencana_item', $item + ['rencana_id' => $id]);
        }
        foreach ($recipients as $recipient) {
            Database::insert('rencana_penerima', $recipient + ['rencana_id' => $id]);
        }
    }
}

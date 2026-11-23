<?php

declare(strict_types=1);

namespace App\Mcp;

use App\Database;
use App\ValidationException;
use App\Services\BelanjaService;
use App\Services\MutasiService;
use App\Services\PenerimaanService;
use App\Services\RencanaService;

/**
 * Tool MCP untuk mencatat/mengubah/menghapus data. Validasi memakai service
 * yang sama dengan form web. Tanpa `id` = catat baru, dengan `id` = ubah.
 */
final class ToolsTulis
{
    public static function daftar(): array
    {
        return array_merge(
            self::penerimaan(), self::mutasi(), self::belanja(), self::rencana(), self::rencanaPendapatan()
        );
    }

    private static function p(string $type, string $desc, array $extra = []): array
    {
        return ['type' => $type, 'description' => $desc] + $extra;
    }

    private static function obj(array $props, array $wajib = []): array
    {
        return ['type' => 'object', 'properties' => $props] + ($wajib ? ['required' => $wajib] : []);
    }

    private static function hapusSkema(string $apa): array
    {
        return self::obj(['id' => self::p('integer', "ID $apa yang dihapus.")], ['id']);
    }

    /** Tahun untuk validasi: dari tgl isian, lalu baris lama, lalu param/default. */
    private static function tahunDari(array $a, ?array $lama = null): int
    {
        $tahun = ToolsBaca::tahun($a);
        foreach ([$a['tgl'] ?? null, $lama['tgl'] ?? null] as $tgl) {
            if (is_string($tgl) && valid_date($tgl)) {
                $tahun = (int) substr($tgl, 0, 4);
                break;
            }
        }
        if (!in_array($tahun, tahun_list(), true)) {
            throw new \RuntimeException("Tahun anggaran $tahun belum terdaftar. Tambahkan dulu di menu Tahun anggaran.");
        }
        return $tahun;
    }

    private static function wajibAda(?array $row, string $apa, int $id): array
    {
        if (!$row) {
            throw new \RuntimeException("$apa dengan id $id tidak ditemukan.");
        }
        return $row;
    }

    private static function penerimaan(): array
    {
        return [
            'penerimaan_simpan' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Catat penerimaan dana BOSP (tanpa id) atau ubah penerimaan (dengan id). Tahap: ' . implode(', ', array_keys(PenerimaanService::TAHAP)) . '.',
                'skema' => self::obj([
                    'id' => self::p('integer', 'Isi untuk mengubah data yang sudah ada.'),
                    'tgl' => self::p('string', 'Tanggal YYYY-MM-DD.'),
                    'tahap' => self::p('string', 'Tahap penerimaan.', ['enum' => array_keys(PenerimaanService::TAHAP)]),
                    'uraian' => self::p('string', 'Uraian penerimaan.'),
                    'jumlah' => self::p('number', 'Jumlah rupiah.'),
                    'no_bukti' => self::p('string', 'Nomor bukti (opsional).'),
                    'rencana_pendapatan_id' => self::p('integer', 'ID rencana pendapatan (opsional, tahap harus sama).'),
                ], ['tgl', 'tahap', 'uraian', 'jumlah']),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0) ?: null;
                    $lama = $id ? self::wajibAda(PenerimaanService::find($id), 'Penerimaan', $id) : null;
                    $data = PenerimaanService::validate($a, $id, self::tahunDari($a, $lama));
                    $id ? PenerimaanService::update($id, $data) : $id = PenerimaanService::create($data);
                    return ['ok' => true, 'id' => $id, 'data' => PenerimaanService::find($id)];
                },
            ],
            'penerimaan_hapus' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Hapus satu penerimaan dana.',
                'skema' => self::hapusSkema('penerimaan'),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0);
                    PenerimaanService::delete(self::wajibAda(PenerimaanService::find($id), 'Penerimaan', $id));
                    return ['ok' => true, 'dihapus' => $id];
                },
            ],
        ];
    }

    private static function mutasi(): array
    {
        return [
            'mutasi_simpan' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Catat (tanpa id) atau ubah (dengan id) mutasi kas: tarik tunai, setor ke bank, biaya admin bank, bunga.',
                'skema' => self::obj([
                    'id' => self::p('integer', 'Isi untuk mengubah data yang sudah ada.'),
                    'tgl' => self::p('string', 'Tanggal YYYY-MM-DD.'),
                    'jenis' => self::p('string', 'Jenis mutasi.', ['enum' => array_keys(MutasiService::JENIS)]),
                    'jumlah' => self::p('number', 'Jumlah rupiah.'),
                    'uraian' => self::p('string', 'Uraian (opsional, ada default per jenis).'),
                    'no_bukti' => self::p('string', 'Nomor bukti (opsional).'),
                ], ['tgl', 'jenis', 'jumlah']),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0) ?: null;
                    $lama = $id ? self::wajibAda(MutasiService::find($id), 'Mutasi', $id) : null;
                    $data = MutasiService::validate($a, $id, self::tahunDari($a, $lama));
                    $id ? MutasiService::update($id, $data) : $id = MutasiService::create($data);
                    return ['ok' => true, 'id' => $id, 'data' => MutasiService::find($id)];
                },
            ],
            'mutasi_hapus' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Hapus satu mutasi kas.',
                'skema' => self::hapusSkema('mutasi kas'),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0);
                    MutasiService::delete(self::wajibAda(MutasiService::find($id), 'Mutasi', $id));
                    return ['ok' => true, 'dihapus' => $id];
                },
            ],
        ];
    }

    private static function belanja(): array
    {
        $item = self::obj([
            'nama' => self::p('string', 'Nama barang/jasa.'),
            'volume' => self::p('number', 'Jumlah.'),
            'satuan' => self::p('string', 'Satuan, mis. rim, pcs.'),
            'harga' => self::p('number', 'Harga satuan rupiah.'),
        ], ['nama', 'volume', 'harga']);
        $pen = self::obj([
            'penerima_id' => self::p('integer', 'ID master penerima.'),
            'volume' => self::p('number', 'Volume, mis. jumlah OJ/hari.'),
            'satuan' => self::p('string', 'Satuan, mis. OJ, OK.'),
            'tarif' => self::p('number', 'Tarif per satuan rupiah.'),
        ], ['penerima_id', 'volume', 'tarif']);

        return [
            'belanja_simpan' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Catat (tanpa id) atau ubah (dengan id) transaksi belanja dari satu paket rencana. '
                    . 'Paket jenis barang memakai `items` (+ pajak_ppn/pph22/pph23, toko_id); jenis honor/transport memakai `pen`. '
                    . 'Pajak dihitung otomatis. Ditolak bila melebihi sisa pagu. Saldo kas boleh minus (belanja sebelum dana cair); kuitansi mencatat tanggal lunas saat dana cair.',
                'skema' => self::obj([
                    'id' => self::p('integer', 'Isi untuk mengubah data yang sudah ada.'),
                    'rencana_id' => self::p('integer', 'ID paket rencana belanja.'),
                    'tgl' => self::p('string', 'Tanggal YYYY-MM-DD.'),
                    'jumlah_periode' => self::p('integer', 'Jumlah bulan yang dicakup (1-12).'),
                    'uraian' => self::p('string', 'Uraian transaksi.'),
                    'metode' => self::p('string', 'Cara bayar.', ['enum' => array_keys(BelanjaService::METODE)]),
                    'no_bukti' => self::p('string', 'Nomor bukti (opsional, otomatis bila kosong).'),
                    'toko_id' => self::p('integer', 'ID toko/penyedia (barang).'),
                    'items' => self::p('array', 'Rincian barang/jasa.', ['items' => $item]),
                    'pajak_ppn' => self::p('boolean', 'Harga sudah termasuk PPN.'),
                    'pajak_pph22' => self::p('boolean', 'Pungut PPh 22.'),
                    'pajak_pph23' => self::p('boolean', 'Pungut PPh 23.'),
                    'pen' => self::p('array', 'Penerima honor/transport.', ['items' => $pen]),
                ], ['rencana_id', 'tgl', 'jumlah_periode', 'uraian', 'metode']),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0) ?: null;
                    $lama = $id ? self::wajibAda(BelanjaService::find($id), 'Belanja', $id) : null;
                    [$data, $children] = BelanjaService::validate($a, $id, self::tahunDari($a, $lama));
                    $id ? BelanjaService::update($id, $data, $children) : $id = BelanjaService::create($data, $children);
                    return ['ok' => true, 'id' => $id, 'data' => BelanjaService::detail($id)];
                },
            ],
            'belanja_hapus' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Hapus satu transaksi belanja beserta rincian, pajak, dan lampirannya.',
                'skema' => self::hapusSkema('belanja'),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0);
                    BelanjaService::delete(self::wajibAda(BelanjaService::find($id), 'Belanja', $id));
                    return ['ok' => true, 'dihapus' => $id];
                },
            ],
        ];
    }

    private static function rencana(): array
    {
        $item = self::obj([
            'nama' => self::p('string', 'Nama barang/jasa.'),
            'volume' => self::p('number', 'Jumlah per bulan.'),
            'satuan' => self::p('string', 'Satuan.'),
            'harga' => self::p('number', 'Harga satuan rupiah.'),
        ], ['nama', 'volume', 'harga']);
        $pen = self::obj([
            'penerima_id' => self::p('integer', 'ID master penerima.'),
            'volume' => self::p('number', 'Volume per bulan.'),
            'satuan' => self::p('string', 'Satuan.'),
            'tarif' => self::p('number', 'Tarif rupiah.'),
        ], ['penerima_id', 'volume', 'tarif']);

        return [
            'rencana_simpan' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Buat (tanpa id) atau ubah (dengan id) paket rencana belanja RKAS. '
                    . 'Pagu = nilai per bulan x frekuensi_bulan. Jenis barang memakai `items`, honor/transport memakai `penerima`.',
                'skema' => self::obj([
                    'id' => self::p('integer', 'Isi untuk mengubah paket yang sudah ada.'),
                    'tahun' => self::p('integer', 'Tahun anggaran paket baru (default tahun default).'),
                    'komponen_kode' => self::p('string', 'Kode komponen BOSP (lihat tool komponen).'),
                    'kegiatan' => self::p('string', 'Kegiatan (opsional).'),
                    'rekening_kode' => self::p('string', 'Kode rekening belanja (opsional).'),
                    'uraian' => self::p('string', 'Nama paket, mis. Belanja ATK.'),
                    'jenis' => self::p('string', 'Jenis paket.', ['enum' => array_keys(RencanaService::JENIS)]),
                    'frekuensi_bulan' => self::p('integer', 'Berapa bulan (1-12).'),
                    'jadwal' => self::p('array', 'Bulan rencana pelaksanaan 1-12.', ['items' => ['type' => 'integer']]),
                    'toko_id' => self::p('integer', 'ID toko (barang, opsional).'),
                    'items' => self::p('array', 'Rincian barang per bulan.', ['items' => $item]),
                    'penerima' => self::p('array', 'Penerima honor/transport.', ['items' => $pen]),
                ], ['komponen_kode', 'uraian', 'jenis', 'frekuensi_bulan']),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0) ?: null;
                    if ($id) {
                        self::wajibAda(RencanaService::find($id), 'Rencana', $id);
                    }
                    [$data, $items, $rec] = RencanaService::validate($a, $id, self::tahunDari(['tahun' => $a['tahun'] ?? null]));
                    $id ? RencanaService::update($id, $data, $items, $rec) : $id = RencanaService::create($data, $items, $rec);
                    return ['ok' => true, 'id' => $id, 'data' => RencanaService::detail($id)];
                },
            ],
            'rencana_hapus' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Hapus paket rencana belanja yang belum dipakai di BKU.',
                'skema' => self::hapusSkema('rencana'),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0);
                    self::wajibAda(RencanaService::find($id), 'Rencana', $id);
                    RencanaService::delete($id);
                    return ['ok' => true, 'dihapus' => $id];
                },
            ],
        ];
    }

    private static function rencanaPendapatan(): array
    {
        $cari = fn (int $id): ?array => Database::one('SELECT * FROM rencana_pendapatan WHERE id = ?', [$id]);
        $dipakai = fn (int $id): int => (int) Database::value(
            'SELECT COUNT(*) FROM penerimaan WHERE rencana_pendapatan_id = ?', [$id]
        );

        return [
            'rencana_pendapatan_simpan' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Buat (tanpa id) atau ubah (dengan id) rencana/target pendapatan BOSP per tahap. Tahap: '
                    . implode(', ', array_keys(PenerimaanService::TAHAP)) . '.',
                'skema' => self::obj([
                    'id' => self::p('integer', 'Isi untuk mengubah rencana yang sudah ada.'),
                    'tahun' => self::p('integer', 'Tahun anggaran rencana baru (default tahun default).'),
                    'tahap' => self::p('string', 'Tahap pendapatan.', ['enum' => array_keys(PenerimaanService::TAHAP)]),
                    'uraian' => self::p('string', 'Nama rencana, mis. BOSP Reguler Tahap 2.'),
                    'target_bulan' => self::p('integer', 'Bulan target cair 1-12 (opsional).'),
                    'target_jumlah' => self::p('number', 'Target jumlah rupiah.'),
                ], ['tahap', 'uraian', 'target_jumlah']),
                'handler' => function (array $a) use ($cari, $dipakai): array {
                    $id = (int) ($a['id'] ?? 0) ?: null;
                    $lama = $id ? self::wajibAda($cari($id), 'Rencana pendapatan', $id) : null;
                    $bulan = $a['target_bulan'] ?? null;
                    $data = [
                        'tahun' => $lama ? (int) $lama['tahun'] : self::tahunDari(['tahun' => $a['tahun'] ?? null]),
                        'tahap' => (string) ($a['tahap'] ?? ''),
                        'uraian' => trim((string) ($a['uraian'] ?? '')),
                        'target_bulan' => ($bulan === null || $bulan === '') ? null : (int) $bulan,
                        'target_jumlah' => uang($a['target_jumlah'] ?? 0),
                    ];
                    $err = [];
                    if (!isset(PenerimaanService::TAHAP[$data['tahap']])) {
                        $err['tahap'] = 'Pilih tahap pendapatan.';
                    }
                    if ($data['uraian'] === '') {
                        $err['uraian'] = 'Nama rencana pendapatan wajib diisi.';
                    }
                    if ($data['target_bulan'] !== null && ($data['target_bulan'] < 1 || $data['target_bulan'] > 12)) {
                        $err['target_bulan'] = 'Bulan target harus 1-12.';
                    }
                    if ($data['target_jumlah'] <= 0) {
                        $err['target_jumlah'] = 'Target pendapatan harus lebih dari nol.';
                    }
                    if ($lama && $lama['tahap'] !== $data['tahap'] && $dipakai($id)) {
                        $err['tahap'] = 'Tahap tidak bisa diubah karena sudah dipasangkan dengan penerimaan.';
                    }
                    if ($err) {
                        throw new ValidationException($err);
                    }
                    $id ? Database::update('rencana_pendapatan', $data, $id) : $id = Database::insert('rencana_pendapatan', $data);
                    return ['ok' => true, 'id' => $id, 'data' => $cari($id)];
                },
            ],
            'rencana_pendapatan_hapus' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Hapus rencana pendapatan yang belum dipasangkan dengan penerimaan.',
                'skema' => self::hapusSkema('rencana pendapatan'),
                'handler' => function (array $a) use ($cari, $dipakai): array {
                    $id = (int) ($a['id'] ?? 0);
                    self::wajibAda($cari($id), 'Rencana pendapatan', $id);
                    if ($dipakai($id)) {
                        throw new \RuntimeException('Rencana ini sudah dipasangkan dengan penerimaan. Lepaskan pasangannya dulu.');
                    }
                    Database::run('DELETE FROM rencana_pendapatan WHERE id = ?', [$id]);
                    return ['ok' => true, 'dihapus' => $id];
                },
            ],
        ];
    }
}

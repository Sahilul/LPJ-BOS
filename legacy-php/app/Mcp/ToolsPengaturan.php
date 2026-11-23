<?php

declare(strict_types=1);

namespace App\Mcp;

use App\Controllers\PejabatController;
use App\Controllers\SekolahController;
use App\Database;
use App\Services\ImageUpload;
use App\ValidationException;

/** Tool MCP untuk profil sekolah, pejabat, tahun anggaran, dan tanggal BKU. */
final class ToolsPengaturan
{
    public static function daftar(): array
    {
        return array_merge(self::sekolah(), self::pejabat(), self::tahun(), self::bku());
    }

    private static function p(string $type, string $desc, array $extra = []): array
    {
        return ['type' => $type, 'description' => $desc] + $extra;
    }

    private static function obj(array $props, array $wajib = []): array
    {
        $s = ['type' => 'object', 'properties' => $props ?: (object) []];
        if ($wajib) {
            $s['required'] = $wajib;
        }
        return $s;
    }

    private static function sekolah(): array
    {
        $props = [];
        foreach (SekolahController::FIELDS as $f) {
            $props[$f] = self::p('string', str_replace('_', ' ', $f));
        }
        $props['npsn']['description'] = 'NPSN, 8 angka';
        return [
            'sekolah_ubah' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Ubah profil sekolah. Kirim hanya field yang berubah. Kop, logo, dan stempel diunggah lewat web.',
                'skema' => self::obj($props),
                'handler' => function (array $a): array {
                    $s = sekolah();
                    $in = [];
                    foreach (SekolahController::FIELDS as $f) {
                        $in[$f] = trim((string) (array_key_exists($f, $a) ? $a[$f] : ($s[$f] ?? '')));
                    }
                    $err = [];
                    if ($in['nama'] === '') {
                        $err['nama'] = 'Isi nama sekolah.';
                    }
                    if ($in['npsn'] !== '' && !preg_match('/^\d{8}$/', $in['npsn'])) {
                        $err['npsn'] = 'NPSN terdiri dari 8 angka.';
                    }
                    if ($in['email'] !== '' && !filter_var($in['email'], FILTER_VALIDATE_EMAIL)) {
                        $err['email'] = 'Alamat surel tidak valid.';
                    }
                    if ($err) {
                        throw new ValidationException($err);
                    }
                    $in['updated_at'] = date('Y-m-d H:i:s');
                    Database::update('sekolah', $in, 1);
                    return ['ok' => true, 'sekolah' => Database::one('SELECT * FROM sekolah WHERE id = 1')];
                },
            ],
        ];
    }

    private static function pejabat(): array
    {
        $peran = array_keys(PejabatController::PERAN);
        return [
            'pejabat_list' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Daftar semua pejabat (kepala sekolah, bendahara) beserta masa jabatan.',
                'skema' => self::obj([]),
                'handler' => fn (array $a): array => Database::all(
                    'SELECT id, peran, nama, nip, pangkat, mulai, selesai, ttd_path IS NOT NULL AS ada_ttd FROM pejabat ORDER BY peran, mulai DESC'
                ),
            ],
            'pejabat_simpan' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Tambah (tanpa id) atau ubah (dengan id) pejabat. Saat ubah cukup kirim field yang berubah. Tanda tangan diunggah lewat web.',
                'skema' => self::obj([
                    'id' => self::p('integer', 'ID pejabat untuk mengubah.'),
                    'peran' => self::p('string', 'Peran.', ['enum' => $peran]),
                    'nama' => self::p('string', 'Nama lengkap beserta gelar.'),
                    'nip' => self::p('string', 'NIP 18 angka, boleh kosong.'),
                    'pangkat' => self::p('string', 'Pangkat/golongan.'),
                    'mulai' => self::p('string', 'Tanggal mulai menjabat YYYY-MM-DD.'),
                    'selesai' => self::p('string', 'Tanggal selesai YYYY-MM-DD; kosong = masih menjabat.'),
                ]),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0) ?: null;
                    $lama = [];
                    if ($id && !($lama = Database::one('SELECT * FROM pejabat WHERE id = ?', [$id]))) {
                        throw new \RuntimeException("Pejabat dengan id $id tidak ditemukan.");
                    }
                    $ambil = fn (string $k) => array_key_exists($k, $a) ? $a[$k] : ($lama[$k] ?? '');
                    $in = [
                        'peran' => (string) $ambil('peran'),
                        'nama' => trim((string) $ambil('nama')),
                        'nip' => digits((string) $ambil('nip')),
                        'pangkat' => trim((string) $ambil('pangkat')),
                        'mulai' => (string) $ambil('mulai'),
                        'selesai' => ((string) $ambil('selesai')) ?: null,
                    ];
                    $err = [];
                    if (!isset(PejabatController::PERAN[$in['peran']])) {
                        $err['peran'] = 'Pilih kepala atau bendahara.';
                    }
                    if ($in['nama'] === '') {
                        $err['nama'] = 'Isi nama lengkap beserta gelar.';
                    }
                    if ($in['nip'] !== '' && strlen($in['nip']) !== 18) {
                        $err['nip'] = 'NIP terdiri dari 18 angka.';
                    }
                    if (!valid_date($in['mulai'])) {
                        $err['mulai'] = 'Isi tanggal mulai menjabat (YYYY-MM-DD).';
                    }
                    if ($in['selesai'] !== null && !valid_date($in['selesai'])) {
                        $err['selesai'] = 'Format tanggal YYYY-MM-DD.';
                    } elseif ($in['selesai'] && $in['selesai'] < $in['mulai']) {
                        $err['selesai'] = 'Tanggal selesai tidak boleh sebelum tanggal mulai.';
                    }
                    if ($err) {
                        throw new ValidationException($err);
                    }
                    if ($id) {
                        Database::update('pejabat', $in, $id);
                    } else {
                        $id = Database::insert('pejabat', $in);
                    }
                    return ['ok' => true, 'id' => $id, 'aktif' => [
                        'kepala' => pejabat_aktif('kepala'),
                        'bendahara' => pejabat_aktif('bendahara'),
                    ]];
                },
            ],
            'pejabat_hapus' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Hapus pejabat beserta gambar tanda tangannya.',
                'skema' => self::obj(['id' => self::p('integer', 'ID pejabat.')], ['id']),
                'handler' => function (array $a): array {
                    $id = (int) ($a['id'] ?? 0);
                    $p = Database::one('SELECT * FROM pejabat WHERE id = ?', [$id]);
                    if (!$p) {
                        throw new \RuntimeException("Pejabat dengan id $id tidak ditemukan.");
                    }
                    ImageUpload::delete($p['ttd_path']);
                    Database::run('DELETE FROM pejabat WHERE id = ?', [$id]);
                    return ['ok' => true, 'dihapus' => $id];
                },
            ],
        ];
    }

    private static function cekTahun(array $a): int
    {
        $t = (int) ($a['tahun'] ?? 0);
        if (!in_array($t, tahun_list(), true)) {
            throw new \RuntimeException("Tahun anggaran $t tidak terdaftar.");
        }
        return $t;
    }

    private static function jadikanDefault(int $t): void
    {
        Database::transaction(function () use ($t) {
            Database::run('UPDATE tahun_anggaran SET is_default = 0');
            Database::run('UPDATE tahun_anggaran SET is_default = 1 WHERE tahun = ?', [$t]);
            Database::run('UPDATE sekolah SET tahun_anggaran = ? WHERE id = 1', [$t]);
        });
    }

    private static function tahun(): array
    {
        $tahun = self::p('integer', 'Tahun anggaran, mis. 2026.');
        return [
            'tahun_tambah' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Tambah tahun anggaran baru.',
                'skema' => self::obj([
                    'tahun' => $tahun,
                    'keterangan' => self::p('string', 'Keterangan singkat.'),
                    'jadikan_default' => self::p('boolean', 'Jadikan tahun default.'),
                ], ['tahun']),
                'handler' => function (array $a): array {
                    $t = (int) ($a['tahun'] ?? 0);
                    if ($t < 2000 || $t > 2100) {
                        throw new ValidationException(['tahun' => 'Isi tahun empat angka antara 2000 dan 2100.']);
                    }
                    if (in_array($t, tahun_list(), true)) {
                        throw new ValidationException(['tahun' => "Tahun $t sudah ada."]);
                    }
                    $pertama = !tahun_list();
                    Database::insert('tahun_anggaran', [
                        'tahun' => $t,
                        'keterangan' => trim((string) ($a['keterangan'] ?? '')),
                        'is_default' => $pertama ? 1 : 0,
                    ]);
                    if (!empty($a['jadikan_default'])) {
                        self::jadikanDefault($t);
                    }
                    return ['ok' => true, 'tahun' => $t, 'default' => tahun_default()];
                },
            ],
            'tahun_ubah' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Ubah keterangan tahun anggaran dan/atau jadikan tahun default.',
                'skema' => self::obj([
                    'tahun' => $tahun,
                    'keterangan' => self::p('string', 'Keterangan baru.'),
                    'jadikan_default' => self::p('boolean', 'true = jadikan tahun default.'),
                ], ['tahun']),
                'handler' => function (array $a): array {
                    $t = self::cekTahun($a);
                    if (array_key_exists('keterangan', $a)) {
                        Database::run('UPDATE tahun_anggaran SET keterangan = ? WHERE tahun = ?', [trim((string) $a['keterangan']), $t]);
                    }
                    if (!empty($a['jadikan_default'])) {
                        self::jadikanDefault($t);
                    }
                    return ['ok' => true, 'tahun' => $t, 'default' => tahun_default()];
                },
            ],
            'tahun_hapus' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Hapus tahun anggaran yang masih kosong dan bukan default.',
                'skema' => self::obj(['tahun' => $tahun], ['tahun']),
                'handler' => function (array $a): array {
                    $t = self::cekTahun($a);
                    $y = (string) $t;
                    if ((int) Database::value('SELECT is_default FROM tahun_anggaran WHERE tahun = ?', [$t])) {
                        throw new \RuntimeException("Tahun $t adalah tahun default. Jadikan tahun lain default dulu.");
                    }
                    $dipakai = (int) Database::value('SELECT COUNT(*) FROM rencana_belanja WHERE tahun = ?', [$t])
                        + (int) Database::value("SELECT COUNT(*) FROM belanja WHERE strftime('%Y', tgl) = ?", [$y])
                        + (int) Database::value("SELECT COUNT(*) FROM penerimaan WHERE strftime('%Y', tgl) = ?", [$y])
                        + (int) Database::value("SELECT COUNT(*) FROM mutasi_kas WHERE strftime('%Y', tgl) = ?", [$y]);
                    if ($dipakai) {
                        throw new \RuntimeException("Tahun $t sudah berisi data, jadi tidak bisa dihapus.");
                    }
                    Database::run('DELETE FROM tahun_anggaran WHERE tahun = ?', [$t]);
                    return ['ok' => true, 'dihapus' => $t];
                },
            ],
        ];
    }

    private static function bku(): array
    {
        $tables = ['trm' => 'penerimaan', 'bel' => 'belanja', 'mut' => 'mutasi_kas'];
        return [
            'bku_ubah_tanggal' => [
                'akses' => ApiAuth::TULIS,
                'deskripsi' => 'Pindahkan tanggal satu baris BKU (penerimaan/belanja/mutasi) di tahun yang sama. Nomor bukti dan saldo disusun ulang otomatis.',
                'skema' => self::obj([
                    'jenis' => self::p('string', 'trm = penerimaan, bel = belanja, mut = mutasi kas.', ['enum' => array_keys($tables)]),
                    'id' => self::p('integer', 'ID transaksi.'),
                    'tgl' => self::p('string', 'Tanggal baru YYYY-MM-DD.'),
                ], ['jenis', 'id', 'tgl']),
                'handler' => function (array $a) use ($tables): array {
                    $src = (string) ($a['jenis'] ?? '');
                    if (!isset($tables[$src])) {
                        throw new ValidationException(['jenis' => 'Pilih trm, bel, atau mut.']);
                    }
                    $table = $tables[$src];
                    $id = (int) ($a['id'] ?? 0);
                    $row = Database::one("SELECT id, tgl FROM $table WHERE id = ?", [$id]);
                    if (!$row) {
                        throw new \RuntimeException("Transaksi $src id $id tidak ditemukan.");
                    }
                    $tgl = (string) ($a['tgl'] ?? '');
                    if (!valid_date($tgl)) {
                        throw new ValidationException(['tgl' => 'Tanggal tidak valid (YYYY-MM-DD).']);
                    }
                    if (substr($tgl, 0, 4) !== substr($row['tgl'], 0, 4)) {
                        throw new ValidationException(['tgl' => 'Tanggal harus di tahun ' . substr($row['tgl'], 0, 4) . '.']);
                    }
                    Database::transaction(function () use ($src, $table, $id, $row, $tgl) {
                        Database::run("UPDATE $table SET tgl = ? WHERE id = ?", [$tgl, $id]);
                        if ($src === 'bel') {
                            Database::run(
                                'UPDATE pajak SET tgl_pungut = ? WHERE belanja_id = ? AND tgl_pungut = ?',
                                [$tgl, $id, $row['tgl']]
                            );
                        }
                        susun_nomor_bukti_bku((int) substr($tgl, 0, 4));
                    });
                    return ['ok' => true, 'jenis' => $src, 'id' => $id, 'tgl' => $tgl];
                },
            ],
        ];
    }
}

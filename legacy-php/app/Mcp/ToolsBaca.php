<?php

declare(strict_types=1);

namespace App\Mcp;

use App\Database;

/** Tool MCP untuk membaca data: status, master, rencana, penerimaan, BKU, laporan. */
final class ToolsBaca
{
    public static function daftar(): array
    {
        return array_merge(self::ringkasan(), self::master(), self::keuangan(), self::laporan());
    }

    private static function obj(array $props, array $wajib = []): array
    {
        $s = ['type' => 'object', 'properties' => $props ?: (object) []];
        if ($wajib) {
            $s['required'] = $wajib;
        }
        return $s;
    }

    private static function s(string $desc): array
    {
        return ['type' => 'string', 'description' => $desc];
    }

    private static function i(string $desc): array
    {
        return ['type' => 'integer', 'description' => $desc];
    }

    /** Tahun anggaran aktif untuk tool ini: param tahun, atau tahun default. */
    public static function tahun(array $a): int
    {
        $t = (int) ($a['tahun'] ?? 0);
        return $t >= 2000 && $t <= 2100 ? $t : tahun_default();
    }

    private static function ringkasan(): array
    {
        return [
            'status' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Ringkasan aplikasi: profil sekolah, tahun anggaran, pagu, realisasi, penerimaan, dan saldo kas.',
                'skema' => self::obj(['tahun' => self::i('Tahun anggaran, default tahun default aplikasi.')]),
                'handler' => function (array $a): array {
                    $tahun = self::tahun($a);
                    $s = sekolah();
                    $pagu = (float) Database::value('SELECT COALESCE(SUM(pagu), 0) FROM rencana_belanja WHERE tahun = ?', [$tahun]);
                    $realisasi = (float) Database::value(
                        'SELECT COALESCE(SUM(b.total), 0) FROM belanja b JOIN rencana_belanja r ON r.id = b.rencana_id WHERE r.tahun = ?',
                        [$tahun]
                    );
                    $diterima = (float) Database::value("SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan WHERE strftime('%Y', tgl) = ?", [(string) $tahun]);
                    $saldo = saldo_kas($tahun . '-12-31');
                    return [
                        'sekolah' => ['nama' => $s['nama'] ?? '', 'npsn' => $s['npsn'] ?? '', 'jenjang' => $s['jenjang'] ?? ''],
                        'tahun' => $tahun,
                        'pagu' => $pagu,
                        'realisasi' => $realisasi,
                        'sisa_pagu' => round($pagu - $realisasi, 2),
                        'penerimaan' => $diterima,
                        'saldo_kas' => $saldo,
                        'jumlah' => [
                            'rencana' => (int) Database::value('SELECT COUNT(*) FROM rencana_belanja WHERE tahun = ?', [$tahun]),
                            'belanja' => (int) Database::value("SELECT COUNT(*) FROM belanja WHERE strftime('%Y', tgl) = ?", [(string) $tahun]),
                            'penerimaan' => (int) Database::value("SELECT COUNT(*) FROM penerimaan WHERE strftime('%Y', tgl) = ?", [(string) $tahun]),
                            'mutasi' => (int) Database::value("SELECT COUNT(*) FROM mutasi_kas WHERE strftime('%Y', tgl) = ?", [(string) $tahun]),
                        ],
                    ];
                },
            ],
            'tahun_list' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Daftar tahun anggaran yang terdaftar beserta penanda default.',
                'skema' => self::obj([]),
                'handler' => fn (array $a): array => Database::all(
                    'SELECT tahun, is_default, keterangan FROM tahun_anggaran ORDER BY tahun DESC'
                ),
            ],
        ];
    }

    private static function master(): array
    {
        $cari = self::s('Kata kunci pencarian nama/kode.');
        return [
            'sekolah' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Profil sekolah, pejabat aktif (kepala/bendahara), dan daftar rekening.',
                'skema' => self::obj([]),
                'handler' => fn (array $a): array => [
                    'sekolah' => sekolah(),
                    'kepala' => pejabat_aktif('kepala'),
                    'bendahara' => pejabat_aktif('bendahara'),
                ],
            ],
            'komponen_list' => self::listTool(
                'Daftar komponen BOSP (kode dan nama).',
                'SELECT id, kode, nama, urutan FROM komponen ORDER BY urutan, kode',
                []
            ),
            'rekening_list' => self::listTool(
                'Daftar kode rekening belanja beserta jenisnya.',
                'SELECT id, kode, nama, jenis FROM kode_rekening ORDER BY kode',
                []
            ),
            'pajak_list' => self::listTool(
                'Daftar tarif pajak yang berlaku di aplikasi.',
                'SELECT id, kode, nama, tarif, keterangan FROM tarif_pajak ORDER BY kode',
                []
            ),
            'toko_list' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Daftar toko/penyedia, bisa dicari dan difilter status aktif.',
                'skema' => self::obj([
                    'q' => $cari,
                    'aktif' => ['type' => 'boolean', 'description' => 'Hanya yang aktif bila true.'],
                ]),
                'handler' => function (array $a): array {
                    $w = [];
                    $p = [];
                    if (trim((string) ($a['q'] ?? '')) !== '') {
                        $w[] = '(nama LIKE ? OR alamat LIKE ? OR npwp LIKE ?)';
                        $q = '%' . trim((string) $a['q']) . '%';
                        array_push($p, $q, $q, $q);
                    }
                    if (array_key_exists('aktif', $a) && $a['aktif'] !== null) {
                        $w[] = 'aktif = ?';
                        $p[] = $a['aktif'] ? 1 : 0;
                    }
                    return Database::all(
                        'SELECT id, nama, alamat, npwp, pkp, telepon, aktif FROM toko'
                        . ($w ? ' WHERE ' . implode(' AND ', $w) : '') . ' ORDER BY aktif DESC, nama',
                        $p
                    );
                },
            ],
            'penerima_list' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Daftar penerima honor/transport (guru, tendik, pihak luar) dengan tarif PPh 21.',
                'skema' => self::obj([
                    'q' => $cari,
                    'kategori' => self::s('PNS, PPPK, GTT, PTT, atau Luar.'),
                    'aktif' => ['type' => 'boolean', 'description' => 'Hanya yang aktif bila true.'],
                ]),
                'handler' => function (array $a): array {
                    $w = [];
                    $p = [];
                    if (trim((string) ($a['q'] ?? '')) !== '') {
                        $w[] = '(nama LIKE ? OR nip LIKE ? OR nik LIKE ? OR jabatan LIKE ?)';
                        $q = '%' . trim((string) $a['q']) . '%';
                        array_push($p, $q, $q, $q, $q);
                    }
                    if (in_array($a['kategori'] ?? '', ['PNS', 'PPPK', 'GTT', 'PTT', 'Luar'], true)) {
                        $w[] = 'kategori = ?';
                        $p[] = $a['kategori'];
                    }
                    if (array_key_exists('aktif', $a) && $a['aktif'] !== null) {
                        $w[] = 'aktif = ?';
                        $p[] = $a['aktif'] ? 1 : 0;
                    }
                    $rows = Database::all(
                        'SELECT * FROM penerima' . ($w ? ' WHERE ' . implode(' AND ', $w) : '') . ' ORDER BY aktif DESC, nama',
                        $p
                    );
                    foreach ($rows as &$r) {
                        $r['pph21'] = tarif_pph21($r);
                    }
                    return $rows;
                },
            ],
        ];
    }

    /** Tool sederhana: satu query SELECT tanpa parameter. */
    private static function listTool(string $deskripsi, string $sql, array $p): array
    {
        return [
            'akses' => ApiAuth::BACA,
            'deskripsi' => $deskripsi,
            'skema' => self::obj([]),
            'handler' => fn (array $a): array => Database::all($sql, $p),
        ];
    }

    private static function keuangan(): array
    {
        return [
            'rencana_list' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Paket rencana belanja (RKAS) satu tahun: pagu, terpakai, dan sisa.',
                'skema' => self::obj([
                    'tahun' => self::i('Tahun anggaran.'),
                    'komponen' => self::s('Filter kode komponen, mis. K01.'),
                    'q' => self::s('Kata kunci uraian/kegiatan/rekening.'),
                ]),
                'handler' => function (array $a): array {
                    $tahun = self::tahun($a);
                    $w = ['r.tahun = ?'];
                    $p = [$tahun];
                    if (trim((string) ($a['komponen'] ?? '')) !== '') {
                        $w[] = 'r.komponen_kode = ?';
                        $p[] = trim((string) $a['komponen']);
                    }
                    if (trim((string) ($a['q'] ?? '')) !== '') {
                        $w[] = '(r.uraian LIKE ? OR r.kegiatan LIKE ? OR r.rekening_kode LIKE ?)';
                        $q = '%' . trim((string) $a['q']) . '%';
                        array_push($p, $q, $q, $q);
                    }
                    $rows = Database::all(
                        'SELECT r.id, r.komponen_kode, r.kegiatan, r.rekening_kode, r.uraian, r.jenis,
                                r.frekuensi_bulan, r.toko_id, r.pagu, r.satuan, r.jadwal_bulan,
                                COALESCE(b.pakai, 0) AS terpakai,
                                r.pagu - COALESCE(b.pakai, 0) AS sisa,
                                r.frekuensi_bulan - COALESCE(b.periode, 0) AS periode_sisa
                         FROM rencana_belanja r
                         LEFT JOIN (SELECT rencana_id, SUM(total) AS pakai, SUM(jumlah_periode) AS periode
                                    FROM belanja GROUP BY rencana_id) b ON b.rencana_id = r.id
                         WHERE ' . implode(' AND ', $w) . '
                         ORDER BY r.komponen_kode, r.id',
                        $p
                    );
                    return ['tahun' => $tahun, 'jumlah' => count($rows), 'rencana' => $rows];
                },
            ],
            'rencana_detail' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Rincian satu paket rencana: daftar barang/penerima, belanja yang sudah tercatat, sisa pagu dan periode.',
                'skema' => self::obj(['id' => self::i('ID paket rencana.')], ['id']),
                'handler' => function (array $a): array {
                    $r = \App\Services\RencanaService::detail((int) ($a['id'] ?? 0));
                    if (!$r) {
                        throw new \RuntimeException('Rencana belanja id ' . (int) ($a['id'] ?? 0) . ' tidak ditemukan.');
                    }
                    $r['belanja'] = Database::all(
                        'SELECT id, tgl, no_bukti, uraian, jumlah_periode, total, metode FROM belanja WHERE rencana_id = ? ORDER BY tgl, id',
                        [$r['id']]
                    );
                    return $r;
                },
            ],
            'penerimaan_list' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Penerimaan dana satu tahun beserta rencana pendapatan yang dipasangkan.',
                'skema' => self::obj(['tahun' => self::i('Tahun anggaran.')]),
                'handler' => function (array $a): array {
                    $tahun = self::tahun($a);
                    $rows = Database::all(
                        "SELECT p.id, p.tgl, p.no_bukti, p.uraian, p.tahap, p.jumlah, p.rencana_pendapatan_id,
                                rp.uraian AS rencana_uraian
                         FROM penerimaan p LEFT JOIN rencana_pendapatan rp ON rp.id = p.rencana_pendapatan_id
                         WHERE strftime('%Y', p.tgl) = ? ORDER BY p.tgl, p.id",
                        [(string) $tahun]
                    );
                    return ['tahun' => $tahun, 'total' => array_sum(array_column($rows, 'jumlah')), 'penerimaan' => $rows];
                },
            ],
            'rencana_pendapatan_list' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Target pendapatan per tahap/sumber beserta realisasi yang sudah terpasang.',
                'skema' => self::obj(['tahun' => self::i('Tahun anggaran.')]),
                'handler' => function (array $a): array {
                    $tahun = self::tahun($a);
                    return Database::all(
                        'SELECT r.id, r.tahap, r.uraian, r.target_bulan, r.target_jumlah,
                                COALESCE(p.realisasi, 0) AS realisasi,
                                r.target_jumlah - COALESCE(p.realisasi, 0) AS selisih
                         FROM rencana_pendapatan r
                         LEFT JOIN (SELECT rencana_pendapatan_id, SUM(jumlah) AS realisasi FROM penerimaan
                                    WHERE rencana_pendapatan_id IS NOT NULL GROUP BY rencana_pendapatan_id) p
                                ON p.rencana_pendapatan_id = r.id
                         WHERE r.tahun = ? ORDER BY COALESCE(r.target_bulan, 13), r.id',
                        [$tahun]
                    );
                },
            ],
            'bku' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Buku kas umum satu bulan: transaksi berurutan, debit/kredit, dan saldo berjalan.',
                'skema' => self::obj([
                    'tahun' => self::i('Tahun anggaran.'),
                    'bulan' => self::i('Bulan 1–12.'),
                ]),
                'handler' => function (array $a): array {
                    $tahun = self::tahun($a);
                    $bulan = (int) ($a['bulan'] ?? 0);
                    $bulan = $bulan >= 1 && $bulan <= 12 ? $bulan : 1;
                    $awal = sprintf('%04d-%02d-01', $tahun, $bulan);
                    $akhir = date('Y-m-t', strtotime($awal));
                    $rows = Database::all(
                        "SELECT * FROM (
                           SELECT 'trm' AS src, id, tgl, no_bukti, uraian, jumlah AS debit, 0 AS kredit
                             FROM penerimaan WHERE tgl BETWEEN :a AND :b
                           UNION ALL
                           SELECT 'bel', id, tgl, no_bukti, uraian, 0, total FROM belanja WHERE tgl BETWEEN :a AND :b
                           UNION ALL
                           SELECT 'mut', id, tgl, no_bukti, uraian,
                                  CASE jenis WHEN 'bunga' THEN jumlah ELSE 0 END,
                                  CASE jenis WHEN 'admin_bank' THEN jumlah ELSE 0 END
                             FROM mutasi_kas WHERE tgl BETWEEN :a AND :b
                         ) ORDER BY tgl, CASE src WHEN 'trm' THEN 0 WHEN 'mut' THEN 1 ELSE 2 END, id",
                        ['a' => $awal, 'b' => $akhir]
                    );
                    $saldo = saldo_kas(date('Y-m-d', strtotime("$awal -1 day")))['total'];
                    foreach ($rows as &$r) {
                        $saldo += (float) $r['debit'] - (float) $r['kredit'];
                        $r['saldo'] = $saldo;
                    }
                    return [
                        'tahun' => $tahun,
                        'bulan' => $bulan,
                        'saldo_awal' => saldo_kas(date('Y-m-d', strtotime("$awal -1 day"))),
                        'saldo_akhir' => saldo_kas($akhir),
                        'transaksi' => $rows,
                    ];
                },
            ],
            'saldo' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Saldo kas tunai, bank, dan total sampai tanggal tertentu.',
                'skema' => self::obj(['tgl' => self::s('Tanggal YYYY-MM-DD, default hari ini.')]),
                'handler' => function (array $a): array {
                    $tgl = trim((string) ($a['tgl'] ?? ''));
                    if (!valid_date($tgl)) {
                        $tgl = date('Y-m-d');
                    }
                    return ['tgl' => $tgl, 'saldo' => saldo_kas($tgl)];
                },
            ],
        ];
    }

    private static function laporan(): array
    {
        return [
            'laporan_list' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Jenis laporan yang tersedia beserta penjelasannya.',
                'skema' => self::obj([]),
                'handler' => fn (array $a): array => array_map(
                    fn ($k, $v) => ['jenis' => $k, 'nama' => $v[0], 'keterangan' => $v[1]],
                    array_keys(\App\Services\Laporan::JENIS),
                    array_values(\App\Services\Laporan::JENIS)
                ),
            ],
            'laporan' => [
                'akses' => ApiAuth::BACA,
                'deskripsi' => 'Data satu laporan (tabel siap tampil) untuk periode tertentu.',
                'skema' => self::obj([
                    'jenis' => self::s('bku, tunai, bank, pajak, komponen, realisasi, atau kas.'),
                    'periode' => self::s('1–12, T1, T2, atau Y. Default Y.'),
                    'tahun' => self::i('Tahun anggaran.'),
                ], ['jenis']),
                'handler' => function (array $a): array {
                    $jenis = (string) ($a['jenis'] ?? '');
                    if (!isset(\App\Services\Laporan::JENIS[$jenis])) {
                        throw new \RuntimeException('Jenis laporan tidak dikenal: ' . $jenis);
                    }
                    $kode = strtoupper(trim((string) ($a['periode'] ?? 'Y')));
                    if (!preg_match('/^(10|11|12|[1-9]|T1|T2|Y)$/', $kode)) {
                        throw new \RuntimeException('Periode tidak dikenal: ' . $kode);
                    }
                    $tahun = self::tahun($a);
                    return \App\Services\Laporan::build($jenis, \App\Services\Laporan::periode($kode, $tahun));
                },
            ],
        ];
    }
}

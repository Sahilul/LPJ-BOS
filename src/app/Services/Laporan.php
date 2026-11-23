<?php

declare(strict_types=1);

namespace App\Services;

use App\Database;

/**
 * Penyusun data laporan. Semua angka dihitung dari penerimaan, belanja, pajak, dan mutasi kas
 * (tidak ada yang disimpan), sehingga selalu konsisten dengan BKU.
 *
 * Setiap laporan tabel berbentuk:
 *   title, columns [key => [label, type text|money|date|pct]], rows (baris boleh punya _class: group|info|sub),
 *   foot (baris jumlah), summary [[label, nilai]], orientation.
 */
final class Laporan
{
    public const JENIS = [
        'bku' => ['Buku kas umum', 'Semua penerimaan dan pengeluaran dengan saldo berjalan.', 'book'],
        'tunai' => ['Buku pembantu kas tunai', 'Tarik tunai, setor, dan belanja yang dibayar tunai.', 'wallet'],
        'bank' => ['Buku pembantu bank', 'Penerimaan dana, bunga, biaya admin, dan belanja via bank.', 'layers'],
        'pajak' => ['Buku pembantu pajak', 'PPN, PPh 21, 22, dan 23 yang dipungut serta disetor.', 'percent'],
        'komponen' => ['Rekap per komponen', 'Pagu dan realisasi untuk setiap komponen BOSP.', 'chart'],
        'realisasi' => ['Realisasi vs RKAS', 'Pagu, realisasi, dan sisa setiap baris rencana belanja.', 'list'],
        'kas' => ['Register penutupan kas & BA pemeriksaan kas', 'Saldo kas tunai dan bank di akhir periode, siap ditandatangani.', 'file'],
    ];

    private const JENIS_BELANJA = ['barang' => 'Barang/jasa', 'honor' => 'Honor', 'transport' => 'Transport'];
    private const JENIS_PAJAK = ['PPN' => 'PPN', 'PPH21' => 'PPh 21', 'PPH22' => 'PPh 22', 'PPH23' => 'PPh 23'];

    /** Periode: '1'..'12' (bulan), 'T1'/'T2' (tahap/semester), 'Y' (setahun). */
    public static function periode(?string $kode, int $tahun): array
    {
        $kode = strtoupper(trim((string) $kode));
        if ($kode === '') {
            $kode = (string) ((int) date('Y') === $tahun ? (int) date('n') : 12);
        }
        [$m1, $m2, $label] = match (true) {
            $kode === 'T1' => [1, 6, "Tahap I (Januari–Juni) $tahun"],
            $kode === 'T2' => [7, 12, "Tahap II (Juli–Desember) $tahun"],
            $kode === 'Y' => [1, 12, "Tahun Anggaran $tahun"],
            ctype_digit($kode) && (int) $kode >= 1 && (int) $kode <= 12 => [(int) $kode, (int) $kode, bulan_indo((int) $kode) . " $tahun"],
            default => abort(404, 'Periode laporan tidak dikenal.'),
        };
        $awal = sprintf('%04d-%02d-01', $tahun, $m1);
        $akhir = date('Y-m-t', strtotime(sprintf('%04d-%02d-01', $tahun, $m2)));
        return [
            'kode' => ctype_digit($kode) ? (string) (int) $kode : $kode,
            'awal' => $awal,
            'akhir' => $akhir,
            'sebelum' => date('Y-m-d', strtotime("$awal -1 day")),
            'label' => $label,
            'bulanan' => $m1 === $m2,
            'slug' => ctype_digit($kode) ? sprintf('%04d-%02d', $tahun, (int) $kode) : "$tahun-$kode",
        ];
    }

    public static function build(string $jenis, array $p): array
    {
        return match ($jenis) {
            'bku' => self::bku($p),
            'tunai' => self::bukuPembantu($p, 'tunai'),
            'bank' => self::bukuPembantu($p, 'bank'),
            'pajak' => self::pajak($p),
            'komponen' => self::komponen($p),
            'realisasi' => self::realisasi($p),
            'kas' => self::kas($p),
            default => abort(404, 'Jenis laporan tidak dikenal.'),
        };
    }

    /** Baris BKU periode, urutan sama dengan halaman BKU. Saldo awal dihitung terpisah. */
    private static function transaksi(array $p): array
    {
        return Database::all(
            "SELECT * FROM (
               SELECT 'trm' AS src, id, tgl, no_bukti, uraian, '' AS rekening, jumlah AS nilai, 'bank' AS metode, 'penerimaan' AS jenis
                 FROM penerimaan WHERE tgl BETWEEN :a AND :b
               UNION ALL
               SELECT 'bel', b.id, b.tgl, b.no_bukti, b.uraian, COALESCE(r.rekening_kode, ''), b.total, b.metode, b.jenis
                 FROM belanja b LEFT JOIN rencana_belanja r ON r.id = b.rencana_id WHERE b.tgl BETWEEN :a AND :b
               UNION ALL
               SELECT 'mut', id, tgl, no_bukti, uraian, '', jumlah, '', jenis
                 FROM mutasi_kas WHERE tgl BETWEEN :a AND :b
             ) ORDER BY tgl, CASE src WHEN 'trm' THEN 0 WHEN 'mut' THEN 1 ELSE 2 END, id",
            ['a' => $p['awal'], 'b' => $p['akhir']]
        );
    }

    private static function columnsKas(): array
    {
        return [
            'tgl' => ['Tanggal', 'date'],
            'no_bukti' => ['No. bukti', 'text'],
            'rekening' => ['Kode rekening', 'text'],
            'uraian' => ['Uraian', 'text'],
            'debit' => ['Penerimaan', 'money'],
            'kredit' => ['Pengeluaran', 'money'],
            'saldo' => ['Saldo', 'money'],
        ];
    }

    private static function bku(array $p): array
    {
        $awal = saldo_kas($p['sebelum']);
        $saldo = (float) $awal['total'];
        $rows = [['_class' => 'group', 'uraian' => 'Saldo awal', 'saldo' => $saldo]];
        $debit = $kredit = 0.0;
        foreach (self::transaksi($p) as $t) {
            [$d, $k] = match (true) {
                $t['src'] === 'trm' => [(float) $t['nilai'], 0.0],
                $t['src'] === 'bel' => [0.0, (float) $t['nilai']],
                $t['jenis'] === 'bunga' => [(float) $t['nilai'], 0.0],
                $t['jenis'] === 'admin_bank' => [0.0, (float) $t['nilai']],
                default => [0.0, 0.0],
            };
            $saldo += $d - $k;
            $debit += $d;
            $kredit += $k;
            $info = $d == 0 && $k == 0;
            $rows[] = [
                '_class' => $info ? 'info' : '',
                'tgl' => $t['tgl'],
                'no_bukti' => $t['no_bukti'],
                'rekening' => $t['rekening'],
                'uraian' => $t['uraian'] . ($info ? ' (pemindahbukuan ' . rupiah((float) $t['nilai']) . ', tidak mengubah saldo)' : ''),
                'debit' => $d ?: null,
                'kredit' => $k ?: null,
                'saldo' => $saldo,
            ];
        }
        $akhir = saldo_kas($p['akhir']);
        return [
            'title' => 'BUKU KAS UMUM',
            'columns' => self::columnsKas(),
            'rows' => $rows,
            'foot' => ['uraian' => 'Jumlah periode ini', 'debit' => $debit, 'kredit' => $kredit, 'saldo' => $saldo],
            'summary' => [
                ['Saldo awal', $awal['total']],
                ['Jumlah penerimaan', $debit],
                ['Jumlah pengeluaran', $kredit],
                ['Saldo akhir', $akhir['total']],
                ['— terdiri dari kas tunai', $akhir['tunai']],
                ['— terdiri dari saldo bank', $akhir['bank']],
            ],
            'orientation' => 'landscape',
        ];
    }

    private static function bukuPembantu(array $p, string $kas): array
    {
        $awal = (float) saldo_kas($p['sebelum'])[$kas];
        $saldo = $awal;
        $rows = [['_class' => 'group', 'uraian' => 'Saldo awal', 'saldo' => $saldo]];
        $debit = $kredit = 0.0;
        foreach (self::transaksi($p) as $t) {
            $n = (float) $t['nilai'];
            [$d, $k] = $kas === 'tunai'
                ? match (true) {
                    $t['src'] === 'bel' && $t['metode'] === 'tunai' => [0.0, $n],
                    $t['src'] === 'mut' && $t['jenis'] === 'tarik' => [$n, 0.0],
                    $t['src'] === 'mut' && $t['jenis'] === 'setor' => [0.0, $n],
                    default => [null, null],
                }
                : match (true) {
                    $t['src'] === 'trm' => [$n, 0.0],
                    $t['src'] === 'bel' && $t['metode'] === 'bank' => [0.0, $n],
                    $t['src'] === 'mut' && in_array($t['jenis'], ['setor', 'bunga'], true) => [$n, 0.0],
                    $t['src'] === 'mut' && in_array($t['jenis'], ['tarik', 'admin_bank'], true) => [0.0, $n],
                    default => [null, null],
                };
            if ($d === null) {
                continue;
            }
            $saldo += $d - $k;
            $debit += $d;
            $kredit += $k;
            $rows[] = ['tgl' => $t['tgl'], 'no_bukti' => $t['no_bukti'], 'rekening' => $t['rekening'], 'uraian' => $t['uraian'],
                'debit' => $d ?: null, 'kredit' => $k ?: null, 'saldo' => $saldo];
        }
        return [
            'title' => $kas === 'tunai' ? 'BUKU PEMBANTU KAS TUNAI' : 'BUKU PEMBANTU BANK',
            'columns' => self::columnsKas(),
            'rows' => $rows,
            'foot' => ['uraian' => 'Jumlah periode ini', 'debit' => $debit, 'kredit' => $kredit, 'saldo' => $saldo],
            'summary' => [['Saldo awal', $awal], ['Jumlah penerimaan', $debit], ['Jumlah pengeluaran', $kredit], ['Saldo akhir', $saldo]],
            'orientation' => 'landscape',
        ];
    }

    private static function pajak(array $p): array
    {
        $tahunAwal = substr($p['awal'], 0, 4) . '-01-01';
        $pungutSql = 'SELECT COALESCE(p.tgl_pungut, b.tgl) AS tgl, b.no_bukti, b.uraian, p.jenis, p.nilai, p.ntpn
                      FROM pajak p JOIN belanja b ON b.id = p.belanja_id
                      WHERE COALESCE(p.tgl_pungut, b.tgl) BETWEEN ? AND ?';
        $setorSql = 'SELECT p.tgl_setor AS tgl, b.no_bukti, b.uraian, p.jenis, p.nilai, p.ntpn
                     FROM pajak p JOIN belanja b ON b.id = p.belanja_id
                     WHERE p.tgl_setor IS NOT NULL AND p.tgl_setor != "" AND p.tgl_setor BETWEEN ? AND ?';
        $sumBefore = fn (string $sql) => array_sum(array_column(Database::all($sql, [$tahunAwal, $p['sebelum']]), 'nilai'));
        $awal = $sumBefore($pungutSql) - $sumBefore($setorSql);

        $events = [];
        foreach (Database::all($pungutSql, [$p['awal'], $p['akhir']]) as $r) {
            $events[] = $r + ['arah' => 0];
        }
        foreach (Database::all($setorSql, [$p['awal'], $p['akhir']]) as $r) {
            $events[] = $r + ['arah' => 1];
        }
        usort($events, fn ($a, $b) => [$a['tgl'], $a['arah'], $a['no_bukti']] <=> [$b['tgl'], $b['arah'], $b['no_bukti']]);

        $saldo = (float) $awal;
        $rows = [['_class' => 'group', 'uraian' => 'Saldo awal (belum disetor)', 'saldo' => $saldo]];
        $pungut = $setor = 0.0;
        $perJenis = [];
        foreach ($events as $e) {
            $n = (float) $e['nilai'];
            $label = self::JENIS_PAJAK[$e['jenis']] ?? $e['jenis'];
            if ($e['arah'] === 0) {
                $saldo += $n;
                $pungut += $n;
                $perJenis[$label][0] = ($perJenis[$label][0] ?? 0) + $n;
            } else {
                $saldo -= $n;
                $setor += $n;
                $perJenis[$label][1] = ($perJenis[$label][1] ?? 0) + $n;
            }
            $rows[] = [
                'tgl' => $e['tgl'],
                'no_bukti' => $e['no_bukti'],
                'jenis' => $label,
                'uraian' => ($e['arah'] ? 'Setor ' . $label . ($e['ntpn'] ? ' NTPN ' . $e['ntpn'] : '') . ' atas ' : 'Pungut ' . $label . ' atas ') . $e['uraian'],
                'debit' => $e['arah'] ? null : $n,
                'kredit' => $e['arah'] ? $n : null,
                'saldo' => $saldo,
            ];
        }
        $summary = [['Saldo awal belum disetor', $awal]];
        foreach ($perJenis as $label => $v) {
            $summary[] = ["$label dipungut / disetor", rupiah($v[0] ?? 0) . ' / ' . rupiah($v[1] ?? 0)];
        }
        $summary[] = ['Saldo akhir belum disetor', $saldo];
        return [
            'title' => 'BUKU PEMBANTU PAJAK',
            'columns' => [
                'tgl' => ['Tanggal', 'date'],
                'no_bukti' => ['No. bukti', 'text'],
                'jenis' => ['Jenis', 'text'],
                'uraian' => ['Uraian', 'text'],
                'debit' => ['Dipungut', 'money'],
                'kredit' => ['Disetor', 'money'],
                'saldo' => ['Saldo', 'money'],
            ],
            'rows' => $rows,
            'foot' => ['uraian' => 'Jumlah periode ini', 'debit' => $pungut, 'kredit' => $setor, 'saldo' => $saldo],
            'summary' => $summary,
            'orientation' => 'landscape',
        ];
    }

    /** Realisasi belanja per rencana: dalam periode dan kumulatif sejak awal tahun s.d. akhir periode. */
    private static function realisasiData(array $p): array
    {
        $tahun = (int) substr($p['awal'], 0, 4);
        return Database::all(
            'SELECT r.id, r.komponen_kode, k.nama AS komponen_nama, r.kegiatan, r.rekening_kode, r.uraian, r.pagu,
                    COALESCE(SUM(CASE WHEN b.tgl BETWEEN :a AND :b THEN b.total END), 0) AS periode,
                    COALESCE(SUM(b.total), 0) AS kumulatif
             FROM rencana_belanja r
             LEFT JOIN komponen k ON k.kode = r.komponen_kode
             LEFT JOIN belanja b ON b.rencana_id = r.id AND b.tgl BETWEEN :y AND :b
             WHERE r.tahun = :t
             GROUP BY r.id
             ORDER BY r.komponen_kode, r.id',
            ['a' => $p['awal'], 'b' => $p['akhir'], 'y' => "$tahun-01-01", 't' => $tahun]
        );
    }

    private static function tanpaRencana(array $p): array
    {
        $tahun = (int) substr($p['awal'], 0, 4);
        return Database::one(
            'SELECT COALESCE(SUM(CASE WHEN tgl BETWEEN ? AND ? THEN total END), 0) AS periode, COALESCE(SUM(total), 0) AS kumulatif
             FROM belanja WHERE rencana_id IS NULL AND tgl BETWEEN ? AND ?',
            [$p['awal'], $p['akhir'], "$tahun-01-01", $p['akhir']]
        ) ?: ['periode' => 0, 'kumulatif' => 0];
    }

    private static function pct(float $real, float $pagu): ?float
    {
        return $pagu > 0 ? round($real / $pagu * 100, 2) : null;
    }

    private static function komponen(array $p): array
    {
        $agg = [];
        foreach (self::realisasiData($p) as $r) {
            $a = &$agg[$r['komponen_kode']];
            $a['pagu'] = ($a['pagu'] ?? 0) + (float) $r['pagu'];
            $a['periode'] = ($a['periode'] ?? 0) + (float) $r['periode'];
            $a['kumulatif'] = ($a['kumulatif'] ?? 0) + (float) $r['kumulatif'];
            unset($a);
        }
        $rows = [];
        $tot = ['pagu' => 0.0, 'periode' => 0.0, 'kumulatif' => 0.0];
        foreach (Database::all('SELECT kode, nama FROM komponen ORDER BY urutan, kode') as $k) {
            $a = $agg[$k['kode']] ?? ['pagu' => 0, 'periode' => 0, 'kumulatif' => 0];
            unset($agg[$k['kode']]);
            $rows[] = self::rekapRow($k['kode'], $k['nama'], $a);
            foreach ($tot as $key => $v) {
                $tot[$key] += $a[$key];
            }
        }
        $lain = self::tanpaRencana($p);
        foreach ($agg as $kode => $a) {
            $lain['periode'] += $a['periode'];
            $lain['kumulatif'] += $a['kumulatif'];
            $tot['pagu'] += $a['pagu'];
        }
        if ($lain['kumulatif'] > 0) {
            $rows[] = self::rekapRow('—', 'Belanja tanpa rencana / komponen lain', ['pagu' => 0] + $lain);
            $tot['periode'] += (float) $lain['periode'];
            $tot['kumulatif'] += (float) $lain['kumulatif'];
        }
        return [
            'title' => 'REKAPITULASI REALISASI PER KOMPONEN',
            'columns' => self::rekapColumns(['kode' => ['Kode', 'text'], 'nama' => ['Komponen', 'text']]),
            'rows' => $rows,
            'foot' => self::rekapRow('', 'Jumlah', $tot),
            'summary' => [['Total pagu', $tot['pagu']], ['Realisasi s.d. periode', $tot['kumulatif']], ['Sisa anggaran', $tot['pagu'] - $tot['kumulatif']]],
            'orientation' => 'landscape',
        ];
    }

    private static function rekapColumns(array $lead): array
    {
        return $lead + [
            'pagu' => ['Pagu', 'money'],
            'periode' => ['Realisasi periode', 'money'],
            'kumulatif' => ['Realisasi s.d. periode', 'money'],
            'sisa' => ['Sisa', 'money'],
            'pct' => ['%', 'pct'],
        ];
    }

    private static function rekapRow(string $kode, string $nama, array $a, string $class = ''): array
    {
        return [
            '_class' => $class,
            'kode' => $kode,
            'nama' => $nama,
            'pagu' => (float) $a['pagu'],
            'periode' => (float) $a['periode'],
            'kumulatif' => (float) $a['kumulatif'],
            'sisa' => (float) $a['pagu'] - (float) $a['kumulatif'],
            'pct' => self::pct((float) $a['kumulatif'], (float) $a['pagu']),
        ];
    }

    private static function realisasi(array $p): array
    {
        $rows = [];
        $tot = ['pagu' => 0.0, 'periode' => 0.0, 'kumulatif' => 0.0];
        $grup = null;
        $sub = $tot;
        $flush = function () use (&$rows, &$grup, &$sub) {
            if ($grup !== null) {
                $rows[] = self::rekapRow('', 'Subtotal komponen ' . $grup, $sub, 'sub');
            }
        };
        foreach (self::realisasiData($p) as $r) {
            if ($r['komponen_kode'] !== $grup) {
                $flush();
                $grup = $r['komponen_kode'];
                $sub = ['pagu' => 0.0, 'periode' => 0.0, 'kumulatif' => 0.0];
                $rows[] = ['_class' => 'group', 'kode' => $grup, 'nama' => $r['komponen_nama'] ?: 'Komponen ' . $grup];
            }
            $row = self::rekapRow($r['rekening_kode'], trim($r['uraian'] . ($r['kegiatan'] !== '' ? ' — ' . $r['kegiatan'] : '')), $r);
            $rows[] = $row;
            foreach ($tot as $k => $v) {
                $tot[$k] += (float) $r[$k];
                $sub[$k] += (float) $r[$k];
            }
        }
        $flush();
        $lain = self::tanpaRencana($p);
        if ($lain['kumulatif'] > 0) {
            $rows[] = self::rekapRow('—', 'Belanja tanpa rencana', ['pagu' => 0] + $lain);
            $tot['periode'] += (float) $lain['periode'];
            $tot['kumulatif'] += (float) $lain['kumulatif'];
        }
        return [
            'title' => 'LAPORAN REALISASI PENGGUNAAN DANA (RKAS)',
            'columns' => self::rekapColumns(['kode' => ['Kode rekening', 'text'], 'nama' => ['Uraian', 'text']]),
            'rows' => $rows,
            'foot' => self::rekapRow('', 'Jumlah', $tot),
            'summary' => [['Total pagu', $tot['pagu']], ['Realisasi s.d. periode', $tot['kumulatif']], ['Sisa anggaran', $tot['pagu'] - $tot['kumulatif']]],
            'orientation' => 'landscape',
        ];
    }

    /** Register penutupan kas & berita acara pemeriksaan kas pada tanggal akhir periode. */
    private static function kas(array $p): array
    {
        $y = substr($p['awal'], 0, 4) . '-01-01';
        $tgl = $p['akhir'];
        $masuk = (float) Database::value('SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan WHERE tgl BETWEEN ? AND ?', [$y, $tgl])
            + (float) Database::value("SELECT COALESCE(SUM(jumlah), 0) FROM mutasi_kas WHERE jenis = 'bunga' AND tgl BETWEEN ? AND ?", [$y, $tgl]);
        $keluar = (float) Database::value('SELECT COALESCE(SUM(total), 0) FROM belanja WHERE tgl BETWEEN ? AND ?', [$y, $tgl])
            + (float) Database::value("SELECT COALESCE(SUM(jumlah), 0) FROM mutasi_kas WHERE jenis = 'admin_bank' AND tgl BETWEEN ? AND ?", [$y, $tgl]);
        $saldo = saldo_kas($tgl);
        $lastBel = Database::one('SELECT tgl, no_bukti FROM belanja WHERE tgl BETWEEN ? AND ? ORDER BY tgl DESC, id DESC LIMIT 1', [$y, $tgl]);
        $rows = [
            ['uraian' => 'Jumlah penerimaan s.d. ' . tanggal_indo($tgl), 'nilai' => $masuk],
            ['uraian' => 'Jumlah pengeluaran s.d. ' . tanggal_indo($tgl), 'nilai' => $keluar],
            ['_class' => 'group', 'uraian' => 'Saldo menurut Buku Kas Umum', 'nilai' => $saldo['total']],
            ['uraian' => 'Kas tunai di brankas', 'nilai' => $saldo['tunai']],
            ['uraian' => 'Saldo rekening bank' . (!empty(sekolah()['bank_nama']) ? ' (' . sekolah()['bank_nama'] . ' ' . sekolah()['bank_rekening'] . ')' : ''), 'nilai' => $saldo['bank']],
            ['_class' => 'group', 'uraian' => 'Jumlah kas (tunai + bank)', 'nilai' => $saldo['tunai'] + $saldo['bank']],
        ];
        return [
            'title' => 'REGISTER PENUTUPAN KAS',
            'columns' => ['uraian' => ['Uraian', 'text'], 'nilai' => ['Jumlah', 'money']],
            'rows' => $rows,
            'foot' => null,
            'summary' => [],
            'orientation' => 'portrait',
            'kas' => [
                'tgl' => $tgl,
                'masuk' => $masuk,
                'keluar' => $keluar,
                'saldo' => $saldo,
                'bukti_terakhir' => $lastBel ? $lastBel['no_bukti'] . ' (' . tanggal_indo($lastBel['tgl']) . ')' : '—',
            ],
        ];
    }

    /** Transaksi belanja periode beserta rincian untuk lampiran SPJ. */
    public static function belanjaPeriode(array $p): array
    {
        $rows = Database::all(
            'SELECT b.*, t.nama AS toko_nama, r.rekening_kode, r.uraian AS rencana_uraian
             FROM belanja b LEFT JOIN toko t ON t.id = b.toko_id LEFT JOIN rencana_belanja r ON r.id = b.rencana_id
             WHERE b.tgl BETWEEN ? AND ? ORDER BY b.tgl, b.id',
            [$p['awal'], $p['akhir']]
        );
        foreach ($rows as &$b) {
            $b['items'] = Database::all('SELECT * FROM item_belanja WHERE belanja_id = ? ORDER BY id', [$b['id']]);
            $b['penerima'] = Database::all('SELECT * FROM penerima_belanja WHERE belanja_id = ? ORDER BY id', [$b['id']]);
            $b['pajak'] = Database::all('SELECT * FROM pajak WHERE belanja_id = ? ORDER BY id', [$b['id']]);
            $b['jenis_label'] = self::JENIS_BELANJA[$b['jenis']] ?? $b['jenis'];
        }
        unset($b);
        return $rows;
    }

    public static function jenisPajak(string $kode): string
    {
        return self::JENIS_PAJAK[$kode] ?? $kode;
    }
}

<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;

final class BkuController
{
    public function index(): void
    {
        $tahun = tahun_aktif();
        $bulan = max(1, min(12, (int) ($_GET['bulan'] ?? (date('Y') == $tahun ? date('n') : 1))));
        $awal = sprintf('%04d-%02d-01', $tahun, $bulan);
        $akhir = date('Y-m-t', strtotime($awal));
        $sebelum = date('Y-m-d', strtotime("$awal -1 day"));

        $rows = Database::all(
            "SELECT * FROM (SELECT 'trm' AS src, id, tgl, no_bukti, uraian, jumlah AS debit, 0 AS kredit, 'bank' AS metode, '' AS jenis, jumlah AS nilai, 0 AS lampiran
               FROM penerimaan WHERE tgl BETWEEN :a AND :b
             UNION ALL
             SELECT 'bel', b.id, b.tgl, b.no_bukti, b.uraian, 0, b.total, b.metode, b.jenis, b.total,
                    (SELECT COUNT(*) FROM lampiran l WHERE l.belanja_id = b.id) AS lampiran
               FROM belanja b WHERE b.tgl BETWEEN :a AND :b
             UNION ALL
             SELECT 'mut', id, tgl, no_bukti, uraian,
                    CASE jenis WHEN 'bunga' THEN jumlah ELSE 0 END,
                    CASE jenis WHEN 'admin_bank' THEN jumlah ELSE 0 END,
                    CASE jenis WHEN 'tarik' THEN 'tunai' ELSE 'bank' END, jenis, jumlah, 0 AS lampiran
               FROM mutasi_kas WHERE tgl BETWEEN :a AND :b
             ) ORDER BY tgl, CASE src WHEN 'trm' THEN 0 WHEN 'mut' THEN 1 ELSE 2 END, id",
            ['a' => $awal, 'b' => $akhir]
        );

        $saldoAwal = saldo_kas($sebelum);
        $saldo = $saldoAwal['total'];
        $debit = $kredit = 0.0;
        foreach ($rows as &$r) {
            $saldo += $r['debit'] - $r['kredit'];
            $debit += $r['debit'];
            $kredit += $r['kredit'];
            $r['saldo'] = $saldo;
        }
        unset($r);

        $perBulan = array_column(Database::all(
            "SELECT CAST(strftime('%m', tgl) AS INTEGER) AS m, COUNT(*) AS n FROM (
                SELECT tgl FROM penerimaan UNION ALL SELECT tgl FROM belanja UNION ALL SELECT tgl FROM mutasi_kas
             ) WHERE strftime('%Y', tgl) = ? GROUP BY m",
            [(string) $tahun]
        ), 'n', 'm');

        view('bku/index', [
            'title' => 'Buku kas umum ' . bulan_indo($bulan) . " $tahun",
            'tahun' => $tahun,
            'bulan' => $bulan,
            'rows' => $rows,
            'saldoAwal' => $saldoAwal,
            'saldoAkhir' => saldo_kas($akhir),
            'debit' => $debit,
            'kredit' => $kredit,
            'perBulan' => $perBulan,
        ]);
    }

    public function updateDate(string $src, string $id): void
    {
        $tables = ['trm' => 'penerimaan', 'bel' => 'belanja', 'mut' => 'mutasi_kas'];
        if (!isset($tables[$src])) {
            abort(404, 'Jenis transaksi BKU tidak dikenal.');
        }
        $table = $tables[$src];
        $row = Database::one("SELECT id, tgl FROM $table WHERE id = ?", [(int) $id]);
        if (!$row) {
            abort(404, 'Transaksi BKU tidak ditemukan.');
        }

        $tgl = (string) ($_POST['tgl'] ?? '');
        if (!valid_date($tgl)) {
            flash('danger', 'Tanggal transaksi tidak valid.');
            redirect('/bku?bulan=' . (int) substr($row['tgl'], 5, 2) . "#$src-" . (int) $id);
        }
        if (!dalam_tahun_aktif($tgl)) {
            flash('danger', 'Tanggal harus berada di tahun anggaran ' . tahun_aktif() . '.');
            redirect('/bku?bulan=' . (int) substr($row['tgl'], 5, 2) . "#$src-" . (int) $id);
        }

        Database::transaction(function () use ($src, $table, $id, $row, $tgl) {
            Database::run("UPDATE $table SET tgl = ? WHERE id = ?", [$tgl, (int) $id]);
            if ($src === 'bel') {
                Database::run(
                    'UPDATE pajak SET tgl_pungut = ? WHERE belanja_id = ? AND tgl_pungut = ?',
                    [$tgl, (int) $id, $row['tgl']]
                );
            }
            susun_nomor_bukti_bku((int) substr($tgl, 0, 4));
        });

        flash('success', 'Tanggal diubah. Urutan nomor bukti dan saldo BKU diperbarui otomatis.');
        redirect('/bku?bulan=' . (int) substr($tgl, 5, 2) . "#$src-" . (int) $id);
    }
}

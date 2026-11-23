<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;

final class DashboardController
{
    public function index(): void
    {
        $s = sekolah();
        $kepala = pejabat_aktif('kepala');
        $bendahara = pejabat_aktif('bendahara');

        $steps = [
            ['label' => 'Profil sekolah', 'done' => $s['nama'] !== '' && $s['npsn'] !== '' && $s['kabupaten'] !== '', 'href' => '/sekolah', 'hint' => 'Nama, NPSN, alamat dan rekening BOS'],
            ['label' => 'Kop surat', 'done' => !empty($s['kop_path']), 'href' => '/sekolah#gambar', 'hint' => 'Gambar kop yang sudah jadi'],
            ['label' => 'Kepala sekolah', 'done' => (bool) $kepala, 'href' => '/pejabat', 'hint' => 'Nama, NIP dan tanda tangan'],
            ['label' => 'Bendahara', 'done' => (bool) $bendahara, 'href' => '/pejabat', 'hint' => 'Nama, NIP dan tanda tangan'],
            ['label' => 'Database penerima', 'done' => (bool) Database::value('SELECT COUNT(*) FROM penerima'), 'href' => '/penerima', 'hint' => 'Guru dan tendik penerima honor'],
            ['label' => 'Toko langganan', 'done' => (bool) Database::value('SELECT COUNT(*) FROM toko'), 'href' => '/toko', 'hint' => 'Tempat belanja ATK, konsumsi, dll.'],
        ];

        $counts = [
            'penerima' => (int) Database::value('SELECT COUNT(*) FROM penerima WHERE aktif = 1'),
            'toko' => (int) Database::value('SELECT COUNT(*) FROM toko WHERE aktif = 1'),
            'komponen' => (int) Database::value('SELECT COUNT(*) FROM komponen'),
            'rekening' => (int) Database::value('SELECT COUNT(*) FROM kode_rekening'),
        ];

        $tahun = tahun_aktif();
        $perTgl = max("$tahun-01-01", min(date('Y-m-d'), "$tahun-12-31"));
        $keu = [
            'pagu' => (float) Database::value('SELECT COALESCE(SUM(pagu), 0) FROM rencana_belanja WHERE tahun = ?', [$tahun]),
            'realisasi' => (float) Database::value("SELECT COALESCE(SUM(b.total), 0) FROM belanja b JOIN rencana_belanja r ON r.id = b.rencana_id WHERE r.tahun = ?", [$tahun]),
            'diterima' => (float) Database::value("SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan WHERE strftime('%Y', tgl) = ?", [(string) $tahun]),
            'saldo' => saldo_kas($perTgl),
            'per' => $perTgl,
        ];

        view('dashboard', [
            'keu' => $keu,
            'title' => 'Beranda',
            's' => $s,
            'tahun' => $tahun,
            'kepala' => $kepala,
            'bendahara' => $bendahara,
            'steps' => $steps,
            'counts' => $counts,
        ]);
    }
}

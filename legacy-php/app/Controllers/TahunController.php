<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;

final class TahunController
{
    public function index(): void
    {
        $rows = Database::all(
            "SELECT t.*,
                (SELECT COUNT(*) FROM rencana_belanja r WHERE r.tahun = t.tahun) AS n_rencana,
                (SELECT COALESCE(SUM(pagu), 0) FROM rencana_belanja r WHERE r.tahun = t.tahun) AS pagu,
                (SELECT COUNT(*) FROM belanja b WHERE strftime('%Y', b.tgl) = CAST(t.tahun AS TEXT)) AS n_belanja,
                (SELECT COUNT(*) FROM penerimaan p WHERE strftime('%Y', p.tgl) = CAST(t.tahun AS TEXT)) AS n_penerimaan,
                (SELECT COUNT(*) FROM mutasi_kas m WHERE strftime('%Y', m.tgl) = CAST(t.tahun AS TEXT)) AS n_mutasi
             FROM tahun_anggaran t ORDER BY t.tahun DESC"
        );
        $list = tahun_list();
        $saran = $list ? max(max($list) + 1, (int) date('Y')) : (int) date('Y');
        view('tahun/index', ['title' => 'Tahun anggaran', 'rows' => $rows, 'aktif' => tahun_aktif(), 'saran' => $saran]);
    }

    public function store(): void
    {
        $tahun = (int) ($_POST['tahun'] ?? 0);
        $ket = trim((string) ($_POST['keterangan'] ?? ''));
        $err = [];
        if ($tahun < 2000 || $tahun > 2100) {
            $err['tahun'] = 'Isi tahun empat angka, misalnya ' . date('Y') . '.';
        } elseif (in_array($tahun, tahun_list(), true)) {
            $err['tahun'] = "Tahun $tahun sudah ada.";
        }
        if ($err) {
            with_old($_POST, $err);
            redirect('/tahun');
        }
        $pertama = !tahun_list();
        Database::insert('tahun_anggaran', ['tahun' => $tahun, 'keterangan' => $ket, 'is_default' => $pertama ? 1 : 0]);
        if (!empty($_POST['jadikan_default'])) {
            $this->setDefault($tahun);
        }
        if (!empty($_POST['pakai'])) {
            $_SESSION['tahun'] = $tahun;
        }
        flash('success', "Tahun anggaran $tahun ditambahkan.");
        redirect('/tahun');
    }

    public function makeDefault(string $tahun): void
    {
        $t = $this->find($tahun);
        $this->setDefault($t);
        flash('success', "Tahun $t sekarang terpilih otomatis saat masuk.");
        redirect('/tahun');
    }

    /** Ganti tahun sesi tanpa keluar. */
    public function switch(): void
    {
        $t = $this->find((string) ($_POST['tahun'] ?? ''));
        $_SESSION['tahun'] = $t;
        flash('success', "Sekarang bekerja di tahun anggaran $t.");
        back('/');
    }

    public function destroy(string $tahun): void
    {
        $t = $this->find($tahun);
        $row = Database::one('SELECT is_default FROM tahun_anggaran WHERE tahun = ?', [$t]);
        $y = (string) $t;
        $dipakai = (int) Database::value('SELECT COUNT(*) FROM rencana_belanja WHERE tahun = ?', [$t])
            + (int) Database::value("SELECT COUNT(*) FROM belanja WHERE strftime('%Y', tgl) = ?", [$y])
            + (int) Database::value("SELECT COUNT(*) FROM penerimaan WHERE strftime('%Y', tgl) = ?", [$y])
            + (int) Database::value("SELECT COUNT(*) FROM mutasi_kas WHERE strftime('%Y', tgl) = ?", [$y]);
        if ($row['is_default']) {
            flash('danger', "Tahun $t adalah tahun default. Jadikan tahun lain default dulu.");
        } elseif ($dipakai) {
            flash('danger', "Tahun $t sudah berisi data, jadi tidak bisa dihapus.");
        } else {
            Database::run('DELETE FROM tahun_anggaran WHERE tahun = ?', [$t]);
            if ((int) ($_SESSION['tahun'] ?? 0) === $t) {
                unset($_SESSION['tahun']);
            }
            flash('success', "Tahun $t dihapus.");
        }
        redirect('/tahun');
    }

    private function setDefault(int $tahun): void
    {
        Database::transaction(function () use ($tahun) {
            Database::run('UPDATE tahun_anggaran SET is_default = 0');
            Database::run('UPDATE tahun_anggaran SET is_default = 1 WHERE tahun = ?', [$tahun]);
            Database::run('UPDATE sekolah SET tahun_anggaran = ? WHERE id = 1', [$tahun]);
        });
    }

    private function find(string $tahun): int
    {
        $t = (int) $tahun;
        if (!in_array($t, tahun_list(), true)) {
            abort(404, 'Tahun anggaran tidak terdaftar.');
        }
        return $t;
    }
}

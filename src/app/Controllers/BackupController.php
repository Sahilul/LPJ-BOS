<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use ZipArchive;

final class BackupController
{
    public function index(): void
    {
        $db = config('db_path');
        $uploads = glob(config('upload_dir') . '/*') ?: [];
        view('backup/index', [
            'title' => 'Cadangkan & pulihkan',
            'dbSize' => is_file($db) ? filesize($db) : 0,
            'uploadCount' => count($uploads),
            'uploadSize' => array_sum(array_map('filesize', $uploads)),
        ]);
    }

    public function download(): void
    {
        $pdo = Database::pdo();
        $pdo->exec('PRAGMA wal_checkpoint(TRUNCATE)');

        $tmpDb = config('tmp_dir') . '/snapshot_' . bin2hex(random_bytes(4)) . '.sqlite';
        $pdo->exec('VACUUM INTO ' . $pdo->quote($tmpDb));

        $zipPath = config('tmp_dir') . '/backup_' . bin2hex(random_bytes(4)) . '.zip';
        $zip = new ZipArchive();
        $zip->open($zipPath, ZipArchive::CREATE | ZipArchive::OVERWRITE);
        $zip->addFile($tmpDb, 'lpjbos.sqlite');
        foreach (glob(config('upload_dir') . '/*') ?: [] as $f) {
            if (is_file($f)) {
                $zip->addFile($f, 'uploads/' . basename($f));
            }
        }
        $zip->addFromString('backup.json', json_encode([
            'app' => 'lpjbos',
            'dibuat' => date('c'),
            'sekolah' => sekolah()['nama'] ?? '',
        ], JSON_PRETTY_PRINT));
        $zip->close();
        @unlink($tmpDb);

        $slug = trim(preg_replace('/[^a-z0-9]+/', '-', strtolower(sekolah()['nama'] ?? 'sekolah')), '-') ?: 'sekolah';
        header('Content-Type: application/zip');
        header('Content-Disposition: attachment; filename="lpjbos-' . $slug . '-' . date('Ymd-His') . '.zip"');
        header('Content-Length: ' . filesize($zipPath));
        header('Cache-Control: no-store');
        readfile($zipPath);
        @unlink($zipPath);
        exit;
    }

    public function restore(): void
    {
        $file = $_FILES['berkas'] ?? null;
        if (!$file || $file['error'] !== UPLOAD_ERR_OK) {
            flash('danger', 'Pilih berkas cadangan (.zip) lebih dulu.');
            redirect('/backup');
        }
        if (($_POST['konfirmasi'] ?? '') !== 'PULIHKAN') {
            flash('danger', 'Ketik PULIHKAN untuk mengonfirmasi.');
            redirect('/backup');
        }

        $zip = new ZipArchive();
        if ($zip->open($file['tmp_name']) !== true || $zip->locateName('lpjbos.sqlite') === false) {
            flash('danger', 'Berkas bukan cadangan LPJ BOS yang valid.');
            redirect('/backup');
        }

        $work = config('tmp_dir') . '/restore_' . bin2hex(random_bytes(4));
        mkdir($work, 0775, true);
        $entries = [];
        for ($i = 0; $i < $zip->numFiles; $i++) {
            $name = $zip->getNameIndex($i);
            if ($name === 'lpjbos.sqlite' || preg_match('#^uploads/[a-z0-9_.-]+$#i', $name)) {
                $entries[] = $name;
            }
        }
        $zip->extractTo($work, $entries);
        $zip->close();

        try {
            $check = new \PDO('sqlite:' . $work . '/lpjbos.sqlite');
            $ok = $check->query("SELECT COUNT(*) FROM sqlite_master WHERE name IN ('sekolah', 'users', 'penerima')")->fetchColumn();
            $check = null;
            if ((int) $ok !== 3) {
                throw new \RuntimeException();
            }
        } catch (\Throwable) {
            $this->rrmdir($work);
            flash('danger', 'Database di dalam cadangan rusak atau bukan milik LPJ BOS.');
            redirect('/backup');
        }

        $db = config('db_path');
        Database::close();
        @copy($db, config('tmp_dir') . '/sebelum-pulihkan-' . date('Ymd-His') . '.sqlite');
        foreach (['-wal', '-shm'] as $sfx) {
            @unlink($db . $sfx);
        }
        rename($work . '/lpjbos.sqlite', $db);

        foreach (glob(config('upload_dir') . '/*') ?: [] as $f) {
            @unlink($f);
        }
        foreach (glob($work . '/uploads/*') ?: [] as $f) {
            rename($f, config('upload_dir') . '/' . basename($f));
        }
        $this->rrmdir($work);

        Database::migrate();
        $_SESSION = [];
        session_regenerate_id(true);
        flash('success', 'Data berhasil dipulihkan. Silakan masuk dengan akun dari cadangan.');
        redirect('/login');
    }

    private function rrmdir(string $dir): void
    {
        foreach (glob($dir . '/{,.}[!.,!..]*', GLOB_BRACE) ?: [] as $f) {
            is_dir($f) ? $this->rrmdir($f) : @unlink($f);
        }
        @rmdir($dir);
    }
}

<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Mcp\ApiAuth;

/** Kelola token akses MCP untuk asisten AI (Claude, Copilot, dll.). */
final class ApiTokenController
{
    public function index(): void
    {
        $tokens = Database::all(
            'SELECT t.*, (SELECT COUNT(*) FROM api_log l WHERE l.token_id = t.id) AS n_panggil
             FROM api_token t ORDER BY t.id DESC'
        );
        $log = Database::all(
            'SELECT l.*, t.nama AS token_nama FROM api_log l LEFT JOIN api_token t ON t.id = l.token_id
             ORDER BY l.id DESC LIMIT 30'
        );
        $baru = $_SESSION['api_token_baru'] ?? null;
        unset($_SESSION['api_token_baru']);
        $url = (is_https() ? 'https' : 'http') . '://' . ($_SERVER['HTTP_HOST'] ?? 'localhost:8080') . '/mcp';
        view('api/index', ['title' => 'Akses AI (MCP)', 'tokens' => $tokens, 'log' => $log, 'baru' => $baru, 'url' => $url]);
    }

    public function store(): void
    {
        $nama = trim((string) ($_POST['nama'] ?? ''));
        $akses = ($_POST['akses'] ?? '') === ApiAuth::TULIS ? ApiAuth::TULIS : ApiAuth::BACA;
        if (mb_strlen($nama) > 120) {
            with_old($_POST, ['nama' => 'Nama maksimal 120 karakter.']);
            redirect('/api');
        }
        $hasil = ApiAuth::create($nama, $akses);
        // Token asli hanya ditampilkan sekali di halaman berikutnya.
        $_SESSION['api_token_baru'] = ['token' => $hasil['token'], 'nama' => $hasil['row']['nama'] ?? $nama];
        flash('success', 'Token dibuat. Salin sekarang, token tidak akan ditampilkan lagi.');
        redirect('/api');
    }

    public function destroy(string $id): void
    {
        $row = Database::one('SELECT * FROM api_token WHERE id = ?', [(int) $id]);
        if (!$row) {
            abort(404, 'Token tidak ditemukan.');
        }
        Database::run('DELETE FROM api_token WHERE id = ?', [(int) $row['id']]);
        flash('success', 'Token “' . $row['nama'] . '” dicabut. Aplikasi yang memakainya tidak bisa masuk lagi.');
        redirect('/api');
    }
}

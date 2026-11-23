<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;

final class AuthController
{
    public function setupForm(): void
    {
        if (Database::value('SELECT COUNT(*) FROM users')) {
            redirect('/login');
        }
        view('auth/setup', ['title' => 'Siapkan akun'], 'guest');
    }

    public function setup(): void
    {
        if (Database::value('SELECT COUNT(*) FROM users')) {
            redirect('/login');
        }
        $in = [
            'nama' => trim($_POST['nama'] ?? ''),
            'username' => strtolower(trim($_POST['username'] ?? '')),
            'sekolah' => trim($_POST['sekolah'] ?? ''),
        ];
        $pass = (string) ($_POST['password'] ?? '');
        $errors = $this->validateAccount($in, $pass, (string) ($_POST['password2'] ?? ''), true);
        if ($errors) {
            with_old($in, $errors);
            redirect('/setup');
        }
        $id = Database::insert('users', [
            'username' => $in['username'],
            'nama' => $in['nama'],
            'password_hash' => password_hash($pass, PASSWORD_DEFAULT),
        ]);
        if ($in['sekolah'] !== '') {
            Database::run('UPDATE sekolah SET nama = ? WHERE id = 1', [$in['sekolah']]);
        }
        session_regenerate_id(true);
        $_SESSION['user_id'] = $id;
        $_SESSION['tahun'] = tahun_default();
        flash('success', 'Akun siap. Lengkapi profil sekolah lebih dulu.');
        redirect('/sekolah');
    }

    public function loginForm(): void
    {
        if (auth_user()) {
            redirect('/');
        }
        $list = tahun_list() ?: [tahun_default()];
        view('auth/login', [
            'title' => 'Masuk',
            'tahunOptions' => array_combine($list, array_map(fn ($t) => $t === tahun_default() ? "$t (default)" : (string) $t, $list)),
        ], 'guest');
    }

    public function login(): void
    {
        $username = strtolower(trim($_POST['username'] ?? ''));
        $tahun = (int) ($_POST['tahun'] ?? 0);
        if (!in_array($tahun, tahun_list(), true)) {
            $tahun = tahun_default();
        }
        $user = Database::one('SELECT * FROM users WHERE username = ?', [$username]);
        if (!$user || !password_verify((string) ($_POST['password'] ?? ''), $user['password_hash'])) {
            usleep(400000);
            with_old(['username' => $username, 'tahun' => $tahun], ['password' => 'Nama pengguna atau kata sandi salah.']);
            redirect('/login');
        }
        session_regenerate_id(true);
        $_SESSION['user_id'] = (int) $user['id'];
        $_SESSION['tahun'] = $tahun;
        redirect('/');
    }

    public function logout(): void
    {
        $_SESSION = [];
        session_destroy();
        redirect('/login');
    }

    public function akunForm(): void
    {
        view('auth/akun', ['title' => 'Akun', 'user' => auth_user()]);
    }

    public function akun(): void
    {
        $user = auth_user();
        $row = Database::one('SELECT * FROM users WHERE id = ?', [$user['id']]);
        $in = [
            'nama' => trim($_POST['nama'] ?? ''),
            'username' => strtolower(trim($_POST['username'] ?? '')),
        ];
        $pass = (string) ($_POST['password'] ?? '');
        $errors = $this->validateAccount($in, $pass, (string) ($_POST['password2'] ?? ''), $pass !== '');
        if (!password_verify((string) ($_POST['password_lama'] ?? ''), $row['password_hash'])) {
            $errors['password_lama'] = 'Kata sandi lama tidak cocok.';
        }
        $dup = Database::value('SELECT id FROM users WHERE username = ? AND id <> ?', [$in['username'], $user['id']]);
        if ($dup) {
            $errors['username'] = 'Nama pengguna sudah dipakai.';
        }
        if ($errors) {
            with_old($in, $errors);
            redirect('/akun');
        }
        $data = $in;
        if ($pass !== '') {
            $data['password_hash'] = password_hash($pass, PASSWORD_DEFAULT);
        }
        Database::update('users', $data, (int) $user['id']);
        flash('success', 'Akun diperbarui.');
        redirect('/akun');
    }

    private function validateAccount(array $in, string $pass, string $pass2, bool $checkPass): array
    {
        $errors = [];
        if ($in['nama'] === '') {
            $errors['nama'] = 'Isi nama Anda.';
        }
        if (!preg_match('/^[a-z0-9._-]{3,32}$/', $in['username'])) {
            $errors['username'] = 'Gunakan 3–32 huruf kecil, angka, titik atau garis.';
        }
        if ($checkPass) {
            if (strlen($pass) < 8) {
                $errors['password'] = 'Minimal 8 karakter.';
            } elseif ($pass !== $pass2) {
                $errors['password2'] = 'Ulangi kata sandi yang sama.';
            }
        }
        return $errors;
    }
}

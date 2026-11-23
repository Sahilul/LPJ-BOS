<?php

declare(strict_types=1);

use App\Database;

function config(string $key, mixed $default = null): mixed
{
    static $cfg = null;
    $cfg ??= require __DIR__ . '/config.php';
    return $cfg[$key] ?? $default;
}

function e(mixed $value): string
{
    return htmlspecialchars((string) ($value ?? ''), ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
}

function rupiah(float|int|string|null $n, bool $prefix = true): string
{
    $n = (float) ($n ?? 0);
    $s = number_format(abs($n), 0, ',', '.');
    $s = ($prefix ? 'Rp' : '') . $s;
    return $n < 0 ? '-' . $s : $s;
}

function angka(float|int|string|null $n, int $dec = 0): string
{
    return number_format((float) ($n ?? 0), $dec, ',', '.');
}

function persen(float|int|string|null $n): string
{
    $n = (float) $n;
    return rtrim(rtrim(number_format($n, 2, ',', '.'), '0'), ',') . '%';
}

function terbilang(float|int $n): string
{
    $n = (int) floor(abs($n));
    $satuan = ['', 'satu', 'dua', 'tiga', 'empat', 'lima', 'enam', 'tujuh', 'delapan', 'sembilan', 'sepuluh', 'sebelas'];
    $f = function (int $x) use (&$f, $satuan): string {
        return match (true) {
            $x < 12 => $satuan[$x],
            $x < 20 => $f($x - 10) . ' belas',
            $x < 100 => $f(intdiv($x, 10)) . ' puluh' . ($x % 10 ? ' ' . $f($x % 10) : ''),
            $x < 200 => 'seratus' . ($x - 100 ? ' ' . $f($x - 100) : ''),
            $x < 1000 => $f(intdiv($x, 100)) . ' ratus' . ($x % 100 ? ' ' . $f($x % 100) : ''),
            $x < 2000 => 'seribu' . ($x - 1000 ? ' ' . $f($x - 1000) : ''),
            $x < 1000000 => $f(intdiv($x, 1000)) . ' ribu' . ($x % 1000 ? ' ' . $f($x % 1000) : ''),
            $x < 1000000000 => $f(intdiv($x, 1000000)) . ' juta' . ($x % 1000000 ? ' ' . $f($x % 1000000) : ''),
            $x < 1000000000000 => $f(intdiv($x, 1000000000)) . ' miliar' . ($x % 1000000000 ? ' ' . $f($x % 1000000000) : ''),
            default => $f(intdiv($x, 1000000000000)) . ' triliun' . ($x % 1000000000000 ? ' ' . $f($x % 1000000000000) : ''),
        };
    };
    return $n === 0 ? 'nol' : trim(preg_replace('/\s+/', ' ', $f($n)));
}

function bulan_indo(int $m): string
{
    return ['', 'Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'][$m] ?? '';
}

function tanggal_indo(?string $date, bool $short = false): string
{
    if (!$date) {
        return '';
    }
    $ts = strtotime($date);
    if ($ts === false) {
        return $date;
    }
    $bulan = bulan_indo((int) date('n', $ts));
    return date('j', $ts) . ' ' . ($short ? substr($bulan, 0, 3) : $bulan) . ' ' . date('Y', $ts);
}

function url(string $path = '/', array $query = []): string
{
    $path = '/' . ltrim($path, '/');
    return $query ? $path . '?' . http_build_query($query) : $path;
}

function asset(string $path): string
{
    $file = __DIR__ . '/../public/assets/' . ltrim($path, '/');
    $v = is_file($file) ? filemtime($file) : 0;
    return '/assets/' . ltrim($path, '/') . '?v=' . $v;
}

function upload_url(?string $name): string
{
    return $name ? '/file/' . rawurlencode($name) . '?v=' . (@filemtime(upload_path($name)) ?: 0) : '';
}

function upload_path(string $name): string
{
    return config('upload_dir') . '/' . basename($name);
}

function redirect(string $to): never
{
    header('Location: ' . $to);
    exit;
}

/** HTTPS langsung atau di balik reverse proxy (Traefik/Dokploy, Nginx). */
function is_https(): bool
{
    if (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') {
        return true;
    }
    return strtolower((string) ($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? '')) === 'https';
}

function back(string $fallback = '/'): never
{
    $ref = $_SERVER['HTTP_REFERER'] ?? '';
    $host = $_SERVER['HTTP_HOST'] ?? '';
    if ($ref && parse_url($ref, PHP_URL_HOST) === parse_url('http://' . $host, PHP_URL_HOST)) {
        redirect($ref);
    }
    redirect($fallback);
}

function flash(string $type, string $message): void
{
    $_SESSION['_flash'][] = ['type' => $type, 'message' => $message];
}

function take_flash(): array
{
    $f = $_SESSION['_flash'] ?? [];
    unset($_SESSION['_flash']);
    return $f;
}

function with_old(array $input, array $errors = []): void
{
    $_SESSION['_old'] = $input;
    $_SESSION['_errors'] = $errors;
}

function old(string $key, mixed $default = ''): mixed
{
    return $GLOBALS['_old'][$key] ?? $default;
}

function field_error(string $key): ?string
{
    return $GLOBALS['_errors'][$key] ?? null;
}

function csrf_token(): string
{
    if (empty($_SESSION['_csrf'])) {
        $_SESSION['_csrf'] = bin2hex(random_bytes(32));
    }
    return $_SESSION['_csrf'];
}

function csrf_field(): string
{
    return '<input type="hidden" name="_csrf" value="' . e(csrf_token()) . '">';
}

function csrf_valid(): bool
{
    $sent = $_POST['_csrf'] ?? ($_SERVER['HTTP_X_CSRF_TOKEN'] ?? '');
    return is_string($sent) && hash_equals(csrf_token(), $sent);
}

function auth_user(): ?array
{
    $id = $_SESSION['user_id'] ?? null;
    if (!$id) {
        return null;
    }
    static $user = null;
    $user ??= Database::one('SELECT id, username, nama FROM users WHERE id = ?', [$id]);
    return $user ?: null;
}

function sekolah(): array
{
    static $s = null;
    $s ??= Database::one('SELECT * FROM sekolah WHERE id = 1') ?: [];
    return $s;
}

/** Daftar tahun anggaran yang terdaftar, terbaru dulu. */
function tahun_list(): array
{
    return array_map('intval', array_column(Database::all('SELECT tahun FROM tahun_anggaran ORDER BY tahun DESC'), 'tahun'));
}

function tahun_default(): int
{
    $t = Database::value('SELECT tahun FROM tahun_anggaran ORDER BY is_default DESC, tahun DESC LIMIT 1');
    return $t ? (int) $t : (int) date('Y');
}

/** Tahun anggaran sesi ini: dipilih saat masuk, bisa diganti dari menu. */
function tahun_aktif(): int
{
    $t = (int) ($_SESSION['tahun'] ?? 0);
    if ($t && in_array($t, tahun_list(), true)) {
        return $t;
    }
    return $_SESSION['tahun'] = tahun_default();
}

/** Tanggal isian bawaan: hari ini bila masih di tahun aktif, selain itu 1 Januari tahun aktif. */
function tgl_default(): string
{
    $t = tahun_aktif();
    return (int) date('Y') === $t ? date('Y-m-d') : "$t-01-01";
}

function dalam_tahun_aktif(string $tgl): bool
{
    return valid_date($tgl) && (int) substr($tgl, 0, 4) === tahun_aktif();
}

function pejabat_aktif(string $peran, ?string $tanggal = null): ?array
{
    $tanggal ??= date('Y-m-d');
    return Database::one(
        'SELECT * FROM pejabat WHERE peran = ? AND mulai <= ? AND (selesai IS NULL OR selesai = "" OR selesai >= ?)
         ORDER BY mulai DESC LIMIT 1',
        [$peran, $tanggal, $tanggal]
    ) ?: null;
}

function view(string $name, array $data = [], ?string $layout = 'app'): void
{
    $GLOBALS['_old'] = $_SESSION['_old'] ?? [];
    $GLOBALS['_errors'] = $_SESSION['_errors'] ?? [];
    unset($_SESSION['_old'], $_SESSION['_errors']);

    $render = function (string $__file, array $__data): string {
        extract($__data, EXTR_SKIP);
        ob_start();
        require $__file;
        return (string) ob_get_clean();
    };

    $content = $render(__DIR__ . '/views/' . $name . '.php', $data);
    if ($layout === null) {
        echo $content;
        return;
    }
    echo $render(__DIR__ . '/views/layouts/' . $layout . '.php', $data + ['content' => $content]);
}

function partial(string $__partial, array $__data = []): void
{
    (static function (string $__file, array $__vars): void {
        extract($__vars);
        require $__file;
    })(__DIR__ . '/views/' . $__partial . '.php', $__data);
}

function abort(int $code, string $message = ''): never
{
    http_response_code($code);
    view('errors/error', ['code' => $code, 'message' => $message, 'title' => 'Galat ' . $code], auth_user() ? 'app' : 'guest');
    exit;
}

function is_active(string $prefix): bool
{
    $path = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?: '/';
    return $prefix === '/' ? $path === '/' : str_starts_with($path, $prefix);
}

function digits(?string $s): string
{
    return preg_replace('/\D+/', '', (string) $s);
}

function icon(string $name, int $size = 18): string
{
    $paths = [
        'home' => '<path d="M3 10.5 12 3l9 7.5"/><path d="M5 9.5V21h14V9.5"/><path d="M10 21v-6h4v6"/>',
        'school' => '<path d="M3 21h18"/><path d="M5 21V10l7-5 7 5v11"/><path d="M10 21v-5h4v5"/><circle cx="12" cy="11" r="1.5"/>',
        'image' => '<rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="10" r="2"/><path d="m21 16-5-5-9 9"/>',
        'badge' => '<circle cx="12" cy="8" r="4"/><path d="M5 21c1.5-4 4-6 7-6s5.5 2 7 6"/>',
        'users' => '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20c1-3.5 3.5-5.5 6.5-5.5s5.5 2 6.5 5.5"/><path d="M16 4.5a3.5 3.5 0 0 1 0 7"/><path d="M18 14.8c1.8.8 3 2.6 3.5 5.2"/>',
        'store' => '<path d="M4 9h16l-1.5-5h-13z"/><path d="M5 9v11h14V9"/><path d="M9 20v-6h6v6"/>',
        'layers' => '<path d="m12 3 9 5-9 5-9-5z"/><path d="m3 13 9 5 9-5"/>',
        'hash' => '<path d="M5 9h15M4 15h15M10 3 8 21M16 3l-2 18"/>',
        'percent' => '<path d="M19 5 5 19"/><circle cx="7" cy="7" r="2.5"/><circle cx="17" cy="17" r="2.5"/>',
        'book' => '<path d="M4 4h6a3 3 0 0 1 3 3v14a2 2 0 0 0-2-2H4z"/><path d="M20 4h-6a3 3 0 0 0-3 3"/><path d="M20 4v15h-7"/>',
        'wallet' => '<rect x="3" y="6" width="18" height="14" rx="2"/><path d="M3 10h18"/><path d="M16 15h2"/><path d="M6 6V5a1 1 0 0 1 1-1h11"/>',
        'list' => '<path d="M9 6h11M9 12h11M9 18h11"/><circle cx="4.5" cy="6" r="1"/><circle cx="4.5" cy="12" r="1"/><circle cx="4.5" cy="18" r="1"/>',
        'chart' => '<path d="M4 20V4"/><path d="M4 20h16"/><path d="M8 16v-5M12 16V8M16 16v-3"/>',
        'database' => '<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5"/><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/>',
        'key' => '<circle cx="8" cy="15" r="4"/><path d="m11 12 9-9"/><path d="m17 6 3 3"/>',
        'plug' => '<path d="M9 3v5M15 3v5"/><path d="M6 8h12v3a6 6 0 0 1-12 0z"/><path d="M12 17v4"/>',
        'logout' => '<path d="M15 4h4v16h-4"/><path d="M10 8l-4 4 4 4"/><path d="M6 12h11"/>',
        'plus' => '<path d="M12 5v14M5 12h14"/>',
        'search' => '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
        'download' => '<path d="M12 4v11"/><path d="m7 10 5 5 5-5"/><path d="M4 20h16"/>',
        'upload' => '<path d="M12 20V9"/><path d="m7 14 5-5 5 5"/><path d="M4 4h16"/>',
        'edit' => '<path d="M4 20h4L19 9l-4-4L4 16z"/><path d="m13.5 6.5 4 4"/>',
        'trash' => '<path d="M4 7h16"/><path d="M9 7V4h6v3"/><path d="M6 7l1 13h10l1-13"/>',
        'check' => '<path d="m5 12 5 5 9-10"/>',
        'x' => '<path d="M6 6l12 12M18 6 6 18"/>',
        'arrow' => '<path d="M5 12h14"/><path d="m13 6 6 6-6 6"/>',
        'menu' => '<path d="M4 7h16M4 12h16M4 17h16"/>',
        'eye' => '<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>',
        'pen' => '<path d="M3 21c3-1 5-3 7-6"/><path d="M14 4l6 6-8 8-6-6z"/>',
        'power' => '<path d="M12 3v8"/><path d="M6.3 7.5a8 8 0 1 0 11.4 0"/>',
        'file' => '<path d="M6 3h8l5 5v13H6z"/><path d="M14 3v5h5"/>',
        'cart' => '<path d="M3 4h2.5l2.2 11h10.6L20.5 8H7"/><circle cx="9.5" cy="19.5" r="1.3"/><circle cx="17" cy="19.5" r="1.3"/>',
        'inbox' => '<path d="M3 13h5l1.5 3h5l1.5-3h5"/><path d="M5.5 5h13L21 13v6H3v-6z"/>',
        'calendar' => '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/>',
        'swap' => '<path d="M4 8h14"/><path d="m14 4 4 4-4 4"/><path d="M20 16H6"/><path d="m10 12-4 4 4 4"/>',
    ];
    $p = $paths[$name] ?? $paths['file'];
    return '<svg class="ic" width="' . $size . '" height="' . $size . '" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' . $p . '</svg>';
}

/** Angka dari isian rupiah "1.200.000" atau desimal "1,5". */
function uang(mixed $raw): float
{
    return (float) digits((string) $raw);
}

function desimal(mixed $raw): float
{
    $s = str_replace([' ', '.'], '', (string) $raw);
    return (float) str_replace(',', '.', $s);
}

/** Sisa pagu satu baris rencana (pagu − total belanja), tanpa menghitung belanja $kecuali. */
function sisa_rencana(int $rencanaId, ?int $kecuali = null): float
{
    $r = App\Database::one('SELECT pagu FROM rencana_belanja WHERE id = ?', [$rencanaId]);
    if (!$r) {
        return 0.0;
    }
    $pakai = (float) App\Database::value('SELECT COALESCE(SUM(total), 0) FROM belanja WHERE rencana_id = ? AND id != ?', [$rencanaId, $kecuali ?? 0]);
    return (float) $r['pagu'] - $pakai;
}

function tarif_pajak(string $kode): float
{
    static $cache = [];
    return $cache[$kode] ??= (float) App\Database::value('SELECT tarif FROM tarif_pajak WHERE kode = ?', [$kode]);
}

/** Tarif PPh 21 (persen) untuk penerima honor berdasarkan kategori, golongan, NPWP. */
function tarif_pph21(array $p): float
{
    if (($p['kategori'] ?? '') === 'PNS') {
        // "III/a", "III.b" dan "III" dianggap sama
        $gol = preg_match('#^(IV|III|II|I)(?![A-Z])#', strtoupper(trim((string) ($p['golongan'] ?? ''))), $m) ? $m[1] : '';
        return $gol !== '' ? tarif_pajak('PPH21_GOL_' . $gol) : 0.0;
    }
    return digits($p['npwp'] ?? '') !== '' ? tarif_pajak('PPH21_NPWP') : tarif_pajak('PPH21_NON_NPWP');
}


function valid_date(string $s): bool
{
    $d = DateTimeImmutable::createFromFormat('!Y-m-d', $s);
    return $d !== false && $d->format('Y-m-d') === $s;
}

/**
 * Saldo kas tunai dan bank sampai dengan $tgl (inklusif), dihitung sejak 1 Januari tahun yang sama:
 * tiap tahun anggaran berdiri sendiri, sisa tahun lalu dicatat sebagai penerimaan SiLPA.
 * Penerimaan & bunga masuk bank, biaya admin keluar dari bank, tarik/setor memindahkan
 * saldo antar keduanya, belanja keluar sesuai metodenya.
 */
function saldo_kas(string $tgl, ?int $kecualiMutasi = null, ?int $kecualiBelanja = null): array
{
    $db = App\Database::class;
    $awal = substr($tgl, 0, 4) . '-01-01';
    $masuk = (float) $db::value('SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan WHERE tgl BETWEEN ? AND ?', [$awal, $tgl]);
    $m = $db::all('SELECT jenis, COALESCE(SUM(jumlah), 0) AS j FROM mutasi_kas WHERE tgl BETWEEN ? AND ? AND id != ? GROUP BY jenis', [$awal, $tgl, $kecualiMutasi ?? 0]);
    $mut = array_column($m, 'j', 'jenis') + ['tarik' => 0, 'setor' => 0, 'admin_bank' => 0, 'bunga' => 0];
    $b = $db::all('SELECT metode, COALESCE(SUM(total), 0) AS j FROM belanja WHERE tgl BETWEEN ? AND ? AND id != ? GROUP BY metode', [$awal, $tgl, $kecualiBelanja ?? 0]);
    $bel = array_column($b, 'j', 'metode') + ['tunai' => 0, 'bank' => 0];
    $tunai = $mut['tarik'] - $mut['setor'] - $bel['tunai'];
    $bank = $masuk + $mut['bunga'] - $mut['admin_bank'] - $mut['tarik'] + $mut['setor'] - $bel['bank'];
    return ['tunai' => round((float) $tunai, 2), 'bank' => round((float) $bank, 2), 'total' => round((float) ($tunai + $bank), 2)];
}

/**
 * Tanggal belanja dianggap lunas dibayar: tanggal transaksi bila saldo kas
 * (sesuai metode) per tanggal itu tidak minus; bila minus, tanggal pertama
 * sesudahnya saat dana cair (penerimaan/mutasi) membuat saldo tidak minus lagi.
 * Null bila sampai akhir tahun saldo masih minus.
 */
function tgl_lunas(string $tgl, string $metode): ?string
{
    $metode = $metode === 'tunai' ? 'tunai' : 'bank';
    if (saldo_kas($tgl)[$metode] > -0.005) {
        return $tgl;
    }
    $akhir = substr($tgl, 0, 4) . '-12-31';
    $tanggal = App\Database::all(
        'SELECT tgl FROM penerimaan WHERE tgl > ? AND tgl <= ?
         UNION SELECT tgl FROM mutasi_kas WHERE tgl > ? AND tgl <= ?
         ORDER BY tgl',
        [$tgl, $akhir, $tgl, $akhir]
    );
    foreach (array_column($tanggal, 'tgl') as $d) {
        if (saldo_kas($d)[$metode] > -0.005) {
            return $d;
        }
    }
    return null;
}

/** Nomor bukti BKU berikutnya; nomor akhir mengikuti urutan tanggal semua transaksi tahun berjalan. */
function no_bukti_berikut(string $tgl): string
{
    $tahun = substr($tgl, 0, 4);
    $count = (int) App\Database::value(
        "SELECT COUNT(*) FROM (
            SELECT id FROM penerimaan WHERE strftime('%Y', tgl) = ?
            UNION ALL SELECT id FROM belanja WHERE strftime('%Y', tgl) = ?
            UNION ALL SELECT id FROM mutasi_kas WHERE strftime('%Y', tgl) = ?
        )",
        [$tahun, $tahun, $tahun]
    );
    return sprintf('BKU-%03d/%s/%s', $count + 1, substr($tgl, 5, 2), $tahun);
}

/** Rebuild the shared BKU number sequence in chronological order for one year. */
function susun_nomor_bukti_bku(int $tahun): void
{
    $rows = App\Database::all(
        "SELECT * FROM (
            SELECT 'trm' AS src, id, tgl FROM penerimaan
            UNION ALL SELECT 'bel', id, tgl FROM belanja
            UNION ALL SELECT 'mut', id, tgl FROM mutasi_kas
        )
        WHERE strftime('%Y', tgl) = ?
        ORDER BY tgl, CASE src WHEN 'trm' THEN 0 WHEN 'mut' THEN 1 ELSE 2 END, id",
        [(string) $tahun]
    );
    $tables = ['trm' => 'penerimaan', 'bel' => 'belanja', 'mut' => 'mutasi_kas'];
    foreach ($rows as $index => $row) {
        $nomor = sprintf('BKU-%03d/%s/%d', $index + 1, substr($row['tgl'], 5, 2), $tahun);
        App\Database::run('UPDATE ' . $tables[$row['src']] . ' SET no_bukti = ? WHERE id = ?', [$nomor, (int) $row['id']]);
    }
}

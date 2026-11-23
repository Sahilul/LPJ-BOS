<?php

declare(strict_types=1);

namespace App\Mcp;

use App\Database;

/**
 * Token akses server MCP. Token hanya ditampilkan sekali saat dibuat;
 * yang disimpan adalah hash SHA-256-nya.
 *
 * Format token: lpbos_<prefix>_<rahasia>
 *   prefix  : 8 karakter heksadesimal untuk mencari baris tanpa membocorkan rahasia
 *   rahasia : 32 byte acak (base64url)
 */
final class ApiAuth
{
    public const BACA = 'baca';
    public const TULIS = 'tulis';

    /** @return array{token: string, row: array} */
    public static function create(string $nama, string $akses = self::BACA): array
    {
        $akses = $akses === self::TULIS ? self::TULIS : self::BACA;
        $nama = trim($nama);
        if ($nama === '') {
            $nama = 'Token ' . date('d/m/Y H:i');
        }
        $prefix = bin2hex(random_bytes(4));
        $rahasia = rtrim(strtr(base64_encode(random_bytes(32)), '+/', '-_'), '=');
        $token = "lpbos_{$prefix}_{$rahasia}";
        $id = Database::insert('api_token', [
            'nama' => mb_substr($nama, 0, 120),
            'prefix' => $prefix,
            'token_hash' => self::hash($token),
            'akses' => $akses,
        ]);
        return ['token' => $token, 'row' => Database::one('SELECT * FROM api_token WHERE id = ?', [$id]) ?: []];
    }

    public static function hash(string $token): string
    {
        return hash('sha256', $token);
    }

    /** Token dari header Authorization: Bearer … atau X-API-Token. */
    public static function tokenDariHeader(array $server): ?string
    {
        $auth = (string) ($server['HTTP_AUTHORIZATION'] ?? $server['REDIRECT_HTTP_AUTHORIZATION'] ?? '');
        if ($auth !== '' && preg_match('/^Bearer\s+(\S+)$/i', $auth, $m)) {
            return $m[1];
        }
        $alt = (string) ($server['HTTP_X_API_TOKEN'] ?? '');
        return $alt !== '' ? trim($alt) : null;
    }

    /** @return array|null baris token bila valid */
    public static function verify(?string $token): ?array
    {
        if (!$token || !preg_match('/^lpbos_([0-9a-f]{8})_/', $token, $m)) {
            return null;
        }
        $row = Database::one('SELECT * FROM api_token WHERE prefix = ?', [$m[1]]);
        if (!$row || !hash_equals((string) $row['token_hash'], self::hash($token))) {
            return null;
        }
        return $row;
    }

    public static function sentuh(array $row): void
    {
        Database::run('UPDATE api_token SET last_used_at = datetime(\'now\', \'localtime\') WHERE id = ?', [(int) $row['id']]);
    }

    public static function log(?array $token, string $tool, array $args, bool $ok, string $pesan = ''): void
    {
        $args = array_filter($args, fn ($v) => $v !== null && $v !== '' && $v !== []);
        Database::insert('api_log', [
            'token_id' => $token['id'] ?? null,
            'tool' => mb_substr($tool, 0, 120),
            'args' => mb_substr(json_encode($args, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) ?: '', 0, 4000),
            'ok' => $ok ? 1 : 0,
            'pesan' => mb_substr($pesan, 0, 500),
        ]);
    }
}

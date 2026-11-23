<?php

declare(strict_types=1);

namespace App\Mcp;

/**
 * Transport HTTP untuk MCP (Streamable HTTP, mode JSON tanpa SSE).
 * POST /mcp dengan header Authorization: Bearer <token>.
 */
final class Http
{
    public static function handle(string $method): void
    {
        $_SESSION = [];
        header('Cache-Control: no-store');

        if ($method === 'OPTIONS') {
            http_response_code(204);
            header('Allow: POST, OPTIONS');
            return;
        }
        if ($method !== 'POST') {
            // Server tidak membuka aliran SSE.
            self::json(405, ['jsonrpc' => '2.0', 'id' => null, 'error' => ['code' => -32000, 'message' => 'Gunakan POST.']]);
            return;
        }

        $token = ApiAuth::verify(ApiAuth::tokenDariHeader($_SERVER));
        if (!$token) {
            header('WWW-Authenticate: Bearer realm="lpjbos"');
            self::json(401, ['jsonrpc' => '2.0', 'id' => null, 'error' => ['code' => -32001, 'message' => 'Token API tidak valid.']]);
            return;
        }
        ApiAuth::sentuh($token);

        $body = json_decode((string) file_get_contents('php://input'), true);
        if (!is_array($body)) {
            self::json(400, ['jsonrpc' => '2.0', 'id' => null, 'error' => ['code' => -32700, 'message' => 'JSON tidak valid.']]);
            return;
        }

        $server = new Server(new Registry(), $token);
        $batch = array_is_list($body);
        $jawaban = [];
        foreach ($batch ? $body : [$body] as $msg) {
            $r = is_array($msg) ? $server->handle($msg) : null;
            if ($r !== null) {
                $jawaban[] = $r;
            }
        }

        if (!$jawaban) {
            http_response_code(202); // hanya notifikasi
            return;
        }
        self::json(200, $batch ? $jawaban : $jawaban[0]);
    }

    private static function json(int $kode, array $data): void
    {
        http_response_code($kode);
        header('Content-Type: application/json; charset=utf-8');
        echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    }
}

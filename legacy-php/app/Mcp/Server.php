<?php

declare(strict_types=1);

namespace App\Mcp;

/**
 * Protokol JSON-RPC 2.0 versi MCP (Model Context Protocol).
 * Hanya menangani bungkus pesan; daftar tool ada di App\Mcp\Registry.
 */
final class Server
{
    public const VERSI = '2024-11-05';

    public function __construct(private Registry $registry, private array $token)
    {
    }

    /** @param array $msg satu pesan JSON-RPC */
    public function handle(array $msg): ?array
    {
        $id = $msg['id'] ?? null;
        $method = (string) ($msg['method'] ?? '');
        $params = (array) ($msg['params'] ?? []);

        if ($method === '') {
            return $this->error($id, -32600, 'Pesan bukan permintaan JSON-RPC.');
        }
        // Notifikasi (tanpa id) tidak dibalas.
        $isNotif = !array_key_exists('id', $msg);

        return match ($method) {
            'initialize' => $this->ok($id, [
                'protocolVersion' => self::VERSI,
                'capabilities' => ['tools' => ['listChanged' => false]],
                'serverInfo' => ['name' => 'lpjbos', 'version' => '1.0.0'],
                'instructions' => 'Server MCP LPJ BOS: baca dan catat data RKAS, penerimaan, belanja, mutasi kas, BKU, dan laporan BOSP.',
            ]),
            'ping' => $this->ok($id, (object) []),
            'tools/list' => $this->ok($id, ['tools' => $this->registry->deskripsi()]),
            'tools/call' => $this->panggilTool($id, $params),
            default => $isNotif ? null : $this->error($id, -32601, 'Metode tidak dikenal: ' . $method),
        };
    }

    private function panggilTool(mixed $id, array $params): array
    {
        $nama = (string) ($params['name'] ?? '');
        $args = (array) ($params['arguments'] ?? []);
        $tool = $this->registry->cari($nama);
        if (!$tool) {
            ApiAuth::log($this->token, $nama, $args, false, 'Tool tidak dikenal');
            return $this->error($id, -32602, 'Tool tidak dikenal: ' . $nama);
        }
        if ($tool['akses'] === ApiAuth::TULIS && ($this->token['akses'] ?? ApiAuth::BACA) !== ApiAuth::TULIS) {
            ApiAuth::log($this->token, $nama, $args, false, 'Token hanya baca');
            return $this->hasil($id, 'Token ini hanya boleh membaca. Buat token dengan akses tulis untuk mencatat data.', true);
        }

        try {
            $data = ($tool['handler'])($args);
            ApiAuth::log($this->token, $nama, $args, true);
            return $this->hasil($id, $data);
        } catch (\App\ValidationException $e) {
            $pesan = 'Isian belum valid: ' . implode(' ', array_values($e->errors));
            ApiAuth::log($this->token, $nama, $args, false, $pesan);
            return $this->hasil($id, $pesan, true);
        } catch (\Throwable $e) {
            ApiAuth::log($this->token, $nama, $args, false, $e->getMessage());
            return $this->hasil($id, 'Gagal: ' . $e->getMessage(), true);
        }
    }

    /** Jawaban sukses tool: teks JSON yang mudah dibaca model. */
    private function hasil(mixed $id, mixed $data, bool $galat = false): array
    {
        $teks = is_string($data) ? $data : $this->json($data);
        return $this->ok($id, [
            'content' => [['type' => 'text', 'text' => $teks]],
            'isError' => $galat,
        ]);
    }

    private function json(mixed $data): string
    {
        return json_encode($data, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) ?: 'null';
    }

    private function ok(mixed $id, array $result): array
    {
        return ['jsonrpc' => '2.0', 'id' => $id, 'result' => $result];
    }

    private function error(mixed $id, int $kode, string $pesan): array
    {
        return ['jsonrpc' => '2.0', 'id' => $id, 'error' => ['code' => $kode, 'message' => $pesan]];
    }
}

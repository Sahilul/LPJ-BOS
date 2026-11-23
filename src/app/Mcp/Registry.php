<?php

declare(strict_types=1);

namespace App\Mcp;

/**
 * Daftar tool MCP. Semua tool membaca/menulis lewat service aplikasi
 * supaya aturan validasi tetap satu dengan form web.
 */
final class Registry
{
    /** @var array<string, array>|null */
    private static ?array $tools = null;

    public static function semua(): array
    {
        if (self::$tools !== null) {
            return self::$tools;
        }
        $t = [];
        foreach ([ToolsBaca::daftar(), ToolsTulis::daftar()] as $grup) {
            foreach ($grup as $nama => $tool) {
                $t[$nama] = $tool;
            }
        }
        return self::$tools = $t;
    }

    public static function cari(string $nama): ?array
    {
        return self::semua()[$nama] ?? null;
    }

    /** Bentuk tools/list untuk MCP: tanpa handler. */
    public static function deskripsi(): array
    {
        $out = [];
        foreach (self::semua() as $nama => $tool) {
            $out[] = [
                'name' => $nama,
                'description' => $tool['deskripsi'],
                'inputSchema' => $tool['skema'] ?: ['type' => 'object', 'properties' => (object) []],
            ];
        }
        return $out;
    }
}

<?php

declare(strict_types=1);

namespace App\Mcp;

use App\Controllers\MasterController;
use App\Database;
use App\ValidationException;

/**
 * Tool tulis untuk data master (penerima, toko, komponen, rekening, pajak).
 * Field & aturan diambil dari MasterController::resources() agar sama dengan form web.
 * Ubah bersifat parsial: field yang tidak dikirim tetap memakai nilai lama.
 */
final class ToolsMaster
{
    public static function daftar(): array
    {
        $out = [];
        foreach (MasterController::resources() as $slug => $res) {
            $out[$slug . '_simpan'] = self::simpan($slug, $res);
            $out[$slug . '_hapus'] = self::hapus($slug, $res);
        }
        return $out;
    }

    private static function tipe(array $f): array
    {
        $type = $f['type'] ?? 'text';
        $s = match ($type) {
            'bool' => ['type' => 'boolean'],
            'money', 'percent' => ['type' => 'number'],
            'int' => ['type' => 'integer'],
            default => ['type' => 'string'],
        };
        $desc = $f['label'] . (!empty($f['hint']) ? '. ' . $f['hint'] : '');
        if (isset($f['len'])) {
            $desc .= " ({$f['len']} angka)";
        }
        $s['description'] = $desc;
        if ($type === 'select') {
            $s['enum'] = array_map('strval', array_keys($f['options']));
        }
        return $s;
    }

    private static function simpan(string $slug, array $res): array
    {
        $props = ['id' => ['type' => 'integer', 'description' => 'Isi untuk mengubah data yang sudah ada (cukup kirim field yang berubah).']];
        $wajib = [];
        foreach ($res['fields'] as $k => $f) {
            $props[$k] = self::tipe($f);
            if (!empty($f['required'])) {
                $wajib[] = $k;
            }
        }
        if (!empty($res['toggle'])) {
            $props['aktif'] = ['type' => 'boolean', 'description' => 'false = nonaktifkan, true = aktifkan.'];
        }

        return [
            'akses' => ApiAuth::TULIS,
            'deskripsi' => "Tambah (tanpa id) atau ubah (dengan id) {$res['singular']}. " . $res['lead'],
            'skema' => ['type' => 'object', 'properties' => $props],
            'handler' => function (array $a) use ($res, $wajib): array {
                $id = (int) ($a['id'] ?? 0) ?: null;
                $lama = [];
                if ($id) {
                    $lama = Database::one("SELECT * FROM {$res['table']} WHERE id = ?", [$id]);
                    if (!$lama) {
                        throw new \RuntimeException(ucfirst($res['singular']) . " dengan id $id tidak ditemukan.");
                    }
                } else {
                    foreach ($wajib as $k) {
                        if (!array_key_exists($k, $a)) {
                            throw new ValidationException([$k => 'Wajib diisi.']);
                        }
                    }
                }
                $data = self::validasi($res, $a, $lama, $id);
                if (!empty($res['toggle']) && array_key_exists('aktif', $a)) {
                    $data['aktif'] = $a['aktif'] ? 1 : 0;
                }
                if ($id) {
                    Database::update($res['table'], $data, $id);
                } else {
                    $id = Database::insert($res['table'], $data);
                }
                return ['ok' => true, 'id' => $id, 'data' => Database::one("SELECT * FROM {$res['table']} WHERE id = ?", [$id])];
            },
        ];
    }

    private static function validasi(array $res, array $a, array $lama, ?int $id): array
    {
        $data = [];
        $err = [];
        foreach ($res['fields'] as $k => $f) {
            $type = $f['type'] ?? 'text';
            $raw = array_key_exists($k, $a) ? $a[$k] : ($lama[$k] ?? ($f['default'] ?? ''));
            $v = match ($type) {
                'bool' => empty($raw) ? 0 : 1,
                'money' => is_int($raw) || is_float($raw) ? (float) $raw : (float) digits((string) $raw),
                'int' => (int) $raw,
                'percent' => (float) str_replace(',', '.', (string) $raw),
                'digits' => digits((string) $raw),
                default => trim((string) $raw),
            };
            if (!empty($f['required']) && $v === '') {
                $err[$k] = 'Wajib diisi.';
            }
            if ($type === 'select' && !array_key_exists((string) $v, $f['options'])) {
                $err[$k] = 'Pilihan tidak valid: ' . implode(', ', array_keys($f['options']));
            }
            if ($type === 'digits' && $v !== '' && isset($f['len']) && strlen($v) !== $f['len']) {
                $err[$k] = $f['label'] . ' terdiri dari ' . $f['len'] . ' angka.';
            }
            if ($type === 'percent' && ($v < 0 || $v > 100)) {
                $err[$k] = 'Isi antara 0 dan 100.';
            }
            if (!empty($f['unique']) && $v !== '' && Database::value(
                "SELECT id FROM {$res['table']} WHERE $k = ? AND id <> ?", [$v, $id ?? 0]
            )) {
                $err[$k] = 'Sudah ada data dengan ' . strtolower($f['label']) . ' ini.';
            }
            $data[$k] = $v;
        }
        if ($err) {
            throw new ValidationException($err);
        }
        return $data;
    }

    private static function hapus(string $slug, array $res): array
    {
        $saran = !empty($res['toggle']) ? " Bila sudah dipakai, nonaktifkan saja lewat {$slug}_simpan aktif=false." : '';
        return [
            'akses' => ApiAuth::TULIS,
            'deskripsi' => "Hapus {$res['singular']} yang belum dipakai di transaksi.$saran",
            'skema' => ['type' => 'object', 'properties' => ['id' => ['type' => 'integer', 'description' => "ID {$res['singular']}."]], 'required' => ['id']],
            'handler' => function (array $a) use ($slug, $res, $saran): array {
                $id = (int) ($a['id'] ?? 0);
                $row = Database::one("SELECT * FROM {$res['table']} WHERE id = ?", [$id]);
                if (!$row) {
                    throw new \RuntimeException(ucfirst($res['singular']) . " dengan id $id tidak ditemukan.");
                }
                try {
                    if ($slug === 'komponen' && Database::value('SELECT COUNT(*) FROM rencana_belanja WHERE komponen_kode = ?', [$row['kode']])) {
                        throw new \RuntimeException('dipakai');
                    }
                    Database::run("DELETE FROM {$res['table']} WHERE id = ?", [$id]);
                } catch (\Throwable) {
                    throw new \RuntimeException('Tidak bisa dihapus karena sudah dipakai.' . $saran);
                }
                return ['ok' => true, 'dihapus' => $id];
            },
        ];
    }
}

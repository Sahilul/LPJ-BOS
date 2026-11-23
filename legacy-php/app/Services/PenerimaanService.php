<?php

declare(strict_types=1);

namespace App\Services;

use App\Database;
use App\ValidationException;

final class PenerimaanService
{
    public const TAHAP = ['Tahap 1' => 'Tahap 1', 'Tahap 2' => 'Tahap 2', 'Sisa tahun lalu' => 'Sisa tahun lalu (SiLPA)', 'Lainnya' => 'Lainnya'];

    public static function find(int $id): ?array
    {
        return Database::one('SELECT * FROM penerimaan WHERE id = ?', [$id]) ?: null;
    }

    /** @throws ValidationException */
    public static function validate(array $p, ?int $id, int $tahun): array
    {
        $planId = (int) ($p['rencana_pendapatan_id'] ?? 0);
        $plan = $planId ? Database::one('SELECT * FROM rencana_pendapatan WHERE id = ?', [$planId]) : null;
        $data = [
            'tgl' => (string) ($p['tgl'] ?? ''),
            'no_bukti' => trim((string) ($p['no_bukti'] ?? '')),
            'uraian' => trim((string) ($p['uraian'] ?? '')),
            'tahap' => (string) ($p['tahap'] ?? ''),
            'jumlah' => uang($p['jumlah'] ?? 0),
            'rencana_pendapatan_id' => $plan ? (int) $plan['id'] : null,
        ];
        $err = [];
        if (!valid_date($data['tgl'])) {
            $err['tgl'] = 'Isi tanggal yang valid.';
        } elseif ((int) substr($data['tgl'], 0, 4) !== $tahun) {
            $err['tgl'] = "Tanggal harus di tahun anggaran $tahun. Ganti tahun dari menu bila perlu.";
        }
        if ($data['uraian'] === '') {
            $err['uraian'] = 'Uraian wajib diisi.';
        }
        if (!isset(self::TAHAP[$data['tahap']])) {
            $err['tahap'] = 'Pilih tahap.';
        }
        if ($planId && !$plan) {
            $err['rencana_pendapatan_id'] = 'Pilih rencana pendapatan yang valid.';
        } elseif ($plan && (int) $plan['tahun'] !== $tahun) {
            $err['rencana_pendapatan_id'] = 'Rencana pendapatan bukan milik tahun anggaran yang sedang dibuka.';
        } elseif ($plan && $data['tahap'] !== $plan['tahap']) {
            $err['rencana_pendapatan_id'] = 'Sumber/tahap penerimaan harus sama dengan rencana yang dipilih.';
        }
        if ($data['jumlah'] <= 0) {
            $err['jumlah'] = 'Jumlah harus lebih dari 0.';
        }
        if ($err) {
            throw new ValidationException($err);
        }
        return $data;
    }

    public static function create(array $data): int
    {
        return Database::transaction(function () use ($data) {
            $id = Database::insert('penerimaan', $data);
            susun_nomor_bukti_bku((int) substr($data['tgl'], 0, 4));
            return $id;
        });
    }

    public static function update(int $id, array $data): void
    {
        Database::transaction(function () use ($data, $id) {
            Database::update('penerimaan', $data, $id);
            susun_nomor_bukti_bku((int) substr($data['tgl'], 0, 4));
        });
    }

    public static function delete(array $row): void
    {
        Database::transaction(function () use ($row) {
            Database::run('DELETE FROM penerimaan WHERE id = ?', [(int) $row['id']]);
            susun_nomor_bukti_bku((int) substr($row['tgl'], 0, 4));
        });
    }
}

<?php

declare(strict_types=1);

namespace App\Services;

use App\Database;
use App\ValidationException;

final class MutasiService
{
    public const JENIS = [
        'tarik' => 'Tarik tunai (bank → kas)',
        'setor' => 'Setor ke bank (kas → bank)',
        'admin_bank' => 'Biaya administrasi bank',
        'bunga' => 'Bunga / jasa giro',
    ];

    public const URAIAN = [
        'tarik' => 'Pengambilan uang tunai dari rekening BOSP',
        'setor' => 'Penyetoran sisa kas tunai ke rekening BOSP',
        'admin_bank' => 'Biaya administrasi bank',
        'bunga' => 'Penerimaan jasa giro',
    ];

    public static function find(int $id): ?array
    {
        return Database::one('SELECT * FROM mutasi_kas WHERE id = ?', [$id]) ?: null;
    }

    /** @throws ValidationException */
    public static function validate(array $p, ?int $id, int $tahun): array
    {
        $data = [
            'tgl' => (string) ($p['tgl'] ?? ''),
            'no_bukti' => trim((string) ($p['no_bukti'] ?? '')),
            'jenis' => (string) ($p['jenis'] ?? ''),
            'uraian' => trim((string) ($p['uraian'] ?? '')),
            'jumlah' => uang($p['jumlah'] ?? 0),
        ];
        $err = [];
        if (!valid_date($data['tgl'])) {
            $err['tgl'] = 'Isi tanggal yang valid.';
        } elseif ((int) substr($data['tgl'], 0, 4) !== $tahun) {
            $err['tgl'] = "Tanggal harus di tahun anggaran $tahun. Ganti tahun dari menu bila perlu.";
        }
        if (!isset(self::JENIS[$data['jenis']])) {
            $err['jenis'] = 'Pilih jenis mutasi.';
        }
        if ($data['jumlah'] <= 0) {
            $err['jumlah'] = 'Jumlah harus lebih dari 0.';
        }
        if ($data['uraian'] === '' && isset(self::URAIAN[$data['jenis']])) {
            $data['uraian'] = self::URAIAN[$data['jenis']];
        }
        // Saldo boleh minus sementara, misalnya tarik tunai sebelum penerimaan dicatat.
        if ($err) {
            throw new ValidationException($err);
        }
        return $data;
    }

    public static function create(array $data): int
    {
        return Database::transaction(function () use ($data) {
            $id = Database::insert('mutasi_kas', $data);
            susun_nomor_bukti_bku((int) substr($data['tgl'], 0, 4));
            return $id;
        });
    }

    public static function update(int $id, array $data): void
    {
        Database::transaction(function () use ($data, $id) {
            Database::update('mutasi_kas', $data, $id);
            susun_nomor_bukti_bku((int) substr($data['tgl'], 0, 4));
        });
    }

    public static function delete(array $row): void
    {
        Database::transaction(function () use ($row) {
            Database::run('DELETE FROM mutasi_kas WHERE id = ?', [(int) $row['id']]);
            susun_nomor_bukti_bku((int) substr($row['tgl'], 0, 4));
        });
    }
}

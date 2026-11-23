<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\Pdf;

final class DokumenController
{
    public function kuitansi(string $id): void
    {
        $data = $this->dokumenData($id);
        $receivers = $data['penerima'];

        if ($data['belanja']['jenis'] === 'barang') {
            $receivers = [[
                'nama' => $data['belanja']['toko_nama'] ?: 'Penyedia',
                'nip' => '',
                'jabatan' => '',
                'bruto' => (float) $data['belanja']['total'],
                'pph21' => 0.0,
                'netto' => (float) $data['belanja']['total'],
            ]];
        } elseif (!$receivers) {
            abort(422, 'Belanja ini belum memiliki data penerima untuk kuitansi.');
        } else {
            $receivers = [[
                'nama' => count($receivers) === 1 ? $receivers[0]['nama'] : 'Para penerima sebagaimana daftar terlampir',
                'nip' => count($receivers) === 1 ? $receivers[0]['nip'] : '',
                'jabatan' => '',
                'bruto' => array_sum(array_column($receivers, 'bruto')),
                'pph21' => array_sum(array_column($receivers, 'pph21')),
                'netto' => array_sum(array_column($receivers, 'netto')),
            ]];
        }

        ob_start();
        view('dokumen/kuitansi', $data + ['receivers' => $receivers, 'receiverCount' => count($data['penerima'])], null);
        Pdf::render((string) ob_get_clean(), 'Kuitansi-' . $data['belanja']['no_bukti'] . '.pdf', false);
    }

    public function buktiTerima(string $id): void
    {
        $data = $this->dokumenData($id);
        if (!in_array($data['belanja']['jenis'], ['honor', 'transport'], true)) {
            abort(404, 'Bukti terima hanya tersedia untuk belanja honor atau transport.');
        }
        if (!$data['penerima']) {
            abort(422, 'Belanja ini belum memiliki data penerima untuk bukti terima.');
        }

        ob_start();
        view('dokumen/bukti-terima', $data, null);
        Pdf::render((string) ob_get_clean(), 'Bukti-Terima-' . $data['belanja']['no_bukti'] . '.pdf', true);
    }

    private function dokumenData(string $id): array
    {
        $tahun = tahun_aktif();
        $belanja = Database::one(
            'SELECT b.*, t.nama AS toko_nama, t.alamat AS toko_alamat, r.uraian AS rencana_uraian
             FROM belanja b
             LEFT JOIN toko t ON t.id = b.toko_id
             LEFT JOIN rencana_belanja r ON r.id = b.rencana_id
             WHERE b.id = ? AND b.tgl BETWEEN ? AND ?',
            [(int) $id, "$tahun-01-01", "$tahun-12-31"]
        );
        if (!$belanja) {
            abort(404, 'Transaksi tidak ditemukan di tahun anggaran yang sedang dibuka.');
        }

        $penerima = Database::all('SELECT * FROM penerima_belanja WHERE belanja_id = ? ORDER BY id', [(int) $id]);
        $items = Database::all('SELECT * FROM item_belanja WHERE belanja_id = ? ORDER BY id', [(int) $id]);
        $pajak = Database::all('SELECT * FROM pajak WHERE belanja_id = ? ORDER BY id', [(int) $id]);
        $s = sekolah();
        $kepala = pejabat_aktif('kepala', $belanja['tgl']);
        $bendahara = pejabat_aktif('bendahara', $belanja['tgl']);

        return [
            'belanja' => $belanja,
            'penerima' => $penerima,
            'items' => $items,
            'pajak' => $pajak,
            'sekolah' => $s,
            'kop' => Pdf::imageData($s['kop_path'] ?? null),
            'stempel' => Pdf::imageData($s['stempel_path'] ?? null),
            'kepala' => Pdf::official($kepala),
            'bendahara' => Pdf::official($bendahara),
            'tanggal' => tanggal_indo($belanja['tgl']),
            'tglLunas' => tgl_lunas($belanja['tgl'], (string) $belanja['metode']),
            'lokasiTanggal' => trim(($s['kabupaten'] ?? '') . ', ' . tanggal_indo($belanja['tgl']), ', '),
            'jenisLabel' => $belanja['jenis'] === 'honor' ? 'Honor' : ($belanja['jenis'] === 'transport' ? 'Transport' : 'Belanja barang/jasa'),
        ];
    }
}

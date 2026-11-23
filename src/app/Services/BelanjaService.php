<?php

declare(strict_types=1);

namespace App\Services;

use App\Database;
use App\ValidationException;

final class BelanjaService
{
    public const METODE = ['tunai' => 'Tunai', 'bank' => 'Transfer / non tunai'];

    public static function find(int $id): ?array
    {
        return Database::one('SELECT * FROM belanja WHERE id = ?', [$id]) ?: null;
    }

    /** Belanja beserta item, penerima, pajak, dan lampiran. */
    public static function detail(int $id): ?array
    {
        $row = self::find($id);
        if (!$row) {
            return null;
        }
        $row['items'] = Database::all('SELECT nama, volume, satuan, harga, subtotal FROM item_belanja WHERE belanja_id = ? ORDER BY id', [$id]);
        $row['penerima'] = Database::all('SELECT penerima_id, nama, kategori, golongan, volume, satuan, tarif, bruto, pph21, netto FROM penerima_belanja WHERE belanja_id = ? ORDER BY id', [$id]);
        $row['pajak'] = Database::all('SELECT jenis, dpp, tarif, nilai, tgl_pungut, tgl_setor, ntpn FROM pajak WHERE belanja_id = ? ORDER BY id', [$id]);
        $row['lampiran'] = Database::all('SELECT jenis, nama_asli FROM lampiran WHERE belanja_id = ? ORDER BY id', [$id]);
        return $row;
    }

    /**
     * Input: rencana_id, tgl, no_bukti, jumlah_periode, uraian, metode, toko_id,
     * items[]{nama,volume,satuan,harga}, pajak_ppn/pajak_pph22/pajak_pph23, pen[]{penerima_id,volume,satuan,tarif}.
     *
     * @return array{0: array, 1: array} data belanja dan baris anak (item, penerima, pajak)
     * @throws ValidationException
     */
    public static function validate(array $p, ?int $id, int $tahun): array
    {
        $err = [];
        $rencana = Database::one('SELECT * FROM rencana_belanja WHERE id = ?', [(int) ($p['rencana_id'] ?? 0)]);
        $data = [
            'tgl' => (string) ($p['tgl'] ?? ''),
            'no_bukti' => trim((string) ($p['no_bukti'] ?? '')),
            'rencana_id' => $rencana ? (int) $rencana['id'] : 0,
            'jumlah_periode' => (int) ($p['jumlah_periode'] ?? 0),
            'jenis' => $rencana['jenis'] ?? 'barang',
            'uraian' => trim((string) ($p['uraian'] ?? '')),
            'metode' => isset(self::METODE[$p['metode'] ?? '']) ? $p['metode'] : 'tunai',
            'toko_id' => null,
            'total' => 0.0,
        ];
        $children = ['items' => [], 'penerima' => [], 'pajak' => []];

        if (!valid_date($data['tgl'])) {
            $err['tgl'] = 'Isi tanggal yang valid.';
        } elseif ((int) substr($data['tgl'], 0, 4) !== $tahun) {
            $err['tgl'] = "Tanggal harus di tahun anggaran $tahun. Ganti tahun dari menu bila perlu.";
        }
        if (!$rencana) {
            $err['rencana_id'] = 'Pilih baris rencana belanja.';
        } elseif ((int) $rencana['tahun'] !== $tahun) {
            $err['rencana_id'] = 'Rencana ini milik tahun ' . $rencana['tahun'] . ', bukan tahun anggaran yang sedang dibuka.';
        }
        if ($data['jumlah_periode'] < 1 || $data['jumlah_periode'] > 12) {
            $err['jumlah_periode'] = 'Jumlah periode harus antara 1 sampai 12 bulan.';
        } elseif ($rencana) {
            $periodeSisa = (int) Database::value(
                'SELECT frekuensi_bulan - COALESCE((SELECT SUM(jumlah_periode) FROM belanja WHERE rencana_id = ? AND id != ?), 0)
                 FROM rencana_belanja WHERE id = ?',
                [(int) $rencana['id'], $id ?? 0, (int) $rencana['id']]
            );
            if ($data['jumlah_periode'] > $periodeSisa) {
                $err['jumlah_periode'] = 'Paket ini hanya memiliki ' . max(0, $periodeSisa) . ' bulan yang belum dicatat.';
            }
        }
        if ($data['uraian'] === '') {
            $err['uraian'] = 'Uraian wajib diisi.';
        }
        if ($data['no_bukti'] !== '' && Database::value('SELECT COUNT(*) FROM belanja WHERE no_bukti = ? AND id != ?', [$data['no_bukti'], $id ?? 0])) {
            $err['no_bukti'] = 'Nomor bukti ini sudah dipakai.';
        }

        if ($data['jenis'] === 'barang') {
            $tokoId = (int) ($p['toko_id'] ?? 0);
            $toko = $tokoId ? Database::one('SELECT * FROM toko WHERE id = ?', [$tokoId]) : null;
            $data['toko_id'] = $toko ? (int) $toko['id'] : null;
            foreach ((array) ($p['items'] ?? []) as $it) {
                $nama = trim((string) ($it['nama'] ?? ''));
                $vol = desimal($it['volume'] ?? 0);
                $harga = uang($it['harga'] ?? 0);
                if ($nama === '' && $harga <= 0) {
                    continue;
                }
                if ($nama === '' || $vol <= 0 || $harga <= 0) {
                    $err['items'] = 'Setiap barang perlu nama, volume dan harga.';
                    continue;
                }
                $sub = round($vol * $harga);
                $children['items'][] = ['nama' => $nama, 'volume' => $vol, 'satuan' => trim((string) ($it['satuan'] ?? '')), 'harga' => $harga, 'subtotal' => $sub];
                $data['total'] += $sub;
            }
            if (!$children['items'] && !isset($err['items'])) {
                $err['items'] = 'Tambahkan minimal satu barang atau jasa.';
            }
            $ppn = !empty($p['pajak_ppn']);
            $dpp = $ppn ? round($data['total'] * 100 / (100 + tarif_pajak('PPN'))) : $data['total'];
            foreach (['PPN' => $ppn, 'PPH22' => !empty($p['pajak_pph22']), 'PPH23' => !empty($p['pajak_pph23'])] as $jenis => $on) {
                if ($on && $data['total'] > 0) {
                    $t = tarif_pajak($jenis);
                    $children['pajak'][] = ['jenis' => $jenis, 'dpp' => $dpp, 'tarif' => $t, 'nilai' => round($dpp * $t / 100), 'tgl_pungut' => $data['tgl']];
                }
            }
        } else {
            $pph = ['dpp' => 0.0, 'nilai' => 0.0];
            $seen = [];
            foreach ((array) ($p['pen'] ?? []) as $r) {
                $pid = (int) ($r['penerima_id'] ?? 0);
                if (!$pid) {
                    continue;
                }
                $m = Database::one('SELECT * FROM penerima WHERE id = ?', [$pid]);
                $vol = desimal($r['volume'] ?? 0);
                $tarif = uang($r['tarif'] ?? 0);
                if (!$m || $vol <= 0 || $tarif <= 0) {
                    $err['pen'] = 'Setiap penerima perlu volume dan tarif.';
                    continue;
                }
                if (isset($seen[$pid])) {
                    $err['pen'] = $m['nama'] . ' tercantum dua kali.';
                    continue;
                }
                $seen[$pid] = true;
                $bruto = round($vol * $tarif);
                $t = $data['jenis'] === 'honor' ? tarif_pph21($m) : 0.0;
                $pajak = round($bruto * $t / 100);
                $children['penerima'][] = [
                    'penerima_id' => $pid, 'nama' => $m['nama'], 'nip' => $m['nip'], 'npwp' => $m['npwp'],
                    'kategori' => $m['kategori'], 'golongan' => $m['golongan'], 'jabatan' => $m['jabatan'],
                    'bank' => $m['bank'], 'no_rekening' => $m['no_rekening'],
                    'volume' => $vol, 'satuan' => trim((string) ($r['satuan'] ?? '')), 'tarif' => $tarif,
                    'bruto' => $bruto, 'pph21' => $pajak, 'netto' => $bruto - $pajak,
                ];
                $data['total'] += $bruto;
                if ($pajak > 0) {
                    $pph['dpp'] += $bruto;
                    $pph['nilai'] += $pajak;
                }
            }
            if (!$children['penerima'] && !isset($err['pen'])) {
                $err['pen'] = 'Tambahkan minimal satu penerima.';
            }
            if ($pph['nilai'] > 0) {
                $children['pajak'][] = ['jenis' => 'PPH21', 'dpp' => $pph['dpp'], 'tarif' => round($pph['nilai'] / $pph['dpp'] * 100, 2), 'nilai' => $pph['nilai'], 'tgl_pungut' => $data['tgl']];
            }
        }

        if ($rencana && !$err) {
            $sisa = sisa_rencana((int) $rencana['id'], $id);
            if ($data['total'] > $sisa + 0.001) {
                $err['rencana_id'] = 'Melebihi sisa anggaran ' . rupiah($sisa) . ' (transaksi ' . rupiah($data['total']) . ').';
            }
        }
        if (!$err) {
            $saldo = saldo_kas($data['tgl'], null, $id)[$data['metode']];
            if ($data['total'] > $saldo + 0.001) {
                $err['metode'] = 'Saldo ' . ($data['metode'] === 'tunai' ? 'kas tunai' : 'bank') . ' per ' . tanggal_indo($data['tgl'], true) . ' hanya ' . rupiah($saldo) . '. '
                    . ($data['metode'] === 'tunai' ? 'Catat tarik tunai dulu di Mutasi kas.' : 'Catat penerimaan dana dulu.');
            }
        }
        if ($err) {
            throw new ValidationException($err, 'Belanja belum disimpan. Periksa isian yang ditandai.');
        }
        return [$data, $children];
    }

    public static function create(array $data, array $children): int
    {
        return Database::transaction(function () use ($data, $children) {
            $data['no_bukti'] = $data['no_bukti'] !== '' ? $data['no_bukti'] : no_bukti_berikut($data['tgl']);
            $id = Database::insert('belanja', $data);
            self::saveChildren($id, $children);
            susun_nomor_bukti_bku((int) substr($data['tgl'], 0, 4));
            return $id;
        });
    }

    public static function update(int $id, array $data, array $children): void
    {
        Database::transaction(function () use ($id, $data, $children) {
            if ($data['no_bukti'] === '') {
                unset($data['no_bukti']);
            }
            Database::update('belanja', $data, $id);
            foreach (['item_belanja', 'penerima_belanja', 'pajak'] as $t) {
                Database::run("DELETE FROM $t WHERE belanja_id = ?", [$id]);
            }
            self::saveChildren($id, $children);
            susun_nomor_bukti_bku((int) substr($data['tgl'], 0, 4));
        });
    }

    public static function delete(array $row): void
    {
        $id = (int) $row['id'];
        $files = Database::all('SELECT path FROM lampiran WHERE belanja_id = ?', [$id]);
        Database::transaction(function () use ($id, $row) {
            Database::run('DELETE FROM belanja WHERE id = ?', [$id]);
            susun_nomor_bukti_bku((int) substr($row['tgl'], 0, 4));
        });
        foreach ($files as $file) {
            $path = upload_path($file['path']);
            if (is_file($path) && !unlink($path)) {
                throw new \RuntimeException('Transaksi dihapus, tetapi lampiran ' . basename($path) . ' tidak dapat dibersihkan.');
            }
        }
    }

    private static function saveChildren(int $id, array $c): void
    {
        foreach ($c['items'] as $it) {
            Database::insert('item_belanja', $it + ['belanja_id' => $id]);
        }
        foreach ($c['penerima'] as $p) {
            Database::insert('penerima_belanja', $p + ['belanja_id' => $id]);
        }
        foreach ($c['pajak'] as $p) {
            Database::insert('pajak', $p + ['belanja_id' => $id]);
        }
    }
}

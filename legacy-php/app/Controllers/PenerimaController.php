<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use PhpOffice\PhpSpreadsheet\Cell\DataType;
use PhpOffice\PhpSpreadsheet\IOFactory;
use PhpOffice\PhpSpreadsheet\Spreadsheet;
use PhpOffice\PhpSpreadsheet\Style\Fill;
use PhpOffice\PhpSpreadsheet\Writer\Xlsx;

/** Impor dan ekspor Excel untuk database penerima. */
final class PenerimaController
{
    private const COLUMNS = [
        'nama' => 'Nama',
        'nip' => 'NIP',
        'nik' => 'NIK',
        'npwp' => 'NPWP',
        'kategori' => 'Kategori (PNS/PPPK/GTT/PTT/Luar)',
        'golongan' => 'Golongan (I/II/III/IV/IX)',
        'jabatan' => 'Jabatan',
        'no_hp' => 'No HP',
        'alamat' => 'Alamat',
        'bank' => 'Bank',
        'no_rekening' => 'No Rekening',
        'atas_nama' => 'Atas Nama',
        'tarif_default' => 'Tarif Bawaan',
    ];

    private const TEXT_COLS = ['nip', 'nik', 'npwp', 'no_hp', 'no_rekening'];

    public function template(): void
    {
        $this->send($this->sheet([[
            'nama' => 'Siti Aminah, S.Pd.', 'nip' => '', 'nik' => '3501010101900001', 'npwp' => '',
            'kategori' => 'GTT', 'golongan' => '', 'jabatan' => 'Guru Kelas 4', 'no_hp' => '',
            'alamat' => '', 'bank' => 'BRI', 'no_rekening' => '012301000123456', 'atas_nama' => 'Siti Aminah',
            'tarif_default' => 500000,
        ]]), 'template-penerima.xlsx');
    }

    public function export(): void
    {
        $rows = Database::all('SELECT * FROM penerima ORDER BY nama');
        $this->send($this->sheet($rows), 'penerima-' . date('Ymd') . '.xlsx');
    }

    public function import(): void
    {
        $file = $_FILES['berkas'] ?? null;
        if (!$file || $file['error'] !== UPLOAD_ERR_OK) {
            flash('danger', 'Pilih berkas Excel (.xlsx) lebih dulu.');
            redirect('/penerima');
        }
        $ext = strtolower(pathinfo($file['name'], PATHINFO_EXTENSION));
        if (!in_array($ext, ['xlsx', 'xls', 'csv'], true)) {
            flash('danger', 'Format berkas harus .xlsx, .xls atau .csv.');
            redirect('/penerima');
        }

        try {
            $sheet = IOFactory::load($file['tmp_name'])->getActiveSheet();
            $data = $sheet->toArray(null, true, false, false);
        } catch (\Throwable) {
            flash('danger', 'Berkas tidak bisa dibaca. Gunakan template dari tombol Unduh template.');
            redirect('/penerima');
        }

        $header = array_map(fn ($h) => strtolower(trim((string) $h)), array_shift($data) ?? []);
        $map = [];
        foreach (self::COLUMNS as $key => $label) {
            $idx = array_search(strtolower($label), $header, true);
            if ($idx === false) {
                $idx = array_search(strtolower(strtok($label, ' (')), $header, true);
            }
            if ($idx !== false) {
                $map[$key] = $idx;
            }
        }
        if (!isset($map['nama'])) {
            flash('danger', 'Kolom "Nama" tidak ditemukan di baris pertama.');
            redirect('/penerima');
        }

        $added = $updated = $skipped = 0;
        $kategoriValid = ['PNS', 'PPPK', 'GTT', 'PTT', 'Luar'];
        Database::transaction(function () use ($data, $map, $kategoriValid, &$added, &$updated, &$skipped) {
            foreach ($data as $line) {
                $row = [];
                foreach ($map as $key => $idx) {
                    $row[$key] = trim((string) ($line[$idx] ?? ''));
                }
                if (($row['nama'] ?? '') === '') {
                    continue;
                }
                foreach (['nip', 'nik', 'no_rekening'] as $d) {
                    if (isset($row[$d])) {
                        $row[$d] = digits($row[$d]);
                    }
                }
                if (isset($row['kategori'])) {
                    $k = strtoupper($row['kategori']) === 'LUAR' ? 'Luar' : strtoupper($row['kategori']);
                    $row['kategori'] = in_array($k, $kategoriValid, true) ? $k : 'GTT';
                }
                if (isset($row['golongan'])) {
                    $row['golongan'] = strtoupper(preg_replace('#[/.].*$#', '', $row['golongan']));
                }
                if (isset($row['tarif_default'])) {
                    $row['tarif_default'] = (float) digits($row['tarif_default']);
                }

                $existing = null;
                if (!empty($row['nip'])) {
                    $existing = Database::value('SELECT id FROM penerima WHERE nip = ?', [$row['nip']]);
                }
                if (!$existing && !empty($row['nik'])) {
                    $existing = Database::value('SELECT id FROM penerima WHERE nik = ?', [$row['nik']]);
                }
                if (!$existing) {
                    $existing = Database::value('SELECT id FROM penerima WHERE lower(nama) = lower(?)', [$row['nama']]);
                }
                if ($existing) {
                    Database::update('penerima', $row, (int) $existing);
                    $updated++;
                } else {
                    Database::insert('penerima', $row);
                    $added++;
                }
            }
        });

        flash('success', "Impor selesai: {$added} penerima baru, {$updated} diperbarui.");
        redirect('/penerima');
    }

    private function sheet(array $rows): Spreadsheet
    {
        $book = new Spreadsheet();
        $ws = $book->getActiveSheet();
        $ws->setTitle('Penerima');
        $col = 1;
        foreach (self::COLUMNS as $label) {
            $ws->setCellValue([$col++, 1], $label);
        }
        $last = $ws->getHighestColumn();
        $ws->getStyle("A1:{$last}1")->getFont()->setBold(true);
        $ws->getStyle("A1:{$last}1")->getFill()->setFillType(Fill::FILL_SOLID)->getStartColor()->setRGB('E8EDFC');

        $r = 2;
        foreach ($rows as $row) {
            $col = 1;
            foreach (array_keys(self::COLUMNS) as $key) {
                $v = $row[$key] ?? '';
                if (in_array($key, self::TEXT_COLS, true)) {
                    $ws->setCellValueExplicit([$col, $r], (string) $v, DataType::TYPE_STRING);
                } else {
                    $ws->setCellValue([$col, $r], $v);
                }
                $col++;
            }
            $r++;
        }
        foreach (range('A', $last) as $c) {
            $ws->getColumnDimension($c)->setAutoSize(true);
        }
        $ws->freezePane('A2');
        return $book;
    }

    private function send(Spreadsheet $book, string $filename): never
    {
        header('Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet');
        header('Content-Disposition: attachment; filename="' . $filename . '"');
        header('Cache-Control: no-store');
        (new Xlsx($book))->save('php://output');
        exit;
    }
}

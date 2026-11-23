<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;
use App\Services\Laporan;
use App\Services\Pdf;
use PhpOffice\PhpSpreadsheet\Cell\DataType;
use PhpOffice\PhpSpreadsheet\Spreadsheet;
use PhpOffice\PhpSpreadsheet\Style\Border;
use PhpOffice\PhpSpreadsheet\Style\Fill;
use PhpOffice\PhpSpreadsheet\Worksheet\Worksheet;
use PhpOffice\PhpSpreadsheet\Writer\Xlsx;

final class LaporanController
{
    private const SPJ_BAGIAN = ['bku', 'tunai', 'bank', 'pajak', 'realisasi', 'kas'];

    public function index(): void
    {
        $tahun = tahun_aktif();
        $p = Laporan::periode($_GET['periode'] ?? null, $tahun);
        $saldo = saldo_kas($p['akhir']);
        $n = (int) Database::value(
            'SELECT (SELECT COUNT(*) FROM penerimaan WHERE tgl BETWEEN :a AND :b) + (SELECT COUNT(*) FROM belanja WHERE tgl BETWEEN :a AND :b) + (SELECT COUNT(*) FROM mutasi_kas WHERE tgl BETWEEN :a AND :b)',
            ['a' => $p['awal'], 'b' => $p['akhir']]
        );
        $nBelanja = (int) Database::value('SELECT COUNT(*) FROM belanja WHERE tgl BETWEEN ? AND ?', [$p['awal'], $p['akhir']]);

        view('laporan/index', [
            'title' => 'Laporan & SPJ',
            'tahun' => $tahun,
            'p' => $p,
            'saldo' => $saldo,
            'nTransaksi' => $n,
            'nBelanja' => $nBelanja,
        ]);
    }

    public function show(string $jenis): void
    {
        if (!isset(Laporan::JENIS[$jenis])) {
            abort(404, 'Jenis laporan tidak dikenal.');
        }
        $tahun = tahun_aktif();
        $p = Laporan::periode($_GET['periode'] ?? null, $tahun);
        $data = Laporan::build($jenis, $p);
        $nama = Laporan::JENIS[$jenis][0];
        $file = str_replace(' ', '-', ucwords(str_replace('&', 'dan', $nama))) . '-' . $p['slug'];

        match ($_GET['format'] ?? 'html') {
            'pdf' => $this->pdf([['jenis' => $jenis, 'data' => $data]], $p, "$file.pdf"),
            'xlsx' => $this->xlsx([['jenis' => $jenis, 'data' => $data]], $p, "$file.xlsx"),
            default => view('laporan/show', [
                'title' => $nama . ' · ' . $p['label'],
                'tahun' => $tahun,
                'jenis' => $jenis,
                'nama' => $nama,
                'p' => $p,
                'data' => $data,
            ]),
        };
    }

    /** SPJ lengkap: semua buku, realisasi, register kas, dan rincian bukti belanja dalam satu dokumen. */
    public function spj(): void
    {
        $p = Laporan::periode($_GET['periode'] ?? null, tahun_aktif());
        $parts = array_map(fn ($j) => ['jenis' => $j, 'data' => Laporan::build($j, $p)], self::SPJ_BAGIAN);
        $belanja = Laporan::belanjaPeriode($p);
        if (($_GET['format'] ?? 'pdf') === 'xlsx') {
            $this->xlsx($parts, $p, "SPJ-{$p['slug']}.xlsx", $belanja);
        }
        $this->pdf($parts, $p, "SPJ-{$p['slug']}.pdf", $belanja, true);
    }

    private function pdf(array $parts, array $p, string $filename, ?array $belanja = null, bool $spj = false): never
    {
        $ident = Pdf::identity($p['akhir']);
        $landscape = $spj || count(array_filter($parts, fn ($x) => $x['data']['orientation'] === 'landscape')) > 0;
        ob_start();
        view('laporan/pdf', $ident + [
            'parts' => $parts,
            'p' => $p,
            'belanja' => $belanja,
            'spj' => $spj,
            'landscape' => $landscape,
        ], null);
        Pdf::render((string) ob_get_clean(), $filename, (bool) ($_GET['unduh'] ?? false), $landscape ? 'landscape' : 'portrait');
    }

    private function xlsx(array $parts, array $p, string $filename, ?array $belanja = null): never
    {
        $s = sekolah();
        $book = new Spreadsheet();
        $book->removeSheetByIndex(0);
        foreach ($parts as $part) {
            $ws = new Worksheet($book, mb_substr(Laporan::JENIS[$part['jenis']][0], 0, 31));
            $book->addSheet($ws);
            $this->fillSheet($ws, $part['data'], $p, $s);
        }
        if ($belanja !== null) {
            $ws = new Worksheet($book, 'Rincian belanja');
            $book->addSheet($ws);
            $this->fillBelanja($ws, $belanja, $p, $s);
        }
        $book->setActiveSheetIndex(0);

        header('Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet');
        header('Content-Disposition: attachment; filename="' . preg_replace('/[^A-Za-z0-9._-]/', '-', $filename) . '"');
        header('Cache-Control: no-store');
        (new Xlsx($book))->save('php://output');
        exit;
    }

    private function header(Worksheet $ws, string $title, array $p, array $s, int $cols): int
    {
        $last = $this->col($cols);
        foreach ([$title, $s['nama'] ?? '', 'Periode: ' . $p['label']] as $i => $text) {
            $ws->setCellValue('A' . ($i + 1), $text);
            $ws->mergeCells('A' . ($i + 1) . ":{$last}" . ($i + 1));
        }
        $ws->getStyle('A1')->getFont()->setBold(true)->setSize(13);
        $ws->getStyle("A1:{$last}3")->getAlignment()->setHorizontal('center');
        return 5;
    }

    private function fillSheet(Worksheet $ws, array $data, array $p, array $s): void
    {
        $cols = $data['columns'];
        $keys = array_keys($cols);
        $r = $this->header($ws, $data['title'], $p, $s, count($cols));
        $last = $this->col(count($cols));
        $head = $r;
        foreach ($keys as $i => $k) {
            $ws->setCellValue([$i + 1, $r], $cols[$k][0]);
        }
        $ws->getStyle("A{$r}:{$last}{$r}")->getFont()->setBold(true);
        $ws->getStyle("A{$r}:{$last}{$r}")->getFill()->setFillType(Fill::FILL_SOLID)->getStartColor()->setRGB('E8EDFC');
        $r++;
        $rows = $data['rows'];
        if ($data['foot']) {
            $rows[] = $data['foot'] + ['_class' => 'foot'];
        }
        foreach ($rows as $row) {
            foreach ($keys as $i => $k) {
                $v = $row[$k] ?? null;
                if ($v === null || $v === '') {
                    continue;
                }
                match ($cols[$k][1]) {
                    'money', 'pct' => $ws->setCellValue([$i + 1, $r], (float) $v),
                    'date' => $ws->setCellValue([$i + 1, $r], tanggal_indo((string) $v, true)),
                    default => $ws->setCellValueExplicit([$i + 1, $r], (string) $v, DataType::TYPE_STRING),
                };
            }
            if (in_array($row['_class'] ?? '', ['group', 'sub', 'foot'], true)) {
                $ws->getStyle("A{$r}:{$last}{$r}")->getFont()->setBold(true);
            }
            if (($row['_class'] ?? '') === 'info') {
                $ws->getStyle("A{$r}:{$last}{$r}")->getFont()->setItalic(true)->getColor()->setRGB('777777');
            }
            $r++;
        }
        $ws->getStyle("A{$head}:{$last}" . ($r - 1))->getBorders()->getAllBorders()->setBorderStyle(Border::BORDER_THIN);
        foreach ($keys as $i => $k) {
            $c = $this->col($i + 1);
            if ($cols[$k][1] === 'money') {
                $ws->getStyle("{$c}" . ($head + 1) . ":{$c}{$r}")->getNumberFormat()->setFormatCode('#,##0');
            } elseif ($cols[$k][1] === 'pct') {
                $ws->getStyle("{$c}" . ($head + 1) . ":{$c}{$r}")->getNumberFormat()->setFormatCode('0.00');
            }
            if ($k === 'uraian' || $k === 'nama') {
                $ws->getColumnDimension($c)->setWidth(60);
            } else {
                $ws->getColumnDimension($c)->setAutoSize(true);
            }
        }
        $r++;
        foreach ($data['summary'] as [$label, $value]) {
            $ws->setCellValue("B{$r}", $label);
            $ws->setCellValue("D{$r}", $value);
            if (is_numeric($value)) {
                $ws->getStyle("D{$r}")->getNumberFormat()->setFormatCode('#,##0');
            }
            $r++;
        }
        $ws->freezePane('A' . ($head + 1));
    }

    private function fillBelanja(Worksheet $ws, array $belanja, array $p, array $s): void
    {
        $head = ['Tanggal', 'No. bukti', 'Kode rekening', 'Jenis', 'Uraian', 'Rincian / penerima', 'Volume', 'Satuan', 'Harga / bruto', 'Pajak', 'Jumlah'];
        $r = $this->header($ws, 'RINCIAN BUKTI BELANJA', $p, $s, count($head));
        $start = $r;
        foreach ($head as $i => $h) {
            $ws->setCellValue([$i + 1, $r], $h);
        }
        $ws->getStyle("A{$r}:K{$r}")->getFont()->setBold(true);
        $ws->getStyle("A{$r}:K{$r}")->getFill()->setFillType(Fill::FILL_SOLID)->getStartColor()->setRGB('E8EDFC');
        $r++;
        foreach ($belanja as $b) {
            $ws->fromArray([tanggal_indo($b['tgl'], true), $b['no_bukti'], $b['rekening_kode'] ?? '', $b['jenis_label'], $b['uraian'], $b['toko_nama'] ?? '', null, null, null,
                array_sum(array_column($b['pajak'], 'nilai')), (float) $b['total']], null, "A{$r}");
            $ws->getStyle("A{$r}:K{$r}")->getFont()->setBold(true);
            $r++;
            foreach ($b['items'] as $it) {
                $ws->fromArray([null, null, null, null, null, $it['nama'], (float) $it['volume'], $it['satuan'], (float) $it['harga'], null, (float) $it['subtotal']], null, "A{$r}");
                $r++;
            }
            foreach ($b['penerima'] as $pn) {
                $ws->fromArray([null, null, null, null, null, $pn['nama'] . ($pn['nip'] ? ' / ' . $pn['nip'] : ''), (float) $pn['volume'], $pn['satuan'], (float) $pn['bruto'], (float) $pn['pph21'], (float) $pn['netto']], null, "A{$r}");
                $r++;
            }
        }
        $ws->getStyle("A{$start}:K" . ($r - 1))->getBorders()->getAllBorders()->setBorderStyle(Border::BORDER_THIN);
        $ws->getStyle("I{$start}:K{$r}")->getNumberFormat()->setFormatCode('#,##0');
        foreach (range('A', 'K') as $c) {
            $ws->getColumnDimension($c)->setAutoSize(true);
        }
        $ws->getColumnDimension('E')->setAutoSize(false)->setWidth(50);
        $ws->freezePane('A' . ($start + 1));
    }

    private function col(int $n): string
    {
        return \PhpOffice\PhpSpreadsheet\Cell\Coordinate::stringFromColumnIndex($n);
    }
}

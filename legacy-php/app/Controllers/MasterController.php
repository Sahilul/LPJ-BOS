<?php

declare(strict_types=1);

namespace App\Controllers;

use App\Database;

/**
 * CRUD generik untuk data master. Setiap resource dideskripsikan
 * oleh daftar field; view crud/index dan crud/form merendernya.
 */
final class MasterController
{
    private array $res;

    public function __construct(private string $slug)
    {
        $this->res = self::resources()[$slug];
    }

    public static function resources(): array
    {
        $kategori = ['PNS' => 'PNS', 'PPPK' => 'PPPK', 'GTT' => 'GTT', 'PTT' => 'PTT', 'Luar' => 'Pihak luar'];
        $golongan = ['' => '—', 'I' => 'I', 'II' => 'II', 'III' => 'III', 'IV' => 'IV', 'IX' => 'IX (PPPK)'];

        return [
            'penerima' => [
                'table' => 'penerima',
                'title' => 'Penerima',
                'singular' => 'penerima',
                'lead' => 'Guru, tenaga kependidikan dan pihak luar yang menerima honor atau transport. Data dicatat ulang ke setiap transaksi, jadi mengubahnya tidak mengubah kuitansi lama.',
                'icon' => 'users',
                'order' => 'aktif DESC, nama',
                'search' => ['nama', 'nip', 'nik', 'jabatan'],
                'toggle' => true,
                'excel' => true,
                'filter' => ['kategori' => $kategori],
                'list' => ['nama', 'kategori', 'golongan', 'jabatan', 'no_rekening', 'tarif_default'],
                'fields' => [
                    'nama' => ['label' => 'Nama lengkap', 'required' => true, 'hint' => 'Tulis beserta gelar, seperti di kuitansi.', 'span' => 2],
                    'kategori' => ['label' => 'Kategori', 'type' => 'select', 'options' => $kategori, 'default' => 'GTT'],
                    'golongan' => ['label' => 'Golongan', 'type' => 'select', 'options' => $golongan, 'hint' => 'Menentukan tarif PPh 21 untuk PNS.'],
                    'nip' => ['label' => 'NIP', 'type' => 'digits', 'len' => 18],
                    'nik' => ['label' => 'NIK', 'type' => 'digits', 'len' => 16],
                    'npwp' => ['label' => 'NPWP'],
                    'jabatan' => ['label' => 'Jabatan / tugas', 'hint' => 'Contoh: Guru Kelas 4, Operator Dapodik.'],
                    'no_hp' => ['label' => 'Nomor HP'],
                    'alamat' => ['label' => 'Alamat', 'type' => 'textarea', 'span' => 2],
                    'bank' => ['label' => 'Bank'],
                    'no_rekening' => ['label' => 'Nomor rekening', 'mono' => true],
                    'atas_nama' => ['label' => 'Atas nama rekening'],
                    'tarif_default' => ['label' => 'Tarif bawaan', 'type' => 'money', 'hint' => 'Terisi otomatis saat memilih penerima di transaksi.'],
                ],
            ],
            'toko' => [
                'table' => 'toko',
                'title' => 'Toko & penyedia',
                'singular' => 'toko',
                'lead' => 'Toko dan penyedia jasa tempat sekolah berbelanja. Status PKP menentukan apakah PPN dipungut.',
                'icon' => 'store',
                'order' => 'aktif DESC, nama',
                'search' => ['nama', 'alamat', 'npwp'],
                'toggle' => true,
                'list' => ['nama', 'alamat', 'npwp', 'pkp'],
                'fields' => [
                    'nama' => ['label' => 'Nama toko', 'required' => true, 'span' => 2],
                    'alamat' => ['label' => 'Alamat', 'type' => 'textarea', 'span' => 2],
                    'npwp' => ['label' => 'NPWP'],
                    'telepon' => ['label' => 'Telepon'],
                    'pkp' => ['label' => 'Pengusaha Kena Pajak (PKP)', 'type' => 'bool', 'hint' => 'Centang bila toko memungut PPN.'],
                ],
            ],
            'komponen' => [
                'table' => 'komponen',
                'title' => 'Komponen BOSP',
                'singular' => 'komponen',
                'lead' => 'Komponen penggunaan dana BOSP Reguler. Kode dipakai di RKAS, BKU dan rekap.',
                'icon' => 'layers',
                'order' => 'urutan, kode',
                'search' => ['kode', 'nama'],
                'list' => ['kode', 'nama'],
                'fields' => [
                    'kode' => ['label' => 'Kode', 'required' => true, 'unique' => true, 'mono' => true],
                    'urutan' => ['label' => 'Urutan', 'type' => 'int', 'default' => 0],
                    'nama' => ['label' => 'Nama komponen', 'required' => true, 'span' => 2],
                ],
            ],
            'rekening' => [
                'table' => 'kode_rekening',
                'title' => 'Kode rekening',
                'singular' => 'kode rekening',
                'lead' => 'Kode rekening belanja sesuai ARKAS. Jenis dipakai untuk memilih pajak yang biasa dipungut.',
                'icon' => 'hash',
                'order' => 'kode',
                'search' => ['kode', 'nama'],
                'list' => ['kode', 'nama', 'jenis'],
                'fields' => [
                    'kode' => ['label' => 'Kode rekening', 'required' => true, 'unique' => true, 'mono' => true],
                    'jenis' => ['label' => 'Jenis', 'type' => 'select', 'options' => ['barang' => 'Barang', 'jasa' => 'Jasa', 'honor' => 'Honor', 'modal' => 'Modal', 'lainnya' => 'Lainnya']],
                    'nama' => ['label' => 'Nama rekening', 'required' => true, 'span' => 2],
                ],
            ],
            'pajak' => [
                'table' => 'tarif_pajak',
                'title' => 'Tarif pajak',
                'singular' => 'tarif pajak',
                'lead' => 'Tarif yang dipakai saat menghitung pajak transaksi. Sesuaikan bila aturan daerah Anda berbeda.',
                'icon' => 'percent',
                'order' => 'kode',
                'search' => ['kode', 'nama'],
                'list' => ['kode', 'nama', 'tarif', 'keterangan'],
                'fields' => [
                    'kode' => ['label' => 'Kode', 'required' => true, 'unique' => true, 'mono' => true],
                    'tarif' => ['label' => 'Tarif (%)', 'type' => 'percent', 'default' => 0],
                    'nama' => ['label' => 'Nama pajak', 'required' => true, 'span' => 2],
                    'keterangan' => ['label' => 'Keterangan', 'type' => 'textarea', 'span' => 2],
                ],
            ],
        ];
    }

    public function index(): void
    {
        $r = $this->res;
        $q = trim((string) ($_GET['q'] ?? ''));
        $where = [];
        $params = [];
        if ($q !== '') {
            $where[] = '(' . implode(' OR ', array_map(fn ($c) => "$c LIKE ?", $r['search'])) . ')';
            array_push($params, ...array_fill(0, count($r['search']), "%$q%"));
        }
        foreach ($r['filter'] ?? [] as $col => $opts) {
            $v = (string) ($_GET[$col] ?? '');
            if ($v !== '' && isset($opts[$v])) {
                $where[] = "$col = ?";
                $params[] = $v;
            }
        }
        if (!empty($r['toggle']) && ($_GET['status'] ?? '') !== '') {
            $where[] = 'aktif = ?';
            $params[] = $_GET['status'] === 'nonaktif' ? 0 : 1;
        }
        $sql = "SELECT * FROM {$r['table']}" . ($where ? ' WHERE ' . implode(' AND ', $where) : '') . " ORDER BY {$r['order']}";
        $rows = Database::all($sql, $params);
        $total = (int) Database::value("SELECT COUNT(*) FROM {$r['table']}");

        view('crud/index', [
            'title' => $r['title'],
            'res' => $r,
            'slug' => $this->slug,
            'rows' => $rows,
            'total' => $total,
            'q' => $q,
        ]);
    }

    public function create(): void
    {
        $row = [];
        foreach ($this->res['fields'] as $k => $f) {
            $row[$k] = $f['default'] ?? '';
        }
        view('crud/form', ['title' => 'Tambah ' . $this->res['singular'], 'res' => $this->res, 'slug' => $this->slug, 'row' => $row]);
    }

    public function store(): void
    {
        $data = $this->validated(null);
        $id = Database::insert($this->res['table'], $data);
        flash('success', ucfirst($this->res['singular']) . ' “' . ($data['nama'] ?? $data['kode']) . '” ditambahkan.');
        if (!empty($_POST['_lagi'])) {
            redirect("/{$this->slug}/baru");
        }
        redirect("/{$this->slug}#row-{$id}");
    }

    public function edit(string $id): void
    {
        $row = $this->find($id);
        view('crud/form', ['title' => 'Ubah ' . $this->res['singular'], 'res' => $this->res, 'slug' => $this->slug, 'row' => $row]);
    }

    public function update(string $id): void
    {
        $this->find($id);
        $data = $this->validated((int) $id);
        Database::update($this->res['table'], $data, (int) $id);
        flash('success', 'Perubahan disimpan.');
        redirect("/{$this->slug}#row-{$id}");
    }

    public function destroy(string $id): void
    {
        $row = $this->find($id);
        try {
            if ($this->slug === 'komponen' && Database::value('SELECT COUNT(*) FROM rencana_belanja WHERE komponen_kode = ?', [$row['kode']])) {
                throw new \RuntimeException('dipakai');
            }
            Database::run("DELETE FROM {$this->res['table']} WHERE id = ?", [(int) $id]);
            flash('success', ucfirst($this->res['singular']) . ' dihapus.');
        } catch (\Throwable) {
            $msg = !empty($this->res['toggle']) ? ' Nonaktifkan saja agar riwayatnya tetap utuh.' : '';
            flash('danger', 'Tidak bisa dihapus karena sudah dipakai di transaksi.' . $msg);
        }
        redirect("/{$this->slug}");
    }

    public function toggle(string $id): void
    {
        $row = $this->find($id);
        if (empty($this->res['toggle'])) {
            abort(404);
        }
        $new = $row['aktif'] ? 0 : 1;
        Database::run("UPDATE {$this->res['table']} SET aktif = ? WHERE id = ?", [$new, (int) $id]);
        flash('success', ($row['nama'] ?? '') . ($new ? ' diaktifkan kembali.' : ' dinonaktifkan.'));
        back("/{$this->slug}");
    }

    private function find(string $id): array
    {
        return Database::one("SELECT * FROM {$this->res['table']} WHERE id = ?", [(int) $id]) ?: abort(404, 'Data tidak ditemukan.');
    }

    private function validated(?int $id): array
    {
        $data = [];
        $errors = [];
        foreach ($this->res['fields'] as $k => $f) {
            $type = $f['type'] ?? 'text';
            $raw = $_POST[$k] ?? '';
            $v = match ($type) {
                'bool' => empty($raw) ? 0 : 1,
                'money' => (float) digits((string) $raw),
                'int' => (int) $raw,
                'percent' => (float) str_replace(',', '.', (string) $raw),
                'digits' => digits((string) $raw),
                default => trim((string) $raw),
            };
            if (!empty($f['required']) && $v === '') {
                $errors[$k] = 'Wajib diisi.';
            }
            if ($type === 'select' && !array_key_exists((string) $v, $f['options'])) {
                $errors[$k] = 'Pilihan tidak valid.';
            }
            if ($type === 'digits' && $v !== '' && isset($f['len']) && strlen($v) !== $f['len']) {
                $errors[$k] = $f['label'] . ' terdiri dari ' . $f['len'] . ' angka.';
            }
            if ($type === 'percent' && ($v < 0 || $v > 100)) {
                $errors[$k] = 'Isi antara 0 dan 100.';
            }
            if (!empty($f['unique']) && $v !== '') {
                $dup = Database::value("SELECT id FROM {$this->res['table']} WHERE $k = ? AND id <> ?", [$v, $id ?? 0]);
                if ($dup) {
                    $errors[$k] = 'Sudah ada data dengan ' . strtolower($f['label']) . ' ini.';
                }
            }
            $data[$k] = $v;
        }
        if ($errors) {
            with_old($_POST, $errors);
            redirect($id ? "/{$this->slug}/{$id}/edit" : "/{$this->slug}/baru");
        }
        return $data;
    }
}

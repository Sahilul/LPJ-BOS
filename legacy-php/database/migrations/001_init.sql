CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    nama TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE sekolah (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    nama TEXT NOT NULL DEFAULT '',
    npsn TEXT NOT NULL DEFAULT '',
    jenjang TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'Negeri',
    alamat TEXT NOT NULL DEFAULT '',
    desa TEXT NOT NULL DEFAULT '',
    kecamatan TEXT NOT NULL DEFAULT '',
    kabupaten TEXT NOT NULL DEFAULT '',
    provinsi TEXT NOT NULL DEFAULT '',
    kode_pos TEXT NOT NULL DEFAULT '',
    telepon TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    npwp TEXT NOT NULL DEFAULT '',
    bank_nama TEXT NOT NULL DEFAULT '',
    bank_rekening TEXT NOT NULL DEFAULT '',
    bank_atas_nama TEXT NOT NULL DEFAULT '',
    tahun_anggaran INTEGER NOT NULL DEFAULT (CAST(strftime('%Y', 'now') AS INTEGER)),
    logo_path TEXT,
    kop_path TEXT,
    stempel_path TEXT,
    updated_at TEXT
);
INSERT INTO sekolah (id) VALUES (1);

CREATE TABLE pejabat (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    peran TEXT NOT NULL CHECK (peran IN ('kepala', 'bendahara')),
    nama TEXT NOT NULL,
    nip TEXT NOT NULL DEFAULT '',
    pangkat TEXT NOT NULL DEFAULT '',
    ttd_path TEXT,
    mulai TEXT NOT NULL,
    selesai TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE komponen (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kode TEXT NOT NULL UNIQUE,
    nama TEXT NOT NULL,
    urutan INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE kode_rekening (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kode TEXT NOT NULL UNIQUE,
    nama TEXT NOT NULL,
    jenis TEXT NOT NULL DEFAULT 'barang' CHECK (jenis IN ('barang', 'jasa', 'honor', 'modal', 'lainnya'))
);

CREATE TABLE tarif_pajak (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kode TEXT NOT NULL UNIQUE,
    nama TEXT NOT NULL,
    tarif REAL NOT NULL DEFAULT 0,
    keterangan TEXT NOT NULL DEFAULT ''
);

CREATE TABLE toko (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL,
    alamat TEXT NOT NULL DEFAULT '',
    npwp TEXT NOT NULL DEFAULT '',
    pkp INTEGER NOT NULL DEFAULT 0,
    telepon TEXT NOT NULL DEFAULT '',
    aktif INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE penerima (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nama TEXT NOT NULL,
    nip TEXT NOT NULL DEFAULT '',
    nik TEXT NOT NULL DEFAULT '',
    npwp TEXT NOT NULL DEFAULT '',
    kategori TEXT NOT NULL DEFAULT 'GTT' CHECK (kategori IN ('PNS', 'PPPK', 'GTT', 'PTT', 'Luar')),
    golongan TEXT NOT NULL DEFAULT '',
    jabatan TEXT NOT NULL DEFAULT '',
    no_hp TEXT NOT NULL DEFAULT '',
    alamat TEXT NOT NULL DEFAULT '',
    bank TEXT NOT NULL DEFAULT '',
    no_rekening TEXT NOT NULL DEFAULT '',
    atas_nama TEXT NOT NULL DEFAULT '',
    tarif_default REAL NOT NULL DEFAULT 0,
    aktif INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);
CREATE INDEX idx_penerima_nama ON penerima(nama);

CREATE TABLE rencana_belanja (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tahun INTEGER NOT NULL,
    komponen_kode TEXT NOT NULL REFERENCES komponen(kode) ON UPDATE CASCADE,
    kegiatan TEXT NOT NULL DEFAULT '',
    rekening_kode TEXT NOT NULL DEFAULT '',
    uraian TEXT NOT NULL,
    jenis TEXT NOT NULL DEFAULT 'barang' CHECK (jenis IN ('barang', 'honor', 'transport')),
    volume REAL NOT NULL DEFAULT 1,
    satuan TEXT NOT NULL DEFAULT '',
    harga_satuan REAL NOT NULL DEFAULT 0,
    pagu REAL NOT NULL DEFAULT 0,
    jadwal_bulan TEXT NOT NULL DEFAULT '',
    sumber TEXT NOT NULL DEFAULT 'manual'
);
CREATE INDEX idx_rencana_tahun ON rencana_belanja(tahun);

CREATE TABLE penerimaan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tgl TEXT NOT NULL,
    no_bukti TEXT NOT NULL DEFAULT '',
    uraian TEXT NOT NULL,
    tahap TEXT NOT NULL DEFAULT '',
    jumlah REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE belanja (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    no_bukti TEXT NOT NULL,
    tgl TEXT NOT NULL,
    rencana_id INTEGER REFERENCES rencana_belanja(id) ON DELETE RESTRICT,
    jenis TEXT NOT NULL DEFAULT 'barang' CHECK (jenis IN ('barang', 'honor', 'transport')),
    toko_id INTEGER REFERENCES toko(id) ON DELETE SET NULL,
    uraian TEXT NOT NULL,
    metode TEXT NOT NULL DEFAULT 'tunai' CHECK (metode IN ('tunai', 'bank')),
    total REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);
CREATE INDEX idx_belanja_tgl ON belanja(tgl);
CREATE INDEX idx_belanja_rencana ON belanja(rencana_id);

CREATE TABLE item_belanja (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    belanja_id INTEGER NOT NULL REFERENCES belanja(id) ON DELETE CASCADE,
    nama TEXT NOT NULL,
    volume REAL NOT NULL DEFAULT 1,
    satuan TEXT NOT NULL DEFAULT '',
    harga REAL NOT NULL DEFAULT 0,
    subtotal REAL NOT NULL DEFAULT 0
);

CREATE TABLE penerima_belanja (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    belanja_id INTEGER NOT NULL REFERENCES belanja(id) ON DELETE CASCADE,
    penerima_id INTEGER REFERENCES penerima(id) ON DELETE SET NULL,
    nama TEXT NOT NULL,
    nip TEXT NOT NULL DEFAULT '',
    npwp TEXT NOT NULL DEFAULT '',
    kategori TEXT NOT NULL DEFAULT '',
    golongan TEXT NOT NULL DEFAULT '',
    jabatan TEXT NOT NULL DEFAULT '',
    bank TEXT NOT NULL DEFAULT '',
    no_rekening TEXT NOT NULL DEFAULT '',
    volume REAL NOT NULL DEFAULT 1,
    satuan TEXT NOT NULL DEFAULT '',
    tarif REAL NOT NULL DEFAULT 0,
    bruto REAL NOT NULL DEFAULT 0,
    pph21 REAL NOT NULL DEFAULT 0,
    netto REAL NOT NULL DEFAULT 0
);

CREATE TABLE pajak (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    belanja_id INTEGER NOT NULL REFERENCES belanja(id) ON DELETE CASCADE,
    jenis TEXT NOT NULL,
    dpp REAL NOT NULL DEFAULT 0,
    tarif REAL NOT NULL DEFAULT 0,
    nilai REAL NOT NULL DEFAULT 0,
    tgl_pungut TEXT,
    tgl_setor TEXT,
    ntpn TEXT NOT NULL DEFAULT '',
    kode_billing TEXT NOT NULL DEFAULT ''
);

CREATE TABLE lampiran (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    belanja_id INTEGER NOT NULL REFERENCES belanja(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    nama_asli TEXT NOT NULL DEFAULT ''
);

CREATE TABLE mutasi_kas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tgl TEXT NOT NULL,
    no_bukti TEXT NOT NULL DEFAULT '',
    jenis TEXT NOT NULL CHECK (jenis IN ('tarik', 'setor', 'admin_bank', 'bunga')),
    uraian TEXT NOT NULL DEFAULT '',
    jumlah REAL NOT NULL DEFAULT 0
);

CREATE TABLE tutup_buku (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tahun INTEGER NOT NULL,
    bulan INTEGER NOT NULL,
    saldo_tunai REAL NOT NULL DEFAULT 0,
    saldo_bank REAL NOT NULL DEFAULT 0,
    ditutup_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime')),
    UNIQUE (tahun, bulan)
);

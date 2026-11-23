CREATE TABLE tahun_anggaran (
    tahun INTEGER PRIMARY KEY CHECK (tahun BETWEEN 2000 AND 2100),
    is_default INTEGER NOT NULL DEFAULT 0,
    keterangan TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

INSERT INTO tahun_anggaran (tahun, is_default)
SELECT tahun_anggaran, 1 FROM sekolah WHERE id = 1;

INSERT OR IGNORE INTO tahun_anggaran (tahun)
SELECT DISTINCT tahun FROM rencana_belanja
UNION SELECT DISTINCT CAST(strftime('%Y', tgl) AS INTEGER) FROM penerimaan
UNION SELECT DISTINCT CAST(strftime('%Y', tgl) AS INTEGER) FROM belanja
UNION SELECT DISTINCT CAST(strftime('%Y', tgl) AS INTEGER) FROM mutasi_kas;

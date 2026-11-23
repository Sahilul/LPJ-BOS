CREATE TABLE rencana_pendapatan (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tahun INTEGER NOT NULL CHECK (tahun BETWEEN 2000 AND 2100),
    tahap TEXT NOT NULL CHECK (tahap IN ('Tahap 1', 'Tahap 2', 'Sisa tahun lalu', 'Lainnya')),
    uraian TEXT NOT NULL,
    target_bulan INTEGER CHECK (target_bulan IS NULL OR target_bulan BETWEEN 1 AND 12),
    target_jumlah REAL NOT NULL CHECK (target_jumlah > 0),
    created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

ALTER TABLE penerimaan
ADD COLUMN rencana_pendapatan_id INTEGER REFERENCES rencana_pendapatan(id) ON DELETE SET NULL;

CREATE INDEX idx_rencana_pendapatan_tahun ON rencana_pendapatan(tahun);
CREATE INDEX idx_penerimaan_rencana_pendapatan ON penerimaan(rencana_pendapatan_id);

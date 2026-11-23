ALTER TABLE rencana_belanja
ADD COLUMN frekuensi_bulan INTEGER NOT NULL DEFAULT 1
CHECK (frekuensi_bulan BETWEEN 1 AND 12);

ALTER TABLE rencana_belanja
ADD COLUMN toko_id INTEGER REFERENCES toko(id) ON DELETE SET NULL;

ALTER TABLE belanja
ADD COLUMN jumlah_periode INTEGER NOT NULL DEFAULT 1
CHECK (jumlah_periode BETWEEN 1 AND 12);

CREATE TABLE rencana_item (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rencana_id INTEGER NOT NULL REFERENCES rencana_belanja(id) ON DELETE CASCADE,
    nama TEXT NOT NULL,
    volume REAL NOT NULL DEFAULT 1 CHECK (volume > 0),
    satuan TEXT NOT NULL DEFAULT '',
    harga_satuan REAL NOT NULL DEFAULT 0 CHECK (harga_satuan > 0)
);

CREATE TABLE rencana_penerima (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rencana_id INTEGER NOT NULL REFERENCES rencana_belanja(id) ON DELETE CASCADE,
    penerima_id INTEGER REFERENCES penerima(id) ON DELETE SET NULL,
    volume REAL NOT NULL DEFAULT 1 CHECK (volume > 0),
    satuan TEXT NOT NULL DEFAULT '',
    tarif REAL NOT NULL DEFAULT 0 CHECK (tarif > 0),
    UNIQUE (rencana_id, penerima_id)
);

CREATE INDEX idx_rencana_item_parent ON rencana_item(rencana_id);
CREATE INDEX idx_rencana_penerima_parent ON rencana_penerima(rencana_id);

UPDATE rencana_belanja
SET frekuensi_bulan = CASE
    WHEN lower(trim(satuan)) IN ('bulan', 'bln', 'ob', 'orang/bulan')
        THEN MAX(1, MIN(12, CAST(volume AS INTEGER)))
    ELSE 1
END;

INSERT INTO rencana_item (rencana_id, nama, volume, satuan, harga_satuan)
SELECT id, uraian,
       CASE WHEN lower(trim(satuan)) IN ('bulan', 'bln', 'ob', 'orang/bulan') THEN 1 ELSE volume END,
       CASE WHEN lower(trim(satuan)) IN ('bulan', 'bln', 'ob', 'orang/bulan') THEN 'paket' ELSE satuan END,
       harga_satuan
FROM rencana_belanja
WHERE jenis = 'barang' AND harga_satuan > 0;

-- Nomor bukti dipisah per jenis: BKM (penerimaan), BKK (belanja), MUT (mutasi kas).
UPDATE penerimaan
SET no_bukti = (
    SELECT 'BKM-' || printf('%03d', n) || '/' || strftime('%m/%Y', penerimaan.tgl)
    FROM (SELECT id, ROW_NUMBER() OVER (PARTITION BY strftime('%Y', tgl) ORDER BY tgl, id) AS n FROM penerimaan) x
    WHERE x.id = penerimaan.id
);

UPDATE belanja
SET no_bukti = (
    SELECT 'BKK-' || printf('%03d', n) || '/' || strftime('%m/%Y', belanja.tgl)
    FROM (SELECT id, ROW_NUMBER() OVER (PARTITION BY strftime('%Y', tgl) ORDER BY tgl, id) AS n FROM belanja) x
    WHERE x.id = belanja.id
);

UPDATE mutasi_kas
SET no_bukti = (
    SELECT 'MUT-' || printf('%03d', n) || '/' || strftime('%m/%Y', mutasi_kas.tgl)
    FROM (SELECT id, ROW_NUMBER() OVER (PARTITION BY strftime('%Y', tgl) ORDER BY tgl, id) AS n FROM mutasi_kas) x
    WHERE x.id = mutasi_kas.id
);

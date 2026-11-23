CREATE TEMP TABLE bku_nomor_otomatis AS
SELECT
    src,
    id,
    'BKU-' || printf('%03d', ROW_NUMBER() OVER (
        PARTITION BY strftime('%Y', tgl)
        ORDER BY tgl, CASE src WHEN 'trm' THEN 0 WHEN 'mut' THEN 1 ELSE 2 END, id
    )) || '/' || strftime('%m/%Y', tgl) AS no_bukti
FROM (
    SELECT 'trm' AS src, id, tgl FROM penerimaan
    UNION ALL SELECT 'bel', id, tgl FROM belanja
    UNION ALL SELECT 'mut', id, tgl FROM mutasi_kas
);

UPDATE penerimaan
SET no_bukti = (SELECT no_bukti FROM bku_nomor_otomatis WHERE src = 'trm' AND id = penerimaan.id)
WHERE id IN (SELECT id FROM bku_nomor_otomatis WHERE src = 'trm');

UPDATE belanja
SET no_bukti = (SELECT no_bukti FROM bku_nomor_otomatis WHERE src = 'bel' AND id = belanja.id)
WHERE id IN (SELECT id FROM bku_nomor_otomatis WHERE src = 'bel');

UPDATE mutasi_kas
SET no_bukti = (SELECT no_bukti FROM bku_nomor_otomatis WHERE src = 'mut' AND id = mutasi_kas.id)
WHERE id IN (SELECT id FROM bku_nomor_otomatis WHERE src = 'mut');

DROP TABLE bku_nomor_otomatis;

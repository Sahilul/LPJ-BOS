-- Kolom pendukung aturan pajak terbaru.
--
-- jenis_penerima : rezim pemotongan PPh 21 menurut PMK 168/2023 Pasal 3 ayat (1).
--                  Data lama dikosongkan lalu diturunkan dari kategori.
-- ptkp           : kategori tarif efektif bulanan (A/B/C) menurut PP 58/2023 Pasal 2 ayat (4).
-- via_siplah     : transaksi lewat Pihak Lain/SIPLah; PPN dan PPh 22 dipungut platform
--                  (PMK 58/2022 Pasal 5 ayat (6) dan Pasal 7 ayat (5)).
-- dana_bos       : penanda sumber dana untuk pengecualian PPh 22
--                  (PMK 59/2022 Pasal 12 ayat (2) huruf d).

ALTER TABLE penerima ADD COLUMN jenis_penerima TEXT NOT NULL DEFAULT '';
ALTER TABLE penerima ADD COLUMN ptkp TEXT NOT NULL DEFAULT 'A';

ALTER TABLE penerima_belanja ADD COLUMN jenis_penerima TEXT NOT NULL DEFAULT '';
ALTER TABLE penerima_belanja ADD COLUMN ptkp TEXT NOT NULL DEFAULT 'A';
ALTER TABLE penerima_belanja ADD COLUMN hari INTEGER NOT NULL DEFAULT 0;

ALTER TABLE belanja ADD COLUMN via_siplah INTEGER NOT NULL DEFAULT 0;
-- Bawaan 0 agar transaksi lama tidak berubah perlakuan pajaknya saat diedit;
-- transaksi baru mengirim dana_bos=1 dari form (aplikasi ini khusus dana BOS).
ALTER TABLE belanja ADD COLUMN dana_bos INTEGER NOT NULL DEFAULT 0;

-- Turunkan jenis penerima data lama dari kategori yang sudah ada.
-- Pihak luar diperlakukan sebagai bukan pegawai (PMK 168/2023 Pasal 12 ayat (3)).
UPDATE penerima SET jenis_penerima = CASE
    WHEN kategori = 'Luar' THEN 'bukan_pegawai'
    WHEN kategori = 'PNS'  THEN 'tetap'
    ELSE 'tidak_tetap'
END
WHERE jenis_penerima = '';

-- Tarif pajak disesuaikan dengan aturan yang berlaku.
-- PPh 22 tidak berlaku untuk dana BOS, jadi keterangannya diperbaiki.
UPDATE tarif_pajak SET
    keterangan = 'Dipungut bila rekanan PKP dan nilai di luar PPN di atas Rp2.000.000 (PMK 59/2022 Pasal 18)'
WHERE kode = 'PPN';

UPDATE tarif_pajak SET
    keterangan = 'Pembelian barang di atas Rp2.000.000 di luar PPN. Tidak dipungut untuk dana BOS (PMK 59/2022 Pasal 12 ayat (2) huruf d)'
WHERE kode = 'PPH22';

UPDATE tarif_pajak SET
    keterangan = 'Sewa harta dan imbalan jasa. Menjadi 4% bila rekanan tanpa NPWP (PMK 59/2022 Lampiran VI)'
WHERE kode = 'PPH23';

-- PPh 21 non-ASN tidak lagi memakai tarif flat; ganti dengan acuan rezim PMK 168/2023.
DELETE FROM tarif_pajak WHERE kode IN ('PPH21_NPWP', 'PPH21_NON_NPWP');

INSERT OR IGNORE INTO tarif_pajak (kode, nama, tarif, keterangan) VALUES
('PPH21_BUKAN_PEGAWAI', 'PPh 21 bukan pegawai', 2.5, 'Efektif 2,5%: 50% bruto x tarif Pasal 17 lapisan pertama (PMK 168/2023 Pasal 12 ayat (3))'),
('PPH21_TIDAK_TETAP_HARIAN', 'PPh 21 pegawai tidak tetap harian', 0.5, 'Tarif efektif harian: 0% s.d. Rp450.000, 0,5% s.d. Rp2.500.000 sehari (PP 58/2023 Lampiran D)'),
('PPH21_PESERTA', 'PPh 21 peserta kegiatan', 5, 'Tarif Pasal 17 atas jumlah bruto utuh (PMK 168/2023 Pasal 12 ayat (6))');

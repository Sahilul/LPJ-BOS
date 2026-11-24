-- PPh 21 pegawai tidak tetap yang dibayar bulanan memakai tarif efektif (TER)
-- bulanan kategori A/B/C (PP 58/2023 Lampiran A-C jo. PMK 168/2023 Pasal 12 ayat (2)).
-- Penghasilan dari pemberi kerja yang sama dalam satu bulan digabung.
-- Penghitungan ulang data honor lama dijalankan aplikasi (HitungUlangPPh21Sekali).

INSERT OR IGNORE INTO tarif_pajak (kode, nama, tarif, keterangan) VALUES
('PPH21_TER_BULANAN', 'PPh 21 pegawai tidak tetap bulanan', 0, 'TER bulanan: 0% s.d. Rp5.400.000 (A), Rp6.200.000 (B), Rp6.600.000 (C) sebulan; honor sebulan digabung (PP 58/2023 Lampiran A-C)');

UPDATE tarif_pajak SET
    keterangan = 'Hanya untuk upah harian (kolom hari diisi): 0% s.d. Rp450.000, 0,5% s.d. Rp2.500.000 sehari (PP 58/2023 Lampiran D)'
WHERE kode = 'PPH21_TIDAK_TETAP_HARIAN';

UPDATE tarif_pajak SET
    keterangan = 'Dipungut bendahara sekolah negeri bila rekanan PKP dan nilai di luar PPN di atas Rp2.000.000 (PMK 59/2022 Pasal 18). Sekolah swasta bukan pemungut'
WHERE kode = 'PPN';

UPDATE tarif_pajak SET
    keterangan = 'Pembelian barang di atas Rp2.000.000 di luar PPN oleh sekolah negeri. Tidak dipungut untuk dana BOS dan oleh sekolah swasta (PMK 59/2022 Pasal 2 dan Pasal 12 ayat (2) huruf d)'
WHERE kode = 'PPH22';

UPDATE tarif_pajak SET
    keterangan = 'Sewa harta dan imbalan jasa (termasuk katering), juga rekanan orang pribadi, tanpa batas minimal. Menjadi 4% bila tanpa NPWP/NIK (PMK 59/2022 Lampiran VI)'
WHERE kode = 'PPH23';

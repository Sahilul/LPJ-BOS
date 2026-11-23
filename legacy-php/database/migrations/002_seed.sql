INSERT INTO komponen (kode, nama, urutan) VALUES
('01', 'Penerimaan Peserta Didik Baru', 1),
('02', 'Pengembangan Perpustakaan', 2),
('03', 'Pelaksanaan Kegiatan Pembelajaran dan Ekstrakurikuler', 3),
('04', 'Pelaksanaan Kegiatan Asesmen dan Evaluasi Pembelajaran', 4),
('05', 'Pelaksanaan Administrasi Kegiatan Sekolah', 5),
('06', 'Pengembangan Profesi Guru dan Tenaga Kependidikan', 6),
('07', 'Pembiayaan Langganan Daya dan Jasa', 7),
('08', 'Pemeliharaan Sarana dan Prasarana Sekolah', 8),
('09', 'Penyediaan Alat Multimedia Pembelajaran', 9),
('10', 'Penyelenggaraan Bursa Kerja Khusus, Praktik Kerja Industri, dan Pemagangan', 10),
('11', 'Penyelenggaraan Kegiatan Uji Kompetensi Keahlian, Sertifikasi Kompetensi Keahlian, dan Uji Kompetensi Kemampuan Bahasa Inggris Berstandar Internasional', 11),
('12', 'Pembayaran Honor', 12);

INSERT INTO kode_rekening (kode, nama, jenis) VALUES
('5.1.02.01.01.0024', 'Belanja Alat/Bahan untuk Kegiatan Kantor-Alat Tulis Kantor', 'barang'),
('5.1.02.01.01.0025', 'Belanja Alat/Bahan untuk Kegiatan Kantor-Kertas dan Cover', 'barang'),
('5.1.02.01.01.0026', 'Belanja Alat/Bahan untuk Kegiatan Kantor-Bahan Cetak', 'barang'),
('5.1.02.01.01.0029', 'Belanja Alat/Bahan untuk Kegiatan Kantor-Bahan Komputer', 'barang'),
('5.1.02.01.01.0052', 'Belanja Makanan dan Minuman Rapat', 'barang'),
('5.1.02.02.01.0013', 'Belanja Jasa Tenaga Pendidikan', 'honor'),
('5.1.02.02.01.0014', 'Belanja Jasa Tenaga Kependidikan', 'honor'),
('5.1.02.02.01.0026', 'Belanja Jasa Tenaga Administrasi', 'honor'),
('5.1.02.02.01.0061', 'Belanja Langganan Listrik', 'jasa'),
('5.1.02.02.01.0063', 'Belanja Langganan Internet', 'jasa'),
('5.1.02.04.01.0003', 'Belanja Perjalanan Dinas Dalam Kota', 'jasa'),
('5.1.02.03.02.0405', 'Belanja Pemeliharaan Bangunan Gedung', 'jasa'),
('5.2.02.10.01.0002', 'Belanja Modal Personal Computer', 'modal');

INSERT INTO tarif_pajak (kode, nama, tarif, keterangan) VALUES
('PPN', 'Pajak Pertambahan Nilai', 11, 'Dipungut bila toko PKP dan nilai transaksi di atas Rp2.000.000'),
('PPH22', 'PPh Pasal 22', 1.5, 'Pembelian barang di atas Rp2.000.000'),
('PPH23', 'PPh Pasal 23', 2, 'Jasa tertentu (sewa, servis, katering)'),
('PPH21_GOL_I', 'PPh 21 PNS Golongan I', 0, 'Honor PNS golongan I'),
('PPH21_GOL_II', 'PPh 21 PNS Golongan II', 0, 'Honor PNS golongan II'),
('PPH21_GOL_III', 'PPh 21 PNS Golongan III', 5, 'Honor PNS golongan III'),
('PPH21_GOL_IV', 'PPh 21 PNS Golongan IV', 15, 'Honor PNS golongan IV'),
('PPH21_NON_NPWP', 'PPh 21 Non-PNS tanpa NPWP', 6, 'Tarif dapat disesuaikan dengan aturan daerah'),
('PPH21_NPWP', 'PPh 21 Non-PNS ber-NPWP', 5, 'Tarif dapat disesuaikan dengan aturan daerah');

package app

import "testing"

func TestBkuHitungDokumen(t *testing.T) {
	rows := []Row{
		{"src": "bel", "jenis": "barang"}, {"src": "bel", "jenis": "honor"},
		{"src": "bel", "jenis": "transport"}, {"src": "trm"}, {"src": "mut", "jenis": "tarik"},
	}
	if n := bkuHitungDokumen(rows, false); n != 3 {
		t.Fatalf("kuitansi: dapat %d", n)
	}
	if n := bkuHitungDokumen(rows, true); n != 2 {
		t.Fatalf("bukti terima: dapat %d", n)
	}
}

func TestBkuDokumenMassalPeriode(t *testing.T) {
	setupBelanja(t, true, "")
	pid := setupHonor(t, "Luar", "", "bukan_pegawai")
	simpan := func(f Form, tgl string) {
		f["tgl"] = tgl
		data, anak, err := BelanjaValidate(f, 0, 2026)
		if err != nil {
			t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
		}
		BelanjaCreate(data, anak)
	}
	simpan(formHonor(pid, 500000, "0"), "2026-03-05")
	simpan(formHonor(pid, 600000, "0"), "2026-08-05")
	simpan(formHonor(pid, 700000, "0"), "2026-11-05")

	if n := len(bkuDokumenMassal(2026, "2026-07-01", "2026-12-31", true)); n != 2 {
		t.Fatalf("bukti terima Tahap 2: dapat %d, mau 2", n)
	}
	data := bkuDokumenMassal(2026, "2026-01-01", "2026-12-31", false)
	if len(data) != 3 {
		t.Fatalf("kuitansi setahun: dapat %d, mau 3", len(data))
	}
	k := NewPdf(false, PdfIdentity("2026-03-05"))
	for i, d := range data {
		kuitansiTulis(k, d, kuitansiPenerima(d), i == 0)
	}
	if n := k.Pdf.PageNo(); n != 3 {
		t.Fatalf("halaman PDF gabungan: dapat %d, mau 3", n)
	}
}

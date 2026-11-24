package app

import "testing"

func TestPPN11(t *testing.T) {
	// Contoh PMK 59/2022 Lampiran VI.B: pembayaran Rp2.775.000 memuat PPN Rp275.000.
	if got := PPN11(2775000); got != 275000 {
		t.Errorf("PPN11(2775000) = %v, want 275000", got)
	}
	if got := PPN11(0); got != 0 {
		t.Errorf("PPN11(0) = %v, want 0", got)
	}
}

func TestBatasPPNTidakTermasukPPN(t *testing.T) {
	// Contoh PMK 59/2022: pembayaran Rp2.109.000 punya DPP Rp1.900.000,
	// jadi tidak melewati batas Rp2.000.000 dan PPN tidak dipungut.
	if DPPDariBruto(2109000) != 1900000 {
		t.Errorf("DPPDariBruto(2109000) = %v, want 1900000", DPPDariBruto(2109000))
	}
	if PPNPerluDipungut(2109000) {
		t.Error("PPN untuk pembayaran Rp2.109.000 (DPP Rp1,9 juta) seharusnya tidak dipungut")
	}
	// Rp2.220.000 memuat PPN Rp220.000 sehingga DPP tepat Rp2.000.000: tidak dipungut.
	if PPNPerluDipungut(2220000) {
		t.Error("DPP Rp2.000.000 seharusnya belum dipungut PPN (batas 'paling banyak')")
	}
	if !PPNPerluDipungut(2331000) {
		t.Error("DPP Rp2.100.000 seharusnya dipungut PPN")
	}
}

func TestPPh22BOSDikecualikan(t *testing.T) {
	// Dana BOS tidak dipungut PPh 22 (PMK 59/2022 Pasal 12 ayat (2) huruf d).
	if PPh22PerluDipungut(10000000, true, false) {
		t.Error("PPh 22 tidak boleh dipungut untuk pembelian barang dana BOS")
	}
	// Transaksi lewat SIPLah dipungut platform.
	if PPh22PerluDipungut(10000000, false, true) {
		t.Error("PPh 22 tidak boleh dipungut bendahara untuk transaksi SIPLah")
	}
	// Bukan BOS dan di atas batas: dipungut.
	if !PPh22PerluDipungut(3000000, false, false) {
		t.Error("PPh 22 seharusnya dipungut untuk bukan dana BOS di atas Rp2 juta")
	}
	// Bukan BOS tetapi di bawah batas: tidak dipungut.
	if PPh22PerluDipungut(2000000, false, false) {
		t.Error("PPh 22 tidak dipungut bila pembayaran paling banyak Rp2 juta")
	}
}

func TestTarifSurchargeNonNPWP(t *testing.T) {
	if got := TarifPPh22(true); got != 1.5 {
		t.Errorf("TarifPPh22 ber-NPWP = %v, want 1.5", got)
	}
	if got := TarifPPh22(false); got != 3 {
		t.Errorf("TarifPPh22 tanpa NPWP = %v, want 3", got)
	}
	if got := TarifPPh23(true); got != 2 {
		t.Errorf("TarifPPh23 ber-NPWP = %v, want 2", got)
	}
	if got := TarifPPh23(false); got != 4 {
		t.Errorf("TarifPPh23 tanpa NPWP = %v, want 4", got)
	}
}

func TestPPh21PNSFinal(t *testing.T) {
	cases := map[string]float64{"I/a": 0, "II/c": 0, "III/a": 5, "III/d": 5, "IV/b": 15}
	for gol, want := range cases {
		p := Row{"kategori": "PNS", "golongan": gol}
		if !PPh21PNSFinal(p) {
			t.Errorf("PNS golongan %s seharusnya memakai tarif final", gol)
		}
		// Honor Rp1.000.000.
		if got := PPh21UntukPenerima(p, 1000000, 0); got != want*10000 {
			t.Errorf("PPh 21 PNS golongan %s = %v, want %v", gol, got, want*10000)
		}
	}
}

func TestPPPKTidakPakaiTarifFinalPNS(t *testing.T) {
	p := Row{"kategori": "PPPK", "golongan": "IX", "jenis_penerima": "tetap"}
	if PPh21PNSFinal(p) {
		t.Error("PPPK tidak termasuk penerima tarif final PP 80/2010")
	}
	// Honor Rp10.000.000 untuk pegawai tetap: tarif Pasal 17 = 5% x 10 juta.
	if got := PPh21UntukPenerima(p, 10000000, 0); got != 500000 {
		t.Errorf("PPh 21 PPPK = %v, want 500000", got)
	}
}

func TestPPh21BukanPegawai(t *testing.T) {
	// Narasumber: 50% x bruto x tarif Pasal 17, efektif 2,5% pada lapisan pertama
	// (PMK 168/2023 Pasal 12 ayat (3)).
	p := Row{"kategori": "Luar", "jenis_penerima": "bukan_pegawai"}
	if got := PPh21UntukPenerima(p, 3000000, 0); got != 75000 {
		t.Errorf("PPh 21 narasumber Rp3.000.000 = %v, want 75000", got)
	}
	// Tarif tampilan harus 2,5%, bukan 5%.
	if got := TarifPPh21(p); got != 2.5 {
		t.Errorf("TarifPPh21 bukan pegawai = %v, want 2.5", got)
	}
}

func TestPPh21TidakTetapHarian(t *testing.T) {
	p := Row{"kategori": "GTT", "jenis_penerima": "tidak_tetap"}
	// Upah Rp450.000 sehari: tarif efektif harian 0% (PP 58/2023 Lampiran D).
	if got := PPh21UntukPenerima(p, 4500000, 10); got != 0 {
		t.Errorf("PPh 21 upah Rp450.000/hari = %v, want 0", got)
	}
	// Upah Rp500.000 sehari selama 10 hari: 0,5% x Rp5.000.000 = Rp25.000.
	if got := PPh21UntukPenerima(p, 5000000, 10); got != 25000 {
		t.Errorf("PPh 21 upah Rp500.000/hari = %v, want 25000", got)
	}
	// Upah sehari di atas Rp2.500.000: 50% bruto x Pasal 17.
	// 50% x 3.000.000 = 1.500.000; 5% = Rp75.000.
	if got := PPh21UntukPenerima(p, 3000000, 1); got != 75000 {
		t.Errorf("PPh 21 upah Rp3.000.000/hari = %v, want 75000", got)
	}
}

func TestPPh21PesertaKegiatan(t *testing.T) {
	// Peserta kegiatan memakai bruto utuh, bukan 50% (PMK 168/2023 Pasal 12 ayat (6)).
	p := Row{"kategori": "Luar", "jenis_penerima": "peserta"}
	if got := PPh21UntukPenerima(p, 2000000, 0); got != 100000 {
		t.Errorf("PPh 21 peserta kegiatan Rp2.000.000 = %v, want 100000", got)
	}
}

func TestPasal17Progresif(t *testing.T) {
	cases := map[float64]float64{
		0:          0,
		60000000:   3000000,
		100000000:  9000000,  // 5% x 60 jt + 15% x 40 jt
		300000000:  44000000, // + 15% x 190 jt + 25% x 50 jt
		6000000000: 1794000000,
	}
	for pkp, want := range cases {
		if got := pasal17(pkp); got != want {
			t.Errorf("pasal17(%v) = %v, want %v", pkp, got, want)
		}
	}
}

func TestBeaMeterai(t *testing.T) {
	if PerluBeaMeterai(5000000) {
		t.Error("Rp5.000.000 tidak melewati batas bea meterai (UU 10/2020 Pasal 3 ayat (2) huruf g)")
	}
	if !PerluBeaMeterai(5000001) {
		t.Error("di atas Rp5.000.000 seharusnya perlu bea meterai")
	}
}

func TestJenisPenerimaDariKategori(t *testing.T) {
	// Data lama tanpa kolom jenis_penerima diturunkan dari kategori.
	if got := jenisPenerimaDari(Row{"kategori": "Luar"}); got != "bukan_pegawai" {
		t.Errorf("kategori Luar = %q, want bukan_pegawai", got)
	}
	if got := jenisPenerimaDari(Row{"kategori": "PNS"}); got != "tetap" {
		t.Errorf("kategori PNS = %q, want tetap", got)
	}
	if got := jenisPenerimaDari(Row{"kategori": "GTT"}); got != "tidak_tetap" {
		t.Errorf("kategori GTT = %q, want tidak_tetap", got)
	}
	// Nilai tersimpan menang atas turunan kategori.
	if got := jenisPenerimaDari(Row{"kategori": "GTT", "jenis_penerima": "peserta"}); got != "peserta" {
		t.Errorf("jenis_penerima tersimpan = %q, want peserta", got)
	}
}

func TestTarifTERBulanan(t *testing.T) {
	cases := []struct {
		kat   string
		bruto float64
		want  float64
	}{
		{"A", 544000, 0}, {"A", 5400000, 0}, {"A", 5500000, 0.25},
		{"B", 6200000, 0}, {"C", 6600000, 0}, {"C", 6700000, 0.25},
		{"A", 2e9, 34}, {"", 5500000, 0.25},
	}
	for _, c := range cases {
		if got := TarifTERBulanan(c.kat, c.bruto); got != c.want {
			t.Errorf("TER %s %v = %v, want %v", c.kat, c.bruto, got, c.want)
		}
	}
}

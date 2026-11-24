package app

import (
	"fmt"
	"math"
	"strings"
)

// Aturan pajak yang dipakai aplikasi. Rujukan utama:
//   - PMK 59/PMK.03/2022 jo. PMK 81/2024 untuk pemungutan oleh Instansi Pemerintah
//   - PMK 168/2023 jo. PP 58/2023 untuk PPh 21
//   - PP 80/2010 untuk honorarium PNS (masih berlaku, hanya Pasal 2 ayat (3) yang dicabut)
//   - PMK 58/2022 untuk transaksi melalui Pihak Lain (SIPLah)
//   - PMK 131/2024 jo. PMK 11/2025 untuk tarif PPN
//   - UU 10/2020 untuk bea meterai

// BatasPajakPPN: PPN tidak dipungut bila pembayaran paling banyak Rp2.000.000
// tidak termasuk PPN dan bukan pembayaran yang dipecah (PMK 59/2022 Pasal 18 ayat (1) huruf a).
const BatasPajakPPN = 2000000

// BatasPPh22: PPh 22 tidak dipungut bila pembayaran paling banyak Rp2.000.000
// tidak termasuk PPN (PMK 59/2022 Pasal 12 ayat (2) huruf a).
const BatasPPh22 = 2000000

// BatasBeaMeterai: dokumen yang menyatakan penerimaan uang dengan nilai nominal
// lebih dari Rp5.000.000 dikenai bea meterai (UU 10/2020 Pasal 3 ayat (2) huruf g).
const BatasBeaMeterai = 5000000

// TarifBeaMeterai: tarif tetap bea meterai (UU 10/2020 Pasal 5).
const TarifBeaMeterai = 10000

// TarifPPNNilaiLain: PPN dihitung 12% x DPP nilai lain 11/12, sehingga tarif
// efektifnya tetap 11% untuk barang/jasa selain yang tergolong mewah
// (PMK 131/2024 Pasal 3 ayat (2)-(3)).
const TarifPPNNilaiLain = 11

// JenisPenerima menentukan rezim pemotongan PPh 21 (PMK 168/2023 Pasal 3 ayat (1)).
var JenisPenerima = []Opt{
	{"tetap", "Pegawai tetap / PPPK"},
	{"tidak_tetap", "Pegawai tidak tetap (GTT/PTT bulanan atau harian)"},
	{"bukan_pegawai", "Bukan pegawai (narasumber, pihak luar)"},
	{"peserta", "Peserta kegiatan (peserta rapat/diklat)"},
}

// PtkpKategori: kategori tarif efektif bulanan berdasarkan status PTKP (PP 58/2023 Pasal 2 ayat (4)).
var PtkpKategori = []Opt{
	{"A", "A — TK/0, TK/1, K/0"},
	{"B", "B — TK/2, TK/3, K/1, K/2"},
	{"C", "C — K/3"},
}

// TarifPasal17: tarif progresif Pasal 17 ayat (1) huruf a UU PPh.
// Batas bawah, batas atas (0 = tak terbatas), dan tarif dalam persen.
var TarifPasal17 = []struct {
	Bawah, Atas float64
	Tarif       float64
}{
	{0, 60000000, 5},
	{60000000, 250000000, 15},
	{250000000, 500000000, 25},
	{500000000, 5000000000, 30},
	{5000000000, 0, 35},
}

// TerHarian: tarif efektif harian untuk pegawai tidak tetap yang dibayar
// harian/satuan/borongan (PP 58/2023 Lampiran huruf D).
var TerHarian = []struct {
	Batas float64
	Tarif float64
}{
	{450000, 0},
	{2500000, 0.5},
}

// terLapis: batas atas penghasilan bruto sebulan (0 = tak terbatas) dan tarif persen.
type terLapis struct {
	Batas float64
	Tarif float64
}

// TerBulanan: tarif efektif bulanan kategori A/B/C (PP 58/2023 Lampiran huruf A-C).
// Dipakai pegawai tidak tetap yang dibayar bulanan (PMK 168/2023 Pasal 12 ayat (2)).
var TerBulanan = map[string][]terLapis{
	"A": {
		{5400000, 0}, {5650000, 0.25}, {5950000, 0.5}, {6300000, 0.75}, {6750000, 1},
		{7500000, 1.25}, {8550000, 1.5}, {9650000, 1.75}, {10050000, 2}, {10350000, 2.25},
		{10700000, 2.5}, {11050000, 3}, {11600000, 3.5}, {12500000, 4}, {13750000, 5},
		{15100000, 6}, {16950000, 7}, {19750000, 8}, {24150000, 9}, {26450000, 10},
		{28000000, 11}, {30050000, 12}, {32400000, 13}, {35400000, 14}, {39100000, 15},
		{43850000, 16}, {47800000, 17}, {51400000, 18}, {56300000, 19}, {62200000, 20},
		{68600000, 21}, {77500000, 22}, {89000000, 23}, {103000000, 24}, {125000000, 25},
		{157000000, 26}, {206000000, 27}, {337000000, 28}, {454000000, 29}, {550000000, 30},
		{695000000, 31}, {910000000, 32}, {1400000000, 33}, {0, 34},
	},
	"B": {
		{6200000, 0}, {6500000, 0.25}, {6850000, 0.5}, {7300000, 0.75}, {9200000, 1},
		{10750000, 1.5}, {11250000, 2}, {11600000, 2.5}, {12600000, 3}, {13600000, 4},
		{14950000, 5}, {16400000, 6}, {18450000, 7}, {21850000, 8}, {26000000, 9},
		{27700000, 10}, {29350000, 11}, {31450000, 12}, {33950000, 13}, {37100000, 14},
		{41100000, 15}, {45800000, 16}, {49500000, 17}, {53800000, 18}, {58500000, 19},
		{64000000, 20}, {71000000, 21}, {80000000, 22}, {93000000, 23}, {109000000, 24},
		{129000000, 25}, {163000000, 26}, {211000000, 27}, {374000000, 28}, {459000000, 29},
		{555000000, 30}, {704000000, 31}, {957000000, 32}, {1405000000, 33}, {0, 34},
	},
	"C": {
		{6600000, 0}, {6950000, 0.25}, {7350000, 0.5}, {7800000, 0.75}, {8850000, 1},
		{9800000, 1.25}, {10950000, 1.5}, {11200000, 1.75}, {12050000, 2}, {12950000, 3},
		{14150000, 4}, {15550000, 5}, {17050000, 6}, {19500000, 7}, {22700000, 8},
		{26600000, 9}, {28100000, 10}, {30100000, 11}, {32600000, 12}, {35400000, 13},
		{38900000, 14}, {43000000, 15}, {47400000, 16}, {51200000, 17}, {55800000, 18},
		{60400000, 19}, {66700000, 20}, {74500000, 21}, {83200000, 22}, {95600000, 23},
		{110000000, 24}, {134000000, 25}, {169000000, 26}, {221000000, 27}, {390000000, 28},
		{463000000, 29}, {561000000, 30}, {709000000, 31}, {965000000, 32}, {1419000000, 33},
		{0, 34},
	},
}

// TarifTERBulanan: tarif efektif bulanan (persen) atas penghasilan bruto sebulan.
func TarifTERBulanan(kategori string, brutoSebulan float64) float64 {
	tabel, ok := TerBulanan[strings.ToUpper(strings.TrimSpace(kategori))]
	if !ok {
		tabel = TerBulanan["A"]
	}
	for _, l := range tabel {
		if l.Batas == 0 || brutoSebulan <= l.Batas {
			return l.Tarif
		}
	}
	return tabel[len(tabel)-1].Tarif
}

// PPh21TERBulanan menghitung PPh 21 pegawai tidak tetap bulanan untuk satu
// pembayaran. Seluruh penghasilan dari pemberi kerja yang sama dalam satu bulan
// digabung untuk menentukan tarif; pajak yang sudah dipotong pada pembayaran
// lain di bulan itu dikurangkan. brutoLain dan pphLain adalah jumlah dari
// pembayaran lain bulan yang sama.
func PPh21TERBulanan(kategori string, bruto, brutoLain, pphLain float64) float64 {
	if bruto <= 0 {
		return 0
	}
	sebulan := bruto + brutoLain
	terutang := math.Round(sebulan * TarifTERBulanan(kategori, sebulan) / 100)
	return math.Max(0, terutang-pphLain)
}

// PPNPPh22Dipungut: bendahara sekolah swasta bukan pemungut PPN dan PPh 22.
// Kewajiban pemungutan oleh bendahara hanya untuk Instansi Pemerintah
// (PMK 59/2022 Pasal 2), sehingga untuk sekolah swasta PPN dan PPh 22
// menjadi urusan rekanan PKP sendiri.
func PPNPPh22Dipungut(statusSekolah string) bool {
	return !strings.EqualFold(strings.TrimSpace(statusSekolah), "Swasta")
}

// PPN11: PPN yang dipungut dari pembayaran yang sudah termasuk PPN,
// yaitu 11/111 bagian dari jumlah pembayaran (PMK 59/2022 Lampiran VI.B.1).
func PPN11(pembayaran float64) float64 {
	if pembayaran <= 0 {
		return 0
	}
	return math.Round(pembayaran * TarifPPNNilaiLain / (100 + TarifPPNNilaiLain))
}

// DPPDariBruto: harga sebelum PPN dari pembayaran yang sudah termasuk PPN.
func DPPDariBruto(bruto float64) float64 {
	return bruto - PPN11(bruto)
}

// PPNPerluDipungut: apakah bendahara wajib memungut PPN. Pembayaran dibandingkan
// setelah dikeluarkan PPN-nya, sesuai "paling banyak Rp2.000.000 tidak termasuk PPN".
func PPNPerluDipungut(pembayaran float64) bool {
	return DPPDariBruto(pembayaran) > BatasPajakPPN
}

// PPh22PerluDipungut: PPh 22 tidak dipungut untuk pembelian barang dengan dana
// BOS/BOP (PMK 59/2022 Pasal 12 ayat (2) huruf d, dipertahankan PMK 81/2024
// Pasal 219 ayat (1) huruf e angka 1 huruf d), untuk pembayaran paling banyak
// Rp2.000.000, dan untuk transaksi lewat Pihak Lain (SIPLah).
func PPh22PerluDipungut(dpp float64, danaBOS, viaSiplah bool) bool {
	if danaBOS || viaSiplah {
		return false
	}
	return dpp > BatasPPh22
}

// TarifPPh22: 1,5% dari harga pembelian tidak termasuk PPN, menjadi 3% bila
// rekanan tidak dapat menunjukkan NPWP (PMK 59/2022 Lampiran V.A.1-2).
func TarifPPh22(punyaNPWP bool) float64 { return tarifDenganSurcharge(1.5, punyaNPWP) }

// TarifPPh23: 2% dari jumlah bruto untuk sewa harta dan imbalan jasa, menjadi
// 4% bila rekanan tidak memiliki NPWP (PMK 59/2022 Lampiran VI.A.1-2).
func TarifPPh23(punyaNPWP bool) float64 { return tarifDenganSurcharge(2, punyaNPWP) }

// tarifDenganSurcharge: tarif lebih tinggi 100% bagi Wajib Pajak tanpa NPWP.
func tarifDenganSurcharge(tarif float64, punyaNPWP bool) float64 {
	if punyaNPWP {
		return tarif
	}
	return tarif * 2
}

// TarifPPh21PNS: tarif final honorarium PNS berdasarkan golongan
// (PP 80/2010 Pasal 4 ayat (2), masih berlaku).
func TarifPPh21PNS(golongan string) float64 {
	switch golonganRompok(golongan) {
	case "I", "II":
		return 0
	case "III":
		return 5
	case "IV":
		return 15
	}
	return 0
}

// golonganRompok mengambil rompok golongan dari teks seperti "III/a" atau "IX".
func golonganRompok(golongan string) string {
	gol := strings.ToUpper(strings.TrimSpace(golongan))
	for _, cand := range []string{"IV", "III", "II", "I"} {
		if strings.HasPrefix(gol, cand) {
			next := gol[len(cand):]
			if next == "" || next[0] < 'A' || next[0] > 'Z' {
				return cand
			}
		}
	}
	return ""
}

// PPh21PNSFinal: true bila penerima memakai tarif final PNS. PP 80/2010 hanya
// berlaku untuk PNS/TNI/Polri, sehingga PPPK tidak termasuk.
func PPh21PNSFinal(p Row) bool {
	return p.S("kategori") == "PNS"
}

// TarifPPh21 menaksir tarif PPh 21 efektif (persen) untuk ditampilkan di form.
// Angka ini hanya perkiraan; perhitungan final memakai PPh21UntukPenerima.
func TarifPPh21(p Row) float64 {
	if PPh21PNSFinal(p) {
		return TarifPPh21PNS(p.S("golongan"))
	}
	switch jenisPenerimaDari(p) {
	case "bukan_pegawai":
		// 50% x bruto dikali tarif Pasal 17 lapisan pertama (5%).
		return 2.5
	case "peserta":
		return 5
	default:
		// Pegawai tidak tetap bulanan: TER bulanan 0% sampai Rp5.400.000 sebulan
		// (kategori A). Tarif pastinya bergantung pada jumlah sebulan.
		return 0
	}
}

// jenisPenerimaDari: jenis penerima tersimpan, atau diturunkan dari kategori
// untuk data lama yang belum punya kolom jenis_penerima.
func jenisPenerimaDari(p Row) string {
	if j := p.S("jenis_penerima"); j != "" {
		return j
	}
	switch p.S("kategori") {
	case "Luar":
		return "bukan_pegawai"
	case "PNS":
		return "tetap"
	default:
		return "tidak_tetap"
	}
}

// PPh21UntukPenerima menghitung PPh 21 satu penerima tanpa memperhitungkan
// pembayaran lain di bulan yang sama. bruto adalah jumlah bruto (volume x tarif),
// hari adalah jumlah hari kerja untuk pegawai tidak tetap yang dibayar harian;
// hari kosong berarti dibayar bulanan sehingga dipakai TER bulanan.
func PPh21UntukPenerima(p Row, bruto float64, hari float64) float64 {
	return PPh21UntukPenerimaBulan(p, bruto, hari, 0, 0)
}

// PPh21UntukPenerimaBulan sama dengan PPh21UntukPenerima, ditambah bruto dan
// PPh 21 dari pembayaran lain bulan yang sama untuk pegawai tidak tetap bulanan.
func PPh21UntukPenerimaBulan(p Row, bruto, hari, brutoLain, pphLain float64) float64 {
	if bruto <= 0 {
		return 0
	}
	if PPh21PNSFinal(p) {
		return math.Round(bruto * TarifPPh21PNS(p.S("golongan")) / 100)
	}
	switch jenisPenerimaDari(p) {
	case "bukan_pegawai":
		// DPP = 50% bruto, dikali tarif Pasal 17 (PMK 168/2023 Pasal 12 ayat (3)).
		return pasal17(0.5 * bruto)
	case "peserta":
		// Peserta kegiatan: DPP = jumlah bruto utuh (PMK 168/2023 Pasal 12 ayat (6)).
		return pasal17(bruto)
	case "tetap":
		// Pegawai tetap/PPPK dengan honor tidak teratur memakai tarif Pasal 17
		// atas penghasilan kena pajak; tanpa data PTKP lengkap, DPP = bruto.
		return pasal17(bruto)
	default:
		if hari <= 0 {
			return PPh21TERBulanan(defaultStr(p.S("ptkp"), "A"), bruto, brutoLain, pphLain)
		}
		return pph21TidakTetap(bruto, hari)
	}
}

// pph21TidakTetap menghitung PPh 21 pegawai tidak tetap yang dibayar harian
// (PMK 168/2023 Pasal 12 ayat (2)). Bila upah sehari tidak lebih dari Rp2.500.000,
// dipakai tarif efektif harian (PP 58/2023 Lampiran huruf D); bila melebihi,
// dipakai tarif Pasal 17 atas 50% bruto.
func pph21TidakTetap(bruto, hari float64) float64 {
	if hari <= 0 {
		hari = 1
	}
	sehari := bruto / hari
	if sehari > 2500000 {
		return pasal17(0.5 * bruto)
	}
	for _, t := range TerHarian {
		if sehari <= t.Batas {
			return math.Round(bruto * t.Tarif / 100)
		}
	}
	return pasal17(0.5 * bruto)
}

// pasal17 menerapkan tarif progresif Pasal 17 atas penghasilan kena pajak.
func pasal17(pkp float64) float64 {
	if pkp <= 0 {
		return 0
	}
	var pajak float64
	for _, l := range TarifPasal17 {
		if pkp <= l.Bawah {
			break
		}
		atas := pkp
		if l.Atas > 0 && pkp > l.Atas {
			atas = l.Atas
		}
		pajak += (atas - l.Bawah) * l.Tarif / 100
	}
	return math.Round(pajak)
}

// PerluBeaMeterai: kuitansi/bukti penerimaan uang di atas Rp5.000.000 perlu meterai.
func PerluBeaMeterai(nilai float64) bool { return nilai > BatasBeaMeterai }

// CatatanPajakBelanja merangkai keterangan singkat aturan pajak yang berlaku
// untuk satu transaksi, dipakai sebagai petunjuk di form.
func CatatanPajakBelanja(statusSekolah string, dpp float64, pkp, danaBOS, viaSiplah bool) string {
	var b strings.Builder
	if !PPNPPh22Dipungut(statusSekolah) {
		return "Sekolah swasta: bendahara bukan pemungut PPN dan PPh 22. PPh 23 tetap dipotong untuk jasa/sewa."
	}
	switch {
	case viaSiplah:
		b.WriteString("Transaksi lewat SIPLah: PPN dan PPh 22 dipungut platform, bendahara tidak memungut lagi.")
	case !pkp:
		b.WriteString("Rekanan bukan PKP: PPN tidak dipungut.")
	case dpp <= BatasPajakPPN:
		b.WriteString(fmt.Sprintf("Nilai di luar PPN paling banyak %s: PPN tidak dipungut.", Rupiah(BatasPajakPPN)))
	default:
		b.WriteString(fmt.Sprintf("Rekanan PKP dan nilai di luar PPN di atas %s: PPN dipungut.", Rupiah(BatasPajakPPN)))
	}
	if danaBOS {
		b.WriteString(" Pembelian barang dengan dana BOS tidak dipungut PPh 22.")
	}
	return b.String()
}

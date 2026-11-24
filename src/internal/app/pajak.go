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
	{"tidak_tetap", "Pegawai tidak tetap (GTT/PTT harian)"},
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
		// Pegawai tidak tetap harian: tarif efektif harian tertinggi yang berlaku
		// untuk upah di atas Rp450.000 sampai Rp2.500.000.
		return 0.5
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

// PPh21UntukPenerima menghitung PPh 21 satu penerima.
// bruto adalah jumlah bruto (volume x tarif), hari adalah jumlah hari kerja
// untuk pegawai tidak tetap yang dibayar harian.
func PPh21UntukPenerima(p Row, bruto float64, hari float64) float64 {
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
		return pph21TidakTetap(bruto, hari)
	}
}

// pph21TidakTetap menghitung PPh 21 pegawai tidak tetap (PMK 168/2023 Pasal 12 ayat (2)).
// Bila dibayar harian/satuan dan upah sehari tidak lebih dari Rp2.500.000, dipakai
// tarif efektif harian (PP 58/2023 Lampiran huruf D). Bila upah sehari melebihi
// Rp2.500.000, atau dibayar bulanan, dipakai tarif Pasal 17 atas 50% bruto.
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
func CatatanPajakBelanja(dpp float64, pkp, danaBOS, viaSiplah bool) string {
	var b strings.Builder
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

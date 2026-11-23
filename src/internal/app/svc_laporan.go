package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// LapJenis: daftar laporan berurutan (kode, nama, keterangan, ikon).
var LapJenis = []struct{ Kode, Nama, Desc, Icon string }{
	{"bku", "Buku kas umum", "Semua penerimaan dan pengeluaran dengan saldo berjalan.", "book"},
	{"tunai", "Buku pembantu kas tunai", "Tarik tunai, setor, dan belanja yang dibayar tunai.", "wallet"},
	{"bank", "Buku pembantu bank", "Penerimaan dana, bunga, biaya admin, dan belanja via bank.", "layers"},
	{"pajak", "Buku pembantu pajak", "PPN, PPh 21, 22, dan 23 yang dipungut serta disetor.", "percent"},
	{"komponen", "Rekap per komponen", "Pagu dan realisasi untuk setiap komponen BOSP.", "chart"},
	{"realisasi", "Realisasi vs RKAS", "Pagu, realisasi, dan sisa setiap baris rencana belanja.", "list"},
	{"kas", "Register penutupan kas & BA pemeriksaan kas", "Saldo kas tunai dan bank di akhir periode, siap ditandatangani.", "file"},
}

var lapJenisBelanja = map[string]string{"barang": "Barang/jasa", "honor": "Honor", "transport": "Transport"}

var lapJenisPajak = map[string]string{"PPN": "PPN", "PPH21": "PPh 21", "PPH22": "PPh 22", "PPH23": "PPh 23"}

// LapCol: satu kolom laporan (kunci, label, tipe).
type LapCol struct{ Key, Label, Type string }

// LapData: hasil satu laporan tabel.
type LapData struct {
	Title       string
	Cols        []LapCol
	Rows        []Row
	Foot        Row
	Summary     [][2]any
	Orientation string
	Kas         Row
}

func lapJenisNama(kode string) (string, bool) {
	for _, j := range LapJenis {
		if j.Kode == kode {
			return j.Nama, true
		}
	}
	return "", false
}

// LapPeriode: periode laporan '1'..'12', 'T1'/'T2', 'Y'. ok=false bila kode tak dikenal.
func LapPeriode(kode string, tahun int) (Row, bool) {
	kode = strings.ToUpper(strings.TrimSpace(kode))
	if kode == "" {
		if ThisYear() == tahun {
			kode = strconv.Itoa(int(time.Now().Month()))
		} else {
			kode = "12"
		}
	}
	m1, m2 := 0, 0
	label := ""
	monthly := false
	switch {
	case kode == "T1":
		m1, m2, label = 1, 6, fmt.Sprintf("Tahap I (Januari–Juni) %d", tahun)
	case kode == "T2":
		m1, m2, label = 7, 12, fmt.Sprintf("Tahap II (Juli–Desember) %d", tahun)
	case kode == "Y":
		m1, m2, label = 1, 12, fmt.Sprintf("Tahun Anggaran %d", tahun)
	case isDigits(kode) && toInt(kode) >= 1 && toInt(kode) <= 12:
		m1, m2 = int(toInt(kode)), int(toInt(kode))
		label = fmt.Sprintf("%s %d", BulanIndo(m1), tahun)
		monthly = true
	default:
		return nil, false
	}
	awal := fmt.Sprintf("%04d-%02d-01", tahun, m1)
	akhir := fmt.Sprintf("%04d-%02d-%02d", tahun, m2, daysInMonth(tahun, m2))
	slug := fmt.Sprintf("%d-%s", tahun, kode)
	if isDigits(kode) {
		kode = strconv.Itoa(int(toInt(kode)))
		slug = fmt.Sprintf("%04d-%02d", tahun, m1)
	}
	return Row{
		"kode": kode, "awal": awal, "akhir": akhir, "sebelum": addDays(awal, -1),
		"label": label, "bulanan": monthly, "slug": slug,
	}, true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// lapTransaksi: transaksi BKU periode, urutan sama dengan halaman BKU.
func lapTransaksi(p Row) []Row {
	return Q.All(`SELECT * FROM (
           SELECT 'trm' AS src, id, tgl, no_bukti, uraian, '' AS rekening, jumlah AS nilai, 'bank' AS metode, 'penerimaan' AS jenis
             FROM penerimaan WHERE tgl BETWEEN ? AND ?
           UNION ALL
           SELECT 'bel', b.id, b.tgl, b.no_bukti, b.uraian, COALESCE(r.rekening_kode, ''), b.total, b.metode, b.jenis
             FROM belanja b LEFT JOIN rencana_belanja r ON r.id = b.rencana_id WHERE b.tgl BETWEEN ? AND ?
           UNION ALL
           SELECT 'mut', id, tgl, no_bukti, uraian, '', jumlah, '', jenis
             FROM mutasi_kas WHERE tgl BETWEEN ? AND ?
         ) ORDER BY tgl, CASE src WHEN 'trm' THEN 0 WHEN 'mut' THEN 1 ELSE 2 END, id`,
		p.S("awal"), p.S("akhir"), p.S("awal"), p.S("akhir"), p.S("awal"), p.S("akhir"))
}

func lapColumnsKas() []LapCol {
	return []LapCol{
		{"tgl", "Tanggal", "date"}, {"no_bukti", "No. bukti", "text"}, {"rekening", "Kode rekening", "text"},
		{"uraian", "Uraian", "text"},
		{"debit", "Penerimaan", "money"}, {"kredit", "Pengeluaran", "money"}, {"saldo", "Saldo", "money"},
	}
}

// LapBuild menyusun data satu laporan.
func LapBuild(jenis string, p Row) *LapData {
	switch jenis {
	case "bku":
		return lapBKU(p)
	case "tunai":
		return lapBukuPembantu(p, "tunai")
	case "bank":
		return lapBukuPembantu(p, "bank")
	case "pajak":
		return lapPajak(p)
	case "komponen":
		return lapKomponen(p)
	case "realisasi":
		return lapRealisasi(p)
	case "kas":
		return lapKas(p)
	}
	return nil
}

func lapBKU(p Row) *LapData {
	awal := SaldoKas(p.S("sebelum"), 0, 0)
	saldo := awal.Total
	rows := []Row{{"_class": "group", "uraian": "Saldo awal", "saldo": saldo}}
	debit, kredit := 0.0, 0.0
	for _, t := range lapTransaksi(p) {
		n := t.F("nilai")
		var d, k float64
		switch {
		case t.S("src") == "trm":
			d = n
		case t.S("src") == "bel":
			k = n
		case t.S("jenis") == "bunga":
			d = n
		case t.S("jenis") == "admin_bank":
			k = n
		}
		saldo += d - k
		debit += d
		kredit += k
		info := d == 0 && k == 0
		class := ""
		uraian := t.S("uraian")
		if info {
			class = "info"
			uraian += " (pemindahbukuan " + Rupiah(n) + ", tidak mengubah saldo)"
		}
		rows = append(rows, Row{
			"_class": class, "tgl": t["tgl"], "no_bukti": t["no_bukti"], "rekening": t["rekening"],
			"uraian": uraian, "debit": lapOrNil(d), "kredit": lapOrNil(k), "saldo": saldo,
		})
	}
	akhir := SaldoKas(p.S("akhir"), 0, 0)
	return &LapData{
		Title: "BUKU KAS UMUM", Cols: lapColumnsKas(), Rows: rows,
		Foot: Row{"uraian": "Jumlah periode ini", "debit": debit, "kredit": kredit, "saldo": saldo},
		Summary: [][2]any{
			{"Saldo awal", awal.Total}, {"Jumlah penerimaan", debit}, {"Jumlah pengeluaran", kredit},
			{"Saldo akhir", akhir.Total}, {"— terdiri dari kas tunai", akhir.Tunai}, {"— terdiri dari saldo bank", akhir.Bank},
		},
		Orientation: "landscape",
	}
}

// lapOrNil: angka nol ditampilkan kosong (seperti $d ?: null di PHP).
func lapOrNil(v float64) any {
	if v == 0 {
		return nil
	}
	return v
}

func lapBukuPembantu(p Row, kas string) *LapData {
	sawal := SaldoKas(p.S("sebelum"), 0, 0)
	awal := sawal.Bank
	if kas == "tunai" {
		awal = sawal.Tunai
	}
	saldo := awal
	rows := []Row{{"_class": "group", "uraian": "Saldo awal", "saldo": saldo}}
	debit, kredit := 0.0, 0.0
	for _, t := range lapTransaksi(p) {
		n := t.F("nilai")
		var d, k float64
		skip := false
		if kas == "tunai" {
			switch {
			case t.S("src") == "bel" && t.S("metode") == "tunai":
				k = n
			case t.S("src") == "mut" && t.S("jenis") == "tarik":
				d = n
			case t.S("src") == "mut" && t.S("jenis") == "setor":
				k = n
			default:
				skip = true
			}
		} else {
			switch {
			case t.S("src") == "trm":
				d = n
			case t.S("src") == "bel" && t.S("metode") == "bank":
				k = n
			case t.S("src") == "mut" && (t.S("jenis") == "setor" || t.S("jenis") == "bunga"):
				d = n
			case t.S("src") == "mut" && (t.S("jenis") == "tarik" || t.S("jenis") == "admin_bank"):
				k = n
			default:
				skip = true
			}
		}
		if skip {
			continue
		}
		saldo += d - k
		debit += d
		kredit += k
		rows = append(rows, Row{
			"tgl": t["tgl"], "no_bukti": t["no_bukti"], "rekening": t["rekening"], "uraian": t["uraian"],
			"debit": lapOrNil(d), "kredit": lapOrNil(k), "saldo": saldo,
		})
	}
	title := "BUKU PEMBANTU BANK"
	if kas == "tunai" {
		title = "BUKU PEMBANTU KAS TUNAI"
	}
	return &LapData{
		Title: title, Cols: lapColumnsKas(), Rows: rows,
		Foot:        Row{"uraian": "Jumlah periode ini", "debit": debit, "kredit": kredit, "saldo": saldo},
		Summary:     [][2]any{{"Saldo awal", awal}, {"Jumlah penerimaan", debit}, {"Jumlah pengeluaran", kredit}, {"Saldo akhir", saldo}},
		Orientation: "landscape",
	}
}

// lapKas: register penutupan kas & data BA pemeriksaan kas.
func lapKas(p Row) *LapData {
	y := p.S("awal")[:4] + "-01-01"
	tgl := p.S("akhir")
	masuk := Q.Float("SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan WHERE tgl BETWEEN ? AND ?", y, tgl) +
		Q.Float("SELECT COALESCE(SUM(jumlah), 0) FROM mutasi_kas WHERE jenis = 'bunga' AND tgl BETWEEN ? AND ?", y, tgl)
	keluar := Q.Float("SELECT COALESCE(SUM(total), 0) FROM belanja WHERE tgl BETWEEN ? AND ?", y, tgl) +
		Q.Float("SELECT COALESCE(SUM(jumlah), 0) FROM mutasi_kas WHERE jenis = 'admin_bank' AND tgl BETWEEN ? AND ?", y, tgl)
	saldo := SaldoKas(tgl, 0, 0)
	last := Q.One("SELECT tgl, no_bukti FROM belanja WHERE tgl BETWEEN ? AND ? ORDER BY tgl DESC, id DESC LIMIT 1", y, tgl)
	s := Sekolah()
	rows := []Row{
		{"uraian": "Jumlah penerimaan s.d. " + TanggalIndo(tgl), "nilai": masuk},
		{"uraian": "Jumlah pengeluaran s.d. " + TanggalIndo(tgl), "nilai": keluar},
		{"_class": "group", "uraian": "Saldo menurut Buku Kas Umum", "nilai": saldo.Total},
		{"uraian": "Kas tunai di brankas", "nilai": saldo.Tunai},
		{"uraian": "Saldo rekening bank" + lapBankLabel(s), "nilai": saldo.Bank},
		{"_class": "group", "uraian": "Jumlah kas (tunai + bank)", "nilai": saldo.Tunai + saldo.Bank},
	}
	bukti := "—"
	if last != nil {
		bukti = last.S("no_bukti") + " (" + TanggalIndo(last.S("tgl")) + ")"
	}
	return &LapData{
		Title: "REGISTER PENUTUPAN KAS",
		Cols:  []LapCol{{"uraian", "Uraian", "text"}, {"nilai", "Jumlah", "money"}},
		Rows:  rows, Orientation: "portrait",
		Kas: Row{"tgl": tgl, "masuk": masuk, "keluar": keluar, "saldo": saldo, "bukti_terakhir": bukti},
	}
}

// lapBankLabel: keterangan rekening bank pada register kas.
func lapBankLabel(s Row) string {
	if s.S("bank_nama") == "" {
		return ""
	}
	return " (" + strings.TrimSpace(s.S("bank_nama")+" "+s.S("bank_rekening")) + ")"
}

// lapRekapColumns: kolom standar rekap (pagu, periode, kumulatif, sisa, %).
func lapRekapColumns(lead ...LapCol) []LapCol {
	cols := append([]LapCol{}, lead...)
	return append(cols,
		LapCol{"pagu", "Pagu", "money"}, LapCol{"periode", "Realisasi periode", "money"},
		LapCol{"kumulatif", "Realisasi s.d. periode", "money"}, LapCol{"sisa", "Sisa", "money"},
		LapCol{"pct", "%", "pct"})
}

func lapPct(real, pagu float64) any {
	if pagu <= 0 {
		return nil
	}
	return math.Round(real/pagu*10000) / 100
}

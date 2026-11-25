package app

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// dokumenData: data belanja + identitas untuk kuitansi/bukti terima.
func dokumenData(c *Ctx) Row {
	tahun := c.TahunAktif()
	id := c.ParamInt("id")
	b := Q.One(`SELECT b.*, t.nama AS toko_nama, t.alamat AS toko_alamat, r.uraian AS rencana_uraian,
            r.komponen_kode, r.kegiatan, k.nama AS komponen_nama
         FROM belanja b
         LEFT JOIN toko t ON t.id = b.toko_id
         LEFT JOIN rencana_belanja r ON r.id = b.rencana_id
         LEFT JOIN komponen k ON k.kode = r.komponen_kode
         WHERE b.id = ? AND b.tgl BETWEEN ? AND ?`,
		id, fmt.Sprintf("%d-01-01", tahun), fmt.Sprintf("%d-12-31", tahun))
	if b == nil {
		c.Abort(404, "Transaksi tidak ditemukan di tahun anggaran yang sedang dibuka.")
	}
	penerima := Q.All("SELECT * FROM penerima_belanja WHERE belanja_id = ? ORDER BY id", id)
	s := Sekolah()
	lunas := TglLunas(b.S("tgl"), b.S("metode"))
	tahapTgl := b.S("tgl")
	if lunas != "" {
		tahapTgl = lunas
	}
	tahap := Q.Str(`SELECT tahap FROM penerimaan WHERE tgl <= ? AND strftime('%Y', tgl) = ? AND tahap <> ''
        ORDER BY tgl DESC, id DESC LIMIT 1`, tahapTgl, b.S("tgl")[:4])
	jenisLabel := "Belanja barang/jasa"
	if b.S("jenis") == "honor" {
		jenisLabel = "Honor"
	} else if b.S("jenis") == "transport" {
		jenisLabel = "Transport"
	}
	return Row{
		"belanja": b, "penerima": penerima,
		"items":   Q.All("SELECT * FROM item_belanja WHERE belanja_id = ? ORDER BY id", id),
		"pajak":   Q.All("SELECT * FROM pajak WHERE belanja_id = ? ORDER BY id", id),
		"sekolah": s, "kepala": PejabatAktif("kepala", b.S("tgl")), "bendahara": PejabatAktif("bendahara", b.S("tgl")),
		"tanggal": TanggalIndo(b.S("tgl")), "tglLunas": lunas, "tahap": tahap,
		"lokasiTanggal": strings.Trim(strings.TrimSpace(s.S("kabupaten"))+", "+TanggalIndo(b.S("tgl")), ", "),
		"jenisLabel":    jenisLabel,
		// Kuitansi di atas Rp5 juta wajib bermeterai Rp10.000 (UU 10/2020).
		"perluMeterai": PerluBeaMeterai(b.F("total")),
		"nilaiMeterai": TarifBeaMeterai,
	}
}

// rp2: format "Rp. 1.234,56" seperti kuitansi.
func rp2(v float64) string {
	return "Rp. " + AngkaD(v, 2)
}

// tglPendek: "5 Jan 2026".
func tglPendek(tgl string) string {
	t, ok := ParseTime(tgl)
	if !ok {
		return tgl
	}
	return fmt.Sprintf("%d %s %d", t.Day(), BulanIndo(int(t.Month()))[:3], t.Year())
}

// kuitansiPenerima: satu baris penerima uang (barang → toko, honor → penerima/agregat).
func kuitansiPenerima(d Row) []Row {
	b := toRow(d["belanja"])
	penerima := toRows(d["penerima"])
	if b.S("jenis") == "barang" {
		nama := b.S("toko_nama")
		if nama == "" {
			nama = "Penyedia"
		}
		return []Row{{"nama": nama, "nip": "", "jabatan": "", "bruto": b.F("total"), "pph21": 0.0, "netto": b.F("total")}}
	}
	if len(penerima) == 0 {
		return nil
	}
	// Penerima pertama di daftar mewakili; lebih dari satu orang diberi "dkk".
	nama := penerima[0].S("nama")
	nip := penerima[0].S("nip")
	if len(penerima) > 1 {
		nama += ", dkk."
	}
	total := Row{"nama": nama, "nip": nip, "jabatan": "", "bruto": 0.0, "pph21": 0.0, "netto": 0.0}
	for _, p := range penerima {
		total["bruto"] = total.F("bruto") + p.F("bruto")
		total["pph21"] = total.F("pph21") + p.F("pph21")
		total["netto"] = total.F("netto") + p.F("netto")
	}
	return []Row{total}
}

// dokKuitansi mendaftarkan rute kuitansi PDF per belanja.
func dokKuitansi(c *Ctx) {
	d := dokumenData(c)
	receivers := kuitansiPenerima(d)
	if receivers == nil {
		c.Abort(422, "Belanja ini belum memiliki data penerima untuk kuitansi.")
	}
	b := toRow(d["belanja"])
	c.Download("application/pdf", "Kuitansi-"+safeFile(b.S("no_bukti"))+".pdf", kuitansiPdf(d, receivers), true)
}

// kuitansiPdf menyusun satu lembar kuitansi per penerima.
func kuitansiPdf(d Row, receivers []Row) []byte {
	ident := PdfIdentity(toRow(d["belanja"]).S("tgl"))
	k := NewPdf(false, ident)
	pdf := k.Pdf
	s := toRow(d["sekolah"])
	b := toRow(d["belanja"])
	lembaga := k.Lembaga()
	tahun := b.S("tgl")[:4]
	tahapNo := ""
	if strings.HasPrefix(strings.ToLower(d.S("tahap")), "tahap") {
		tahapNo = strings.TrimSpace(d.S("tahap")[5:])
	}
	desaKec := strings.Join(nonEmpty(s.S("desa"), s.S("kecamatan")), " / ")
	komponen := strings.Trim(strings.TrimSpace(b.S("komponen_kode")+" — "+b.S("komponen_nama")), " —")
	sumber := "Dana BOS"
	if d.S("tahap") != "" {
		sumber += " " + d.S("tahap")
	}
	sumber += " Tahun " + tahun
	kota := s.S("kabupaten")

	for i, p := range receivers {
		if i > 0 {
			pdf.AddPage()
		}
		// Kuitansi sengaja tanpa kop surat: dokumen ini diserahkan per penerima
		// dan identitas sekolah sudah tercetak pada baris "Sudah terima dari".
		pdf.SetY(pdf.GetY() + 4)
		startY := pdf.GetY()
		pdf.SetXY(20, startY+2)
		pdf.SetFont("Times", "U", 15)
		pdf.CellFormat(k.W-40, 8, k.T("KWITANSI / BUKTI PEMBAYARAN"), "", 1, "C", false, 0, "")
		pdf.Ln(3)
		lbl := func(label, val string, bold bool) {
			pdf.SetX(20)
			pdf.SetFont("Times", "", 12)
			pdf.CellFormat(42, 6, k.T(label), "", 0, "L", false, 0, "")
			pdf.CellFormat(6, 6, ":", "", 0, "L", false, 0, "")
			if bold {
				pdf.SetFont("Times", "B", 12)
			}
			pdf.MultiCell(k.W-40-48, 6, k.T(val), "", "L", false)
		}
		lbl("Tahun Anggaran", tahun, true)
		if tahapNo != "" {
			lbl("Tahap", tahapNo, true)
		}
		lbl("Nomor Bukti", b.S("no_bukti"), true)
		lbl("Sudah terima dari", "Kepala "+lembaga, true)
		lbl(lembaga, s.S("nama"), true)
		if desaKec != "" {
			lbl("Desa / Kecamatan", desaKec, true)
		}
		if kota != "" {
			lbl("Kabupaten", kota, true)
		}
		if s.S("provinsi") != "" {
			lbl("Provinsi", s.S("provinsi"), true)
		}
		pdf.Ln(2)
		// Jumlah uang dalam kotak.
		pdf.SetX(20)
		pdf.SetFont("Times", "", 12)
		pdf.CellFormat(42, 12, k.T("Jumlah Uang"), "", 0, "L", false, 0, "")
		pdf.CellFormat(6, 12, ":", "", 0, "L", false, 0, "")
		amt := rp2(p.F("netto"))
		pdf.SetFont("Times", "B", 14)
		aw := pdf.GetStringWidth(k.T(amt)) + 10
		x, y := pdf.GetX(), pdf.GetY()
		pdf.SetDrawColor(34, 34, 34)
		pdf.SetLineWidth(0.45)
		pdf.Rect(x, y+1, aw, 10, "D")
		pdf.SetXY(x+5, y+2.2)
		pdf.CellFormat(aw-8, 8, k.T(amt), "", 1, "L", false, 0, "")
		pdf.SetLineWidth(0.2)
		pdf.Ln(1)
		pdf.SetX(20)
		pdf.SetFont("Times", "", 12)
		pdf.CellFormat(42, 6, k.T("Terbilang"), "", 0, "L", false, 0, "")
		pdf.CellFormat(6, 6, ":", "", 0, "L", false, 0, "")
		pdf.SetFont("Times", "BI", 12)
		terbilang := strings.ToUpper(Terbilang(p["netto"])[:1]) + Terbilang(p["netto"])[1:]
		pdf.MultiCell(k.W-40-48, 6, k.T(terbilang+" rupiah"), "", "L", false)
		pdf.Ln(1)
		lbl("Untuk pembayaran", b.S("uraian"), true)
		if komponen != "" {
			pdf.SetX(20 + 48)
			pdf.SetFont("Times", "", 11)
			pdf.MultiCell(k.W-40-48, 5.4, k.T("("+komponen+")"), "", "L", false)
		}
		recCount := len(toRows(d["penerima"]))
		if b.S("jenis") != "barang" && recCount > 1 {
			pdf.SetX(20 + 48)
			pdf.SetFont("Times", "", 11)
			pdf.MultiCell(k.W-40-48, 5.4, k.T(fmt.Sprintf("%d orang; rincian pada daftar bukti terima.", recCount)), "", "L", false)
		}
		if p.F("pph21") > 0 {
			pdf.SetX(20 + 48)
			pdf.SetFont("Times", "", 11)
			pdf.MultiCell(k.W-40-48, 5.4, k.T("Bruto "+Rupiah(p["bruto"])+", PPh 21 "+Rupiah(p["pph21"])), "", "L", false)
		}
		lbl("Sumber Dana", sumber, true)

		// Tanda tangan penerima uang (kanan).
		pdf.Ln(6)
		label := "Penerima Uang,"
		if b.S("jenis") == "barang" {
			label = "Penyedia,"
		}
		rightW := (k.W - 40) / 2
		pdf.SetX(k.W - 20 - rightW)
		pdf.SetFont("Times", "", 12)
		pdf.CellFormat(rightW, 6, k.T(label), "", 1, "C", false, 0, "")
		// Kuitansi di atas Rp5.000.000 wajib bermeterai (UU 10/2020 Pasal 3 ayat (2) huruf g):
		// beri ruang tanda tangan lebih lebar dan tulisan kecil tempat meterai di atas nama.
		if PerluBeaMeterai(p.F("netto")) {
			mw, mh := 26.0, 18.0
			mx := k.W - 20 - rightW/2 - mw/2 - 12
			my := pdf.GetY() + 2
			pdf.SetTextColor(130, 130, 130)
			pdf.SetFont("Times", "", 8)
			pdf.SetXY(mx, my+mh/2-4)
			pdf.MultiCell(mw, 4, k.T(fmt.Sprintf("Meterai\nRp%s", Angka(TarifBeaMeterai))), "", "C", false)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetY(my + mh + 2)
		} else {
			pdf.Ln(14)
		}
		pdf.SetX(k.W - 20 - rightW)
		pdf.SetFont("Times", "B", 12)
		pdf.CellFormat(rightW, 6, k.T(p.S("nama")), "", 1, "C", false, 0, "")
		if p.S("nip") != "" {
			pdf.SetX(k.W - 20 - rightW)
			pdf.SetFont("Times", "", 10)
			pdf.CellFormat(rightW, 5, k.T("NIP. "+p.S("nip")), "", 1, "C", false, 0, "")
		}
		pdf.Ln(2)
		pdf.SetFont("Times", "", 12)
		pdf.CellFormat(k.W-40, 6, k.T(strings.Trim(strings.TrimSpace(kota)+", "+tglPendek(b.S("tgl")), ", ")), "", 1, "R", false, 0, "")

		// Mengetahui kepala + bendahara.
		kepala := toRow(d["kepala"])
		bendahara := toRow(d["bendahara"])
		yTTD := pdf.GetY()
		half := (k.W - 40) / 2
		pdf.SetXY(20, yTTD)
		pdf.MultiCell(half, 6, k.T("Mengetahui,\nKepala "+lembaga), "", "C", false)
		pdf.SetXY(20+half, yTTD)
		pdf.MultiCell(half, 6, k.T("\nBendahara "+lembaga), "", "C", false)
		ySig := yTTD + 18
		if img := ImagePath(kepala.S("ttd_path")); img != "" {
			pdf.ImageOptions(img, 20+half/2-15, ySig, 30, 0, false, fpdf.ImageOptions{ImageType: imgType(img), ReadDpi: true}, 0, "")
		}
		if img := ImagePath(ident.S("stempel")); img != "" {
			pdf.ImageOptions(img, 20+half/2-14, ySig-2, 26, 0, false, fpdf.ImageOptions{ImageType: imgType(img), ReadDpi: true}, 0, "")
		}
		if img := ImagePath(bendahara.S("ttd_path")); img != "" {
			pdf.ImageOptions(img, 20+half+half/2-15, ySig, 30, 0, false, fpdf.ImageOptions{ImageType: imgType(img), ReadDpi: true}, 0, "")
		}
		pdf.SetXY(20, ySig+18)
		pdf.SetFont("Times", "B", 12)
		pdf.CellFormat(half, 6, k.T(strings.ToUpper(defStr(kepala.S("nama"), "................................"))), "", 0, "C", false, 0, "")
		pdf.CellFormat(half, 6, k.T(strings.ToUpper(defStr(bendahara.S("nama"), "................................"))), "", 1, "C", false, 0, "")
		pdf.SetFont("Times", "", 10)
		nip := func(r Row) string {
			if r.S("nip") == "" {
				return ""
			}
			return "NIP. " + r.S("nip")
		}
		pdf.SetX(20)
		pdf.CellFormat(half, 5, k.T(nip(kepala)), "", 0, "C", false, 0, "")
		pdf.CellFormat(half, 5, k.T(nip(bendahara)), "", 1, "C", false, 0, "")
		pdf.Ln(3)

		// Kotak LUNAS.
		lunas := d.S("tglLunas")
		txt := "LUNAS DIBAYAR — " + "......................."
		if lunas != "" {
			txt = "LUNAS DIBAYAR — " + TanggalIndo(lunas)
		}
		bw := 80.0
		bx := (k.W - bw) / 2
		by := pdf.GetY()
		pdf.SetTextColor(180, 35, 42)
		pdf.SetDrawColor(180, 35, 42)
		pdf.SetLineWidth(0.5)
		pdf.Rect(bx, by, bw, 9, "D")
		pdf.SetXY(bx, by+1.4)
		pdf.SetFont("Times", "B", 12)
		pdf.CellFormat(bw, 6, k.T(txt), "", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetDrawColor(0, 0, 0)
		pdf.SetLineWidth(0.2)

		// Bingkai lembar.
		endY := pdf.GetY() + 3
		pdf.Rect(14, startY-3, k.W-28, endY-(startY-3), "D")
	}
	var buf bytes.Buffer
	_ = k.Output(&buf)
	return buf.Bytes()
}

func safeFile(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, s)
}

func init() {
	Register(85, func(rt *Router) {
		rt.Get("/dokumen/belanja/{id}/kuitansi.pdf", dokKuitansi)
		rt.Get("/dokumen/belanja/{id}/bukti-terima.pdf", dokBuktiTerima)
	})
}

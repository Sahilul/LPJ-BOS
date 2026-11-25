package app

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
)

// dokBuktiTerima mendaftarkan rute bukti terima honor/transport.
func dokBuktiTerima(c *Ctx) {
	d := dokumenData(c)
	b := toRow(d["belanja"])
	if b.S("jenis") != "honor" && b.S("jenis") != "transport" {
		c.Abort(404, "Bukti terima hanya tersedia untuk belanja honor atau transport.")
	}
	if len(toRows(d["penerima"])) == 0 {
		c.Abort(422, "Belanja ini belum memiliki data penerima untuk bukti terima.")
	}
	c.Download("application/pdf", "Bukti-Terima-"+safeFile(b.S("no_bukti"))+".pdf", buktiTerimaPdf(d), true)
}

// buktiTerimaPdf: daftar bukti penerimaan honor/transport siap tanda tangan.
func buktiTerimaPdf(d Row) []byte {
	k := NewPdf(false, PdfIdentity(toRow(d["belanja"]).S("tgl")))
	buktiTerimaTulis(k, d, true)
	var buf bytes.Buffer
	_ = k.Output(&buf)
	return buf.Bytes()
}

// buktiTerimaTulis menggambar daftar bukti terima ke k; first=false berarti mulai di halaman baru.
func buktiTerimaTulis(k *PdfKit, d Row, first bool) {
	b := toRow(d["belanja"])
	ident := PdfIdentity(b.S("tgl"))
	k.Ident = ident
	pdf := k.Pdf
	lembaga := k.Lembaga()

	if !first {
		pdf.AddPage()
	}
	k.Kop()
	pdf.SetFont("Times", "BU", 14)
	pdf.CellFormat(k.W-28, 7, k.T("DAFTAR BUKTI PENERIMAAN "+strings.ToUpper(d.S("jenisLabel"))), "", 1, "C", false, 0, "")
	pdf.SetFont("Times", "", 11)
	pdf.CellFormat(k.W-28, 5.5, k.T("Tahun Anggaran "+b.S("tgl")[:4]), "", 1, "C", false, 0, "")
	pdf.Ln(3)

	meta := func(label, val string) {
		pdf.SetFont("Times", "", 11)
		pdf.CellFormat(34, 5.6, k.T(label), "", 0, "L", false, 0, "")
		pdf.MultiCell(k.W-28-34, 5.6, k.T(": "+val), "", "L", false)
	}
	meta("Nomor bukti", b.S("no_bukti"))
	meta("Tanggal", d.S("tanggal"))
	meta("Uraian", b.S("uraian"))
	pdf.Ln(3)

	// Tabel penerima.
	widths := []float64{8, 76, 26, 24, 26, 22}
	heads := []string{"No.", "Nama / identitas penerima", "Bruto", "PPh 21", "Diterima", "Tanda tangan"}
	totalW := 0.0
	for _, w := range widths {
		totalW += w
	}
	writeHead := func() {
		pdf.SetFont("Times", "B", 9.5)
		pdf.SetFillColor(238, 238, 238)
		pdf.SetDrawColor(85, 85, 85)
		for i, h := range heads {
			al := "L"
			if i >= 2 && i <= 4 {
				al = "R"
			}
			pdf.CellFormat(widths[i], 8, k.T(h), "1", 0, al, true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFillColor(255, 255, 255)
		pdf.SetFont("Times", "", 9.5)
	}
	writeHead()

	tBruto, tPPh, tNetto := 0.0, 0.0, 0.0
	for i, p := range toRows(d["penerima"]) {
		nama := p.S("nama")
		if p.S("nip") != "" {
			nama += "\nNIP/NIK: " + p.S("nip")
		}
		if p.S("jabatan") != "" {
			nama += "\n" + p.S("jabatan")
		}
		nama += "\n" + AngkaD(p["volume"], 2) + " " + p.S("satuan") + " × " + Rupiah(p["tarif"])
		lines := pdf.SplitLines([]byte(k.T(nama)), widths[1]-3)
		rowH := float64(len(lines))*4.3 + 2.6
		if rowH < 13 {
			rowH = 13
		}
		if pdf.GetY()+rowH > k.H-30 {
			pdf.AddPage()
			k.Kop()
			writeHead()
		}
		x, y := pdf.GetX(), pdf.GetY()
		vals := []string{
			strconv.Itoa(i + 1), nama,
			Rupiah(p["bruto"]), Rupiah(p["pph21"]), Rupiah(p["netto"]), "",
		}
		cx := x
		for c, w := range widths {
			pdf.Rect(cx, y, w, rowH, "D")
			al := "L"
			if c >= 2 && c <= 4 {
				al = "R"
			}
			pdf.SetXY(cx+1.5, y+1.3)
			if c == 1 || c == 0 {
				pdf.SetFont("Times", "", 9.5)
			} else if c == 4 {
				pdf.SetFont("Times", "B", 9.5)
			} else {
				pdf.SetFont("Times", "", 9.5)
			}
			pdf.MultiCell(w-3, 4.3, k.T(vals[c]), "", al, false)
			cx += w
		}
		tBruto += p.F("bruto")
		tPPh += p.F("pph21")
		tNetto += p.F("netto")
		pdf.SetXY(x, y+rowH)
	}
	pdf.SetFont("Times", "B", 11)
	pdf.CellFormat(totalW, 8, k.T(fmt.Sprintf("Total bruto %s · PPh 21 %s · Total diterima %s", Rupiah(tBruto), Rupiah(tPPh), Rupiah(tNetto))), "", 1, "R", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Times", "", 11)
	pdf.MultiCell(k.W-28, 5.6, k.T("Dengan ini penerima menyatakan telah menerima pembayaran sesuai jumlah bersih yang tercantum pada daftar di atas."), "", "L", false)
	pdf.Ln(1)
	pdf.SetFont("Times", "", 11)
	pdf.CellFormat(k.W-28, 5.6, k.T(d.S("lokasiTanggal")), "", 1, "R", false, 0, "")
	pdf.Ln(2)

	// Tanda tangan kepala (kiri) dan bendahara + stempel (kanan).
	kepala := toRow(d["kepala"])
	bendahara := toRow(d["bendahara"])
	half := (k.W - 28) / 2
	y0 := pdf.GetY()
	pdf.SetX(14)
	pdf.SetFont("Times", "", 11)
	pdf.MultiCell(half, 5.6, k.T("Mengetahui,\nKepala "+lembaga), "", "C", false)
	pdf.SetXY(14+half, y0)
	pdf.MultiCell(half, 5.6, k.T("\nBendahara "+lembaga), "", "C", false)
	ySig := y0 + 18
	if img := ImagePath(kepala.S("ttd_path")); img != "" {
		pdf.ImageOptions(img, 14+half/2-15, ySig, 30, 0, false, fpdf.ImageOptions{ImageType: imgType(img), ReadDpi: true}, 0, "")
	}
	if img := ImagePath(ident.S("stempel")); img != "" {
		pdf.ImageOptions(img, 14+half+half/2-14, ySig-2, 26, 0, false, fpdf.ImageOptions{ImageType: imgType(img), ReadDpi: true}, 0, "")
	}
	if img := ImagePath(bendahara.S("ttd_path")); img != "" {
		pdf.ImageOptions(img, 14+half+half/2-15, ySig, 30, 0, false, fpdf.ImageOptions{ImageType: imgType(img), ReadDpi: true}, 0, "")
	}
	pdf.SetXY(14, ySig+18)
	pdf.SetFont("Times", "B", 11)
	pdf.CellFormat(half, 5.6, k.T(strings.ToUpper(defStr(kepala.S("nama"), "................................"))), "", 0, "C", false, 0, "")
	pdf.CellFormat(half, 5.6, k.T(strings.ToUpper(defStr(bendahara.S("nama"), "................................"))), "", 1, "C", false, 0, "")
	pdf.SetFont("Times", "", 9.5)
	nip := func(r Row) string {
		if r.S("nip") == "" {
			return ""
		}
		return "NIP. " + r.S("nip")
	}
	pdf.SetX(14)
	pdf.CellFormat(half, 5, k.T(nip(kepala)), "", 0, "C", false, 0, "")
	pdf.CellFormat(half, 5, k.T(nip(bendahara)), "", 1, "C", false, 0, "")
}

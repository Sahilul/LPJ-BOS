package app

import (
	"strings"

	"github.com/go-pdf/fpdf"
)

// rapatNotulenTulis: notulen rapat berkop dengan pembukaan, pembahasan,
// keputusan, tindak lanjut, dan tanda tangan notulis + kepala.
func rapatNotulenTulis(k *PdfKit, d Row) {
	pdf := k.Pdf
	lembaga := k.Lembaga()
	k.Kop()
	pdf.SetFont("Times", "B", 13)
	pdf.CellFormat(k.W-28, 7, k.T("NOTULEN RAPAT"), "", 1, "C", false, 0, "")
	pdf.SetFont("Times", "", 10)
	pdf.CellFormat(k.W-28, 5, k.T(d.S("perihal")+" · "+d.S("no_undangan")), "", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Identitas rapat dalam dua kolom.
	label := func(l, v string) {
		pdf.SetX(14)
		pdf.SetFont("Times", "", 10)
		pdf.CellFormat(38, 5.2, k.T(l), "", 0, "L", false, 0, "")
		pdf.CellFormat(4, 5.2, ":", "", 0, "L", false, 0, "")
		pdf.MultiCell(k.W-28-42, 5.2, k.T(v), "", "L", false)
	}
	label("Hari / Tanggal", hariTanggalIndo(d.S("tgl")))
	label("Waktu", jamRapat(d))
	label("Tempat", d.S("tempat"))
	label("Agenda", d.S("agenda"))
	label("Pimpinan rapat", defStr(d.S("pimpinan"), "-"))
	label("Notulis", defStr(d.S("notulis"), "-"))
	if n := d.I("jumlah_peserta"); n > 0 {
		label("Jumlah peserta", itoa(int(n))+" orang")
	}
	if d.S("komponen_nama") != "" {
		label("Komponen", d.S("komponen_nama"))
	}
	pdf.Ln(2)
	garisPemisah(k)

	bagian := func(judul, isi string, lh float64) {
		if strings.TrimSpace(isi) == "" {
			return
		}
		if pdf.GetY() > k.H-45 {
			pdf.AddPage()
		}
		pdf.SetFont("Times", "B", 11)
		pdf.CellFormat(k.W-28, 6, k.T(judul), "", 1, "L", false, 0, "")
		pdf.SetFont("Times", "", 10)
		k.paragraf(isi, lh, "J")
		pdf.Ln(2)
	}
	bagian("A. Pembukaan", d.S("pembukaan"), 5)
	// Pembahasan ditulis per baris sebagai butir bernomor.
	if strings.TrimSpace(d.S("pembahasan")) != "" {
		if pdf.GetY() > k.H-45 {
			pdf.AddPage()
		}
		pdf.SetFont("Times", "B", 11)
		pdf.CellFormat(k.W-28, 6, k.T("B. Pokok Pembahasan"), "", 1, "L", false, 0, "")
		pdf.SetFont("Times", "", 10)
		for i, ln := range barisTeks(d.S("pembahasan")) {
			pdf.SetX(14)
			pdf.CellFormat(7, 5, k.T(itoa(i+1)+"."), "", 0, "L", false, 0, "")
			pdf.MultiCell(k.W-28-7, 5, k.T(ln), "", "J", false)
			pdf.Ln(0.6)
		}
		pdf.Ln(2)
	}
	bagian("C. Keputusan Rapat", d.S("keputusan"), 5)
	bagian("D. Tindak Lanjut", d.S("tindak_lanjut"), 5)

	rapatTandaTangan(k, d, lembaga)
	rapatDasarBelanja(k, d)
}

// garisPemisah: garis tipis selebar teks.
func garisPemisah(k *PdfKit) {
	k.Pdf.SetDrawColor(150, 150, 150)
	k.Pdf.Line(14, k.Pdf.GetY(), k.W-14, k.Pdf.GetY())
	k.Pdf.SetDrawColor(0, 0, 0)
	k.Pdf.Ln(3)
}

// barisTeks: pecah teks per baris, buang baris kosong dan penomoran/butir
// yang diketik pengguna (notulen menomori ulang secara otomatis).
func barisTeks(s string) []string {
	out := []string{}
	for _, ln := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		ln = strings.TrimSpace(ln)
		ln = strings.TrimLeft(ln, "-•*·– ")
		ln = buangNomor(ln)
		if ln != "" {
			out = append(out, ln)
		}
	}
	return out
}

// buangNomor membuang awalan penomoran seperti "1.", "1)", "a." atau "(2)".
func buangNomor(s string) string {
	i := 0
	if strings.HasPrefix(s, "(") {
		i++
	}
	j := i
	for j < len(s) && (s[j] >= '0' && s[j] <= '9' || (i == j && ((s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z')))) {
		j++
	}
	if j == i || j >= len(s) {
		return s
	}
	sisa := s[j:]
	if strings.HasPrefix(sisa, ")") || strings.HasPrefix(sisa, ".") {
		sisa = strings.TrimSpace(sisa[1:])
		if sisa != "" && (j-i <= 2 || (j-i == 1 && i == 0)) {
			return sisa
		}
	}
	return s
}

// rapatTandaTangan: notulis (kiri) dan kepala (kanan), dengan TTD bila ada.
func rapatTandaTangan(k *PdfKit, d Row, lembaga string) {
	pdf := k.Pdf
	if pdf.GetY() > k.H-62 {
		pdf.AddPage()
	}
	pdf.Ln(4)
	lebar := (k.W - 28) / 2
	kepala := toRow(d["kepala"])
	notulis := d.S("notulis")
	pdf.SetFont("Times", "", 10)
	pdf.SetX(k.W - 14 - lebar)
	pdf.CellFormat(lebar, 5, k.T(k.TanggalLokasi(d.S("tgl"))), "", 1, "R", false, 0, "")
	y := pdf.GetY() + 2
	pdf.SetXY(14, y)
	pdf.MultiCell(lebar, 5, k.T("Notulis,"), "", "C", false)
	pdf.SetXY(14+lebar, y)
	pdf.MultiCell(lebar, 5, k.T("Kepala "+lembaga+","), "", "C", false)
	ySig := pdf.GetY() + 14
	if img := ImagePath(kepala.S("ttd_path")); img != "" {
		pdf.ImageOptions(img, 14+lebar+lebar/2-15, ySig-13, 30, 0, false, fpdf.ImageOptions{ImageType: imgType(img), ReadDpi: true}, 0, "")
	}
	pdf.SetXY(14, ySig)
	pdf.SetFont("Times", "B", 10)
	pdf.CellFormat(lebar, 5, k.T(defStr(notulis, "................................")), "", 0, "C", false, 0, "")
	pdf.CellFormat(lebar, 5, k.T(strings.ToUpper(defStr(kepala.S("nama"), "................................"))), "", 1, "C", false, 0, "")
	if kepala.S("nip") != "" {
		pdf.SetX(14 + lebar)
		pdf.SetFont("Times", "", 9)
		pdf.CellFormat(lebar, 4.6, k.T("NIP. "+kepala.S("nip")), "", 1, "C", false, 0, "")
	}
}

// rapatDasarBelanja: catatan kecil tentang belanja yang mendanai rapat.
func rapatDasarBelanja(k *PdfKit, d Row) {
	pdf := k.Pdf
	pdf.Ln(6)
	pdf.SetFont("Times", "I", 8)
	pdf.SetTextColor(120, 120, 120)
	pdf.MultiCell(k.W-28, 4, k.T("Dasar belanja: "+TanggalIndo(d.S("belanja_tgl"))+" · "+
		d.S("no_bukti")+" · "+Rupiah(d["belanja_total"])+" · "+d.S("belanja_uraian")), "", "L", false)
	pdf.SetTextColor(0, 0, 0)
}

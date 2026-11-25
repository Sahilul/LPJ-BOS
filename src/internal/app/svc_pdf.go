package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-pdf/fpdf"
)

// PdfKit membungkus FPDF dengan kop surat, tabel laporan, dan blok tanda tangan.
type PdfKit struct {
	Pdf   *fpdf.Fpdf
	tr    func(string) string
	Ident Row
	W     float64
	H     float64
}

// PdfIdentity: kop, stempel, dan pejabat yang menjabat pada tanggal.
func PdfIdentity(tgl string) Row {
	s := Sekolah()
	return Row{
		"sekolah": s,
		"kop":     s.S("kop_path"), "stempel": s.S("stempel_path"),
		"kepala":        PejabatAktif("kepala", tgl),
		"bendahara":     PejabatAktif("bendahara", tgl),
		"lokasiTanggal": strings.Trim(strings.TrimSpace(s.S("kabupaten"))+", "+TanggalIndo(tgl), ", "),
	}
}

// NewPdf menyiapkan dokumen A4 dengan margin 12/14 mm.
func NewPdf(landscape bool, ident Row) *PdfKit {
	ori := "P"
	if landscape {
		ori = "L"
	}
	pdf := fpdf.New(ori, "mm", "A4", "")
	pdf.SetMargins(14, 12, 14)
	pdf.SetAutoPageBreak(true, 14)
	pdf.SetTitle("LPJ BOS", false)
	w, h := pdf.GetPageSize()
	k := &PdfKit{Pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("cp1252"), Ident: ident, W: w, H: h}
	pdf.AddPage()
	pdf.SetFont("Times", "", 10)
	return k
}

func (k *PdfKit) T(s string) string { return k.tr(s) }

// ImagePath: path berkas unggahan bila ada, "" bila kosong/hilang.
func ImagePath(name string) string {
	if name == "" {
		return ""
	}
	p := UploadPath(name)
	if st, err := os.Stat(p); err != nil || st.IsDir() {
		return ""
	}
	return p
}

// Kop menggambar kop surat. Bila tidak ada gambar, ditulis nama & alamat sekolah.
func (k *PdfKit) Kop() {
	pdf := k.Pdf
	s := toRow(k.Ident["sekolah"])
	if p := ImagePath(k.Ident.S("kop")); p != "" {
		w := k.W - 28
		h := 20.0
		if info := pdf.RegisterImageOptions(p, fpdf.ImageOptions{ImageType: imgType(p), ReadDpi: true}); info != nil && info.Width() > 0 {
			h = w * info.Height() / info.Width()
			if h > 30 {
				h = 30
				w = h * info.Width() / info.Height()
			}
		}
		pdf.ImageOptions(p, (k.W-w)/2, pdf.GetY(), w, h, false, fpdf.ImageOptions{ImageType: imgType(p), ReadDpi: true}, 0, "")
		pdf.SetY(pdf.GetY() + h + 2)
		return
	}
	pdf.SetFont("Times", "B", 14)
	pdf.CellFormat(k.W-28, 6, k.T(s.S("nama")), "", 1, "C", false, 0, "")
	alamat := strings.Join(nonEmpty(s.S("alamat"), s.S("desa"), s.S("kecamatan"), s.S("kabupaten")), ", ")
	pdf.SetFont("Times", "", 9)
	if alamat != "" {
		pdf.CellFormat(k.W-28, 4.5, k.T(alamat), "", 1, "C", false, 0, "")
	}
	pdf.Ln(1)
}

// Judul menulis judul laporan + subjudul (periode/sekolah).
func (k *PdfKit) Judul(judul, sub string) {
	k.Pdf.SetFont("Times", "B", 13)
	k.Pdf.CellFormat(k.W-28, 7, k.T(judul), "", 1, "C", false, 0, "")
	if sub != "" {
		k.Pdf.SetFont("Times", "", 10)
		k.Pdf.CellFormat(k.W-28, 5, k.T(sub), "", 1, "C", false, 0, "")
	}
	k.Pdf.Ln(2)
}

func nonEmpty(vals ...string) []string {
	out := []string{}
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

// LapTabel menggambar tabel laporan lengkap dengan kepala, baris, dan jumlah.
func (k *PdfKit) LapTabel(d *LapData) {
	pdf := k.Pdf
	widths := lapWidths(d.Cols, k.W-28)
	head := func() {
		pdf.SetFont("Times", "B", 9)
		pdf.SetFillColor(232, 237, 252)
		for i, c := range d.Cols {
			al := "L"
			if c.Type == "money" || c.Type == "pct" {
				al = "R"
			}
			pdf.CellFormat(widths[i], 7, k.T(c.Label), "1", 0, al, true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFillColor(255, 255, 255)
	}
	head()
	pdf.SetFont("Times", "", 9)
	const lh = 4.6
	writeRow := func(row Row, bold bool) {
		style := ""
		if bold {
			style = "B"
		}
		pdf.SetFont("Times", style, 9)
		// Teks panjang dibungkus ke baris berikutnya; tinggi baris mengikuti sel terpanjang.
		cells := make([][]string, len(d.Cols))
		n := 1
		for i, c := range d.Cols {
			cells[i] = k.lapLines(row, c, widths[i])
			if len(cells[i]) > n {
				n = len(cells[i])
			}
		}
		h := float64(n)*lh + 1.4
		if pdf.GetY()+h > k.H-24 {
			pdf.AddPage()
			head()
			pdf.SetFont("Times", style, 9)
		}
		if row.S("_class") == "info" {
			pdf.SetTextColor(120, 120, 120)
		} else {
			pdf.SetTextColor(0, 0, 0)
		}
		x0, y0 := pdf.GetX(), pdf.GetY()
		x := x0
		for i, c := range d.Cols {
			al := "L"
			switch c.Type {
			case "money", "pct":
				al = "R"
			}
			for j, ln := range cells[i] {
				pdf.SetXY(x, y0+0.7+float64(j)*lh)
				pdf.CellFormat(widths[i], lh, ln, "", 0, al, false, 0, "")
			}
			x += widths[i]
		}
		pdf.Line(x0, y0+h, x, y0+h)
		pdf.SetXY(x0, y0+h)
	}
	for _, row := range d.Rows {
		writeRow(row, row.S("_class") == "group" || row.S("_class") == "sub")
	}
	if d.Foot != nil {
		pdf.SetDrawColor(120, 120, 120)
		writeRow(d.Foot, true)
	}
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(3)
	if len(d.Summary) > 0 {
		pdf.SetFont("Times", "", 10)
		for _, s := range d.Summary {
			label := toStr(s[0])
			val := ""
			switch v := s[1].(type) {
			case string:
				val = v
			case float64:
				val = Rupiah(v)
			default:
				val = toStr(v)
			}
			pdf.SetFont("Times", "B", 10)
			pdf.CellFormat(70, 5.2, k.T(label), "", 0, "L", false, 0, "")
			pdf.SetFont("Times", "", 10)
			pdf.CellFormat(45, 5.2, k.T(val), "", 1, "R", false, 0, "")
		}
		pdf.Ln(2)
	}
}

// lapLines: teks sel terformat; kolom teks dibungkus per baris, kolom lain dipotong agar muat.
func (k *PdfKit) lapLines(row Row, c LapCol, w float64) []string {
	v := row[c.Key]
	if v == nil {
		return nil
	}
	var s string
	switch c.Type {
	case "money":
		s = Rupiah(v)
	case "pct":
		s = Persen(v)
	case "date":
		s = TanggalIndoShort(v)
	default:
		s = toStr(v)
	}
	if c.Type == "text" {
		lines := k.Pdf.SplitText(s, w-2)
		for i := range lines {
			lines[i] = k.tr(lines[i])
		}
		return lines
	}
	return []string{k.tr(fitText(k.Pdf, s, w-2, "Times", "", 9))}
}

// fitText memotong teks dengan elipsis agar muat dalam lebar w pada font tertentu.
func fitText(pdf *fpdf.Fpdf, s string, w float64, family, style string, size float64) string {
	if w <= 0 || s == "" || pdf.GetStringWidth(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 1 {
		r = r[:len(r)-1]
		t := string(r) + "…"
		if pdf.GetStringWidth(t) <= w {
			return t
		}
	}
	return s
}

// lapWidths membagi lebar tabel: nominal 26mm, tanggal 20mm, % 13mm, nomor bukti 30mm,
// kode 28mm, sisanya untuk kolom teks (uraian/nama).
func lapWidths(cols []LapCol, total float64) []float64 {
	out := make([]float64, len(cols))
	fixed := 0.0
	textIdx := []int{}
	for i, c := range cols {
		switch {
		case c.Type == "money":
			out[i] = 26
		case c.Type == "date":
			out[i] = 20
		case c.Type == "pct":
			out[i] = 13
		case c.Key == "no_bukti":
			out[i] = 30
		case c.Key == "rekening" || c.Key == "kode":
			out[i] = 28
		default:
			textIdx = append(textIdx, i)
		}
		fixed += out[i]
	}
	if len(textIdx) > 0 {
		share := (total - fixed) / float64(len(textIdx))
		if share < 24 {
			share = 24
		}
		for _, i := range textIdx {
			out[i] = share
		}
	}
	return out
}

// TandaTangan menulis blok Mengetahui/Kepala + Bendahara, opsional dengan TTD & stempel.
func (k *PdfKit) TandaTangan(tgl string, stempel bool) {
	pdf := k.Pdf
	if pdf.GetY() > k.H-62 {
		pdf.AddPage()
	}
	pdf.Ln(4)
	lebar := (k.W - 28) / 2
	x0 := 14 + lebar
	pdf.SetFont("Times", "", 10)
	pdf.SetXY(x0, pdf.GetY())
	pdf.CellFormat(lebar, 5, k.T(k.TanggalLokasi(tgl)), "", 1, "R", false, 0, "")
	y := pdf.GetY() + 2
	kepala := toRow(k.Ident["kepala"])
	bendahara := toRow(k.Ident["bendahara"])
	pdf.SetXY(14, y)
	pdf.MultiCell(lebar, 5, k.T("Mengetahui,\nKepala "+k.Lembaga()), "", "C", false)
	kb := pdf.GetY()
	pdf.SetXY(x0, y)
	pdf.MultiCell(lebar, 5, k.T("\nBendahara "+k.Lembaga()), "", "C", false)
	bb := pdf.GetY()
	ySig := max(kb, bb) + 1
	if p := ImagePath(kepala.S("ttd_path")); p != "" {
		pdf.ImageOptions(p, 14+lebar/2-15, ySig, 30, 0, false, fpdf.ImageOptions{ImageType: imgType(p), ReadDpi: true}, 0, "")
	}
	if stempel {
		if p := ImagePath(k.Ident.S("stempel")); p != "" {
			pdf.ImageOptions(p, x0+lebar/2-17, ySig-2, 28, 0, false, fpdf.ImageOptions{ImageType: imgType(p), ReadDpi: true}, 0, "")
		}
	}
	if p := ImagePath(bendahara.S("ttd_path")); p != "" {
		pdf.ImageOptions(p, x0+lebar/2-15, ySig, 30, 0, false, fpdf.ImageOptions{ImageType: imgType(p), ReadDpi: true}, 0, "")
	}
	pdf.SetXY(14, ySig+20)
	pdf.SetFont("Times", "B", 10)
	pdf.CellFormat(lebar, 5, k.T(strings.ToUpper(defStr(kepala.S("nama"), "................................"))), "", 0, "C", false, 0, "")
	pdf.CellFormat(lebar, 5, k.T(strings.ToUpper(defStr(bendahara.S("nama"), "................................"))), "", 1, "C", false, 0, "")
	pdf.SetFont("Times", "", 9)
	nip := func(r Row) string {
		if r.S("nip") == "" {
			return ""
		}
		return "NIP. " + r.S("nip")
	}
	pdf.CellFormat(lebar, 4.5, k.T(nip(kepala)), "", 0, "C", false, 0, "")
	pdf.CellFormat(lebar, 4.5, k.T(nip(bendahara)), "", 1, "C", false, 0, "")
}

// Lembaga: "Madrasah" atau "Sekolah" sesuai nama/jenjang.
func (k *PdfKit) Lembaga() string {
	nama := strings.ToLower(toRow(k.Ident["sekolah"]).S("nama"))
	jenjang := strings.ToLower(toRow(k.Ident["sekolah"]).S("jenjang"))
	if strings.HasPrefix(nama, "ma") || strings.HasPrefix(nama, "mts") || strings.HasPrefix(nama, "mi") ||
		strings.HasPrefix(nama, "ra") || strings.HasPrefix(nama, "madrasah") || strings.HasPrefix(jenjang, "m") {
		return "Madrasah"
	}
	return "Sekolah"
}

// TanggalLokasi: "Kabupaten, 12 Januari 2026".
func (k *PdfKit) TanggalLokasi(tgl string) string {
	return strings.Trim(strings.TrimSpace(toRow(k.Ident["sekolah"]).S("kabupaten"))+", "+TanggalIndo(tgl), ", ")
}

func toRow(v any) Row {
	if r, ok := v.(Row); ok {
		return r
	}
	return Row{}
}

// imgType: jenis gambar untuk fpdf (tanpa titik, huruf kecil).
func imgType(p string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(p)), ".")
}

func defStr(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// Output menulis PDF ke w.
func (k *PdfKit) Output(w io.Writer) error {
	return k.Pdf.Output(w)
}

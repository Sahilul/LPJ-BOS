package app

import (
	"bytes"
	"fmt"
	"strings"
)

// lapHeaders: kepala tabel untuk template HTML.
type lapHeaderView struct {
	Label string
	Num   bool
}

// lapCellView: satu sel laporan siap tampil.
type lapCellView struct {
	Text string
	Num  bool
	Mono bool
	Rp   bool
}

type lapRowView struct {
	Cells []lapCellView
	Class string
}

type lapSumView struct {
	Label string
	Value string
}

func lapViewCell(row Row, c LapCol) lapCellView {
	v := row[c.Key]
	out := lapCellView{Num: c.Type == "money" || c.Type == "pct"}
	out.Mono = out.Num || c.Key == "tgl" || c.Key == "no_bukti" || c.Key == "kode" || c.Key == "rekening"
	if v == nil || toStr(v) == "" {
		return out
	}
	switch c.Type {
	case "money":
		out.Text, out.Rp = RupiahP(v, false), true
	case "pct":
		out.Text = Persen(v)
	case "date":
		out.Text = TanggalIndoShort(v)
	default:
		out.Text = toStr(v)
	}
	return out
}

func lapRowViews(d *LapData) []lapRowView {
	out := make([]lapRowView, 0, len(d.Rows)+1)
	build := func(row Row) lapRowView {
		v := lapRowView{Class: row.S("_class")}
		for _, c := range d.Cols {
			v.Cells = append(v.Cells, lapViewCell(row, c))
		}
		return v
	}
	for _, r := range d.Rows {
		out = append(out, build(r))
	}
	if d.Foot != nil {
		f := build(d.Foot)
		f.Class = "group"
		out = append(out, f)
	}
	return out
}

func lapSumViews(d *LapData) []lapSumView {
	out := []lapSumView{}
	for _, s := range d.Summary {
		v := ""
		if f, ok := s[1].(float64); ok {
			v = Rupiah(f)
		} else {
			v = toStr(s[1])
		}
		out = append(out, lapSumView{Label: toStr(s[0]), Value: v})
	}
	return out
}

func laporanPeriode(c *Ctx) Row {
	p, ok := LapPeriode(c.Q("periode"), c.TahunAktif())
	if !ok {
		c.Abort(404, "Periode laporan tidak dikenal.")
	}
	return p
}

func laporanIndex(c *Ctx) {
	tahun := c.TahunAktif()
	p := laporanPeriode(c)
	saldo := SaldoKas(p.S("akhir"), 0, 0)
	n := Q.Int(`SELECT (SELECT COUNT(*) FROM penerimaan WHERE tgl BETWEEN ? AND ?)
        + (SELECT COUNT(*) FROM belanja WHERE tgl BETWEEN ? AND ?)
        + (SELECT COUNT(*) FROM mutasi_kas WHERE tgl BETWEEN ? AND ?)`,
		p.S("awal"), p.S("akhir"), p.S("awal"), p.S("akhir"), p.S("awal"), p.S("akhir"))
	nBelanja := Q.Int("SELECT COUNT(*) FROM belanja WHERE tgl BETWEEN ? AND ?", p.S("awal"), p.S("akhir"))
	c.View("laporan/index", D{
		"title": "Laporan & SPJ", "tahun": tahun, "p": p, "saldo": saldo,
		"nTransaksi": n, "nBelanja": nBelanja,
		"jenisList": LapJenis,
	})
}

func laporanShow(c *Ctx) {
	jenis := c.Param("jenis")
	nama, ok := lapJenisNama(jenis)
	if !ok {
		c.Abort(404, "Jenis laporan tidak dikenal.")
	}
	p := laporanPeriode(c)
	d := LapBuild(jenis, p)
	if d == nil {
		c.Abort(404, "Jenis laporan tidak dikenal.")
	}
	file := lapFileName(nama, p)
	switch c.Q("format") {
	case "pdf":
		body := lapPdf([]*LapData{d}, p, nil, false)
		c.Download("application/pdf", file+".pdf", body, c.Q("unduh") == "")
	case "xlsx":
		body, err := lapXlsx([]*LapData{d}, p, nil)
		if err != nil {
			c.Abort(500, "Gagal menyiapkan berkas Excel.")
		}
		c.Download(xlsxMime, file+".xlsx", body, false)
	default:
		c.View("laporan/show", D{
			"title": nama + " · " + p.S("label"), "tahun": c.TahunAktif(),
			"jenis": jenis, "nama": nama, "p": p, "data": d,
			"headers": lapHeadersOf(d), "rows": lapRowViews(d), "summary": lapSumViews(d),
		})
	}
}

func lapHeadersOf(d *LapData) []lapHeaderView {
	out := make([]lapHeaderView, 0, len(d.Cols))
	for _, c := range d.Cols {
		out = append(out, lapHeaderView{Label: c.Label, Num: c.Type == "money" || c.Type == "pct"})
	}
	return out
}

func lapFileName(nama string, p Row) string {
	n := strings.ReplaceAll(nama, "&", "dan")
	parts := strings.Fields(n)
	for i, s := range parts {
		parts[i] = strings.ToUpper(s[:1]) + s[1:]
	}
	return strings.ReplaceAll(strings.Join(parts, " "), " ", "-") + "-" + p.S("slug")
}

// lapPdf: satu atau beberapa bagian laporan dalam satu PDF.
func lapPdf(parts []*LapData, p Row, belanja []Row, spj bool) []byte {
	landscape := spj
	for _, d := range parts {
		if d.Orientation == "landscape" {
			landscape = true
		}
	}
	ident := PdfIdentity(p.S("akhir"))
	k := NewPdf(landscape, ident)
	sub := p.S("label") + " · " + toRow(ident["sekolah"]).S("nama")
	for i, d := range parts {
		if i > 0 {
			k.Pdf.AddPage()
		}
		k.Kop()
		k.Judul(d.Title, sub)
		k.LapTabel(d)
		if d.Kas != nil {
			k.beritaAcaraKas(d.Kas)
		}
	}
	if spj && belanja != nil {
		k.Pdf.AddPage()
		k.Kop()
		k.Judul("RINCIAN BUKTI BELANJA", sub)
		k.rincianBelanja(belanja)
	}
	k.TandaTangan(p.S("akhir"), true)
	var buf bytes.Buffer
	_ = k.Output(&buf)
	return buf.Bytes()
}

// beritaAcaraKas menambahkan BA pemeriksaan kas di bawah register penutupan kas.
func (k *PdfKit) beritaAcaraKas(kas Row) {
	pdf := k.Pdf
	pdf.Ln(4)
	pdf.SetFont("Times", "B", 11)
	pdf.CellFormat(k.W-28, 6, k.T("BERITA ACARA PEMERIKSAAN KAS"), "", 1, "C", false, 0, "")
	pdf.SetFont("Times", "", 10)
	pdf.MultiCell(k.W-28, 5, k.T(fmt.Sprintf(
		"Pada hari ini telah dilakukan pemeriksaan kas terhadap saldo kas tunai sebesar %s dan saldo rekening bank sebesar %s, dengan jumlah keseluruhan %s. Bukti terakhir: %s.",
		Rupiah(kas["saldo"].(Saldo).Tunai), Rupiah(kas["saldo"].(Saldo).Bank),
		Rupiah(kas["saldo"].(Saldo).Total), kas.S("bukti_terakhir"))), "", "L", false)
	_ = pdf
}

// rincianBelanja menulis daftar setiap bukti belanja pada lampiran SPJ.
func (k *PdfKit) rincianBelanja(belanja []Row) {
	pdf := k.Pdf
	for _, b := range belanja {
		if pdf.GetY() > k.H-45 {
			pdf.AddPage()
		}
		pdf.SetFont("Times", "B", 10)
		pdf.MultiCell(k.W-28, 5, k.T(fmt.Sprintf("%s · %s · %s", TanggalIndoShort(b.S("tgl")), b.S("no_bukti"), b.S("uraian"))), "", "L", false)
		pdf.SetFont("Times", "", 9)
		if b.S("toko_nama") != "" {
			pdf.MultiCell(k.W-28, 4.6, k.T("Toko: "+b.S("toko_nama")), "", "L", false)
		}
		for _, it := range toRows(b["items"]) {
			pdf.MultiCell(k.W-28, 4.6, k.T(fmt.Sprintf("   - %s %s %s × %s = %s",
				it.S("nama"), AngkaD(it["volume"], 2), it.S("satuan"), Rupiah(it["harga"]), Rupiah(it["subtotal"]))), "", "L", false)
		}
		for _, pn := range toRows(b["penerima"]) {
			pdf.MultiCell(k.W-28, 4.6, k.T(fmt.Sprintf("   - %s %s %s × %s = %s; PPh 21 %s; diterima %s",
				pn.S("nama"), AngkaD(pn["volume"], 2), pn.S("satuan"), Rupiah(pn["tarif"]),
				Rupiah(pn["bruto"]), Rupiah(pn["pph21"]), Rupiah(pn["netto"]))), "", "L", false)
		}
		for _, pj := range toRows(b["pajak"]) {
			pdf.MultiCell(k.W-28, 4.6, k.T("   - Pajak "+LapJenisPajakLabel(pj.S("jenis"))+": "+Rupiah(pj["nilai"])), "", "L", false)
		}
		pdf.SetFont("Times", "B", 9)
		pdf.CellFormat(k.W-28, 4.8, k.T("   Jumlah: "+Rupiah(b["total"])), "", 1, "R", false, 0, "")
		pdf.Ln(2)
	}
}

func laporanSpj(c *Ctx) {
	p := laporanPeriode(c)
	parts := []*LapData{}
	for _, kode := range []string{"bku", "tunai", "bank", "pajak", "realisasi", "kas"} {
		parts = append(parts, LapBuild(kode, p))
	}
	belanja := LapBelanjaPeriode(p)
	if c.Q("format") == "xlsx" {
		body, err := lapXlsx(parts, p, belanja)
		if err != nil {
			c.Abort(500, "Gagal menyiapkan berkas Excel.")
		}
		c.Download(xlsxMime, "SPJ-"+p.S("slug")+".xlsx", body, false)
	}
	c.Download("application/pdf", "SPJ-"+p.S("slug")+".pdf", lapPdf(parts, p, belanja, true), true)
}

func init() {
	Register(80, func(rt *Router) {
		rt.Get("/laporan", laporanIndex)
		rt.Get("/laporan/spj", laporanSpj)
		rt.Get("/laporan/{jenis}", laporanShow)
	})
}

package app

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// rapatDoc: data satu undangan/notulen + identitas sekolah.
func rapatDoc(id int64, tahun int) Row {
	row := Q.One(`SELECT p.*, b.no_bukti, b.uraian AS belanja_uraian, b.total AS belanja_total,
            b.tgl AS belanja_tgl, r.komponen_kode, k.nama AS komponen_nama
        FROM rapat p
        JOIN belanja b ON b.id = p.belanja_id
        LEFT JOIN rencana_belanja r ON r.id = b.rencana_id
        LEFT JOIN komponen k ON k.kode = r.komponen_kode
        WHERE p.id = ? AND p.tahun = ?`, id, tahun)
	if row == nil {
		return nil
	}
	s := Sekolah()
	row["sekolah"] = s
	row["kepala"] = PejabatAktif("kepala", row.S("tgl_surat"))
	row["ident"] = PdfIdentity(row.S("tgl_surat"))
	return row
}

// rapatPdf: PDF undangan atau notulen satu rapat.
func rapatPdf(d Row, notulen bool) []byte {
	k := NewPdf(false, toRow(d["ident"]))
	if notulen {
		rapatNotulenTulis(k, d)
	} else {
		rapatUndanganTulis(k, d)
	}
	var buf bytes.Buffer
	_ = k.Output(&buf)
	return buf.Bytes()
}

// rapatUntukBelanja: undangan (dan notulen) rapat milik daftar belanja pada lampiran SPJ.
func rapatUntukBelanja(belanja []Row) []Row {
	var out []Row
	for _, b := range belanja {
		r := RapatFindBelanja(b.I("id"))
		if r == nil {
			continue
		}
		if d := rapatDoc(r.I("id"), int(r.I("tahun"))); d != nil {
			out = append(out, d)
		}
	}
	return out
}

// rapatPortraitTulis: halaman baru berorientasi tegak (surat), lalu kembali ke
// ukuran halaman kit saat ini. Dipakai bila undangan disisipkan ke PDF lanskap (SPJ).
func (k *PdfKit) rapatPortraitTulis(d Row, notulen bool) {
	k.rapatHalamanKembali(false)
	w, h := k.W, k.H
	if notulen {
		rapatNotulenTulis(k, d)
	} else {
		rapatUndanganTulis(k, d)
	}
	k.W, k.H = w, h
}

// rapatHalamanKembali: halaman baru dengan orientasi laporan.
func (k *PdfKit) rapatHalamanKembali(landscape bool) {
	ori, w, h := "P", 210.0, 297.0
	if landscape {
		ori, w, h = "L", 297.0, 210.0
	}
	k.Pdf.AddPageFormat(ori, fpdf.SizeType{Wd: w, Ht: h})
	k.Pdf.SetMargins(14, 12, 14)
	k.W, k.H = w, h
}

// paragraf: tulis teks rata kiri-kanan dengan jarak 5mm.
func (k *PdfKit) paragraf(s string, lh float64, align string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	if align == "" {
		align = "J"
	}
	k.Pdf.MultiCell(k.W-28, lh, k.T(s), "", align, false)
}

// rapatJudulSurat: nomor, lampiran, dan perihal di kanan atas surat undangan.
func rapatJudulSurat(k *PdfKit, d Row) {
	pdf := k.Pdf
	pdf.SetFont("Times", "", 11)
	baris := func(label, val string) {
		pdf.SetX(14)
		pdf.CellFormat(35, 5, k.T(label), "", 0, "L", false, 0, "")
		pdf.CellFormat(4, 5, ":", "", 0, "L", false, 0, "")
		pdf.CellFormat(k.W-28-39, 5, k.T(val), "", 1, "L", false, 0, "")
	}
	baris("Nomor", d.S("no_undangan"))
	baris("Lampiran", defStr(d.S("lampiran"), "-"))
	baris("Perihal", d.S("perihal"))
	pdf.Ln(3)
}

// rapatUndanganTulis: surat undangan rapat resmi berkop.
func rapatUndanganTulis(k *PdfKit, d Row) {
	pdf := k.Pdf
	s := toRow(d["sekolah"])
	lembaga := k.Lembaga()
	k.Kop()
	rapatJudulSurat(k, d)
	// Tanggal surat di kanan, sejajar dengan blok nomor.
	pdf.SetFont("Times", "", 11)
	pdf.CellFormat(k.W-28, 5, k.T(k.TanggalLokasi(d.S("tgl_surat"))), "", 1, "R", false, 0, "")
	pdf.Ln(3)
	k.paragraf("Kepada", 5, "L")
	k.paragraf(defStr(d.S("kepada"), "Seluruh Dewan Guru dan Tenaga Kependidikan"), 5, "L")
	k.paragraf("di "+s.S("nama"), 5, "L")
	if kec := strings.Trim(strings.TrimSpace(s.S("kecamatan"))+", "+strings.TrimSpace(s.S("kabupaten")), ", "); kec != "" {
		k.paragraf(kec, 5, "L")
	}
	pdf.Ln(4)

	k.paragraf("Assalamu'alaikum warahmatullahi wabarakatuh.", 5, "L")
	pdf.Ln(1)
	k.paragraf(fmt.Sprintf("Dengan hormat, sehubungan dengan pelaksanaan kegiatan yang didanai BOS pada komponen %s, kami mengharap kehadiran Bapak/Ibu pada:",
		defStr(d.S("komponen_nama"), "yang tercantum")), 5, "J")

	info := [][2]string{
		{"Hari / Tanggal", hariTanggalIndo(d.S("tgl"))},
		{"Waktu", jamRapat(d)},
		{"Tempat", d.S("tempat")},
		{"Agenda", d.S("agenda")},
	}
	pdf.Ln(2)
	for _, it := range info {
		pdf.SetX(20)
		pdf.SetFont("Times", "", 11)
		pdf.CellFormat(30, 5.4, k.T(it[0]), "", 0, "L", false, 0, "")
		pdf.CellFormat(4, 5.4, ":", "", 0, "L", false, 0, "")
		pdf.SetFont("Times", "B", 11)
		pdf.MultiCell(k.W-28-34-20+14, 5.4, k.T(it[1]), "", "L", false)
		pdf.SetFont("Times", "", 11)
	}
	pdf.Ln(2)
	k.paragraf("Mengingat pentingnya acara tersebut, kami mohon Bapak/Ibu dapat hadir tepat waktu. Atas perhatian dan kerja samanya kami sampaikan terima kasih.", 5, "J")
	pdf.Ln(2)
	k.paragraf("Wassalamu'alaikum warahmatullahi wabarakatuh.", 5, "L")

	// Tanda tangan kepala madrasah/sekolah saja.
	pdf.Ln(6)
	kepala := toRow(d["kepala"])
	lebar := (k.W - 28) / 2
	x0 := 14 + lebar
	pdf.SetX(x0)
	pdf.SetFont("Times", "", 11)
	pdf.CellFormat(lebar, 5, k.T("Kepala "+lembaga), "", 1, "C", false, 0, "")
	y := pdf.GetY() + 16
	if img := ImagePath(kepala.S("ttd_path")); img != "" {
		pdf.ImageOptions(img, x0+lebar/2-15, pdf.GetY(), 30, 0, false, fpdf.ImageOptions{ImageType: imgType(img), ReadDpi: true}, 0, "")
	}
	pdf.SetXY(x0, y)
	pdf.SetFont("Times", "B", 11)
	pdf.CellFormat(lebar, 5, k.T(strings.ToUpper(defStr(kepala.S("nama"), "................................"))), "", 1, "C", false, 0, "")
	if kepala.S("nip") != "" {
		pdf.SetX(x0)
		pdf.SetFont("Times", "", 10)
		pdf.CellFormat(lebar, 4.6, k.T("NIP. "+kepala.S("nip")), "", 1, "C", false, 0, "")
	}
	// Tembusan: dasar belanja yang mendanai rapat.
	pdf.Ln(6)
	dasar := fmt.Sprintf("Dasar belanja: %s · %s · %s", TanggalIndo(d.S("belanja_tgl")), d.S("no_bukti"), Rupiah(d["belanja_total"]))
	pdf.SetFont("Times", "I", 8)
	pdf.SetTextColor(120, 120, 120)
	pdf.MultiCell(k.W-28, 4, k.T(dasar), "", "L", false)
	pdf.SetTextColor(0, 0, 0)
}

// hariTanggalIndo: "Senin, 5 Januari 2026".
func hariTanggalIndo(tgl string) string {
	t, ok := ParseTime(tgl)
	if !ok {
		return TanggalIndo(tgl)
	}
	hari := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	return hari[int(t.Weekday())] + ", " + TanggalIndo(tgl)
}

// jamRapat: "09:00 - 11:00 WIB" atau hanya jam mulai. Memakai tanda hubung
// biasa karena font standar PDF tidak selalu punya glyph en dash.
func jamRapat(d Row) string {
	mulai, selesai := d.S("jam_mulai"), d.S("jam_selesai")
	switch {
	case mulai == "" && selesai == "":
		return "Menyesuaikan jadwal"
	case selesai == "":
		return mulai + " WIB"
	default:
		return mulai + " - " + selesai + " WIB"
	}
}

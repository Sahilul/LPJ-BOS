package app

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// rapatIndex: daftar undangan & notulen rapat tahun anggaran aktif.
func rapatIndex(c *Ctx) {
	tahun := c.TahunAktif()
	komponen := c.Q("komponen")
	sql := `SELECT p.*, b.no_bukti, b.uraian AS belanja_uraian, b.total AS belanja_total,
            b.jenis AS belanja_jenis, r.komponen_kode, k.nama AS komponen_nama
        FROM rapat p
        JOIN belanja b ON b.id = p.belanja_id
        LEFT JOIN rencana_belanja r ON r.id = b.rencana_id
        LEFT JOIN komponen k ON k.kode = r.komponen_kode
        WHERE p.tahun = ?`
	args := []any{tahun}
	if komponen != "" {
		sql += " AND r.komponen_kode = ?"
		args = append(args, komponen)
	}
	rows := Q.All(sql+" ORDER BY p.tgl, p.id", args...)
	for _, r := range rows {
		r["ada_notulen"] = r.S("pembahasan") != ""
		r["jam"] = r.S("jam_mulai")
		if r.S("jam_selesai") != "" {
			r["jam"] = r.S("jam_mulai") + "–" + r.S("jam_selesai")
		}
	}
	kandidat := RapatBelanjaKandidat("", tahun, 0)
	c.View("rapat/index", D{
		"title": "Undangan & notulen rapat", "tahun": tahun, "rows": rows,
		"komponen": komponen, "komponenOpts": rapatKomponenFilter(tahun),
		"nKandidat": len(kandidat),
		"nNotulen":  countRapat(rows, true),
	})
}

func countRapat(rows []Row, notulen bool) int {
	n := 0
	for _, r := range rows {
		if !notulen || r["ada_notulen"] == true {
			n++
		}
	}
	return n
}

// rapatKomponenFilter: pilihan komponen untuk penyaring, dengan opsi "semua".
func rapatKomponenFilter(tahun int) []Opt {
	return append([]Opt{{"", "Semua komponen"}}, RapatKomponenList(tahun)...)
}

func rapatFindOr404(c *Ctx) Row {
	row := RapatFind(c.ParamInt("id"))
	if row == nil || row.I("tahun") != int64(c.TahunAktif()) {
		c.Abort(404, "Undangan & notulen rapat tidak ditemukan di tahun anggaran ini.")
	}
	return row
}

// rapatForm: form undangan & notulen. row nil = buat baru.
func rapatForm(c *Ctx, row Row) {
	tahun := c.TahunAktif()
	edit := row != nil
	var kecuali int64
	belanjaID := c.Query.Get("belanja")
	if edit {
		kecuali = row.I("id")
		belanjaID = fmt.Sprint(row.I("belanja_id"))
	}
	komponen := c.Q("komponen")
	var belanja Row
	if b := toInt(belanjaID); b != 0 {
		// Belanja yang sudah punya rapat: lanjutkan ke rapat yang ada.
		if ada := RapatFindBelanja(int64(b)); ada != nil && ada.I("id") != kecuali {
			c.Flash("info", "Belanja ini sudah punya undangan & notulen. Silakan ubah dokumen yang ada.")
			c.Redirect(fmt.Sprintf("/rapat/%d/edit", ada.I("id")))
		}
		belanja = dokumenBelanjaHeader(int64(b))
	}
	if belanja != nil {
		komponen = belanja.S("komponen_kode")
	}
	kandidat := RapatBelanjaKandidat(komponen, tahun, kecuali)
	opts := []Opt{{"", "— Pilih belanja BKU —"}}
	for _, k := range kandidat {
		opts = append(opts, Opt{fmt.Sprint(k.I("id")), rapatBelanjaLabel(k)})
	}

	if row == nil {
		row = Row{"sifat": "Penting", "lampiran": "-", "tgl": c.TglDefault(), "kepada": "Seluruh Dewan Guru dan Tenaga Kependidikan"}
		if belanja != nil {
			row["tgl"] = belanja.S("tgl")
			row["tgl_surat"] = belanja.S("tgl")
			row["agenda"] = belanja.S("uraian")
		}
		row["tempat"] = defStr(Sekolah().S("nama"), "Ruang Guru")
	}
	c.loadOld()
	title := "Buat undangan & notulen"
	if edit {
		title = "Ubah undangan & notulen"
	}
	tglSurat := toStr(c.OldVal("tgl_surat", row["tgl_surat"]))
	noUndangan := toStr(c.OldVal("no_undangan", row.S("no_undangan")))
	if noUndangan == "" {
		noUndangan = RapatNomorBerikut(tahun, tglSurat)
	}
	c.View("rapat/form", D{
		"title": title, "edit": edit, "row": row, "tahun": tahun,
		"belanjaOpts": opts, "komponen": komponen, "komponenOpts": rapatKomponenFilter(tahun),
		"sifatOpts": RapatSifat, "kandidat": len(kandidat),
		"belanjaVal": toStr(c.OldVal("belanja_id", belanjaID)),
		"vals": map[string]any{
			"no_undangan": noUndangan, "tgl_surat": tglSurat,
			"sifat":          c.OldVal("sifat", row.S("sifat")),
			"lampiran":       c.OldVal("lampiran", row.S("lampiran")),
			"perihal":        c.OldVal("perihal", defStr(row.S("perihal"), "Undangan Rapat")),
			"kepada":         c.OldVal("kepada", row.S("kepada")),
			"tgl":            c.OldVal("tgl", row["tgl"]),
			"jam_mulai":      c.OldVal("jam_mulai", row.S("jam_mulai")),
			"jam_selesai":    c.OldVal("jam_selesai", row.S("jam_selesai")),
			"tempat":         c.OldVal("tempat", row.S("tempat")),
			"agenda":         c.OldVal("agenda", row.S("agenda")),
			"pimpinan":       c.OldVal("pimpinan", row.S("pimpinan")),
			"notulis":        c.OldVal("notulis", row.S("notulis")),
			"jumlah_peserta": c.OldVal("jumlah_peserta", row["jumlah_peserta"]),
			"pembukaan":      c.OldVal("pembukaan", row.S("pembukaan")),
			"pembahasan":     c.OldVal("pembahasan", row.S("pembahasan")),
			"keputusan":      c.OldVal("keputusan", row.S("keputusan")),
			"tindak_lanjut":  c.OldVal("tindak_lanjut", row.S("tindak_lanjut")),
		},
	})
}

// rapatBelanjaLabel: teks pilihan belanja pada form.
func rapatBelanjaLabel(k Row) string {
	teks := fmt.Sprintf("%s · %s · %s · %s", TanggalIndoShort(k.S("tgl")), k.S("no_bukti"),
		fitString(k.S("uraian"), 60), Rupiah(k["total"]))
	if k.S("komponen_nama") != "" {
		teks += " · " + k.S("komponen_nama")
	}
	return teks
}

// fitString memotong teks panjang untuk label pilihan.
func fitString(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// dokumenBelanjaHeader: belanja + komponen tanpa syarat tahun.
func dokumenBelanjaHeader(id int64) Row {
	return Q.One(`SELECT b.*, r.tahun, r.komponen_kode, r.uraian AS rencana_uraian, k.nama AS komponen_nama
        FROM belanja b
        LEFT JOIN rencana_belanja r ON r.id = b.rencana_id
        LEFT JOIN komponen k ON k.kode = r.komponen_kode
        WHERE b.id = ?`, id)
}

func rapatValidated(c *Ctx, id int64, back string) Row {
	data, err := RapatValidate(c.Form, id, c.TahunAktif())
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			c.WithOld(c.Form, ve.Errors)
			c.Redirect(back)
		}
		panic(err)
	}
	return data
}

// dokRapatUndangan / dokRapatNotulen: unduh PDF satu rapat.
func dokRapatUndangan(c *Ctx) {
	d := rapatDocOr404(c)
	c.Download("application/pdf", "Undangan-"+safeFile(d.S("no_undangan"))+".pdf", rapatPdf(d, false), true)
}

func dokRapatNotulen(c *Ctx) {
	d := rapatDocOr404(c)
	if strings.TrimSpace(d.S("pembahasan")) == "" {
		c.Abort(422, "Notulen belum diisi. Lengkapi pokok pembahasan rapat terlebih dahulu.")
	}
	c.Download("application/pdf", "Notulen-"+safeFile(d.S("no_undangan"))+".pdf", rapatPdf(d, true), true)
}

func rapatDocOr404(c *Ctx) Row {
	d := rapatDoc(c.ParamInt("id"), c.TahunAktif())
	if d == nil {
		c.Abort(404, "Undangan & notulen rapat tidak ditemukan di tahun anggaran ini.")
	}
	return d
}

// dokBkuRapat: cetak semua undangan (dan notulen yang sudah ada) pada satu periode BKU.
func dokBkuRapat(c *Ctx) {
	tahun := c.TahunAktif()
	p, ok := LapPeriode(c.Q("periode"), tahun)
	if !ok {
		c.Abort(404, "Periode tidak dikenal.")
	}
	var docs []Row
	sql := `SELECT p.id FROM rapat p JOIN belanja b ON b.id = p.belanja_id
        WHERE p.tahun = ?`
	args := []any{tahun}
	if c.Q("belanja") != "" {
		sql += " AND p.belanja_id = ?"
		args = append(args, toInt(c.Q("belanja")))
	} else {
		sql += " AND b.tgl BETWEEN ? AND ?"
		args = append(args, p.S("awal"), p.S("akhir"))
	}
	if c.Q("jenis") == "notulen" {
		sql += " AND p.pembahasan <> ''"
	}
	for _, r := range Q.All(sql+" ORDER BY p.tgl, p.id", args...) {
		if d := rapatDoc(r.I("id"), tahun); d != nil {
			docs = append(docs, d)
		}
	}
	if len(docs) == 0 {
		c.Abort(404, "Tidak ada undangan rapat pada periode "+p.S("label")+".")
	}
	notulen := c.Q("jenis") == "notulen"
	var k *PdfKit
	for _, d := range docs {
		if k == nil {
			k = NewPdf(false, toRow(d["ident"]))
		} else {
			k.Pdf.AddPage()
		}
		if notulen {
			rapatNotulenTulis(k, d)
		} else {
			rapatUndanganTulis(k, d)
		}
	}
	var buf bytes.Buffer
	_ = k.Output(&buf)
	nama := "Undangan-Rapat-"
	if notulen {
		nama = "Notulen-Rapat-"
	}
	c.Download("application/pdf", nama+safeFile(p.S("slug"))+".pdf", buf.Bytes(), true)
}

func init() {
	Register(72, func(rt *Router) {
		rt.Get("/rapat", rapatIndex)
		rt.Get("/rapat/baru", func(c *Ctx) { rapatForm(c, nil) })
		rt.Post("/rapat", func(c *Ctx) {
			data := rapatValidated(c, 0, "/rapat/baru")
			id := RapatCreate(data)
			c.Flash("success", "Undangan & notulen rapat dibuat. Cetak undangan lewat tombol PDF.")
			c.Redirect(fmt.Sprintf("/rapat/%d/edit", id))
		})
		rt.Get("/rapat/{id}/edit", func(c *Ctx) { rapatForm(c, rapatFindOr404(c)) })
		rt.Post("/rapat/{id}", func(c *Ctx) {
			row := rapatFindOr404(c)
			id := row.I("id")
			data := rapatValidated(c, id, fmt.Sprintf("/rapat/%d/edit", id))
			RapatUpdate(id, data)
			c.Flash("success", "Undangan & notulen rapat diperbarui.")
			c.Redirect(fmt.Sprintf("/rapat/%d/edit", id))
		})
		rt.Post("/rapat/{id}/hapus", func(c *Ctx) {
			row := rapatFindOr404(c)
			RapatDelete(row)
			c.Flash("success", "Undangan & notulen rapat dihapus.")
			c.Back("/rapat")
		})
	})
}

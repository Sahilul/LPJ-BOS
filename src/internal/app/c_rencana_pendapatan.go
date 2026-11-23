package app

import (
	"fmt"
	"math"
	"strings"
)

func pendapatanFind(c *Ctx) Row {
	row := Q.One("SELECT * FROM rencana_pendapatan WHERE id = ?", c.ParamInt("id"))
	if row == nil {
		c.Abort(404, "Rencana pendapatan tidak ditemukan.")
	}
	return row
}

func pendapatanIndex(c *Ctx) {
	tahun := c.TahunAktif()
	rows := Q.All(`SELECT r.*, COALESCE(p.realisasi, 0) AS realisasi, COALESCE(p.n, 0) AS n_penerimaan
         FROM rencana_pendapatan r
         LEFT JOIN (
            SELECT rencana_pendapatan_id, SUM(jumlah) AS realisasi, COUNT(*) AS n
            FROM penerimaan
            WHERE rencana_pendapatan_id IS NOT NULL
            GROUP BY rencana_pendapatan_id
         ) p ON p.rencana_pendapatan_id = r.id
         WHERE r.tahun = ?
         ORDER BY COALESCE(r.target_bulan, 13), r.id`, tahun)
	var totalTarget, totalRealisasi float64
	for _, r := range rows {
		target, real := r.F("target_jumlah"), r.F("realisasi")
		r["tahap_label"] = optText(PenerimaanTahap, r.S("tahap"))
		totalTarget += target
		totalRealisasi += real
		r["selisih"] = target - real
		pct := 0
		if target > 0 {
			pct = int(math.Round(real / target * 100))
			if pct > 100 {
				pct = 100
			}
		}
		r["pct"] = pct
	}
	belum := Q.Float(`SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan
         WHERE strftime('%Y', tgl) = ? AND rencana_pendapatan_id IS NULL`, fmt.Sprint(tahun))
	c.View("rencana/pendapatan/index", D{
		"title": "Rencana pendapatan", "rows": rows, "tahun": tahun,
		"totalTarget": totalTarget, "totalRealisasi": totalRealisasi, "belumDipetakan": belum,
	})
}

func pendapatanBulanOpts() []Opt {
	out := []Opt{{"", "— Belum ditentukan —"}}
	for m := 1; m <= 12; m++ {
		out = append(out, Opt{fmt.Sprint(m), BulanIndo(m)})
	}
	return out
}

func pendapatanForm(c *Ctx, row Row) {
	if row == nil {
		row = Row{"tahun": c.TahunAktif(), "tahap": "Tahap 1"}
	}
	edit := row.I("id") != 0
	title := "Tambah target pendapatan"
	head := title
	if edit {
		title = "Ubah rencana pendapatan"
		head = row.S("uraian")
	}
	c.loadOld()
	tahunHint := "Mengikuti tahun anggaran yang sedang dibuka."
	if edit {
		tahunHint = ""
	}
	c.View("rencana/pendapatan/form", D{
		"title": title, "head": head, "res": D{"Singular": "target"}, "edit": edit, "row": row,
		"tahapOpts": PenerimaanTahap, "bulanOpts": pendapatanBulanOpts(),
		"tahapVal":  toStr(c.OldVal("tahap", row.S("tahap"))),
		"uraian":    toStr(c.OldVal("uraian", row.S("uraian"))),
		"bulanVal":  toStr(c.OldVal("target_bulan", row["target_bulan"])),
		"jumlahVal": c.OldVal("target_jumlah", row["target_jumlah"]),
		"tahun":     row.I("tahun"), "tahunHint": tahunHint,
	})
}

func pendapatanValidated(c *Ctx, id int64, back string) Row {
	bulanRaw := strings.TrimSpace(c.Form.Str("target_bulan"))
	tahun := c.TahunAktif()
	if id != 0 {
		tahun = int(Q.Int("SELECT tahun FROM rencana_pendapatan WHERE id = ?", id))
	}
	data := Row{
		"tahun":         tahun,
		"tahap":         c.Form.Str("tahap"),
		"uraian":        c.Form.Trim("uraian"),
		"target_bulan":  nil,
		"target_jumlah": Uang(c.Form.Str("target_jumlah")),
	}
	if bulanRaw != "" {
		data["target_bulan"] = toInt(bulanRaw)
	}
	errs := map[string]string{}
	if !optHas(PenerimaanTahap, data.S("tahap")) {
		errs["tahap"] = "Pilih tahap pendapatan."
	}
	if data.S("uraian") == "" {
		errs["uraian"] = "Nama rencana pendapatan wajib diisi."
	}
	if bulanRaw != "" {
		b := toInt(bulanRaw)
		if b < 1 || b > 12 {
			errs["target_bulan"] = "Pilih bulan target yang valid."
		}
	}
	if data.F("target_jumlah") <= 0 {
		errs["target_jumlah"] = "Target pendapatan harus lebih dari nol."
	}
	if len(errs) > 0 {
		c.WithOld(c.Form, errs)
		c.Redirect(back)
	}
	return data
}

// RegisterRencanaPendapatan: rute target pendapatan (didaftarkan sebelum /rencana/{id}).
func RegisterRencanaPendapatan(rt *Router) {
	rt.Get("/rencana/pendapatan", pendapatanIndex)
	rt.Get("/rencana/pendapatan/baru", func(c *Ctx) { pendapatanForm(c, nil) })
	rt.Post("/rencana/pendapatan", func(c *Ctx) {
		data := pendapatanValidated(c, 0, "/rencana/pendapatan/baru")
		id := Q.Insert("rencana_pendapatan", data)
		c.Flash("success", "Rencana pendapatan “"+data.S("uraian")+"” ditambahkan.")
		c.Redirect(fmt.Sprintf("/rencana/pendapatan#row-%d", id))
	})
	rt.Get("/rencana/pendapatan/{id}/edit", func(c *Ctx) { pendapatanForm(c, pendapatanFind(c)) })
	rt.Post("/rencana/pendapatan/{id}", func(c *Ctx) {
		existing := pendapatanFind(c)
		id := c.ParamInt("id")
		data := pendapatanValidated(c, id, fmt.Sprintf("/rencana/pendapatan/%d/edit", id))
		if Q.Int("SELECT COUNT(*) FROM penerimaan WHERE rencana_pendapatan_id = ?", id) > 0 && existing.S("tahap") != data.S("tahap") {
			c.WithOld(c.Form, map[string]string{"tahap": "Tahap tidak bisa diubah karena sudah dipasangkan dengan penerimaan aktual."})
			c.Redirect(fmt.Sprintf("/rencana/pendapatan/%d/edit", id))
		}
		Q.Update("rencana_pendapatan", data, id)
		c.Flash("success", "Rencana pendapatan diperbarui.")
		c.Redirect(fmt.Sprintf("/rencana/pendapatan#row-%d", id))
	})
	rt.Post("/rencana/pendapatan/{id}/hapus", func(c *Ctx) {
		row := pendapatanFind(c)
		id := c.ParamInt("id")
		if Q.Int("SELECT COUNT(*) FROM penerimaan WHERE rencana_pendapatan_id = ?", id) > 0 {
			c.Flash("danger", "Rencana ini sudah dipasangkan dengan penerimaan aktual. Lepaskan pasangannya sebelum menghapus.")
		} else {
			Q.Run("DELETE FROM rencana_pendapatan WHERE id = ?", id)
			c.Flash("success", "Rencana pendapatan “"+row.S("uraian")+"” dihapus.")
		}
		c.Redirect("/rencana/pendapatan")
	})
}

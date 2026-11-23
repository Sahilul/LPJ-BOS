package app

import (
	"errors"
	"fmt"
	"strings"
)

func penerimaanIndex(c *Ctx) {
	tahun := c.TahunAktif()
	rows := Q.All(`SELECT p.*, rp.uraian AS rencana_uraian
         FROM penerimaan p
         LEFT JOIN rencana_pendapatan rp ON rp.id = p.rencana_pendapatan_id
         WHERE strftime('%Y', p.tgl) = ?
         ORDER BY p.tgl, p.id`, fmt.Sprint(tahun))
	var total float64
	for _, r := range rows {
		total += r.F("jumlah")
	}
	c.View("penerimaan/index", D{
		"title": "Penerimaan dana", "rows": rows, "tahun": tahun, "total": total,
	})
}

func penerimaanFindOr404(c *Ctx) Row {
	row := PenerimaanFind(c.ParamInt("id"))
	if row == nil {
		c.Abort(404, "Penerimaan tidak ditemukan.")
	}
	return row
}

// penerimaanPlans: target pendapatan tahun berjalan beserta realisasinya.
func penerimaanPlans(tahun int) []Row {
	rows := Q.All(`SELECT r.*, COALESCE(p.realisasi, 0) AS realisasi
         FROM rencana_pendapatan r
         LEFT JOIN (
            SELECT rencana_pendapatan_id, SUM(jumlah) AS realisasi
            FROM penerimaan
            WHERE rencana_pendapatan_id IS NOT NULL
            GROUP BY rencana_pendapatan_id
         ) p ON p.rencana_pendapatan_id = r.id
         WHERE r.tahun = ?
         ORDER BY COALESCE(r.target_bulan, 13), r.id`, tahun)
	for _, r := range rows {
		r["tahap_label"] = optText(PenerimaanTahap, r.S("tahap"))
	}
	return rows
}

func penerimaanForm(c *Ctx, row Row) {
	edit := row != nil
	if row == nil {
		tahun := c.TahunAktif()
		n := Q.Int("SELECT COUNT(*) FROM penerimaan WHERE strftime('%Y', tgl) = ?", fmt.Sprint(tahun))
		tahap := "Tahap 1"
		if n > 0 {
			tahap = "Tahap 2"
		}
		row = Row{
			"tgl":                   c.TglDefault(),
			"no_bukti":              NoBuktiBerikut(c.TglDefault(), "trm"),
			"uraian":                fmt.Sprintf("Penerimaan Dana BOSP Reguler %s Tahun %d", tahap, tahun),
			"tahap":                 tahap,
			"rencana_pendapatan_id": "",
		}
		if planID := c.Query.Get("rencana_pendapatan"); planID != "" {
			plan := Q.One("SELECT * FROM rencana_pendapatan WHERE id = ? AND tahun = ?", toInt(planID), tahun)
			if plan != nil {
				row["rencana_pendapatan_id"] = plan.I("id")
				row["uraian"] = plan.S("uraian")
				row["tahap"] = plan.S("tahap")
				row["jumlah"] = plan["target_jumlah"]
			}
		}
	}
	c.loadOld()
	tahun := int(toInt(keuanganTahunDari(toStr(c.OldVal("tgl", row["tgl"])))))
	if tahun == 0 {
		tahun = c.TahunAktif()
	}
	plans := penerimaanPlans(tahun)
	title := "Catat penerimaan"
	head := title
	if edit {
		title = "Ubah penerimaan"
		head = title
	}
	c.View("penerimaan/form", D{
		"title": title, "head": head, "edit": edit, "row": row,
		"plans": plans, "noPlans": len(plans) == 0,
		"tglVal":     toStr(c.OldVal("tgl", row["tgl"])),
		"noBuktiVal": toStr(c.OldVal("no_bukti", row["no_bukti"])),
		"planVal":    toStr(c.OldVal("rencana_pendapatan_id", row["rencana_pendapatan_id"])),
		"uraian":     toStr(c.OldVal("uraian", row.S("uraian"))),
		"tahapVal":   toStr(c.OldVal("tahap", row.S("tahap"))),
		"jumlahVal":  c.OldVal("jumlah", row["jumlah"]),
		"tahapOpts":  PenerimaanTahap,
	})
}

func penerimaanValidated(c *Ctx, id int64, back string) Row {
	data, err := PenerimaanValidate(c.Form, id, c.TahunAktif())
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

func init() {
	Register(40, func(rt *Router) {
		rt.Get("/penerimaan", penerimaanIndex)
		rt.Get("/penerimaan/baru", func(c *Ctx) { penerimaanForm(c, nil) })
		rt.Post("/penerimaan", func(c *Ctx) {
			data := penerimaanValidated(c, 0, "/penerimaan/baru")
			id := PenerimaanCreate(data)
			c.Flash("success", "Penerimaan "+Rupiah(data["jumlah"])+" dicatat.")
			bulan := ""
			if t := data.S("tgl"); len(t) >= 7 {
				bulan = strings.TrimLeft(t[5:7], "0")
			}
			c.Redirect("/bku?bulan=" + bulan + fmt.Sprintf("#trm-%d", id))
		})
		rt.Get("/penerimaan/{id}/edit", func(c *Ctx) { penerimaanForm(c, penerimaanFindOr404(c)) })
		rt.Post("/penerimaan/{id}", func(c *Ctx) {
			penerimaanFindOr404(c)
			id := c.ParamInt("id")
			data := penerimaanValidated(c, id, fmt.Sprintf("/penerimaan/%d/edit", id))
			PenerimaanUpdate(id, data)
			c.Flash("success", "Penerimaan diperbarui.")
			c.Redirect(fmt.Sprintf("/penerimaan#row-%d", id))
		})
		rt.Post("/penerimaan/{id}/hapus", func(c *Ctx) {
			row := penerimaanFindOr404(c)
			PenerimaanDelete(row)
			c.Flash("success", "Penerimaan “"+row.S("uraian")+"” dihapus.")
			c.Back("/penerimaan")
		})
	})
}

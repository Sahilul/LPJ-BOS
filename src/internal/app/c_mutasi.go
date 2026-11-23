package app

import (
	"errors"
	"fmt"
)

func mutasiFindOr404(c *Ctx) Row {
	row := MutasiFind(c.ParamInt("id"))
	if row == nil {
		c.Abort(404, "Mutasi kas tidak ditemukan.")
	}
	return row
}

func mutasiUraianJSON() string {
	m := map[string]string{}
	for _, o := range MutasiUraian {
		m[o.K] = o.V
	}
	b, _ := jsonMarshal(m)
	return string(b)
}

func mutasiIndex(c *Ctx) {
	tahun := c.TahunAktif()
	rows := Q.All("SELECT * FROM mutasi_kas WHERE strftime('%Y', tgl) = ? ORDER BY tgl, id", fmt.Sprint(tahun))
	for _, r := range rows {
		r["label"] = optText(MutasiJenis, r.S("jenis"))
	}
	c.View("mutasi/index", D{
		"title": "Mutasi kas", "rows": rows, "tahun": tahun,
		"saldo": SaldoKas(fmt.Sprintf("%d-12-31", tahun), 0, 0),
	})
}

func mutasiForm(c *Ctx, row Row) {
	edit := row != nil
	if row == nil {
		jenis := c.Q("jenis")
		if !optHas(MutasiJenis, jenis) {
			jenis = "tarik"
		}
		tgl := c.TglDefault()
		row = Row{"tgl": tgl, "no_bukti": NoBuktiBerikut(tgl, "mut"), "jenis": jenis, "uraian": optText(MutasiUraian, jenis)}
	}
	c.loadOld()
	var kecuali int64
	if edit {
		kecuali = row.I("id")
	}
	tglVal := toStr(c.OldVal("tgl", row["tgl"]))
	title := "Catat mutasi kas"
	if edit {
		title = "Ubah mutasi kas"
	}
	c.View("mutasi/form", D{
		"title": title, "edit": edit, "row": row,
		"saldo":      SaldoKas(tglVal, kecuali, 0),
		"tglVal":     tglVal,
		"noBuktiVal": toStr(c.OldVal("no_bukti", row["no_bukti"])),
		"jenisVal":   toStr(c.OldVal("jenis", defaultStr(row.S("jenis"), "tarik"))),
		"uraian":     toStr(c.OldVal("uraian", row.S("uraian"))),
		"jumlahVal":  c.OldVal("jumlah", row["jumlah"]),
		"jenisOpts":  MutasiJenis,
		"uraianJSON": mutasiUraianJSON(),
	})
}

func mutasiValidated(c *Ctx, id int64, back string) Row {
	data, err := MutasiValidate(c.Form, id, c.TahunAktif())
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
	Register(70, func(rt *Router) {
		rt.Get("/mutasi", mutasiIndex)
		rt.Get("/mutasi/baru", func(c *Ctx) { mutasiForm(c, nil) })
		rt.Post("/mutasi", func(c *Ctx) {
			data := mutasiValidated(c, 0, "/mutasi/baru")
			id := MutasiCreate(data)
			c.Flash("success", optText(MutasiJenis, data.S("jenis"))+" "+Rupiah(data["jumlah"])+" dicatat.")
			c.Redirect(fmt.Sprintf("/mutasi#row-%d", id))
		})
		rt.Get("/mutasi/{id}/edit", func(c *Ctx) { mutasiForm(c, mutasiFindOr404(c)) })
		rt.Post("/mutasi/{id}", func(c *Ctx) {
			mutasiFindOr404(c)
			id := c.ParamInt("id")
			data := mutasiValidated(c, id, fmt.Sprintf("/mutasi/%d/edit", id))
			MutasiUpdate(id, data)
			c.Flash("success", "Mutasi kas diperbarui.")
			c.Redirect(fmt.Sprintf("/mutasi#row-%d", id))
		})
		rt.Post("/mutasi/{id}/hapus", func(c *Ctx) {
			row := mutasiFindOr404(c)
			MutasiDelete(row)
			c.Flash("success", "Mutasi kas dihapus.")
			c.Back("/mutasi")
		})
	})
}

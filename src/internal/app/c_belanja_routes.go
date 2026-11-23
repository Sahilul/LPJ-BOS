package app

import (
	"errors"
	"fmt"
	"os"
)

func belanjaValidated(c *Ctx, id int64) (Row, BelanjaAnak) {
	data, anak, err := BelanjaValidate(c.Form, id, c.TahunAktif())
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			c.WithOld(c.Form, ve.Errors)
			c.Flash("danger", err.Error())
			back := "/belanja/baru"
			if id != 0 {
				back = fmt.Sprintf("/belanja/%d/edit", id)
			} else if rid := c.Form.Int("rencana_id"); rid != 0 && Q.Int("SELECT COUNT(*) FROM rencana_belanja WHERE id = ?", rid) > 0 {
				back += fmt.Sprintf("?rencana=%d", rid)
			}
			c.Redirect(back)
		}
		panic(err)
	}
	return data, anak
}

func belanjaBulanRedirect(tgl string, id int64) string {
	return fmt.Sprintf("/bku?bulan=%d#bel-%d", bulanDari(tgl), id)
}

func init() {
	Register(60, func(rt *Router) {
		rt.Get("/belanja/baru", func(c *Ctx) { belanjaForm(c, nil) })
		rt.Post("/belanja", func(c *Ctx) {
			data, anak := belanjaValidated(c, 0)
			id := BelanjaCreate(data, anak)
			c.Flash("success", "Belanja "+Rupiah(data["total"])+" dicatat. Sisa pagu sekarang "+Rupiah(SisaRencana(data.I("rencana_id"), 0))+".")
			if c.Form.Truthy("_lagi") {
				c.Redirect(fmt.Sprintf("/belanja/baru?rencana=%d", data.I("rencana_id")))
			}
			c.Redirect(belanjaBulanRedirect(data.S("tgl"), id))
		})
		rt.Get("/belanja/{id}/edit", func(c *Ctx) { belanjaForm(c, belanjaFindOr404(c)) })
		rt.Post("/belanja/{id}", func(c *Ctx) {
			belanjaFindOr404(c)
			id := c.ParamInt("id")
			data, anak := belanjaValidated(c, id)
			BelanjaUpdate(id, data, anak)
			c.Flash("success", "Belanja diperbarui. Sisa pagu "+Rupiah(SisaRencana(data.I("rencana_id"), id))+".")
			c.Redirect(belanjaBulanRedirect(data.S("tgl"), id))
		})
		rt.Post("/belanja/{id}/hapus", func(c *Ctx) {
			row := belanjaFindOr404(c)
			if err := BelanjaDelete(row); err != nil {
				c.Flash("danger", err.Error())
				c.Redirect(belanjaBulanRedirect(row.S("tgl"), row.I("id")))
			}
			c.Flash("success", "Belanja "+row.S("no_bukti")+" dihapus. Pagu "+Rupiah(row["total"])+" kembali ke rencana.")
			c.Redirect(fmt.Sprintf("/bku?bulan=%d", bulanDari(row.S("tgl"))))
		})
		rt.Post("/belanja/{id}/lampiran", func(c *Ctx) {
			row := belanjaFindOr404(c)
			jenis := c.Form.Str("jenis")
			if jenis != "nota" && jenis != "bast" {
				jenis = "lainnya"
			}
			fh, ok := c.File("berkas")
			if !ok || fh == nil {
				c.Flash("danger", "Pilih berkas nota atau dokumen lebih dulu.")
				c.Redirect(fmt.Sprintf("/belanja/%d/edit#lampiran", row.I("id")))
			}
			name, asli, err := LampiranSimpan(fh)
			if err != nil {
				c.Flash("danger", err.Error())
				c.Redirect(fmt.Sprintf("/belanja/%d/edit#lampiran", row.I("id")))
			}
			Q.Insert("lampiran", Row{"belanja_id": row.I("id"), "path": name, "nama_asli": asli, "jenis": jenis})
			c.Flash("success", "Lampiran diunggah.")
			c.Redirect(fmt.Sprintf("/belanja/%d/edit#lampiran", row.I("id")))
		})
		rt.Post("/belanja/{id}/lampiran/{lid}/hapus", func(c *Ctx) {
			row := belanjaFindOr404(c)
			l := Q.One("SELECT * FROM lampiran WHERE id = ? AND belanja_id = ?", c.ParamInt("lid"), row.I("id"))
			if l != nil {
				if p := UploadPath(l.S("path")); p != "" {
					_ = os.Remove(p)
				}
				Q.Run("DELETE FROM lampiran WHERE id = ?", l.I("id"))
				c.Flash("success", "Lampiran dihapus.")
			}
			c.Redirect(fmt.Sprintf("/belanja/%d/edit#lampiran", row.I("id")))
		})
	})
}

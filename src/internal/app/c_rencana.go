package app

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// rencanaItemView / rencanaPersonView: baris editor paket siap tampil.
type rencanaItemView struct{ Nama, Volume, Satuan, Harga string }

type rencanaPersonView struct{ PenerimaID, Volume, Satuan, Tarif string }

// fmtVol menampilkan jumlah dengan koma desimal (” → 1).
func fmtVol(v any) string {
	s := strings.TrimSpace(toStr(v))
	if s == "" {
		return "1"
	}
	return strings.ReplaceAll(phpFloatStr(Desimal(s)), ".", ",")
}

// fmtHarga menampilkan nominal dengan pemisah ribuan (” → kosong).
func fmtHarga(v any) string {
	s := strings.TrimSpace(toStr(v))
	if s == "" {
		return ""
	}
	n := Uang(s)
	if n <= 0 {
		return ""
	}
	return Angka(n)
}

func rencanaFindOr404(c *Ctx) Row {
	row := RencanaFind(c.ParamInt("id"))
	if row == nil {
		c.Abort(404, "Paket rencana belanja tidak ditemukan.")
	}
	return row
}

// rencanaKomponenOpts: pilihan komponen berurutan.
func rencanaKomponenOpts() []Opt {
	out := []Opt{{"", "— Pilih komponen —"}}
	for _, k := range Q.All("SELECT kode, nama FROM komponen ORDER BY urutan, kode") {
		out = append(out, Opt{k.S("kode"), k.S("kode") + " · " + k.S("nama")})
	}
	return out
}

func rencanaIndex(c *Ctx) {
	tahun := c.TahunAktif()
	q := strings.TrimSpace(c.Q("q"))
	komp := c.Q("komponen")
	where := []string{"r.tahun = ?"}
	args := []any{tahun}
	if komp != "" {
		where = append(where, "r.komponen_kode = ?")
		args = append(args, komp)
	}
	if q != "" {
		where = append(where, "(r.uraian LIKE ? OR r.kegiatan LIKE ? OR r.rekening_kode LIKE ?)")
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	rows := Q.All(`SELECT r.*, k.nama AS komponen_nama, t.nama AS toko_nama,
            COALESCE(b.pakai, 0) AS realisasi, COALESCE(b.n, 0) AS n_belanja,
            COALESCE(p.periode, 0) AS periode_terpakai
         FROM rencana_belanja r
         LEFT JOIN komponen k ON k.kode = r.komponen_kode
         LEFT JOIN toko t ON t.id = r.toko_id
         LEFT JOIN (SELECT rencana_id, SUM(total) AS pakai, COUNT(*) AS n FROM belanja GROUP BY rencana_id) b ON b.rencana_id = r.id
         LEFT JOIN (SELECT rencana_id, SUM(jumlah_periode) AS periode FROM belanja GROUP BY rencana_id) p ON p.rencana_id = r.id
         WHERE `+strings.Join(where, " AND ")+`
         ORDER BY r.komponen_kode, r.id`, args...)

	detail := map[int64]D{}
	for _, it := range Q.All("SELECT * FROM rencana_item ORDER BY id") {
		d := detail[it.I("rencana_id")]
		if d == nil {
			d = D{"items": []Row{}, "penerima": []Row{}}
			detail[it.I("rencana_id")] = d
		}
		d["items"] = append(d["items"].([]Row), it)
	}
	for _, p := range Q.All("SELECT rp.*, p.nama FROM rencana_penerima rp LEFT JOIN penerima p ON p.id = rp.penerima_id ORDER BY rp.id") {
		d := detail[p.I("rencana_id")]
		if d == nil {
			d = D{"items": []Row{}, "penerima": []Row{}}
			detail[p.I("rencana_id")] = d
		}
		d["penerima"] = append(d["penerima"].([]Row), p)
	}

	var totPagu, totReal float64
	for _, r := range rows {
		pagu, real := r.F("pagu"), r.F("realisasi")
		totPagu += pagu
		totReal += real
		r["sisa"] = pagu - real
		if pagu > 0 {
			pct := int(math.Round(real / pagu * 100))
			if pct > 100 {
				pct = 100
			}
			r["pct"] = pct
		} else {
			r["pct"] = 0
		}
		d := detail[r.I("id")]
		if d == nil {
			d = D{"items": []Row{}, "penerima": []Row{}}
		}
		r["items"] = d["items"]
		r["penerima"] = d["penerima"]
	}
	totPct := 0
	if totPagu > 0 {
		totPct = int(math.Round(totReal / totPagu * 100))
		if totPct > 100 {
			totPct = 100
		}
	}
	c.View("rencana/index", D{
		"title": "Rencana belanja", "rows": rows, "tahun": tahun,
		"q": q, "komp": komp, "komponen": rencanaKomponenOpts(),
		"filtered": q != "" || komp != "",
		"totPagu":  totPagu, "totReal": totReal, "totSisa": totPagu - totReal, "totPct": totPct,
	})
}

func rencanaForm(c *Ctx, row Row) {
	edit := row != nil
	c.loadOld()
	posted := c.Posted()
	if row == nil {
		row = Row{"tahun": c.TahunAktif(), "jenis": "barang", "frekuensi_bulan": 1}
	}
	id := row.I("id")
	if edit {
		row["items"] = Q.All("SELECT * FROM rencana_item WHERE rencana_id = ? ORDER BY id", id)
		row["penerima"] = Q.All("SELECT * FROM rencana_penerima WHERE rencana_id = ? ORDER BY id", id)
	}
	items := []rencanaItemView{}
	recipients := []rencanaPersonView{}
	if posted {
		for _, it := range c.Old.List("items") {
			items = append(items, rencanaItemView{
				Nama: it.Str("nama"), Volume: fmtVol(it.Str("volume")),
				Satuan: it.Str("satuan"), Harga: fmtHarga(it.Str("harga")),
			})
		}
		for _, p := range c.Old.List("penerima") {
			recipients = append(recipients, rencanaPersonView{
				PenerimaID: p.Str("penerima_id"), Volume: fmtVol(p.Str("volume")),
				Satuan: p.Str("satuan"), Tarif: fmtHarga(p.Str("tarif")),
			})
		}
	} else {
		for _, it := range toRows(row["items"]) {
			items = append(items, rencanaItemView{
				Nama: it.S("nama"), Volume: fmtVol(it["volume"]),
				Satuan: it.S("satuan"), Harga: fmtHarga(it["harga_satuan"]),
			})
		}
		for _, p := range toRows(row["penerima"]) {
			recipients = append(recipients, rencanaPersonView{
				PenerimaID: toStr(p["penerima_id"]), Volume: fmtVol(p["volume"]),
				Satuan: p.S("satuan"), Tarif: fmtHarga(p["tarif"]),
			})
		}
	}
	if len(items) == 0 {
		items = append(items, rencanaItemView{Volume: "1"})
	}
	if len(recipients) == 0 {
		recipients = append(recipients, rencanaPersonView{Volume: "1"})
	}

	jadwal := map[int]bool{}
	if posted {
		for _, s := range c.Old.Strings("jadwal") {
			jadwal[int(toInt(s))] = true
		}
	} else {
		for _, s := range strings.Split(row.S("jadwal_bulan"), ",") {
			if s = strings.TrimSpace(s); s != "" {
				jadwal[int(toInt(s))] = true
			}
		}
	}

	tersangka := int64(0)
	if edit {
		tersangka = id
	}
	ids := []string{}
	for _, p := range toRows(row["penerima"]) {
		if pid := p.I("penerima_id"); pid != 0 {
			ids = append(ids, fmt.Sprint(pid))
		}
	}
	in := ""
	if len(ids) > 0 {
		in = " OR id IN (" + strings.Join(ids, ",") + ")"
	}
	penerimaList := Q.All("SELECT id, nama, nip, kategori, golongan, jabatan, tarif_default, aktif FROM penerima WHERE aktif = 1" + in + " ORDER BY nama")

	title := "Tambah paket rencana"
	head := title
	if edit {
		title = "Ubah paket rencana"
		head = row.S("uraian")
	}
	c.View("rencana/form", D{
		"title": title, "head": head, "edit": edit, "row": row,
		"komponen":     rencanaKomponenOpts(),
		"rekening":     Q.All("SELECT kode, nama, jenis FROM kode_rekening ORDER BY kode"),
		"toko":         Q.All("SELECT id, nama FROM toko WHERE aktif = 1 OR id = ? ORDER BY nama", row.I("toko_id")),
		"penerimaList": penerimaList,
		"items":        items, "recipients": recipients, "jadwal": jadwal,
		"uraian":          toStr(c.OldVal("uraian", row.S("uraian"))),
		"kegiatan":        toStr(c.OldVal("kegiatan", row.S("kegiatan"))),
		"komponenVal":     toStr(c.OldVal("komponen_kode", row.S("komponen_kode"))),
		"rekeningVal":     toStr(c.OldVal("rekening_kode", row.S("rekening_kode"))),
		"jenis":           toStr(c.OldVal("jenis", defaultStr(row.S("jenis"), "barang"))),
		"frekuensi":       toStr(c.OldVal("frekuensi_bulan", defaultInt(row.I("frekuensi_bulan"), 1))),
		"tokoVal":         toStr(c.OldVal("toko_id", row["toko_id"])),
		"tahun":           row.I("tahun"),
		"jenisOpts":       RencanaJenis,
		"tahunHint":       tahunHint(edit),
		"terpakai":        Q.Float("SELECT COALESCE(SUM(total), 0) FROM belanja WHERE rencana_id = ?", tersangka),
		"periodeTerpakai": Q.Int("SELECT COALESCE(SUM(jumlah_periode), 0) FROM belanja WHERE rencana_id = ?", tersangka),
	})
}

func tahunHint(edit bool) string {
	if edit {
		return ""
	}
	return "Mengikuti tahun yang sedang dibuka."
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func defaultInt(v, def int64) int64 {
	if v == 0 {
		return def
	}
	return v
}

// toRows mengubah lampiran Row/[]Row hasil query menjadi []Row.
func toRows(v any) []Row {
	switch x := v.(type) {
	case []Row:
		return x
	case Row:
		return []Row{x}
	}
	return nil
}

func rencanaValidated(c *Ctx, id int64, back string) (Row, []Row, []Row) {
	data, items, recipients, err := RencanaValidate(c.Form, id, c.TahunAktif())
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			c.WithOld(c.Form, ve.Errors)
			c.Redirect(back)
		}
		panic(err)
	}
	return data, items, recipients
}

func init() {
	Register(30, func(rt *Router) {
		RegisterRencanaPendapatan(rt)
		rt.Get("/rencana", rencanaIndex)
		rt.Get("/rencana/baru", func(c *Ctx) { rencanaForm(c, nil) })
		rt.Post("/rencana", func(c *Ctx) {
			data, items, recipients := rencanaValidated(c, 0, "/rencana/baru")
			id := RencanaCreate(data, items, recipients)
			c.Flash("success", "Paket “"+data.S("uraian")+"” ditambahkan dengan pagu "+Rupiah(data["pagu"])+".")
			if c.Form.Truthy("_lagi") {
				c.Redirect("/rencana/baru")
			}
			c.Redirect(fmt.Sprintf("/rencana#row-%d", id))
		})
		rt.Get("/rencana/{id}/edit", func(c *Ctx) { rencanaForm(c, rencanaFindOr404(c)) })
		rt.Post("/rencana/{id}", func(c *Ctx) {
			rencanaFindOr404(c)
			id := c.ParamInt("id")
			data, items, recipients := rencanaValidated(c, id, fmt.Sprintf("/rencana/%d/edit", id))
			RencanaUpdate(id, data, items, recipients)
			c.Flash("success", "Paket “"+data.S("uraian")+"” diperbarui.")
			c.Redirect(fmt.Sprintf("/rencana#row-%d", id))
		})
		rt.Post("/rencana/{id}/hapus", func(c *Ctx) {
			row := rencanaFindOr404(c)
			if err := RencanaDelete(c.ParamInt("id")); err != nil {
				c.Flash("danger", err.Error())
			} else {
				c.Flash("success", "Paket “"+row.S("uraian")+"” dihapus.")
			}
			c.Redirect("/rencana")
		})
	})
}

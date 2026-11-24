package app

import (
	"fmt"
	"strings"
)

// belanjaItemView / belanjaPenView: baris editor belanja siap tampil.
type belanjaItemView struct{ Nama, Volume, Satuan, Harga, PackageVolume string }

type belanjaPenView struct {
	PenerimaID, Volume, Satuan, Tarif, PackageVolume, Ket, Hari string
}

const batasPajakBarang = 2000000

func belanjaFindOr404(c *Ctx) Row {
	row := BelanjaFind(c.ParamInt("id"))
	if row == nil {
		c.Abort(404, "Transaksi belanja tidak ditemukan.")
	}
	return row
}

// belanjaPaketData: paket tahun aktif (plus paket transaksi yang sedang diedit).
func belanjaPaketData(editId int64, tahun int) ([]Row, map[int64]D) {
	rencana := Q.All(`SELECT r.id, r.tahun, r.uraian, r.jenis, r.satuan, r.pagu, r.komponen_kode, r.harga_satuan,
            r.frekuensi_bulan, r.toko_id,
            r.pagu - COALESCE((SELECT SUM(total) FROM belanja b WHERE b.rencana_id = r.id AND b.id != ?), 0) AS sisa,
            r.frekuensi_bulan - COALESCE((SELECT SUM(jumlah_periode) FROM belanja b WHERE b.rencana_id = r.id AND b.id != ?), 0) AS periode_sisa
         FROM rencana_belanja r
         WHERE r.tahun = ? OR r.id = ?
         ORDER BY r.komponen_kode, r.id`, editId, editId, tahun, editId)
	paket := map[int64]D{}
	for _, it := range Q.All("SELECT * FROM rencana_item ORDER BY id") {
		d := paket[it.I("rencana_id")]
		if d == nil {
			d = D{"items": []D{}, "penerima": []D{}, "satuan": ""}
			paket[it.I("rencana_id")] = d
		}
		d["items"] = append(d["items"].([]D), D{
			"nama": it.S("nama"), "volume": it.F("volume"), "satuan": it.S("satuan"), "harga": it.F("harga_satuan"),
		})
	}
	for _, p := range Q.All("SELECT * FROM rencana_penerima ORDER BY id") {
		d := paket[p.I("rencana_id")]
		if d == nil {
			d = D{"items": []D{}, "penerima": []D{}, "satuan": ""}
			paket[p.I("rencana_id")] = d
		}
		d["penerima"] = append(d["penerima"].([]D), D{
			"penerima_id": p.I("penerima_id"), "volume": p.F("volume"), "satuan": p.S("satuan"), "tarif": p.F("tarif"),
		})
	}
	for _, r := range rencana {
		d := paket[r.I("id")]
		if d == nil {
			d = D{"items": []D{}, "penerima": []D{}, "satuan": ""}
		}
		entries := d["items"].([]D)
		if r.S("jenis") != "barang" {
			entries = d["penerima"].([]D)
		}
		if len(entries) > 0 {
			d["satuan"] = entries[0]["satuan"]
		}
		r["satuan_paket"] = d["satuan"]
		pj, _ := jsonMarshal(d)
		r["paket_json"] = string(pj)
	}
	return rencana, paket
}

func belanjaForm(c *Ctx, row Row) {
	edit := row != nil
	var editId int64
	if edit {
		editId = row.I("id")
		row["items"] = Q.All("SELECT * FROM item_belanja WHERE belanja_id = ? ORDER BY id", editId)
		row["penerima"] = Q.All("SELECT * FROM penerima_belanja WHERE belanja_id = ? ORDER BY id", editId)
		pj := map[string]Row{}
		for _, p := range Q.All("SELECT * FROM pajak WHERE belanja_id = ?", editId) {
			pj[p.S("jenis")] = p
		}
		row["pajak"] = pj
	} else {
		row = Row{"tgl": c.TglDefault(), "rencana_id": "", "jumlah_periode": 1, "metode": "tunai", "items": []Row{}, "penerima": []Row{}}
	}
	c.loadOld()
	posted := c.Posted()

	if !edit {
		if rid := toInt(c.OldVal("rencana_id", c.Query.Get("rencana"))); rid != 0 {
			if plan := Q.One("SELECT uraian, satuan, jenis, toko_id FROM rencana_belanja WHERE id = ?", rid); plan != nil {
				row["rencana_id"] = rid
				row["uraian"] = plan.S("uraian")
				row["satuan_default"] = plan.S("satuan")
				row["toko_id"] = plan["toko_id"]
				if plan.S("jenis") == "barang" {
					items := []Row{}
					for _, it := range Q.All("SELECT * FROM rencana_item WHERE rencana_id = ? ORDER BY id", rid) {
						items = append(items, Row{"nama": it["nama"], "volume": it["volume"], "satuan": it["satuan"], "harga": it["harga_satuan"], "package_volume": it["volume"]})
					}
					row["items"] = items
				} else {
					pen := []Row{}
					for _, p := range Q.All("SELECT * FROM rencana_penerima WHERE rencana_id = ? ORDER BY id", rid) {
						pen = append(pen, Row{"penerima_id": p["penerima_id"], "volume": p["volume"], "satuan": p["satuan"], "tarif": p["tarif"], "package_volume": p["volume"]})
					}
					row["penerima"] = pen
				}
			}
		}
	}

	rencana, _ := belanjaPaketData(editId, c.TahunAktif())
	rid := toStr(c.OldVal("rencana_id", row["rencana_id"]))
	var sel Row
	for _, r := range rencana {
		if fmt.Sprint(r.I("id")) == rid {
			sel = r
		}
	}
	jenis := "barang"
	if sel != nil {
		jenis = sel.S("jenis")
	}
	satDefault := row.S("satuan_default")
	if satDefault == "" && sel != nil {
		satDefault = sel.S("satuan_paket")
	}
	periode := c.OldVal("jumlah_periode", defaultInt(row.I("jumlah_periode"), 1))

	items := []belanjaItemView{}
	pens := []belanjaPenView{}
	if posted {
		for _, it := range c.Old.List("items") {
			items = append(items, belanjaItemView{Nama: it.Str("nama"), Volume: fmtVol(it.Str("volume")), Satuan: it.Str("satuan"), Harga: fmtHarga(it.Str("harga"))})
		}
		for _, p := range c.Old.List("pen") {
			pens = append(pens, belanjaPenView{
				PenerimaID: p.Str("penerima_id"), Volume: fmtVol(p.Str("volume")),
				Satuan: p.Str("satuan"), Tarif: fmtHarga(p.Str("tarif")), Ket: penerimaKet(toInt(p.Str("penerima_id"))),
				Hari: p.Str("hari"),
			})
		}
	} else {
		for _, it := range toRows(row["items"]) {
			pv := ""
			if !it.Null("package_volume") {
				pv = fmtVol(it["package_volume"])
			}
			items = append(items, belanjaItemView{Nama: it.S("nama"), Volume: fmtVol(it["volume"]), Satuan: it.S("satuan"), Harga: fmtHarga(it["harga"]), PackageVolume: pv})
		}
		for _, p := range toRows(row["penerima"]) {
			pv := ""
			if !p.Null("package_volume") {
				pv = fmtVol(p["package_volume"])
			}
			pens = append(pens, belanjaPenView{
				PenerimaID: toStr(p["penerima_id"]), Volume: fmtVol(p["volume"]), Satuan: p.S("satuan"),
				Tarif: fmtHarga(p["tarif"]), PackageVolume: pv, Ket: penerimaKet(p.I("penerima_id")),
				Hari: fmtVol(p["hari"]),
			})
		}
	}
	if len(items) == 0 {
		items = append(items, belanjaItemView{Volume: "1"})
	}
	if len(pens) == 0 {
		pens = append(pens, belanjaPenView{Volume: "1", Satuan: satDefault})
	}

	ids := []string{}
	for _, p := range toRows(row["penerima"]) {
		if pid := p.I("penerima_id"); pid != 0 {
			ids = append(ids, fmt.Sprint(pid))
		}
	}
	for _, p := range Q.All("SELECT penerima_id FROM rencana_penerima WHERE rencana_id = ?", toInt(rid)) {
		if pid := p.I("penerima_id"); pid != 0 {
			ids = append(ids, fmt.Sprint(pid))
		}
	}
	in := ""
	if len(ids) > 0 {
		in = " OR id IN (" + strings.Join(ids, ",") + ")"
	}
	list := []Row{}
	for _, p := range Q.All("SELECT id, nama, nip, npwp, kategori, golongan, jabatan, tarif_default, jenis_penerima, ptkp FROM penerima WHERE aktif = 1" + in + " ORDER BY nama") {
		ket := p.S("kategori")
		if p.S("golongan") != "" {
			ket += " · Gol " + p.S("golongan")
		}
		if p.S("jabatan") != "" {
			ket += " · " + p.S("jabatan")
		}
		p["ket"] = strings.TrimPrefix(ket, " · ")
		p["pph"] = TarifPPh21(p)
		p["jenis_pen"] = jenisPenerimaDari(p)
		list = append(list, p)
	}

	pajakMap, _ := row["pajak"].(map[string]Row)
	chk := func(key, jp string) bool {
		if posted {
			return c.Old.Truthy(key)
		}
		return pajakMap != nil && pajakMap[jp] != nil
	}

	title := "Catat belanja"
	if edit {
		title = "Ubah belanja " + row.S("no_bukti")
	}
	lampiran := []Row{}
	if edit {
		lampiran = Q.All("SELECT * FROM lampiran WHERE belanja_id = ? ORDER BY id DESC", editId)
	}
	saldo := SaldoKas(toStr(c.OldVal("tgl", row["tgl"])), 0, editId)
	danaBOS := true
	viaSiplah := false
	if posted {
		danaBOS = c.Old.Truthy("dana_bos") || !c.Old.Has("dana_bos")
		viaSiplah = c.Old.Truthy("via_siplah")
	} else if edit {
		danaBOS = row.I("dana_bos") == 1
		viaSiplah = row.I("via_siplah") == 1
	}
	c.View("belanja/form", D{
		"title": title, "edit": edit, "row": row,
		"rencana": rencana, "penerimaList": list,
		"toko":     Q.All("SELECT id, nama, pkp, npwp FROM toko WHERE aktif = 1 OR id = ? ORDER BY nama", row.I("toko_id")),
		"lampiran": lampiran,
		"saldo":    saldo,
		"tarif":    D{"PPN": TarifPajak("PPN"), "PPH22": TarifPajak("PPH22"), "PPH23": TarifPajak("PPH23")},
		"batas":    BatasPajakPPN,
		"batasPPh22": BatasPPh22,
		"danaBOS":    danaBOS,
		"viaSiplah":  viaSiplah,
		"items":      items, "pens": pens,
		"jenis": jenis, "periode": periode, "satDefault": satDefault,
		"rencanaVal": rid, "selRencana": sel,
		"tglVal":     toStr(c.OldVal("tgl", row["tgl"])),
		"noBuktiVal": toStr(c.OldVal("no_bukti", row["no_bukti"])),
		"uraian":     toStr(c.OldVal("uraian", row.S("uraian"))),
		"metodeVal":  toStr(c.OldVal("metode", defaultStr(row.S("metode"), "tunai"))),
		"tokoVal":    toStr(c.OldVal("toko_id", row["toko_id"])),
		"chkPPN":     chk("pajak_ppn", "PPN"),
		"chkPPH22":   chk("pajak_pph22", "PPH22"),
		"chkPPH23":   chk("pajak_pph23", "PPH23"),
		"metodeOpts": BelanjaMetode,
	})
}

// penerimaKet: ringkasan kategori/golongan/jabatan untuk opsi penerima.
func penerimaKet(id int64) string {
	if id == 0 {
		return ""
	}
	p := Q.One("SELECT kategori, golongan, jabatan FROM penerima WHERE id = ?", id)
	if p == nil {
		return ""
	}
	ket := p.S("kategori")
	if p.S("golongan") != "" {
		ket += " · Gol " + p.S("golongan")
	}
	if p.S("jabatan") != "" {
		ket += " · " + p.S("jabatan")
	}
	return strings.TrimPrefix(ket, " · ")
}

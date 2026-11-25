package app

import (
	"fmt"
	"strings"
	"time"
)

// BkuJenisLabel: label jenis transaksi di BKU.
var BkuJenisLabel = []Opt{
	{"barang", "Barang / jasa"}, {"honor", "Honor"}, {"transport", "Transport"},
	{"tarik", "Tarik tunai"}, {"setor", "Setor bank"}, {"bunga", "Bunga bank"}, {"admin_bank", "Biaya admin"},
}

func bkuIndex(c *Ctx) {
	tahun := c.TahunAktif()
	kode := c.Q("periode")
	if kode == "" {
		kode = c.Q("bulan")
	}
	p, ok := LapPeriode(kode, tahun)
	if !ok {
		p, _ = LapPeriode("", tahun)
	}
	awal, akhir, sebelum := p.S("awal"), p.S("akhir"), p.S("sebelum")
	bulan := 0
	if p["bulanan"] == true {
		bulan = int(toInt(p.S("kode")))
	}
	namaPeriode := strings.TrimSuffix(p.S("label"), fmt.Sprintf(" %d", tahun))

	rows := Q.All(`SELECT * FROM (
            SELECT 'trm' AS src, id, tgl, no_bukti, uraian, jumlah AS debit, 0 AS kredit, 'bank' AS metode, '' AS jenis, jumlah AS nilai, 0 AS lampiran
              FROM penerimaan WHERE tgl BETWEEN ? AND ?
            UNION ALL
            SELECT 'bel', b.id, b.tgl, b.no_bukti, b.uraian, 0, b.total, b.metode, b.jenis, b.total,
                   (SELECT COUNT(*) FROM lampiran l WHERE l.belanja_id = b.id) AS lampiran
              FROM belanja b WHERE b.tgl BETWEEN ? AND ?
            UNION ALL
            SELECT 'mut', id, tgl, no_bukti, uraian,
                   CASE jenis WHEN 'bunga' THEN jumlah ELSE 0 END,
                   CASE jenis WHEN 'admin_bank' THEN jumlah ELSE 0 END,
                   CASE jenis WHEN 'tarik' THEN 'tunai' ELSE 'bank' END, jenis, jumlah, 0 AS lampiran
              FROM mutasi_kas WHERE tgl BETWEEN ? AND ?
         ) ORDER BY tgl, CASE src WHEN 'trm' THEN 0 WHEN 'mut' THEN 1 ELSE 2 END, id`,
		awal, akhir, awal, akhir, awal, akhir)

	saldoAwal := SaldoKas(sebelum, 0, 0)
	saldo := saldoAwal.Total
	debit, kredit := 0.0, 0.0
	for _, r := range rows {
		saldo += r.F("debit") - r.F("kredit")
		debit += r.F("debit")
		kredit += r.F("kredit")
		r["saldo"] = saldo
		r["jenis_label"] = optText(BkuJenisLabel, r.S("jenis"))
		if r.S("src") == "trm" {
			r["link"] = fmt.Sprintf("/penerimaan/%d/edit", r.I("id"))
		} else if r.S("src") == "bel" {
			r["link"] = fmt.Sprintf("/belanja/%d/edit", r.I("id"))
		} else {
			r["link"] = fmt.Sprintf("/mutasi/%d/edit", r.I("id"))
		}
	}
	perBulan := map[int]int64{}
	for _, r := range Q.All(`SELECT CAST(strftime('%m', tgl) AS INTEGER) AS m, COUNT(*) AS n FROM (
            SELECT tgl FROM penerimaan UNION ALL SELECT tgl FROM belanja UNION ALL SELECT tgl FROM mutasi_kas
         ) WHERE strftime('%Y', tgl) = ? GROUP BY m`, fmt.Sprint(tahun)) {
		perBulan[int(r.I("m"))] = r.I("n")
	}
	c.View("bku/index", D{
		"title": "Buku kas umum " + p.S("label"),
		"tahun": tahun, "bulan": bulan, "periode": p.S("kode"), "rows": rows,
		"bulanNama": namaPeriode, "bulanan": p["bulanan"] == true,
		"nKuitansi": bkuHitungDokumen(rows, false), "nBukti": bkuHitungDokumen(rows, true),
		"saldoAwal": saldoAwal, "saldoAkhir": SaldoKas(akhir, 0, 0),
		"debit": debit, "kredit": kredit, "perBulan": perBulan,
	})
}

// bkuHitungDokumen: jumlah belanja yang punya kuitansi (atau bukti terima bila bukti=true).
func bkuHitungDokumen(rows []Row, bukti bool) int {
	n := 0
	for _, r := range rows {
		if r.S("src") != "bel" {
			continue
		}
		if bukti && r.S("jenis") != "honor" && r.S("jenis") != "transport" {
			continue
		}
		n++
	}
	return n
}

func bkuUpdateTgl(c *Ctx) {
	src := c.Param("src")
	table, ok := NomorBuktiTabel[src]
	if !ok {
		c.Abort(404, "Jenis transaksi BKU tidak dikenal.")
	}
	row := Q.One("SELECT id, tgl FROM "+table+" WHERE id = ?", c.ParamInt("id"))
	if row == nil {
		c.Abort(404, "Transaksi BKU tidak ditemukan.")
	}
	tgl := c.Form.Str("tgl")
	bulanLama := bulanDari(row.S("tgl"))
	if !ValidDate(tgl) {
		c.Flash("danger", "Tanggal transaksi tidak valid.")
		c.Redirect(fmt.Sprintf("/bku?bulan=%d#%s-%d", bulanLama, src, row.I("id")))
	}
	if !c.DalamTahunAktif(tgl) {
		c.Flash("danger", fmt.Sprintf("Tanggal harus berada di tahun anggaran %d.", c.TahunAktif()))
		c.Redirect(fmt.Sprintf("/bku?bulan=%d#%s-%d", bulanLama, src, row.I("id")))
	}
	Q.Tx(func() {
		Q.Run("UPDATE "+table+" SET tgl = ? WHERE id = ?", tgl, row.I("id"))
		if src == "bel" {
			Q.Run("UPDATE pajak SET tgl_pungut = ? WHERE belanja_id = ? AND tgl_pungut = ?", tgl, row.I("id"), row.S("tgl"))
		}
		SusunNomorBuktiBKU(int(toInt(tgl[:4])))
	})
	c.Flash("success", "Tanggal diubah. Urutan nomor bukti dan saldo BKU diperbarui otomatis.")
	c.Redirect(fmt.Sprintf("/bku?bulan=%d#%s-%d", bulanDari(tgl), src, row.I("id")))
}

// bulanDari: komponen bulan dari tanggal YYYY-MM-DD (1–12).
func bulanDari(tgl string) int {
	if len(tgl) < 7 {
		return 1
	}
	m := int(toInt(tgl[5:7]))
	if m < 1 || m > 12 {
		return 1
	}
	return m
}

func daysInMonth(tahun, bulan int) int {
	return time.Date(tahun, time.Month(bulan)+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func addDays(tgl string, n int) string {
	t, err := time.Parse("2006-01-02", tgl)
	if err != nil {
		return tgl
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

// rupiahKosong: rupiah atau string kosong (untuk sel debit/kredit).
func rupiahKosong(v float64) string {
	if v == 0 {
		return ""
	}
	return Rupiah(v)
}

func init() {
	Register(50, func(rt *Router) {
		rt.Get("/bku", bkuIndex)
		rt.Post("/bku/{src}/{id}/tanggal", bkuUpdateTgl)
	})
}

package app

import (
	"sort"
	"strings"
)

// lapPajak: buku pembantu pajak (dipungut & disetor per jenis).
func lapPajak(p Row) *LapData {
	tahunAwal := p.S("awal")[:4] + "-01-01"
	pungutSQL := `SELECT COALESCE(p.tgl_pungut, b.tgl) AS tgl, b.no_bukti, b.uraian, p.jenis, p.nilai, p.ntpn
        FROM pajak p JOIN belanja b ON b.id = p.belanja_id
        WHERE COALESCE(p.tgl_pungut, b.tgl) BETWEEN ? AND ?`
	setorSQL := `SELECT p.tgl_setor AS tgl, b.no_bukti, b.uraian, p.jenis, p.nilai, p.ntpn
        FROM pajak p JOIN belanja b ON b.id = p.belanja_id
        WHERE p.tgl_setor IS NOT NULL AND p.tgl_setor != '' AND p.tgl_setor BETWEEN ? AND ?`
	sumBefore := func(sql string) float64 {
		var t float64
		for _, r := range Q.All(sql, tahunAwal, p.S("sebelum")) {
			t += r.F("nilai")
		}
		return t
	}
	awal := sumBefore(pungutSQL) - sumBefore(setorSQL)

	type ev struct {
		Row
		arah int
	}
	events := []ev{}
	for _, r := range Q.All(pungutSQL, p.S("awal"), p.S("akhir")) {
		events = append(events, ev{r, 0})
	}
	for _, r := range Q.All(setorSQL, p.S("awal"), p.S("akhir")) {
		events = append(events, ev{r, 1})
	}
	sort.SliceStable(events, func(a, b int) bool {
		x, y := events[a], events[b]
		if x.S("tgl") != y.S("tgl") {
			return x.S("tgl") < y.S("tgl")
		}
		if x.arah != y.arah {
			return x.arah < y.arah
		}
		return x.S("no_bukti") < y.S("no_bukti")
	})

	saldo := awal
	rows := []Row{{"_class": "group", "uraian": "Saldo awal (belum disetor)", "saldo": saldo}}
	pungut, setor := 0.0, 0.0
	perJenis := map[string][2]float64{}
	order := []string{}
	for _, e := range events {
		n := e.F("nilai")
		label := lapJenisPajak[e.S("jenis")]
		if label == "" {
			label = e.S("jenis")
		}
		v, ok := perJenis[label]
		if !ok {
			order = append(order, label)
		}
		uraian := ""
		if e.arah == 1 {
			saldo -= n
			setor += n
			v[1] += n
			uraian = "Setor " + label
			if e.S("ntpn") != "" {
				uraian += " NTPN " + e.S("ntpn")
			}
			uraian += " atas " + e.S("uraian")
		} else {
			saldo += n
			pungut += n
			v[0] += n
			uraian = "Pungut " + label + " atas " + e.S("uraian")
		}
		debitNilai := 0.0
		kreditNilai := 0.0
		if e.arah == 0 {
			debitNilai = n
		} else {
			kreditNilai = n
		}
		perJenis[label] = v
		rows = append(rows, Row{
			"tgl": e.Row["tgl"], "no_bukti": e.Row["no_bukti"], "jenis": label, "uraian": uraian,
			"debit": lapOrNil(debitNilai), "kredit": lapOrNil(kreditNilai), "saldo": saldo,
		})
	}
	summary := [][2]any{{"Saldo awal belum disetor", awal}}
	for _, label := range order {
		v := perJenis[label]
		summary = append(summary, [2]any{label + " dipungut / disetor", Rupiah(v[0]) + " / " + Rupiah(v[1])})
	}
	summary = append(summary, [2]any{"Saldo akhir belum disetor", saldo})
	return &LapData{
		Title: "BUKU PEMBANTU PAJAK",
		Cols: []LapCol{
			{"tgl", "Tanggal", "date"}, {"no_bukti", "No. bukti", "text"}, {"jenis", "Jenis", "text"},
			{"uraian", "Uraian", "text"}, {"debit", "Dipungut", "money"}, {"kredit", "Disetor", "money"},
			{"saldo", "Saldo", "money"},
		},
		Rows:    rows,
		Foot:    Row{"uraian": "Jumlah periode ini", "debit": pungut, "kredit": setor, "saldo": saldo},
		Summary: summary, Orientation: "landscape",
	}
}

// lapRekapRow: satu baris rekap komponen/realisasi.
func lapRekapRow(kode, nama string, pagu, periode, kumulatif float64, class string) Row {
	return Row{
		"_class": class, "kode": kode, "nama": nama, "pagu": pagu, "periode": periode,
		"kumulatif": kumulatif, "sisa": pagu - kumulatif, "pct": lapPct(kumulatif, pagu),
	}
}

// lapRealisasiData: pagu, realisasi periode dan kumulatif per baris rencana.
func lapRealisasiData(p Row) []Row {
	tahun := toInt(p.S("awal")[:4])
	return Q.All(`SELECT r.id, r.komponen_kode, k.nama AS komponen_nama, r.kegiatan, r.rekening_kode, r.uraian, r.pagu,
            COALESCE(SUM(CASE WHEN b.tgl BETWEEN ? AND ? THEN b.total END), 0) AS periode,
            COALESCE(SUM(b.total), 0) AS kumulatif
         FROM rencana_belanja r
         LEFT JOIN komponen k ON k.kode = r.komponen_kode
         LEFT JOIN belanja b ON b.rencana_id = r.id AND b.tgl BETWEEN ? AND ?
         WHERE r.tahun = ?
         GROUP BY r.id
         ORDER BY r.komponen_kode, r.id`,
		p.S("awal"), p.S("akhir"), p.S("awal")[:4]+"-01-01", p.S("akhir"), tahun)
}

// lapTanpaRencana: belanja yang tidak terhubung ke rencana.
func lapTanpaRencana(p Row) Row {
	tahun := p.S("awal")[:4]
	r := Q.One(`SELECT COALESCE(SUM(CASE WHEN tgl BETWEEN ? AND ? THEN total END), 0) AS periode,
            COALESCE(SUM(total), 0) AS kumulatif
         FROM belanja WHERE rencana_id IS NULL AND tgl BETWEEN ? AND ?`,
		p.S("awal"), p.S("akhir"), tahun+"-01-01", p.S("akhir"))
	if r == nil {
		return Row{"periode": 0.0, "kumulatif": 0.0}
	}
	return r
}

func lapKomponen(p Row) *LapData {
	type agg struct{ pagu, periode, kumulatif float64 }
	per := map[string]*agg{}
	for _, r := range lapRealisasiData(p) {
		a := per[r.S("komponen_kode")]
		if a == nil {
			a = &agg{}
			per[r.S("komponen_kode")] = a
		}
		a.pagu += r.F("pagu")
		a.periode += r.F("periode")
		a.kumulatif += r.F("kumulatif")
	}
	rows := []Row{}
	totPagu, totPeriode, totKum := 0.0, 0.0, 0.0
	for _, k := range Q.All("SELECT kode, nama FROM komponen ORDER BY urutan, kode") {
		a := per[k.S("kode")]
		if a == nil {
			a = &agg{}
		}
		delete(per, k.S("kode"))
		rows = append(rows, lapRekapRow(k.S("kode"), k.S("nama"), a.pagu, a.periode, a.kumulatif, ""))
		totPagu += a.pagu
		totPeriode += a.periode
		totKum += a.kumulatif
	}
	lain := lapTanpaRencana(p)
	kodeLain := make([]string, 0, len(per))
	for kode := range per {
		kodeLain = append(kodeLain, kode)
	}
	sort.Strings(kodeLain)
	for _, kode := range kodeLain {
		a := per[kode]
		lain["periode"] = lain.F("periode") + a.periode
		lain["kumulatif"] = lain.F("kumulatif") + a.kumulatif
		totPagu += a.pagu
	}
	if lain.F("kumulatif") > 0 {
		rows = append(rows, lapRekapRow("—", "Belanja tanpa rencana / komponen lain", 0, lain.F("periode"), lain.F("kumulatif"), ""))
		totPeriode += lain.F("periode")
		totKum += lain.F("kumulatif")
	}
	return &LapData{
		Title: "REKAPITULASI REALISASI PER KOMPONEN",
		Cols:  lapRekapColumns(LapCol{"kode", "Kode", "text"}, LapCol{"nama", "Komponen", "text"}),
		Rows:  rows,
		Foot:  lapRekapRow("", "Jumlah", totPagu, totPeriode, totKum, ""),
		Summary: [][2]any{
			{"Total pagu", totPagu}, {"Realisasi s.d. periode", totKum}, {"Sisa anggaran", totPagu - totKum},
		},
		Orientation: "landscape",
	}
}

func lapRealisasi(p Row) *LapData {
	rows := []Row{}
	totPagu, totPeriode, totKum := 0.0, 0.0, 0.0
	grup := ""
	subPagu, subPeriode, subKum := 0.0, 0.0, 0.0
	flush := func() {
		if grup != "" {
			rows = append(rows, lapRekapRow("", "Subtotal komponen "+grup, subPagu, subPeriode, subKum, "sub"))
		}
	}
	for _, r := range lapRealisasiData(p) {
		if r.S("komponen_kode") != grup {
			flush()
			grup = r.S("komponen_kode")
			subPagu, subPeriode, subKum = 0, 0, 0
			nama := r.S("komponen_nama")
			if nama == "" {
				nama = "Komponen " + grup
			}
			rows = append(rows, Row{"_class": "group", "kode": grup, "nama": nama})
		}
		uraian := strings.TrimSpace(r.S("uraian"))
		if r.S("kegiatan") != "" {
			uraian += " — " + r.S("kegiatan")
		}
		rows = append(rows, lapRekapRow(r.S("rekening_kode"), uraian, r.F("pagu"), r.F("periode"), r.F("kumulatif"), ""))
		totPagu += r.F("pagu")
		totPeriode += r.F("periode")
		totKum += r.F("kumulatif")
		subPagu += r.F("pagu")
		subPeriode += r.F("periode")
		subKum += r.F("kumulatif")
	}
	flush()
	lain := lapTanpaRencana(p)
	if lain.F("kumulatif") > 0 {
		rows = append(rows, lapRekapRow("—", "Belanja tanpa rencana", 0, lain.F("periode"), lain.F("kumulatif"), ""))
		totPeriode += lain.F("periode")
		totKum += lain.F("kumulatif")
	}
	return &LapData{
		Title: "LAPORAN REALISASI PENGGUNAAN DANA (RKAS)",
		Cols:  lapRekapColumns(LapCol{"kode", "Kode rekening", "text"}, LapCol{"nama", "Uraian", "text"}),
		Rows:  rows,
		Foot:  lapRekapRow("", "Jumlah", totPagu, totPeriode, totKum, ""),
		Summary: [][2]any{
			{"Total pagu", totPagu}, {"Realisasi s.d. periode", totKum}, {"Sisa anggaran", totPagu - totKum},
		},
		Orientation: "landscape",
	}
}

// LapBelanjaPeriode: transaksi belanja periode beserta rincian untuk lampiran SPJ.
func LapBelanjaPeriode(p Row) []Row {
	rows := Q.All(`SELECT b.*, t.nama AS toko_nama, r.rekening_kode, r.uraian AS rencana_uraian
         FROM belanja b LEFT JOIN toko t ON t.id = b.toko_id LEFT JOIN rencana_belanja r ON r.id = b.rencana_id
         WHERE b.tgl BETWEEN ? AND ? ORDER BY b.tgl, b.id`, p.S("awal"), p.S("akhir"))
	for _, b := range rows {
		id := b.I("id")
		b["items"] = Q.All("SELECT * FROM item_belanja WHERE belanja_id = ? ORDER BY id", id)
		b["penerima"] = Q.All("SELECT * FROM penerima_belanja WHERE belanja_id = ? ORDER BY id", id)
		b["pajak"] = Q.All("SELECT * FROM pajak WHERE belanja_id = ? ORDER BY id", id)
		b["jenis_label"] = lapJenisBelanja[b.S("jenis")]
		if b.S("jenis_label") == "" {
			b["jenis_label"] = b.S("jenis")
		}
	}
	return rows
}

// LapJenisPajakLabel: label jenis pajak dari kode.
func LapJenisPajakLabel(kode string) string {
	if l := lapJenisPajak[kode]; l != "" {
		return l
	}
	return kode
}

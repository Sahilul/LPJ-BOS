package app

import (
	"strconv"
)

// PenerimaanTahap: tahap/sumber penerimaan dana (urutan seperti PHP).
var PenerimaanTahap = []Opt{{"Tahap 1", "Tahap 1"}, {"Tahap 2", "Tahap 2"}, {"Sisa tahun lalu", "Sisa tahun lalu (SiLPA)"}, {"Lainnya", "Lainnya"}}

func PenerimaanFind(id int64) Row {
	return Q.One("SELECT * FROM penerimaan WHERE id = ?", id)
}

// keuanganCekTgl: pesan galat tanggal transaksi BKU ("" bila valid).
func keuanganCekTgl(tgl string, tahun int) string {
	if !ValidDate(tgl) {
		return "Isi tanggal yang valid."
	}
	if int(toInt(tgl[:4])) != tahun {
		t := strconv.Itoa(tahun)
		return "Tanggal harus di tahun anggaran " + t + ". Ganti tahun dari menu bila perlu."
	}
	return ""
}

func PenerimaanValidate(p Form, id int64, tahun int) (Row, error) {
	planID := p.Int("rencana_pendapatan_id")
	var plan Row
	if planID != 0 {
		plan = Q.One("SELECT * FROM rencana_pendapatan WHERE id = ?", planID)
	}
	data := Row{
		"tgl":                   p.Str("tgl"),
		"no_bukti":              p.Trim("no_bukti"),
		"uraian":                p.Trim("uraian"),
		"tahap":                 p.Str("tahap"),
		"jumlah":                Uang(p.Str("jumlah")),
		"rencana_pendapatan_id": nil,
	}
	if plan != nil {
		data["rencana_pendapatan_id"] = plan.I("id")
	}
	err := NewValidationOrdered()
	if m := keuanganCekTgl(data.S("tgl"), tahun); m != "" {
		err.Set("tgl", m)
	}
	if data.S("uraian") == "" {
		err.Set("uraian", "Uraian wajib diisi.")
	}
	if _, ok := optLabel(PenerimaanTahap, data.S("tahap")); !ok {
		err.Set("tahap", "Pilih tahap.")
	}
	if planID != 0 && plan == nil {
		err.Set("rencana_pendapatan_id", "Pilih rencana pendapatan yang valid.")
	} else if plan != nil && plan.I("tahun") != int64(tahun) {
		err.Set("rencana_pendapatan_id", "Rencana pendapatan bukan milik tahun anggaran yang sedang dibuka.")
	} else if plan != nil && data.S("tahap") != plan.S("tahap") {
		err.Set("rencana_pendapatan_id", "Sumber/tahap penerimaan harus sama dengan rencana yang dipilih.")
	}
	if data.F("jumlah") <= 0 {
		err.Set("jumlah", "Jumlah harus lebih dari 0.")
	}
	if len(err.Errors) > 0 {
		return nil, err
	}
	return data, nil
}

func PenerimaanCreate(data Row) int64 {
	var id int64
	Q.Tx(func() {
		id = Q.Insert("penerimaan", data)
		SusunNomorBuktiBKU(int(toInt(data.S("tgl")[:4])))
	})
	return id
}

func PenerimaanUpdate(id int64, data Row) {
	Q.Tx(func() {
		Q.Update("penerimaan", data, id)
		SusunNomorBuktiBKU(int(toInt(data.S("tgl")[:4])))
	})
}

func PenerimaanDelete(row Row) {
	Q.Tx(func() {
		Q.Run("DELETE FROM penerimaan WHERE id = ?", row.I("id"))
		SusunNomorBuktiBKU(int(toInt(keuanganTahunDari(row.S("tgl")))))
	})
}

// keuanganTahunDari: substr(tgl, 0, 4) yang aman untuk string pendek.
func keuanganTahunDari(tgl string) string {
	if len(tgl) < 4 {
		return tgl
	}
	return tgl[:4]
}

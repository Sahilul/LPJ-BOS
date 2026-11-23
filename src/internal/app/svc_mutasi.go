package app

var MutasiJenis = []Opt{
	{"tarik", "Tarik tunai (bank → kas)"},
	{"setor", "Setor ke bank (kas → bank)"},
	{"admin_bank", "Biaya administrasi bank"},
	{"bunga", "Bunga / jasa giro"},
}

var MutasiUraian = []Opt{
	{"tarik", "Pengambilan uang tunai dari rekening BOSP"},
	{"setor", "Penyetoran sisa kas tunai ke rekening BOSP"},
	{"admin_bank", "Biaya administrasi bank"},
	{"bunga", "Penerimaan jasa giro"},
}

func MutasiFind(id int64) Row {
	return Q.One("SELECT * FROM mutasi_kas WHERE id = ?", id)
}

func MutasiValidate(p Form, id int64, tahun int) (Row, error) {
	data := Row{
		"tgl":      p.Str("tgl"),
		"no_bukti": p.Trim("no_bukti"),
		"jenis":    p.Str("jenis"),
		"uraian":   p.Trim("uraian"),
		"jumlah":   Uang(p.Str("jumlah")),
	}
	err := NewValidationOrdered()
	if m := keuanganCekTgl(data.S("tgl"), tahun); m != "" {
		err.Set("tgl", m)
	}
	if _, ok := optLabel(MutasiJenis, data.S("jenis")); !ok {
		err.Set("jenis", "Pilih jenis mutasi.")
	}
	if data.F("jumlah") <= 0 {
		err.Set("jumlah", "Jumlah harus lebih dari 0.")
	}
	if u, ok := optLabel(MutasiUraian, data.S("jenis")); ok && data.S("uraian") == "" {
		data["uraian"] = u
	}
	if len(err.Errors) > 0 {
		return nil, err
	}
	return data, nil
}

func MutasiCreate(data Row) int64 {
	var id int64
	Q.Tx(func() {
		id = Q.Insert("mutasi_kas", data)
		SusunNomorBuktiBKU(int(toInt(data.S("tgl")[:4])))
	})
	return id
}

func MutasiUpdate(id int64, data Row) {
	Q.Tx(func() {
		Q.Update("mutasi_kas", data, id)
		SusunNomorBuktiBKU(int(toInt(data.S("tgl")[:4])))
	})
}

func MutasiDelete(row Row) {
	Q.Tx(func() {
		Q.Run("DELETE FROM mutasi_kas WHERE id = ?", row.I("id"))
		SusunNomorBuktiBKU(int(toInt(keuanganTahunDari(row.S("tgl")))))
	})
}

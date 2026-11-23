package app

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
)

// RencanaJenis: jenis paket rencana belanja (urutan seperti PHP).
var RencanaJenis = []Opt{{"barang", "Barang / jasa"}, {"honor", "Honor"}, {"transport", "Transport"}}

func optLabel(opts []Opt, k string) (string, bool) {
	for _, o := range opts {
		if o.K == k {
			return o.V, true
		}
	}
	return "", false
}

func RencanaFind(id int64) Row {
	return Q.One("SELECT * FROM rencana_belanja WHERE id = ?", id)
}

// RencanaDetail: paket beserta barang/penerima dan sisa pagu & periode.
func RencanaDetail(id int64) Row {
	row := RencanaFind(id)
	if row == nil {
		return nil
	}
	row["items"] = Q.All("SELECT nama, volume, satuan, harga_satuan FROM rencana_item WHERE rencana_id = ? ORDER BY id", id)
	row["penerima"] = Q.All(`SELECT rp.penerima_id, p.nama, rp.volume, rp.satuan, rp.tarif
             FROM rencana_penerima rp LEFT JOIN penerima p ON p.id = rp.penerima_id
             WHERE rp.rencana_id = ? ORDER BY rp.id`, id)
	terpakai := Q.Float("SELECT COALESCE(SUM(total), 0) FROM belanja WHERE rencana_id = ?", id)
	periode := Q.Int("SELECT COALESCE(SUM(jumlah_periode), 0) FROM belanja WHERE rencana_id = ?", id)
	row["terpakai"] = terpakai
	row["periode_terpakai"] = periode
	row["sisa"] = row.F("pagu") - terpakai
	row["periode_sisa"] = row.I("frekuensi_bulan") - periode
	return row
}

// RencanaValidate mengembalikan data paket, baris barang, dan baris penerima.
func RencanaValidate(p Form, id int64, tahun int) (Row, []Row, []Row, error) {
	frekuensi := p.Int("frekuensi_bulan")
	jenis := "barang"
	if p.Has("jenis") {
		jenis = p.Str("jenis")
	}
	seenBulan := map[int64]bool{}
	bulan := []int64{}
	for _, s := range p.Strings("jadwal") {
		m := toInt(strings.TrimSpace(s))
		if m >= 1 && m <= 12 && !seenBulan[m] {
			seenBulan[m] = true
			bulan = append(bulan, m)
		}
	}
	sort.Slice(bulan, func(a, b int) bool { return bulan[a] < bulan[b] })
	jadwal := make([]string, len(bulan))
	for i, m := range bulan {
		jadwal[i] = strconv.FormatInt(m, 10)
	}
	tokoID := p.Int("toko_id")
	var toko Row
	if tokoID != 0 {
		toko = Q.One("SELECT id FROM toko WHERE id = ?", tokoID)
	}
	var dataTahun int64 = int64(tahun)
	if id != 0 {
		dataTahun = Q.Int("SELECT tahun FROM rencana_belanja WHERE id = ?", id)
	}
	data := Row{
		"tahun":           dataTahun,
		"komponen_kode":   p.Trim("komponen_kode"),
		"kegiatan":        p.Trim("kegiatan"),
		"rekening_kode":   p.Trim("rekening_kode"),
		"uraian":          p.Trim("uraian"),
		"jenis":           jenis,
		"frekuensi_bulan": frekuensi,
		"toko_id":         nil,
		"volume":          frekuensi,
		"satuan":          "bulan",
		"harga_satuan":    0.0,
		"pagu":            0.0,
		"jadwal_bulan":    strings.Join(jadwal, ","),
	}
	if jenis == "barang" && toko != nil {
		data["toko_id"] = toko.I("id")
	}
	items := []Row{}
	recipients := []Row{}
	perPeriode := 0.0
	err := NewValidationOrdered()
	err.Msg = "Paket rencana belum valid."

	if dataTahun < 2000 || dataTahun > 2100 {
		err.Set("tahun", "Tahun tidak valid.")
	}
	if Q.Int("SELECT COUNT(*) FROM komponen WHERE kode = ?", data.S("komponen_kode")) == 0 {
		err.Set("komponen_kode", "Pilih komponen BOSP.")
	}
	if data.S("uraian") == "" {
		err.Set("uraian", "Nama paket wajib diisi, misalnya “Belanja ATK”.")
	}
	_, jenisOK := optLabel(RencanaJenis, jenis)
	if !jenisOK {
		err.Set("jenis", "Pilih jenis paket.")
	}
	if frekuensi < 1 || frekuensi > 12 {
		err.Set("frekuensi_bulan", "Frekuensi harus antara 1 sampai 12 bulan.")
	}
	if jenis == "barang" {
		for _, it := range p.List("items") {
			nama := it.Trim("nama")
			volume := Desimal(it.Str("volume"))
			satuan := it.Trim("satuan")
			harga := Uang(it.Str("harga"))
			if nama == "" && harga <= 0 {
				continue
			}
			if nama == "" || volume <= 0 || harga <= 0 {
				err.Set("items", "Setiap barang perlu nama, jumlah per bulan, dan harga satuan.")
				continue
			}
			items = append(items, Row{"nama": nama, "volume": volume, "satuan": satuan, "harga_satuan": harga})
			perPeriode += math.Round(volume * harga)
		}
		if _, ada := err.Errors["items"]; len(items) == 0 && !ada {
			err.Set("items", "Tambahkan minimal satu barang atau jasa ke paket.")
		}
		if tokoID != 0 && toko == nil {
			err.Set("toko_id", "Pilih toko/penyedia yang valid.")
		}
	} else if jenisOK {
		seen := map[int64]bool{}
		for _, person := range p.List("penerima") {
			pid := person.Int("penerima_id")
			volume := Desimal(person.Str("volume"))
			satuan := person.Trim("satuan")
			tarif := Uang(person.Str("tarif"))
			if pid == 0 && tarif <= 0 {
				continue
			}
			var master Row
			if pid != 0 {
				master = Q.One("SELECT id FROM penerima WHERE id = ?", pid)
			}
			if master == nil || volume <= 0 || tarif <= 0 {
				err.Set("penerima", "Setiap baris perlu penerima, volume, dan tarif.")
				continue
			}
			if seen[pid] {
				err.Set("penerima", "Penerima yang sama tidak boleh ditambahkan dua kali.")
				continue
			}
			seen[pid] = true
			recipients = append(recipients, Row{"penerima_id": pid, "volume": volume, "satuan": satuan, "tarif": tarif})
			perPeriode += math.Round(volume * tarif)
		}
		if _, ada := err.Errors["penerima"]; len(recipients) == 0 && !ada {
			err.Set("penerima", "Tambahkan minimal satu penerima ke paket.")
		}
	}

	data["harga_satuan"] = math.Round(perPeriode)
	data["pagu"] = math.Round(perPeriode * float64(frekuensi))
	if perPeriode <= 0 {
		key := "penerima"
		if jenis == "barang" {
			key = "items"
		}
		if _, ada := err.Errors[key]; !ada {
			err.Set(key, "Nilai paket harus lebih dari nol.")
		}
	}
	if id != 0 && len(err.Errors) == 0 {
		terpakai := Q.Float("SELECT COALESCE(SUM(total), 0) FROM belanja WHERE rencana_id = ?", id)
		if data.F("pagu")+0.001 < terpakai {
			err.Set("frekuensi_bulan", "Pagu baru "+Rupiah(data["pagu"])+" lebih kecil dari yang sudah dibelanjakan ("+Rupiah(terpakai)+").")
		}
		periodeTerpakai := Q.Int("SELECT COALESCE(SUM(jumlah_periode), 0) FROM belanja WHERE rencana_id = ?", id)
		if frekuensi < periodeTerpakai {
			err.Set("frekuensi_bulan", "Frekuensi tidak boleh lebih kecil dari periode yang sudah dicatat ("+strconv.FormatInt(periodeTerpakai, 10)+" bulan).")
		}
	}
	if len(err.Errors) > 0 {
		return nil, nil, nil, err
	}
	return data, items, recipients, nil
}

func RencanaCreate(data Row, items, recipients []Row) int64 {
	var id int64
	Q.Tx(func() {
		id = Q.Insert("rencana_belanja", data)
		rencanaSavePackage(id, items, recipients)
	})
	return id
}

func RencanaUpdate(id int64, data Row, items, recipients []Row) {
	Q.Tx(func() {
		Q.Update("rencana_belanja", data, id)
		Q.Run("DELETE FROM rencana_item WHERE rencana_id = ?", id)
		Q.Run("DELETE FROM rencana_penerima WHERE rencana_id = ?", id)
		rencanaSavePackage(id, items, recipients)
	})
}

// RencanaDelete gagal bila paket sudah dipakai di BKU.
func RencanaDelete(id int64) error {
	if Q.Int("SELECT COUNT(*) FROM belanja WHERE rencana_id = ?", id) > 0 {
		return errors.New("Rencana ini sudah dipakai di BKU. Hapus transaksinya dulu bila memang ingin menghapus.")
	}
	Q.Run("DELETE FROM rencana_belanja WHERE id = ?", id)
	return nil
}

func rencanaSavePackage(id int64, items, recipients []Row) {
	for _, it := range items {
		r := Row{"rencana_id": id}
		for k, v := range it {
			r[k] = v
		}
		Q.Insert("rencana_item", r)
	}
	for _, rc := range recipients {
		r := Row{"rencana_id": id}
		for k, v := range rc {
			r[k] = v
		}
		Q.Insert("rencana_penerima", r)
	}
}

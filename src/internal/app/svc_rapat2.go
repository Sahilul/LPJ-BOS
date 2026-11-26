package app

import (
	"fmt"
	"strings"
)

// RapatValidate: validasi isian undangan & notulen rapat.
// p: belanja_id, no_undangan, tgl_surat, sifat, lampiran, perihal, kepada,
// tgl, jam_mulai, jam_selesai, tempat, agenda, pimpinan, notulis,
// jumlah_peserta, pembukaan, pembahasan, keputusan, tindak_lanjut.
func RapatValidate(p Form, id int64, tahun int) (Row, error) {
	data := Row{
		"belanja_id":     p.Int("belanja_id"),
		"no_undangan":    p.Trim("no_undangan"),
		"tgl_surat":      p.Str("tgl_surat"),
		"perihal":        p.Trim("perihal"),
		"kepada":         p.Trim("kepada"),
		"tgl":            p.Str("tgl"),
		"jam_mulai":      p.Trim("jam_mulai"),
		"jam_selesai":    p.Trim("jam_selesai"),
		"tempat":         p.Trim("tempat"),
		"agenda":         p.Trim("agenda"),
		"pimpinan":       p.Trim("pimpinan"),
		"notulis":        p.Trim("notulis"),
		"jumlah_peserta": p.Int("jumlah_peserta"),
		"pembukaan":      p.Trim("pembukaan"),
		"pembahasan":     p.Trim("pembahasan"),
		"keputusan":      p.Trim("keputusan"),
		"tindak_lanjut":  p.Trim("tindak_lanjut"),
		"sifat":          defStr(p.Str("sifat"), "Penting"),
		"lampiran":       defStr(p.Trim("lampiran"), "-"),
		"tahun":          tahun,
	}
	err := NewValidationOrdered()

	var belanja Row
	if data.I("belanja_id") != 0 {
		belanja = RapatBelanjaAda(data.I("belanja_id"))
	}
	if belanja == nil {
		err.Set("belanja_id", "Pilih belanja BKU yang menjadi dasar rapat.")
	} else {
		if bt := rapatTahunBelanja(belanja); bt != 0 && bt != tahun {
			err.Set("belanja_id", fmt.Sprintf("Belanja yang dipilih bukan tahun anggaran %d.", tahun))
		}
		if lain := Q.Int("SELECT id FROM rapat WHERE belanja_id = ? AND id != ?", data.I("belanja_id"), id); lain != 0 {
			err.Set("belanja_id", "Belanja ini sudah punya undangan & notulen rapat.")
		}
	}
	if _, ok := optLabel(RapatSifat, data.S("sifat")); !ok {
		data["sifat"] = "Penting"
	}
	if !ValidDate(data.S("tgl_surat")) {
		err.Set("tgl_surat", "Isi tanggal surat undangan yang valid.")
	} else if m := keuanganCekTgl(data.S("tgl_surat"), tahun); m != "" {
		err.Set("tgl_surat", strings.Replace(m, "Tanggal", "Tanggal surat", 1))
	}
	if !ValidDate(data.S("tgl")) {
		err.Set("tgl", "Isi tanggal pelaksanaan rapat yang valid.")
	} else if m := keuanganCekTgl(data.S("tgl"), tahun); m != "" {
		err.Set("tgl", m)
	}
	if ValidDate(data.S("tgl_surat")) && ValidDate(data.S("tgl")) && data.S("tgl") < data.S("tgl_surat") {
		err.Set("tgl", "Tanggal rapat tidak boleh sebelum tanggal surat undangan.")
	}
	for _, k := range []string{"jam_mulai", "jam_selesai"} {
		if v := data.S(k); v != "" && !validJam(v) {
			err.Set(k, "Tulis jam dengan format HH:MM, mis. 09:00.")
		}
	}
	if a, b := data.S("jam_mulai"), data.S("jam_selesai"); a != "" && b != "" && b < a {
		err.Set("jam_selesai", "Jam selesai lebih awal dari jam mulai.")
	}
	if data.S("tempat") == "" {
		err.Set("tempat", "Isi tempat rapat.")
	}
	if data.S("agenda") == "" {
		err.Set("agenda", "Isi agenda rapat.")
	}
	if data.I("jumlah_peserta") < 0 {
		err.Set("jumlah_peserta", "Jumlah peserta tidak boleh negatif.")
	}
	if data.S("no_undangan") == "" {
		data["no_undangan"] = RapatNomorBerikut(tahun, data.S("tgl_surat"))
	}
	if dup := Q.Int("SELECT id FROM rapat WHERE tahun = ? AND no_undangan = ? AND id != ?",
		tahun, data.S("no_undangan"), id); dup != 0 {
		err.Set("no_undangan", "Nomor undangan ini sudah dipakai rapat lain.")
	}
	if len(err.Errors) > 0 {
		return nil, err
	}
	return data, nil
}

// RapatBelanjaAda: belanja beserta tahun anggaran rencananya.
func RapatBelanjaAda(id int64) Row {
	return Q.One(`SELECT b.id, b.tgl, b.uraian, b.total, r.tahun
        FROM belanja b JOIN rencana_belanja r ON r.id = b.rencana_id
        WHERE b.id = ?`, id)
}

// validJam: HH:MM 00:00–23:59.
func validJam(s string) bool {
	if len(s) != 5 || s[2] != ':' {
		return false
	}
	h, m := toInt(s[:2]), toInt(s[3:])
	return h >= 0 && h <= 23 && m >= 0 && m <= 59 && s[0] >= '0' && s[0] <= '9' && s[3] >= '0' && s[3] <= '9'
}

func RapatCreate(data Row) int64 {
	return Q.Insert("rapat", data)
}

func RapatUpdate(id int64, data Row) {
	Q.Update("rapat", data, id)
}

func RapatDelete(row Row) {
	Q.Run("DELETE FROM rapat WHERE id = ?", row.I("id"))
}

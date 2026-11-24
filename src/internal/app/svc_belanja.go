package app

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var BelanjaMetode = []Opt{{"tunai", "Tunai"}, {"bank", "Transfer / non tunai"}}

// BelanjaAnak: baris anak belanja (item, penerima, pajak).
type BelanjaAnak struct {
	Items    []Row
	Penerima []Row
	Pajak    []Row
}

func BelanjaFind(id int64) Row {
	return Q.One("SELECT * FROM belanja WHERE id = ?", id)
}

// BelanjaDetail: belanja beserta item, penerima, pajak, dan lampiran.
func BelanjaDetail(id int64) Row {
	row := BelanjaFind(id)
	if row == nil {
		return nil
	}
	row["items"] = Q.All("SELECT nama, volume, satuan, harga, subtotal FROM item_belanja WHERE belanja_id = ? ORDER BY id", id)
	row["penerima"] = Q.All("SELECT penerima_id, nama, kategori, golongan, volume, satuan, tarif, bruto, pph21, netto FROM penerima_belanja WHERE belanja_id = ? ORDER BY id", id)
	row["pajak"] = Q.All("SELECT jenis, dpp, tarif, nilai, tgl_pungut, tgl_setor, ntpn FROM pajak WHERE belanja_id = ? ORDER BY id", id)
	row["lampiran"] = Q.All("SELECT jenis, nama_asli FROM lampiran WHERE belanja_id = ? ORDER BY id", id)
	return row
}

// BelanjaValidate. Input: rencana_id, tgl, no_bukti, jumlah_periode, uraian, metode, toko_id,
// items[]{nama,volume,satuan,harga}, pajak_ppn/pajak_pph22/pajak_pph23, pen[]{penerima_id,volume,satuan,tarif}.
func BelanjaValidate(p Form, id int64, tahun int) (Row, BelanjaAnak, error) {
	rencana := Q.One("SELECT * FROM rencana_belanja WHERE id = ?", p.Int("rencana_id"))
	metode := p.Str("metode")
	if _, ok := optLabel(BelanjaMetode, metode); !ok {
		metode = "tunai"
	}
	data := Row{
		"tgl":            p.Str("tgl"),
		"no_bukti":       p.Trim("no_bukti"),
		"rencana_id":     int64(0),
		"jumlah_periode": p.Int("jumlah_periode"),
		"jenis":          "barang",
		"uraian":         p.Trim("uraian"),
		"metode":         metode,
		"toko_id":        nil,
		"total":          0.0,
		// Dana BOS adalah sumber dana bawaan aplikasi ini; PPh 22 atas
		// pembelian barang dengan dana BOS dikecualikan.
		"dana_bos":   0,
		"via_siplah": 0,
	}
	// Checkbox yang tidak dicentang tidak dikirim browser, jadi ketiadaan kunci
	// berarti tidak dicentang. Form belanja selalu mengirim dana_bos=1 secara bawaan.
	if p.Truthy("dana_bos") {
		data["dana_bos"] = 1
	}
	if p.Truthy("via_siplah") {
		data["via_siplah"] = 1
	}
	if rencana != nil {
		data["rencana_id"] = rencana.I("id")
		if !rencana.Null("jenis") {
			data["jenis"] = rencana.S("jenis")
		}
	}
	anak := BelanjaAnak{Items: []Row{}, Penerima: []Row{}, Pajak: []Row{}}
	err := NewValidationOrdered()
	err.Msg = "Belanja belum disimpan. Periksa isian yang ditandai."
	total := 0.0

	if m := keuanganCekTgl(data.S("tgl"), tahun); m != "" {
		err.Set("tgl", m)
	}
	if rencana == nil {
		err.Set("rencana_id", "Pilih baris rencana belanja.")
	} else if rencana.I("tahun") != int64(tahun) {
		err.Set("rencana_id", "Rencana ini milik tahun "+rencana.S("tahun")+", bukan tahun anggaran yang sedang dibuka.")
	}
	jp := data.I("jumlah_periode")
	if jp < 1 || jp > 12 {
		err.Set("jumlah_periode", "Jumlah periode harus antara 1 sampai 12 bulan.")
	} else if rencana != nil {
		periodeSisa := Q.Int(`SELECT frekuensi_bulan - COALESCE((SELECT SUM(jumlah_periode) FROM belanja WHERE rencana_id = ? AND id != ?), 0)
                 FROM rencana_belanja WHERE id = ?`, rencana.I("id"), id, rencana.I("id"))
		if jp > periodeSisa {
			err.Set("jumlah_periode", "Paket ini hanya memiliki "+strconv.FormatInt(max(0, periodeSisa), 10)+" bulan yang belum dicatat.")
		}
	}
	if data.S("uraian") == "" {
		err.Set("uraian", "Uraian wajib diisi.")
	}
	if data.S("no_bukti") != "" && Q.Int("SELECT COUNT(*) FROM belanja WHERE no_bukti = ? AND id != ?", data.S("no_bukti"), id) > 0 {
		err.Set("no_bukti", "Nomor bukti ini sudah dipakai.")
	}

	if data.S("jenis") == "barang" {
		tokoID := p.Int("toko_id")
		if tokoID != 0 {
			if toko := Q.One("SELECT * FROM toko WHERE id = ?", tokoID); toko != nil {
				data["toko_id"] = toko.I("id")
			}
		}
		for _, it := range p.List("items") {
			nama := it.Trim("nama")
			vol := Desimal(it.Str("volume"))
			harga := Uang(it.Str("harga"))
			if nama == "" && harga <= 0 {
				continue
			}
			if nama == "" || vol <= 0 || harga <= 0 {
				err.Set("items", "Setiap barang perlu nama, volume dan harga.")
				continue
			}
			sub := math.Round(vol * harga)
			anak.Items = append(anak.Items, Row{"nama": nama, "volume": vol, "satuan": it.Trim("satuan"), "harga": harga, "subtotal": sub})
			total += sub
		}
		if _, ada := err.Errors["items"]; len(anak.Items) == 0 && !ada {
			err.Set("items", "Tambahkan minimal satu barang atau jasa.")
		}
		// Nilai nota sudah termasuk PPN, jadi DPP dihitung 11/111 bagian
		// (PMK 59/2022 Lampiran VI.B.1).
		ppn := p.Truthy("pajak_ppn")
		pph22 := p.Truthy("pajak_pph22")
		if !PPNPPh22Dipungut(Sekolah().S("status")) {
			if ppn {
				err.Set("pajak_ppn", "Sekolah swasta: bendahara bukan pemungut PPN (PMK 59/2022 Pasal 2).")
				ppn = false
			}
			if pph22 {
				err.Set("pajak_pph22", "Sekolah swasta: bendahara bukan pemungut PPh 22 (PMK 59/2022 Pasal 2).")
				pph22 = false
			}
		}
		dpp := total
		if ppn {
			dpp = DPPDariBruto(total)
		}
		if ppn && total > 0 && !PPNPerluDipungut(total) {
			err.Set("pajak_ppn", "PPN tidak dipungut: nilai di luar PPN paling banyak "+Rupiah(BatasPajakPPN)+".")
		}
		punyaNPWP := tokoNPWPAda(data["toko_id"])
		for _, pj := range []struct {
			jenis string
			on    bool
		}{{"PPN", ppn}, {"PPH22", pph22}, {"PPH23", p.Truthy("pajak_pph23")}} {
			if !pj.on || total <= 0 {
				continue
			}
			t := TarifPajak(pj.jenis)
			switch pj.jenis {
			case "PPH22":
				// Dana BOS dikecualikan dari PPh 22 (PMK 59/2022 Pasal 12 ayat (2) huruf d).
				if !PPh22PerluDipungut(dpp, data.I("dana_bos") == 1, data.I("via_siplah") == 1) {
					err.Set("pajak_pph22", "PPh 22 tidak dipungut untuk transaksi ini: pembelian barang dana BOS dikecualikan (PMK 59/2022 Pasal 12 ayat (2) huruf d).")
					continue
				}
				t = TarifPPh22(punyaNPWP)
			case "PPH23":
				t = TarifPPh23(punyaNPWP)
			}
			anak.Pajak = append(anak.Pajak, Row{"jenis": pj.jenis, "dpp": dpp, "tarif": t, "nilai": math.Round(dpp * t / 100), "tgl_pungut": data.S("tgl")})
		}
	} else {
		pphDpp, pphNilai := 0.0, 0.0
		seen := map[int64]bool{}
		for _, r := range p.List("pen") {
			pid := r.Int("penerima_id")
			if pid == 0 {
				continue
			}
			m := Q.One("SELECT * FROM penerima WHERE id = ?", pid)
			vol := Desimal(r.Str("volume"))
			tarif := Uang(r.Str("tarif"))
			if m == nil || vol <= 0 || tarif <= 0 {
				err.Set("pen", "Setiap penerima perlu volume dan tarif.")
				continue
			}
			if seen[pid] {
				err.Set("pen", m.S("nama")+" tercantum dua kali.")
				continue
			}
			seen[pid] = true
			bruto := math.Round(vol * tarif)
			// PPh 21 memakai rezim PMK 168/2023: tarif final golongan hanya untuk PNS,
			// bukan pegawai 50% x Pasal 17, pegawai tidak tetap memakai TER bulanan
			// (hari kosong) atau TER harian (hari diisi).
			hari := Desimal(r.Str("hari"))
			pajak := 0.0
			if data.S("jenis") == "honor" {
				brutoLain, pphLain := honorBulananLain(pid, data.S("tgl"), id)
				pajak = PPh21UntukPenerimaBulan(m, bruto, hari, brutoLain, pphLain)
			}
			anak.Penerima = append(anak.Penerima, Row{
				"penerima_id": pid, "nama": m["nama"], "nip": m["nip"], "npwp": m["npwp"],
				"kategori": m["kategori"], "golongan": m["golongan"], "jabatan": m["jabatan"],
				"jenis_penerima": jenisPenerimaDari(m), "ptkp": defaultStr(m.S("ptkp"), "A"),
				"hari": toInt(hari),
				"bank": m["bank"], "no_rekening": m["no_rekening"],
				"volume": vol, "satuan": r.Trim("satuan"), "tarif": tarif,
				"bruto": bruto, "pph21": pajak, "netto": bruto - pajak,
			})
			total += bruto
			if pajak > 0 {
				pphDpp += bruto
				pphNilai += pajak
			}
		}
		if _, ada := err.Errors["pen"]; len(anak.Penerima) == 0 && !ada {
			err.Set("pen", "Tambahkan minimal satu penerima.")
		}
		if pphNilai > 0 {
			anak.Pajak = append(anak.Pajak, Row{"jenis": "PPH21", "dpp": pphDpp, "tarif": math.Round(pphNilai/pphDpp*100*100) / 100, "nilai": pphNilai, "tgl_pungut": data.S("tgl")})
		}
	}
	data["total"] = total

	if rencana != nil && len(err.Errors) == 0 {
		sisa := SisaRencana(rencana.I("id"), id)
		if total > sisa+0.001 {
			err.Set("rencana_id", "Melebihi sisa anggaran "+Rupiah(sisa)+" (transaksi "+Rupiah(total)+").")
		}
	}
	// Saldo boleh minus: belanja yang mendahului pencairan dana tetap dicatat.
	if len(err.Errors) > 0 {
		return nil, anak, err
	}
	return data, anak, nil
}

func BelanjaCreate(data Row, anak BelanjaAnak) int64 {
	var id int64
	Q.Tx(func() {
		if data.S("no_bukti") == "" {
			data["no_bukti"] = NoBuktiBerikut(data.S("tgl"), "bel")
		}
		id = Q.Insert("belanja", data)
		belanjaSaveChildren(id, anak)
		SusunNomorBuktiBKU(int(toInt(data.S("tgl")[:4])))
	})
	return id
}

func BelanjaUpdate(id int64, data Row, anak BelanjaAnak) {
	Q.Tx(func() {
		d := Row{}
		for k, v := range data {
			d[k] = v
		}
		if d.S("no_bukti") == "" {
			delete(d, "no_bukti")
		}
		Q.Update("belanja", d, id)
		for _, t := range []string{"item_belanja", "penerima_belanja", "pajak"} {
			Q.Run("DELETE FROM "+t+" WHERE belanja_id = ?", id)
		}
		belanjaSaveChildren(id, anak)
		SusunNomorBuktiBKU(int(toInt(data.S("tgl")[:4])))
	})
}

// BelanjaDelete menghapus transaksi beserta berkas lampirannya.
func BelanjaDelete(row Row) error {
	id := row.I("id")
	files := Q.All("SELECT path FROM lampiran WHERE belanja_id = ?", id)
	Q.Tx(func() {
		Q.Run("DELETE FROM belanja WHERE id = ?", id)
		SusunNomorBuktiBKU(int(toInt(keuanganTahunDari(row.S("tgl")))))
	})
	for _, f := range files {
		path := UploadPath(f.S("path"))
		if st, e := os.Stat(path); e == nil && st.Mode().IsRegular() {
			if os.Remove(path) != nil {
				return errors.New("Transaksi dihapus, tetapi lampiran " + filepath.Base(path) + " tidak dapat dibersihkan.")
			}
		}
	}
	return nil
}

// honorBulananLain menjumlah bruto dan PPh 21 honor bulanan (tanpa hari) seorang
// penerima pada bulan yang sama dari transaksi lain, untuk menentukan TER bulanan.
func honorBulananLain(penerimaID int64, tgl string, kecuali int64) (bruto, pph float64) {
	if len(tgl) < 7 {
		return 0, 0
	}
	r := Q.One(`SELECT COALESCE(SUM(pb.bruto), 0) AS bruto, COALESCE(SUM(pb.pph21), 0) AS pph
		FROM penerima_belanja pb JOIN belanja b ON b.id = pb.belanja_id
		WHERE pb.penerima_id = ? AND b.jenis = 'honor' AND pb.hari = 0
		  AND substr(b.tgl, 1, 7) = ? AND b.id != ?`, penerimaID, tgl[:7], kecuali)
	if r == nil {
		return 0, 0
	}
	return r.F("bruto"), r.F("pph")
}

func belanjaSaveChildren(id int64, a BelanjaAnak) {
	simpan := func(tabel string, rows []Row) {
		for _, r := range rows {
			x := Row{"belanja_id": id}
			for k, v := range r {
				x[k] = v
			}
			Q.Insert(tabel, x)
		}
	}
	simpan("item_belanja", a.Items)
	simpan("penerima_belanja", a.Penerima)
	simpan("pajak", a.Pajak)
}

// penandaHitungUlangPPh21 dicatat di tabel migrations agar penghitungan ulang
// hanya berjalan sekali per database.
const penandaHitungUlangPPh21 = "011_pph21_ter_bulanan.recalc"

// HitungUlangPPh21Sekali menghitung ulang PPh 21 honor lama satu kali setelah
// aturan TER bulanan berlaku. Lihat HitungUlangPPh21.
func HitungUlangPPh21Sekali() {
	if Q.Int("SELECT COUNT(*) FROM migrations WHERE name = ?", penandaHitungUlangPPh21) > 0 {
		return
	}
	Q.Tx(func() {
		HitungUlangPPh21()
		Q.Run("INSERT INTO migrations (name, applied_at) VALUES (?, ?)", penandaHitungUlangPPh21, time.Now().Format("2006-01-02T15:04:05-07:00"))
	})
}

// HitungUlangPPh21 menghitung ulang PPh 21 semua transaksi honor memakai data
// penerima yang tersimpan di transaksi, urut tanggal agar honor bulanan digabung
// per bulan. Transaksi yang PPh 21-nya sudah disetor (ada NTPN/tanggal setor)
// tidak diubah, tetapi tetap ikut dijumlahkan. Mengembalikan jumlah transaksi
// yang berubah.
func HitungUlangPPh21() int {
	type kunci struct {
		pid   int64
		bulan string
	}
	sudah := map[kunci][2]float64{}
	berubah := 0
	for _, b := range Q.All("SELECT id, tgl FROM belanja WHERE jenis = 'honor' ORDER BY tgl, id") {
		bid := b.I("id")
		bulan := b.S("tgl")
		if len(bulan) >= 7 {
			bulan = bulan[:7]
		}
		pj := Q.One("SELECT * FROM pajak WHERE belanja_id = ? AND jenis = 'PPH21' ORDER BY id LIMIT 1", bid)
		terkunci := pj != nil && (pj.S("ntpn") != "" || pj.S("tgl_setor") != "")
		pphDpp, pphNilai, ubah := 0.0, 0.0, false
		for _, r := range Q.All("SELECT * FROM penerima_belanja WHERE belanja_id = ? ORDER BY id", bid) {
			k := kunci{r.I("penerima_id"), bulan}
			hari := r.F("hari")
			if !terkunci {
				// Data penerima terbaru dipakai seperti saat transaksi disimpan ulang.
				if m := Q.One("SELECT * FROM penerima WHERE id = ?", k.pid); m != nil {
					r["kategori"], r["golongan"] = m["kategori"], m["golongan"]
					r["jenis_penerima"], r["ptkp"] = jenisPenerimaDari(m), defaultStr(m.S("ptkp"), "A")
				}
				// Form lama menampilkan hari kosong sebagai 1, sehingga honor bulanan
				// yang disimpan ulang tercatat 1 hari dan terkena tarif harian.
				if hari == 1 && !PPh21PNSFinal(r) && jenisPenerimaDari(r) == "tidak_tetap" {
					hari = 0
				}
			}
			bulanan := hari <= 0 && k.pid != 0
			pph := r.F("pph21")
			if !terkunci {
				lain := [2]float64{}
				if bulanan {
					lain = sudah[k]
				}
				pph = PPh21UntukPenerimaBulan(r, r.F("bruto"), hari, lain[0], lain[1])
				Q.Run("UPDATE penerima_belanja SET pph21 = ?, netto = bruto - ?, hari = ?, jenis_penerima = ?, ptkp = ? WHERE id = ?",
					pph, pph, toInt(hari), r.S("jenis_penerima"), r.S("ptkp"), r.I("id"))
				if pph != r.F("pph21") {
					ubah = true
				}
			}
			if bulanan {
				s := sudah[k]
				sudah[k] = [2]float64{s[0] + r.F("bruto"), s[1] + pph}
			}
			if pph > 0 {
				pphDpp += r.F("bruto")
				pphNilai += pph
			}
		}
		if terkunci {
			continue
		}
		switch {
		case pphNilai <= 0 && pj != nil:
			Q.Run("DELETE FROM pajak WHERE belanja_id = ? AND jenis = 'PPH21'", bid)
			ubah = true
		case pphNilai > 0:
			tarif := math.Round(pphNilai/pphDpp*100*100) / 100
			if pj == nil {
				Q.Insert("pajak", Row{"belanja_id": bid, "jenis": "PPH21", "dpp": pphDpp, "tarif": tarif, "nilai": pphNilai, "tgl_pungut": b.S("tgl")})
				ubah = true
			} else if pj.F("nilai") != pphNilai || pj.F("dpp") != pphDpp {
				Q.Run("UPDATE pajak SET dpp = ?, tarif = ?, nilai = ? WHERE id = ?", pphDpp, tarif, pphNilai, pj.I("id"))
				ubah = true
			}
		}
		if ubah {
			berubah++
		}
	}
	return berubah
}

package app

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Tool MCP untuk mencatat/mengubah/menghapus data. Validasi memakai service
// yang sama dengan form web. Tanpa `id` = catat baru, dengan `id` = ubah.
func init() { RegisterMcp(2, mcpTulisDaftar) }

func mcpTulisDaftar() []McpTool {
	var out []McpTool
	out = append(out, mcpTulisPenerimaan()...)
	out = append(out, mcpTulisMutasi()...)
	out = append(out, mcpTulisBelanja()...)
	out = append(out, mcpTulisRencana()...)
	out = append(out, mcpTulisRencanaPendapatan()...)
	return out
}

// mcpForm mengubah argumen JSON menjadi Form seperti isian POST.
func mcpForm(a McpArgs) Form {
	f, _ := mcpFormNilai(map[string]any(a)).(Form)
	return f
}

func mcpFormNilai(v any) any {
	switch x := v.(type) {
	case map[string]any:
		f := Form{}
		for k, e := range x {
			if n := mcpFormNilai(e); n != nil {
				f[k] = n
			}
		}
		return f
	case []any:
		f := Form{}
		for i, e := range x {
			if n := mcpFormNilai(e); n != nil {
				f[strconv.Itoa(i)] = n
			}
		}
		return f
	case nil:
		return nil
	case json.Number:
		return x.String()
	default:
		return toStr(x)
	}
}

func mcpOptKeys(opts []Opt) []string {
	out := make([]string, len(opts))
	for i, o := range opts {
		out[i] = o.K
	}
	return out
}

func mcpEnum(desc string, opts []Opt) OM {
	return append(mcpS(desc), KV{"enum", mcpOptKeys(opts)})
}

func mcpArr(desc string, items any) OM {
	return OM{{"type", "array"}, {"description", desc}, {"items", items}}
}

func mcpHapusSkema(apa string) OM {
	return mcpObj(OM{{"id", mcpI("ID " + apa + " yang dihapus.")}}, "id")
}

// mcpTahunDari: tahun untuk validasi: dari tgl isian, lalu baris lama, lalu param/default.
func mcpTahunDari(a McpArgs, lama Row) (int, error) {
	tahun := mcpTahun(a)
	for _, v := range []any{a["tgl"], lama["tgl"]} {
		if s, ok := v.(string); ok && ValidDate(s) {
			tahun = int(toInt(s[:4]))
			break
		}
	}
	if !InTahunList(tahun) {
		return 0, mcpError("Tahun anggaran %d belum terdaftar. Tambahkan dulu di menu Tahun anggaran.", tahun)
	}
	return tahun, nil
}

func mcpWajibAda(row Row, apa string, id int64) error {
	if row == nil {
		return mcpError("%s dengan id %d tidak ditemukan.", apa, id)
	}
	return nil
}

func mcpOkID(id int64, data any) OM { return OM{{"ok", true}, {"id", id}, {"data", data}} }
func mcpOkHapus(id int64) OM        { return OM{{"ok", true}, {"dihapus", id}} }

func mcpTulisPenerimaan() []McpTool {
	return []McpTool{
		{
			Nama:      "penerimaan_simpan",
			Akses:     AksesTulis,
			Deskripsi: "Catat penerimaan dana BOSP (tanpa id) atau ubah penerimaan (dengan id). Tahap: " + strings.Join(mcpOptKeys(PenerimaanTahap), ", ") + ".",
			Skema: mcpObj(OM{
				{"id", mcpI("Isi untuk mengubah data yang sudah ada.")},
				{"tgl", mcpS("Tanggal YYYY-MM-DD.")},
				{"tahap", mcpEnum("Tahap penerimaan.", PenerimaanTahap)},
				{"uraian", mcpS("Uraian penerimaan.")},
				{"jumlah", mcpN("Jumlah rupiah.")},
				{"no_bukti", mcpS("Nomor bukti (opsional).")},
				{"rencana_pendapatan_id", mcpI("ID rencana pendapatan (opsional, tahap harus sama).")},
			}, "tgl", "tahap", "uraian", "jumlah"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				var lama Row
				if id != 0 {
					lama = PenerimaanFind(id)
					if err := mcpWajibAda(lama, "Penerimaan", id); err != nil {
						return nil, err
					}
				}
				tahun, err := mcpTahunDari(a, lama)
				if err != nil {
					return nil, err
				}
				data, err := PenerimaanValidate(mcpForm(a), id, tahun)
				if err != nil {
					return nil, err
				}
				if id != 0 {
					PenerimaanUpdate(id, data)
				} else {
					id = PenerimaanCreate(data)
				}
				return mcpOkID(id, PenerimaanFind(id)), nil
			},
		},
		{
			Nama:      "penerimaan_hapus",
			Akses:     AksesTulis,
			Deskripsi: "Hapus satu penerimaan dana.",
			Skema:     mcpHapusSkema("penerimaan"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				row := PenerimaanFind(id)
				if err := mcpWajibAda(row, "Penerimaan", id); err != nil {
					return nil, err
				}
				PenerimaanDelete(row)
				return mcpOkHapus(id), nil
			},
		},
	}
}

func mcpTulisMutasi() []McpTool {
	return []McpTool{
		{
			Nama:      "mutasi_simpan",
			Akses:     AksesTulis,
			Deskripsi: "Catat (tanpa id) atau ubah (dengan id) mutasi kas: tarik tunai, setor ke bank, biaya admin bank, bunga.",
			Skema: mcpObj(OM{
				{"id", mcpI("Isi untuk mengubah data yang sudah ada.")},
				{"tgl", mcpS("Tanggal YYYY-MM-DD.")},
				{"jenis", mcpEnum("Jenis mutasi.", MutasiJenis)},
				{"jumlah", mcpN("Jumlah rupiah.")},
				{"uraian", mcpS("Uraian (opsional, ada default per jenis).")},
				{"no_bukti", mcpS("Nomor bukti (opsional).")},
			}, "tgl", "jenis", "jumlah"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				var lama Row
				if id != 0 {
					lama = MutasiFind(id)
					if err := mcpWajibAda(lama, "Mutasi", id); err != nil {
						return nil, err
					}
				}
				tahun, err := mcpTahunDari(a, lama)
				if err != nil {
					return nil, err
				}
				data, err := MutasiValidate(mcpForm(a), id, tahun)
				if err != nil {
					return nil, err
				}
				if id != 0 {
					MutasiUpdate(id, data)
				} else {
					id = MutasiCreate(data)
				}
				return mcpOkID(id, MutasiFind(id)), nil
			},
		},
		{
			Nama:      "mutasi_hapus",
			Akses:     AksesTulis,
			Deskripsi: "Hapus satu mutasi kas.",
			Skema:     mcpHapusSkema("mutasi kas"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				row := MutasiFind(id)
				if err := mcpWajibAda(row, "Mutasi", id); err != nil {
					return nil, err
				}
				MutasiDelete(row)
				return mcpOkHapus(id), nil
			},
		},
	}
}

func mcpTulisBelanja() []McpTool {
	item := mcpObj(OM{
		{"nama", mcpS("Nama barang/jasa.")},
		{"volume", mcpN("Jumlah.")},
		{"satuan", mcpS("Satuan, mis. rim, pcs.")},
		{"harga", mcpN("Harga satuan rupiah.")},
	}, "nama", "volume", "harga")
	pen := mcpObj(OM{
		{"penerima_id", mcpI("ID master penerima.")},
		{"volume", mcpN("Volume, mis. jumlah OJ/hari.")},
		{"satuan", mcpS("Satuan, mis. OJ, OK.")},
		{"tarif", mcpN("Tarif per satuan rupiah.")},
	}, "penerima_id", "volume", "tarif")

	return []McpTool{
		{
			Nama:  "belanja_simpan",
			Akses: AksesTulis,
			Deskripsi: "Catat (tanpa id) atau ubah (dengan id) transaksi belanja dari satu paket rencana. " +
				"Paket jenis barang memakai `items` (+ pajak_ppn/pph22/pph23, toko_id); jenis honor/transport memakai `pen`. " +
				"Pajak dihitung otomatis. Ditolak bila melebihi sisa pagu. Saldo kas boleh minus (belanja sebelum dana cair); kuitansi mencatat tanggal lunas saat dana cair.",
			Skema: mcpObj(OM{
				{"id", mcpI("Isi untuk mengubah data yang sudah ada.")},
				{"rencana_id", mcpI("ID paket rencana belanja.")},
				{"tgl", mcpS("Tanggal YYYY-MM-DD.")},
				{"jumlah_periode", mcpI("Jumlah bulan yang dicakup (1-12).")},
				{"uraian", mcpS("Uraian transaksi.")},
				{"metode", mcpEnum("Cara bayar.", BelanjaMetode)},
				{"no_bukti", mcpS("Nomor bukti (opsional, otomatis bila kosong).")},
				{"toko_id", mcpI("ID toko/penyedia (barang).")},
				{"items", mcpArr("Rincian barang/jasa.", item)},
				{"pajak_ppn", mcpB("Harga sudah termasuk PPN.")},
				{"pajak_pph22", mcpB("Pungut PPh 22.")},
				{"pajak_pph23", mcpB("Pungut PPh 23.")},
				{"pen", mcpArr("Penerima honor/transport.", pen)},
			}, "rencana_id", "tgl", "jumlah_periode", "uraian", "metode"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				var lama Row
				if id != 0 {
					lama = BelanjaFind(id)
					if err := mcpWajibAda(lama, "Belanja", id); err != nil {
						return nil, err
					}
				}
				tahun, err := mcpTahunDari(a, lama)
				if err != nil {
					return nil, err
				}
				data, anak, err := BelanjaValidate(mcpForm(a), id, tahun)
				if err != nil {
					return nil, err
				}
				if id != 0 {
					BelanjaUpdate(id, data, anak)
				} else {
					id = BelanjaCreate(data, anak)
				}
				return mcpOkID(id, BelanjaDetail(id)), nil
			},
		},
		{
			Nama:      "belanja_hapus",
			Akses:     AksesTulis,
			Deskripsi: "Hapus satu transaksi belanja beserta rincian, pajak, dan lampirannya.",
			Skema:     mcpHapusSkema("belanja"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				row := BelanjaFind(id)
				if err := mcpWajibAda(row, "Belanja", id); err != nil {
					return nil, err
				}
				if err := BelanjaDelete(row); err != nil {
					return nil, err
				}
				return mcpOkHapus(id), nil
			},
		},
	}
}

func mcpTulisRencana() []McpTool {
	item := mcpObj(OM{
		{"nama", mcpS("Nama barang/jasa.")},
		{"volume", mcpN("Jumlah per bulan.")},
		{"satuan", mcpS("Satuan.")},
		{"harga", mcpN("Harga satuan rupiah.")},
	}, "nama", "volume", "harga")
	pen := mcpObj(OM{
		{"penerima_id", mcpI("ID master penerima.")},
		{"volume", mcpN("Volume per bulan.")},
		{"satuan", mcpS("Satuan.")},
		{"tarif", mcpN("Tarif rupiah.")},
	}, "penerima_id", "volume", "tarif")

	return []McpTool{
		{
			Nama:  "rencana_simpan",
			Akses: AksesTulis,
			Deskripsi: "Buat (tanpa id) atau ubah (dengan id) paket rencana belanja RKAS. " +
				"Pagu = nilai per bulan x frekuensi_bulan. Jenis barang memakai `items`, honor/transport memakai `penerima`.",
			Skema: mcpObj(OM{
				{"id", mcpI("Isi untuk mengubah paket yang sudah ada.")},
				{"tahun", mcpI("Tahun anggaran paket baru (default tahun default).")},
				{"komponen_kode", mcpS("Kode komponen BOSP (lihat tool komponen).")},
				{"kegiatan", mcpS("Kegiatan (opsional).")},
				{"rekening_kode", mcpS("Kode rekening belanja (opsional).")},
				{"uraian", mcpS("Nama paket, mis. Belanja ATK.")},
				{"jenis", mcpEnum("Jenis paket.", RencanaJenis)},
				{"frekuensi_bulan", mcpI("Berapa bulan (1-12).")},
				{"jadwal", mcpArr("Bulan rencana pelaksanaan 1-12.", OM{{"type", "integer"}})},
				{"toko_id", mcpI("ID toko (barang, opsional).")},
				{"items", mcpArr("Rincian barang per bulan.", item)},
				{"penerima", mcpArr("Penerima honor/transport.", pen)},
			}, "komponen_kode", "uraian", "jenis", "frekuensi_bulan"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				if id != 0 {
					if err := mcpWajibAda(RencanaFind(id), "Rencana", id); err != nil {
						return nil, err
					}
				}
				tahun, err := mcpTahunDari(McpArgs{"tahun": a["tahun"]}, nil)
				if err != nil {
					return nil, err
				}
				data, items, rec, err := RencanaValidate(mcpForm(a), id, tahun)
				if err != nil {
					return nil, err
				}
				if id != 0 {
					RencanaUpdate(id, data, items, rec)
				} else {
					id = RencanaCreate(data, items, rec)
				}
				return mcpOkID(id, RencanaDetail(id)), nil
			},
		},
		{
			Nama:      "rencana_hapus",
			Akses:     AksesTulis,
			Deskripsi: "Hapus paket rencana belanja yang belum dipakai di BKU.",
			Skema:     mcpHapusSkema("rencana"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				if err := mcpWajibAda(RencanaFind(id), "Rencana", id); err != nil {
					return nil, err
				}
				if err := RencanaDelete(id); err != nil {
					return nil, err
				}
				return mcpOkHapus(id), nil
			},
		},
	}
}

func mcpTulisRencanaPendapatan() []McpTool {
	cari := func(id int64) Row { return Q.One("SELECT * FROM rencana_pendapatan WHERE id = ?", id) }
	dipakai := func(id int64) int64 {
		return Q.Int("SELECT COUNT(*) FROM penerimaan WHERE rencana_pendapatan_id = ?", id)
	}

	return []McpTool{
		{
			Nama:  "rencana_pendapatan_simpan",
			Akses: AksesTulis,
			Deskripsi: "Buat (tanpa id) atau ubah (dengan id) rencana/target pendapatan BOSP per tahap. Tahap: " +
				strings.Join(mcpOptKeys(PenerimaanTahap), ", ") + ".",
			Skema: mcpObj(OM{
				{"id", mcpI("Isi untuk mengubah rencana yang sudah ada.")},
				{"tahun", mcpI("Tahun anggaran rencana baru (default tahun default).")},
				{"tahap", mcpEnum("Tahap pendapatan.", PenerimaanTahap)},
				{"uraian", mcpS("Nama rencana, mis. BOSP Reguler Tahap 2.")},
				{"target_bulan", mcpI("Bulan target cair 1-12 (opsional).")},
				{"target_jumlah", mcpN("Target jumlah rupiah.")},
			}, "tahap", "uraian", "target_jumlah"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				var lama Row
				if id != 0 {
					lama = cari(id)
					if err := mcpWajibAda(lama, "Rencana pendapatan", id); err != nil {
						return nil, err
					}
				}
				var tahun int64
				if lama != nil {
					tahun = lama.I("tahun")
				} else {
					t, err := mcpTahunDari(McpArgs{"tahun": a["tahun"]}, nil)
					if err != nil {
						return nil, err
					}
					tahun = int64(t)
				}
				var bulan any
				if b := a.S("target_bulan"); a["target_bulan"] != nil && b != "" {
					bulan = toInt(b)
				}
				data := Row{
					"tahun":         tahun,
					"tahap":         a.S("tahap"),
					"uraian":        mcpTrim(a, "uraian"),
					"target_bulan":  bulan,
					"target_jumlah": Uang(a.S("target_jumlah")),
				}
				e := NewValidationOrdered()
				if _, ok := optLabel(PenerimaanTahap, data.S("tahap")); !ok {
					e.Set("tahap", "Pilih tahap pendapatan.")
				}
				if data.S("uraian") == "" {
					e.Set("uraian", "Nama rencana pendapatan wajib diisi.")
				}
				if b, ok := bulan.(int64); ok && (b < 1 || b > 12) {
					e.Set("target_bulan", "Bulan target harus 1-12.")
				}
				if data.F("target_jumlah") <= 0 {
					e.Set("target_jumlah", "Target pendapatan harus lebih dari nol.")
				}
				if lama != nil && lama.S("tahap") != data.S("tahap") && dipakai(id) > 0 {
					e.Set("tahap", "Tahap tidak bisa diubah karena sudah dipasangkan dengan penerimaan.")
				}
				if len(e.Errors) > 0 {
					return nil, e
				}
				if id != 0 {
					Q.Update("rencana_pendapatan", data, id)
				} else {
					id = Q.Insert("rencana_pendapatan", data)
				}
				return mcpOkID(id, cari(id)), nil
			},
		},
		{
			Nama:      "rencana_pendapatan_hapus",
			Akses:     AksesTulis,
			Deskripsi: "Hapus rencana pendapatan yang belum dipasangkan dengan penerimaan.",
			Skema:     mcpHapusSkema("rencana pendapatan"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				if err := mcpWajibAda(cari(id), "Rencana pendapatan", id); err != nil {
					return nil, err
				}
				if dipakai(id) > 0 {
					return nil, mcpError("Rencana ini sudah dipasangkan dengan penerimaan. Lepaskan pasangannya dulu.")
				}
				Q.Run("DELETE FROM rencana_pendapatan WHERE id = ?", id)
				return mcpOkHapus(id), nil
			},
		},
	}
}

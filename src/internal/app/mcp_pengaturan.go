package app

import (
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Tool MCP untuk profil sekolah, pejabat, tahun anggaran, dan tanggal BKU.

func init() { RegisterMcp(4, mcpPengaturanDaftar) }

var mcpSekolahFields = []string{
	"nama", "npsn", "jenjang", "status", "alamat", "desa", "kecamatan", "kabupaten", "provinsi",
	"kode_pos", "telepon", "email", "npwp", "bank_nama", "bank_rekening", "bank_atas_nama",
}

var mcpPeran = []string{"kepala", "bendahara"}

var mcpNpsnRe = regexp.MustCompile(`^\d{8}$`)

// phpEmpty meniru empty() PHP untuk nilai hasil decode JSON.
func phpEmpty(v any) bool {
	switch x := v.(type) {
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return !truthyValue(v)
}

func mcpEmailValid(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s && strings.Contains(s[strings.LastIndex(s, "@")+1:], ".")
}

// mcpHapusGambar meniru ImageUpload::delete: hapus berkas upload beserta cache PDF-nya.
func mcpHapusGambar(name string) {
	if name == "" {
		return
	}
	p := UploadPath(name)
	_ = os.Remove(p)
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(p), "pdf-cache", base+"-*"))
	for _, m := range matches {
		_ = os.Remove(m)
	}
}

func mcpCekTahun(a McpArgs) (int, error) {
	t := int(a.I("tahun"))
	if !InTahunList(t) {
		return 0, mcpError("Tahun anggaran %d tidak terdaftar.", t)
	}
	return t, nil
}

func mcpJadikanDefault(t int) {
	Q.Tx(func() {
		Q.Run("UPDATE tahun_anggaran SET is_default = 0")
		Q.Run("UPDATE tahun_anggaran SET is_default = 1 WHERE tahun = ?", t)
		Q.Run("UPDATE sekolah SET tahun_anggaran = ? WHERE id = 1", t)
	})
}

func mcpPengaturanDaftar() []McpTool {
	sekolahProps := OM{}
	for _, f := range mcpSekolahFields {
		desc := strings.ReplaceAll(f, "_", " ")
		if f == "npsn" {
			desc = "NPSN, 8 angka"
		}
		sekolahProps = append(sekolahProps, KV{f, mcpS(desc)})
	}
	tahunP := mcpI("Tahun anggaran, mis. 2026.")
	bkuTabel := map[string]string{"trm": "penerimaan", "bel": "belanja", "mut": "mutasi_kas"}

	return []McpTool{
		{
			Nama: "sekolah_ubah", Akses: AksesTulis,
			Deskripsi: "Ubah profil sekolah. Kirim hanya field yang berubah. Kop, logo, dan stempel diunggah lewat web.",
			Skema:     mcpObj(sekolahProps),
			Handler: func(a McpArgs) (any, error) {
				s := Sekolah()
				in := Row{}
				for _, f := range mcpSekolahFields {
					if a.Has(f) {
						in[f] = strings.TrimSpace(a.S(f))
					} else {
						in[f] = strings.TrimSpace(s.S(f))
					}
				}
				e := NewValidationOrdered()
				if in.S("nama") == "" {
					e.Set("nama", "Isi nama sekolah.")
				}
				if n := in.S("npsn"); n != "" && !mcpNpsnRe.MatchString(n) {
					e.Set("npsn", "NPSN terdiri dari 8 angka.")
				}
				if m := in.S("email"); m != "" && !mcpEmailValid(m) {
					e.Set("email", "Alamat surel tidak valid.")
				}
				if len(e.Errors) > 0 {
					return nil, e
				}
				in["updated_at"] = time.Now().Format("2006-01-02 15:04:05")
				Q.Update("sekolah", in, 1)
				return OM{{"ok", true}, {"sekolah", Q.One("SELECT * FROM sekolah WHERE id = 1")}}, nil
			},
		},
		{
			Nama: "pejabat_list", Akses: AksesBaca,
			Deskripsi: "Daftar semua pejabat (kepala sekolah, bendahara) beserta masa jabatan.",
			Skema:     mcpObj(nil),
			Handler: func(a McpArgs) (any, error) {
				return Q.All("SELECT id, peran, nama, nip, pangkat, mulai, selesai, ttd_path IS NOT NULL AS ada_ttd FROM pejabat ORDER BY peran, mulai DESC"), nil
			},
		},
		{
			Nama: "pejabat_simpan", Akses: AksesTulis,
			Deskripsi: "Tambah (tanpa id) atau ubah (dengan id) pejabat. Saat ubah cukup kirim field yang berubah. Tanda tangan diunggah lewat web.",
			Skema: mcpObj(OM{
				{"id", mcpI("ID pejabat untuk mengubah.")},
				{"peran", OM{{"type", "string"}, {"description", "Peran."}, {"enum", mcpPeran}}},
				{"nama", mcpS("Nama lengkap beserta gelar.")},
				{"nip", mcpS("NIP 18 angka, boleh kosong.")},
				{"pangkat", mcpS("Pangkat/golongan.")},
				{"mulai", mcpS("Tanggal mulai menjabat YYYY-MM-DD.")},
				{"selesai", mcpS("Tanggal selesai YYYY-MM-DD; kosong = masih menjabat.")},
			}),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				lama := Row{}
				if id != 0 {
					if lama = Q.One("SELECT * FROM pejabat WHERE id = ?", id); lama == nil {
						return nil, mcpError("Pejabat dengan id %d tidak ditemukan.", id)
					}
				}
				ambil := func(k string) string {
					if a.Has(k) {
						return a.S(k)
					}
					return lama.S(k)
				}
				in := Row{
					"peran":   ambil("peran"),
					"nama":    strings.TrimSpace(ambil("nama")),
					"nip":     Digits(ambil("nip")),
					"pangkat": strings.TrimSpace(ambil("pangkat")),
					"mulai":   ambil("mulai"),
					"selesai": nil,
				}
				selesai := ambil("selesai")
				if selesai != "" && selesai != "0" {
					in["selesai"] = selesai
				}
				e := NewValidationOrdered()
				if !inStrings(in.S("peran"), mcpPeran...) {
					e.Set("peran", "Pilih kepala atau bendahara.")
				}
				if in.S("nama") == "" {
					e.Set("nama", "Isi nama lengkap beserta gelar.")
				}
				if n := in.S("nip"); n != "" && len(n) != 18 {
					e.Set("nip", "NIP terdiri dari 18 angka.")
				}
				if !ValidDate(in.S("mulai")) {
					e.Set("mulai", "Isi tanggal mulai menjabat (YYYY-MM-DD).")
				}
				if in["selesai"] != nil && !ValidDate(in.S("selesai")) {
					e.Set("selesai", "Format tanggal YYYY-MM-DD.")
				} else if in["selesai"] != nil && in.S("selesai") < in.S("mulai") {
					e.Set("selesai", "Tanggal selesai tidak boleh sebelum tanggal mulai.")
				}
				if len(e.Errors) > 0 {
					return nil, e
				}
				if id != 0 {
					Q.Update("pejabat", in, id)
				} else {
					id = Q.Insert("pejabat", in)
				}
				return OM{{"ok", true}, {"id", id}, {"aktif", OM{
					{"kepala", PejabatAktif("kepala", "")},
					{"bendahara", PejabatAktif("bendahara", "")},
				}}}, nil
			},
		},
		{
			Nama: "pejabat_hapus", Akses: AksesTulis,
			Deskripsi: "Hapus pejabat beserta gambar tanda tangannya.",
			Skema:     mcpObj(OM{{"id", mcpI("ID pejabat.")}}, "id"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				p := Q.One("SELECT * FROM pejabat WHERE id = ?", id)
				if p == nil {
					return nil, mcpError("Pejabat dengan id %d tidak ditemukan.", id)
				}
				mcpHapusGambar(p.S("ttd_path"))
				Q.Run("DELETE FROM pejabat WHERE id = ?", id)
				return OM{{"ok", true}, {"dihapus", id}}, nil
			},
		},
		{
			Nama: "tahun_tambah", Akses: AksesTulis,
			Deskripsi: "Tambah tahun anggaran baru.",
			Skema: mcpObj(OM{
				{"tahun", tahunP},
				{"keterangan", mcpS("Keterangan singkat.")},
				{"jadikan_default", mcpB("Jadikan tahun default.")},
			}, "tahun"),
			Handler: func(a McpArgs) (any, error) {
				t := int(a.I("tahun"))
				if t < 2000 || t > 2100 {
					return nil, NewValidationOrdered("tahun", "Isi tahun empat angka antara 2000 dan 2100.")
				}
				if InTahunList(t) {
					return nil, NewValidationOrdered("tahun", fmt.Sprintf("Tahun %d sudah ada.", t))
				}
				isDef := 0
				if len(TahunList()) == 0 {
					isDef = 1
				}
				Q.Insert("tahun_anggaran", Row{"tahun": t, "keterangan": strings.TrimSpace(a.S("keterangan")), "is_default": isDef})
				if !phpEmpty(a["jadikan_default"]) {
					mcpJadikanDefault(t)
				}
				return OM{{"ok", true}, {"tahun", t}, {"default", TahunDefault()}}, nil
			},
		},
		{
			Nama: "tahun_ubah", Akses: AksesTulis,
			Deskripsi: "Ubah keterangan tahun anggaran dan/atau jadikan tahun default.",
			Skema: mcpObj(OM{
				{"tahun", tahunP},
				{"keterangan", mcpS("Keterangan baru.")},
				{"jadikan_default", mcpB("true = jadikan tahun default.")},
			}, "tahun"),
			Handler: func(a McpArgs) (any, error) {
				t, err := mcpCekTahun(a)
				if err != nil {
					return nil, err
				}
				if a.Has("keterangan") {
					Q.Run("UPDATE tahun_anggaran SET keterangan = ? WHERE tahun = ?", strings.TrimSpace(a.S("keterangan")), t)
				}
				if !phpEmpty(a["jadikan_default"]) {
					mcpJadikanDefault(t)
				}
				return OM{{"ok", true}, {"tahun", t}, {"default", TahunDefault()}}, nil
			},
		},
		{
			Nama: "tahun_hapus", Akses: AksesTulis,
			Deskripsi: "Hapus tahun anggaran yang masih kosong dan bukan default.",
			Skema:     mcpObj(OM{{"tahun", tahunP}}, "tahun"),
			Handler: func(a McpArgs) (any, error) {
				t, err := mcpCekTahun(a)
				if err != nil {
					return nil, err
				}
				y := itoa(t)
				if Q.Int("SELECT is_default FROM tahun_anggaran WHERE tahun = ?", t) != 0 {
					return nil, mcpError("Tahun %d adalah tahun default. Jadikan tahun lain default dulu.", t)
				}
				dipakai := Q.Int("SELECT COUNT(*) FROM rencana_belanja WHERE tahun = ?", t) +
					Q.Int("SELECT COUNT(*) FROM belanja WHERE strftime('%Y', tgl) = ?", y) +
					Q.Int("SELECT COUNT(*) FROM penerimaan WHERE strftime('%Y', tgl) = ?", y) +
					Q.Int("SELECT COUNT(*) FROM mutasi_kas WHERE strftime('%Y', tgl) = ?", y)
				if dipakai > 0 {
					return nil, mcpError("Tahun %d sudah berisi data, jadi tidak bisa dihapus.", t)
				}
				Q.Run("DELETE FROM tahun_anggaran WHERE tahun = ?", t)
				return OM{{"ok", true}, {"dihapus", t}}, nil
			},
		},
		{
			Nama: "bku_ubah_tanggal", Akses: AksesTulis,
			Deskripsi: "Pindahkan tanggal satu baris BKU (penerimaan/belanja/mutasi) di tahun yang sama. Nomor bukti dan saldo disusun ulang otomatis.",
			Skema: mcpObj(OM{
				{"jenis", OM{{"type", "string"}, {"description", "trm = penerimaan, bel = belanja, mut = mutasi kas."}, {"enum", []string{"trm", "bel", "mut"}}}},
				{"id", mcpI("ID transaksi.")},
				{"tgl", mcpS("Tanggal baru YYYY-MM-DD.")},
			}, "jenis", "id", "tgl"),
			Handler: func(a McpArgs) (any, error) {
				src := a.S("jenis")
				tabel, ok := bkuTabel[src]
				if !ok {
					return nil, NewValidationOrdered("jenis", "Pilih trm, bel, atau mut.")
				}
				id := a.I("id")
				row := Q.One("SELECT id, tgl FROM "+tabel+" WHERE id = ?", id)
				if row == nil {
					return nil, mcpError("Transaksi %s id %d tidak ditemukan.", src, id)
				}
				tgl := a.S("tgl")
				if !ValidDate(tgl) {
					return nil, NewValidationOrdered("tgl", "Tanggal tidak valid (YYYY-MM-DD).")
				}
				lamaTgl := row.S("tgl")
				if tgl[:4] != lamaTgl[:4] {
					return nil, NewValidationOrdered("tgl", "Tanggal harus di tahun "+lamaTgl[:4]+".")
				}
				Q.Tx(func() {
					Q.Run("UPDATE "+tabel+" SET tgl = ? WHERE id = ?", tgl, id)
					if src == "bel" {
						Q.Run("UPDATE pajak SET tgl_pungut = ? WHERE belanja_id = ? AND tgl_pungut = ?", tgl, id, lamaTgl)
					}
					SusunNomorBuktiBKU(int(toInt(tgl[:4])))
				})
				return OM{{"ok", true}, {"jenis", src}, {"id", id}, {"tgl", tgl}}, nil
			},
		},
	}
}

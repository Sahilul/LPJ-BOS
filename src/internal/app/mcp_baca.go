package app

import (
	"strconv"
	"strings"
	"time"
)

// Tool MCP untuk membaca data: status, master, rencana, penerimaan, BKU, laporan.

func init() { RegisterMcp(1, mcpBacaDaftar) }

// mcpTahun: tahun anggaran untuk tool: param tahun, atau tahun default.
func mcpTahun(a McpArgs) int {
	t := int(a.I("tahun"))
	if t >= 2000 && t <= 2100 {
		return t
	}
	return TahunDefault()
}

func itoa(i int) string { return strconv.Itoa(i) }

func mcpTrim(a McpArgs, k string) string { return strings.TrimSpace(a.S(k)) }

// mcpAktifFilter meniru array_key_exists('aktif') && !== null.
func mcpAktifFilter(a McpArgs) (int, bool) {
	v, ok := a["aktif"]
	if !ok || v == nil {
		return 0, false
	}
	if truthyValue(v) {
		return 1, true
	}
	return 0, true
}

func truthyValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != "" && x != "0"
	default:
		return toFloat(x) != 0
	}
}

func mcpListTool(nama, deskripsi, sql string) McpTool {
	return McpTool{Nama: nama, Akses: AksesBaca, Deskripsi: deskripsi, Skema: mcpObj(nil),
		Handler: func(a McpArgs) (any, error) { return Q.All(sql), nil }}
}

func mcpBacaDaftar() []McpTool {
	cari := mcpS("Kata kunci pencarian nama/kode.")
	aktifB := mcpB("Hanya yang aktif bila true.")
	return []McpTool{
		{
			Nama: "status", Akses: AksesBaca,
			Deskripsi: "Ringkasan aplikasi: profil sekolah, tahun anggaran, pagu, realisasi, penerimaan, dan saldo kas.",
			Skema:     mcpObj(OM{{"tahun", mcpI("Tahun anggaran, default tahun default aplikasi.")}}),
			Handler: func(a McpArgs) (any, error) {
				tahun := mcpTahun(a)
				ts := itoa(tahun)
				s := Sekolah()
				pagu := Q.Float("SELECT COALESCE(SUM(pagu), 0) FROM rencana_belanja WHERE tahun = ?", tahun)
				realisasi := Q.Float("SELECT COALESCE(SUM(b.total), 0) FROM belanja b JOIN rencana_belanja r ON r.id = b.rencana_id WHERE r.tahun = ?", tahun)
				diterima := Q.Float("SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan WHERE strftime('%Y', tgl) = ?", ts)
				return OM{
					{"sekolah", OM{{"nama", s.S("nama")}, {"npsn", s.S("npsn")}, {"jenjang", s.S("jenjang")}}},
					{"tahun", tahun},
					{"pagu", pagu},
					{"realisasi", realisasi},
					{"sisa_pagu", round2(pagu - realisasi)},
					{"penerimaan", diterima},
					{"saldo_kas", SaldoKas(ts+"-12-31", 0, 0)},
					{"jumlah", OM{
						{"rencana", Q.Int("SELECT COUNT(*) FROM rencana_belanja WHERE tahun = ?", tahun)},
						{"belanja", Q.Int("SELECT COUNT(*) FROM belanja WHERE strftime('%Y', tgl) = ?", ts)},
						{"penerimaan", Q.Int("SELECT COUNT(*) FROM penerimaan WHERE strftime('%Y', tgl) = ?", ts)},
						{"mutasi", Q.Int("SELECT COUNT(*) FROM mutasi_kas WHERE strftime('%Y', tgl) = ?", ts)},
					}},
				}, nil
			},
		},
		mcpListTool("tahun_list", "Daftar tahun anggaran yang terdaftar beserta penanda default.",
			"SELECT tahun, is_default, keterangan FROM tahun_anggaran ORDER BY tahun DESC"),
		{
			Nama: "sekolah", Akses: AksesBaca,
			Deskripsi: "Profil sekolah, pejabat aktif (kepala/bendahara), dan daftar rekening.",
			Skema:     mcpObj(nil),
			Handler: func(a McpArgs) (any, error) {
				return OM{
					{"sekolah", Sekolah()},
					{"kepala", PejabatAktif("kepala", "")},
					{"bendahara", PejabatAktif("bendahara", "")},
				}, nil
			},
		},
		mcpListTool("komponen_list", "Daftar komponen BOSP (kode dan nama).",
			"SELECT id, kode, nama, urutan FROM komponen ORDER BY urutan, kode"),
		mcpListTool("rekening_list", "Daftar kode rekening belanja beserta jenisnya.",
			"SELECT id, kode, nama, jenis FROM kode_rekening ORDER BY kode"),
		mcpListTool("pajak_list", "Daftar tarif pajak yang berlaku di aplikasi.",
			"SELECT id, kode, nama, tarif, keterangan FROM tarif_pajak ORDER BY kode"),
		{
			Nama: "toko_list", Akses: AksesBaca,
			Deskripsi: "Daftar toko/penyedia, bisa dicari dan difilter status aktif.",
			Skema:     mcpObj(OM{{"q", cari}, {"aktif", aktifB}}),
			Handler: func(a McpArgs) (any, error) {
				w, p := []string{}, []any{}
				if q := mcpTrim(a, "q"); q != "" {
					w = append(w, "(nama LIKE ? OR alamat LIKE ? OR npwp LIKE ?)")
					q = "%" + q + "%"
					p = append(p, q, q, q)
				}
				if v, ok := mcpAktifFilter(a); ok {
					w = append(w, "aktif = ?")
					p = append(p, v)
				}
				return Q.All("SELECT id, nama, alamat, npwp, pkp, telepon, aktif FROM toko"+mcpWhere(w)+" ORDER BY aktif DESC, nama", p...), nil
			},
		},
		{
			Nama: "penerima_list", Akses: AksesBaca,
			Deskripsi: "Daftar penerima honor/transport (guru, tendik, pihak luar) dengan tarif PPh 21.",
			Skema: mcpObj(OM{
				{"q", cari},
				{"kategori", mcpS("PNS, PPPK, GTT, PTT, atau Luar.")},
				{"aktif", aktifB},
			}),
			Handler: func(a McpArgs) (any, error) {
				w, p := []string{}, []any{}
				if q := mcpTrim(a, "q"); q != "" {
					w = append(w, "(nama LIKE ? OR nip LIKE ? OR nik LIKE ? OR jabatan LIKE ?)")
					q = "%" + q + "%"
					p = append(p, q, q, q, q)
				}
				if k, ok := a["kategori"].(string); ok && inStrings(k, "PNS", "PPPK", "GTT", "PTT", "Luar") {
					w = append(w, "kategori = ?")
					p = append(p, k)
				}
				if v, ok := mcpAktifFilter(a); ok {
					w = append(w, "aktif = ?")
					p = append(p, v)
				}
				rows := Q.All("SELECT * FROM penerima"+mcpWhere(w)+" ORDER BY aktif DESC, nama", p...)
				for _, r := range rows {
					r["pph21"] = TarifPPh21(r)
				}
				return rows, nil
			},
		},
		{
			Nama: "rencana_list", Akses: AksesBaca,
			Deskripsi: "Paket rencana belanja (RKAS) satu tahun: pagu, terpakai, dan sisa.",
			Skema: mcpObj(OM{
				{"tahun", mcpI("Tahun anggaran.")},
				{"komponen", mcpS("Filter kode komponen, mis. K01.")},
				{"q", mcpS("Kata kunci uraian/kegiatan/rekening.")},
			}),
			Handler: func(a McpArgs) (any, error) {
				tahun := mcpTahun(a)
				w, p := []string{"r.tahun = ?"}, []any{tahun}
				if k := mcpTrim(a, "komponen"); k != "" {
					w = append(w, "r.komponen_kode = ?")
					p = append(p, k)
				}
				if q := mcpTrim(a, "q"); q != "" {
					w = append(w, "(r.uraian LIKE ? OR r.kegiatan LIKE ? OR r.rekening_kode LIKE ?)")
					q = "%" + q + "%"
					p = append(p, q, q, q)
				}
				rows := Q.All(`SELECT r.id, r.komponen_kode, r.kegiatan, r.rekening_kode, r.uraian, r.jenis,
                                r.frekuensi_bulan, r.toko_id, r.pagu, r.satuan, r.jadwal_bulan,
                                COALESCE(b.pakai, 0) AS terpakai,
                                r.pagu - COALESCE(b.pakai, 0) AS sisa,
                                r.frekuensi_bulan - COALESCE(b.periode, 0) AS periode_sisa
                         FROM rencana_belanja r
                         LEFT JOIN (SELECT rencana_id, SUM(total) AS pakai, SUM(jumlah_periode) AS periode
                                    FROM belanja GROUP BY rencana_id) b ON b.rencana_id = r.id
                         WHERE `+strings.Join(w, " AND ")+`
                         ORDER BY r.komponen_kode, r.id`, p...)
				return OM{{"tahun", tahun}, {"jumlah", len(rows)}, {"rencana", rows}}, nil
			},
		},
		mcpRencanaDetailTool(),
		{
			Nama: "penerimaan_list", Akses: AksesBaca,
			Deskripsi: "Penerimaan dana satu tahun beserta rencana pendapatan yang dipasangkan.",
			Skema:     mcpObj(OM{{"tahun", mcpI("Tahun anggaran.")}}),
			Handler: func(a McpArgs) (any, error) {
				tahun := mcpTahun(a)
				rows := Q.All(`SELECT p.id, p.tgl, p.no_bukti, p.uraian, p.tahap, p.jumlah, p.rencana_pendapatan_id,
                                rp.uraian AS rencana_uraian
                         FROM penerimaan p LEFT JOIN rencana_pendapatan rp ON rp.id = p.rencana_pendapatan_id
                         WHERE strftime('%Y', p.tgl) = ? ORDER BY p.tgl, p.id`, itoa(tahun))
				var total float64
				for _, r := range rows {
					total += r.F("jumlah")
				}
				return OM{{"tahun", tahun}, {"total", total}, {"penerimaan", rows}}, nil
			},
		},
		{
			Nama: "rencana_pendapatan_list", Akses: AksesBaca,
			Deskripsi: "Target pendapatan per tahap/sumber beserta realisasi yang sudah terpasang.",
			Skema:     mcpObj(OM{{"tahun", mcpI("Tahun anggaran.")}}),
			Handler: func(a McpArgs) (any, error) {
				return Q.All(`SELECT r.id, r.tahap, r.uraian, r.target_bulan, r.target_jumlah,
                                COALESCE(p.realisasi, 0) AS realisasi,
                                r.target_jumlah - COALESCE(p.realisasi, 0) AS selisih
                         FROM rencana_pendapatan r
                         LEFT JOIN (SELECT rencana_pendapatan_id, SUM(jumlah) AS realisasi FROM penerimaan
                                    WHERE rencana_pendapatan_id IS NOT NULL GROUP BY rencana_pendapatan_id) p
                                ON p.rencana_pendapatan_id = r.id
                         WHERE r.tahun = ? ORDER BY COALESCE(r.target_bulan, 13), r.id`, mcpTahun(a)), nil
			},
		},
		{
			Nama: "bku", Akses: AksesBaca,
			Deskripsi: "Buku kas umum satu bulan: transaksi berurutan, debit/kredit, dan saldo berjalan.",
			Skema: mcpObj(OM{
				{"tahun", mcpI("Tahun anggaran.")},
				{"bulan", mcpI("Bulan 1–12.")},
			}),
			Handler: func(a McpArgs) (any, error) {
				tahun := mcpTahun(a)
				bulan := int(a.I("bulan"))
				if bulan < 1 || bulan > 12 {
					bulan = 1
				}
				t0 := time.Date(tahun, time.Month(bulan), 1, 0, 0, 0, 0, time.Local)
				awal := t0.Format("2006-01-02")
				akhir := t0.AddDate(0, 1, -1).Format("2006-01-02")
				sebelum := t0.AddDate(0, 0, -1).Format("2006-01-02")
				rows := Q.All(`SELECT * FROM (
                           SELECT 'trm' AS src, id, tgl, no_bukti, uraian, jumlah AS debit, 0 AS kredit
                             FROM penerimaan WHERE tgl BETWEEN ?1 AND ?2
                           UNION ALL
                           SELECT 'bel', id, tgl, no_bukti, uraian, 0, total FROM belanja WHERE tgl BETWEEN ?1 AND ?2
                           UNION ALL
                           SELECT 'mut', id, tgl, no_bukti, uraian,
                                  CASE jenis WHEN 'bunga' THEN jumlah ELSE 0 END,
                                  CASE jenis WHEN 'admin_bank' THEN jumlah ELSE 0 END
                             FROM mutasi_kas WHERE tgl BETWEEN ?1 AND ?2
                         ) ORDER BY tgl, CASE src WHEN 'trm' THEN 0 WHEN 'mut' THEN 1 ELSE 2 END, id`, awal, akhir)
				saldoAwal := SaldoKas(sebelum, 0, 0)
				saldo := saldoAwal.Total
				for _, r := range rows {
					saldo += r.F("debit") - r.F("kredit")
					r["saldo"] = saldo
				}
				return OM{
					{"tahun", tahun},
					{"bulan", bulan},
					{"saldo_awal", saldoAwal},
					{"saldo_akhir", SaldoKas(akhir, 0, 0)},
					{"transaksi", rows},
				}, nil
			},
		},
		{
			Nama: "saldo", Akses: AksesBaca,
			Deskripsi: "Saldo kas tunai, bank, dan total sampai tanggal tertentu.",
			Skema:     mcpObj(OM{{"tgl", mcpS("Tanggal YYYY-MM-DD, default hari ini.")}}),
			Handler: func(a McpArgs) (any, error) {
				tgl := mcpTrim(a, "tgl")
				if !ValidDate(tgl) {
					tgl = Today()
				}
				return OM{{"tgl", tgl}, {"saldo", SaldoKas(tgl, 0, 0)}}, nil
			},
		},
		mcpLaporanListTool(),
		mcpLaporanTool(),
	}
}

func mcpWhere(w []string) string {
	if len(w) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w, " AND ")
}

func inStrings(v string, list ...string) bool {
	for _, x := range list {
		if v == x {
			return true
		}
	}
	return false
}

package app

import "fmt"

// mcpLaporanListTool: daftar jenis laporan yang tersedia.
func mcpLaporanListTool() McpTool {
	return McpTool{
		Nama: "laporan_list", Akses: AksesBaca,
		Deskripsi: "Jenis laporan yang tersedia beserta penjelasannya.",
		Skema:     mcpObj(OM{}),
		Handler: func(a McpArgs) (any, error) {
			out := []OM{}
			for _, j := range LapJenis {
				out = append(out, OM{{"jenis", j.Kode}, {"nama", j.Nama}, {"keterangan", j.Desc}})
			}
			return out, nil
		},
	}
}

// mcpLaporanTool: data tabel satu laporan untuk periode tertentu.
func mcpLaporanTool() McpTool {
	return McpTool{
		Nama: "laporan", Akses: AksesBaca,
		Deskripsi: "Data satu laporan (tabel siap tampil) untuk periode tertentu.",
		Skema: mcpObj(OM{
			{"jenis", mcpS("bku, tunai, bank, pajak, komponen, realisasi, atau kas.")},
			{"periode", mcpS("1–12, T1, T2, atau Y. Default Y.")},
			{"tahun", mcpI("Tahun anggaran.")},
		}, "jenis"),
		Handler: func(a McpArgs) (any, error) {
			jenis := mcpTrim(a, "jenis")
			if _, ok := lapJenisNama(jenis); !ok {
				return nil, fmt.Errorf("jenis laporan tidak dikenal: %s", jenis)
			}
			kode := mcpTrim(a, "periode")
			if kode == "" {
				kode = "Y"
			}
			tahun := mcpTahun(a)
			p, ok := LapPeriode(kode, tahun)
			if !ok {
				return nil, fmt.Errorf("periode tidak dikenal: %s", kode)
			}
			d := LapBuild(jenis, p)
			cols := OM{}
			for _, c := range d.Cols {
				cols = append(cols, KV{c.Key, []any{c.Label, c.Type}})
			}
			summary := []any{}
			for _, s := range d.Summary {
				summary = append(summary, []any{s[0], s[1]})
			}
			out := OM{
				{"title", d.Title},
				{"periode", p.S("kode")},
				{"tahun", tahun},
				{"label", p.S("label")},
				{"columns", cols},
				{"rows", d.Rows},
				{"foot", d.Foot},
				{"summary", summary},
				{"orientation", d.Orientation},
			}
			if d.Kas != nil {
				out = append(out, KV{"kas", d.Kas})
			}
			return out, nil
		},
	}
}

// mcpRencanaDetailTool: rincian satu paket rencana + transaksi yang memakainya.
func mcpRencanaDetailTool() McpTool {
	return McpTool{
		Nama: "rencana_detail", Akses: AksesBaca,
		Deskripsi: "Rincian satu paket rencana: daftar barang/penerima, belanja yang sudah tercatat, sisa pagu dan periode.",
		Skema:     mcpObj(OM{{"id", mcpI("ID paket rencana.")}}, "id"),
		Handler: func(a McpArgs) (any, error) {
			id := a.I("id")
			r := RencanaDetail(id)
			if r == nil {
				return nil, fmt.Errorf("rencana belanja id %d tidak ditemukan", id)
			}
			r["belanja"] = Q.All(`SELECT id, tgl, no_bukti, uraian, jumlah_periode, total, metode
                FROM belanja WHERE rencana_id = ? ORDER BY tgl, id`, id)
			return r, nil
		},
	}
}

package app

// Tool MCP undangan & notulen rapat: daftar, detail, simpan, hapus.
func init() { RegisterMcp(3, mcpRapatDaftar) }

func mcpRapatDaftar() []McpTool {
	return []McpTool{
		{
			Nama:      "rapat_list",
			Akses:     AksesBaca,
			Deskripsi: "Daftar undangan & notulen rapat pada tahun anggaran, beserta belanja BKU yang mendasarinya.",
			Skema: mcpObj(OM{
				{"tahun", mcpI("Tahun anggaran, default tahun default aplikasi.")},
				{"komponen", mcpS("Kode komponen untuk menyaring (opsional).")},
			}),
			Handler: func(a McpArgs) (any, error) {
				tahun := mcpTahun(a)
				sql := `SELECT p.id, p.tgl, p.tgl_surat, p.no_undangan, p.sifat, p.perihal, p.agenda,
                        p.tempat, p.jam_mulai, p.jam_selesai, p.belanja_id,
                        b.no_bukti, b.uraian AS belanja_uraian, b.total AS belanja_total,
                        r.komponen_kode, k.nama AS komponen_nama
                    FROM rapat p JOIN belanja b ON b.id = p.belanja_id
                    LEFT JOIN rencana_belanja r ON r.id = b.rencana_id
                    LEFT JOIN komponen k ON k.kode = r.komponen_kode
                    WHERE p.tahun = ?`
				args := []any{tahun}
				if k := mcpTrim(a, "komponen"); k != "" {
					sql += " AND r.komponen_kode = ?"
					args = append(args, k)
				}
				rows := Q.All(sql+" ORDER BY p.tgl, p.id", args...)
				return OM{{"tahun", tahun}, {"jumlah", len(rows)}, {"rapat", rows}}, nil
			},
		},
		{
			Nama:      "rapat_detail",
			Akses:     AksesBaca,
			Deskripsi: "Detail satu undangan & notulen rapat, termasuk isi notulen.",
			Skema:     mcpObj(OM{{"id", mcpI("ID rapat.")}}, "id"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				row := RapatFind(id)
				if err := mcpWajibAda(row, "Rapat", id); err != nil {
					return nil, err
				}
				return rapatDoc(id, int(row.I("tahun"))), nil
			},
		},
		{
			Nama:      "rapat_simpan",
			Akses:     AksesTulis,
			Deskripsi: "Buat undangan & notulen rapat (tanpa id) atau ubah (dengan id). Rapat wajib terikat pada satu belanja BKU; nomor undangan otomatis bila kosong.",
			Skema: mcpObj(OM{
				{"id", mcpI("Isi untuk mengubah rapat yang sudah ada.")},
				{"belanja_id", mcpI("ID belanja BKU yang mendanai rapat (wajib).")},
				{"tgl", mcpS("Tanggal rapat YYYY-MM-DD.")},
				{"tgl_surat", mcpS("Tanggal surat undangan YYYY-MM-DD, default sama dengan tgl.")},
				{"no_undangan", mcpS("Nomor undangan; kosongkan agar dibuat otomatis.")},
				{"sifat", mcpEnum("Sifat surat.", RapatSifat)},
				{"lampiran", mcpS("Lampiran surat, mis. 1 (satu) lembar.")},
				{"perihal", mcpS("Perihal surat undangan.")},
				{"kepada", mcpS("Tujuan undangan.")},
				{"jam_mulai", mcpS("Jam mulai HH:MM.")},
				{"jam_selesai", mcpS("Jam selesai HH:MM.")},
				{"tempat", mcpS("Tempat rapat.")},
				{"agenda", mcpS("Agenda/maksud rapat.")},
				{"pimpinan", mcpS("Pimpinan rapat.")},
				{"notulis", mcpS("Notulis rapat.")},
				{"jumlah_peserta", mcpI("Jumlah peserta hadir.")},
				{"pembukaan", mcpS("Isi pembukaan notulen.")},
				{"pembahasan", mcpS("Pokok pembahasan; satu pokok per baris. Bila diisi, notulen PDF tersedia.")},
				{"keputusan", mcpS("Keputusan rapat.")},
				{"tindak_lanjut", mcpS("Rencana tindak lanjut.")},
			}, "belanja_id", "tgl"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				var lama Row
				if id != 0 {
					lama = RapatFind(id)
					if err := mcpWajibAda(lama, "Rapat", id); err != nil {
						return nil, err
					}
				}
				tahun, err := mcpTahunDari(a, lama)
				if err != nil {
					return nil, err
				}
				f := mcpForm(a)
				// Ubah sebagian: isian yang tidak dikirim memakai nilai lama.
				if lama != nil {
					for _, k := range []string{"belanja_id", "no_undangan", "tgl_surat", "sifat", "lampiran",
						"perihal", "kepada", "tgl", "jam_mulai", "jam_selesai", "tempat", "agenda",
						"pimpinan", "notulis", "jumlah_peserta", "pembukaan", "pembahasan",
						"keputusan", "tindak_lanjut"} {
						if _, ok := a[k]; !ok {
							f[k] = toStr(lama[k])
						}
					}
				}
				if f.Str("tgl_surat") == "" {
					f["tgl_surat"] = f.Str("tgl")
				}
				if f.Str("tempat") == "" {
					f["tempat"] = defStr(Sekolah().S("nama"), "Ruang Guru")
				}
				data, err := RapatValidate(f, id, tahun)
				if err != nil {
					return nil, err
				}
				if id != 0 {
					RapatUpdate(id, data)
				} else {
					id = RapatCreate(data)
				}
				return mcpOkID(id, RapatFind(id)), nil
			},
		},
		{
			Nama:      "rapat_hapus",
			Akses:     AksesTulis,
			Deskripsi: "Hapus undangan & notulen rapat. Transaksi belanja BKU tidak ikut terhapus.",
			Skema:     mcpHapusSkema("rapat"),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				row := RapatFind(id)
				if err := mcpWajibAda(row, "Rapat", id); err != nil {
					return nil, err
				}
				RapatDelete(row)
				return mcpOkHapus(id), nil
			},
		},
	}
}

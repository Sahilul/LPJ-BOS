package app

// Tool MCP undangan & notulen rapat: daftar, detail, simpan, hapus.
func init() { RegisterMcp(3, mcpRapatDaftar) }

// mcpRapatTahun: tahun anggaran rapat mengikuti belanja yang mendasarinya
// (belanja_id baru, lalu rapat lama), baru jatuh ke tgl/param tahun.
func mcpRapatTahun(a McpArgs, lama Row) (int, error) {
	bid := a.I("belanja_id")
	if bid == 0 && lama != nil {
		bid = lama.I("belanja_id")
	}
	if t := rapatTahunBelanja(RapatBelanjaAda(bid)); t != 0 {
		if !InTahunList(t) {
			return 0, mcpError("Tahun anggaran %d belum terdaftar. Tambahkan dulu di menu Tahun anggaran.", t)
		}
		return t, nil
	}
	if lama != nil {
		return int(lama.I("tahun")), nil
	}
	return mcpTahunDari(a, nil)
}

// rapatMcpDetail: isi rapat + belanja dasar + tautan PDF, tanpa data kop/sekolah.
func rapatMcpDetail(id int64, tahun int) OM {
	d := rapatDoc(id, tahun)
	if d == nil {
		return nil
	}
	out := OM{}
	for _, k := range []string{"id", "tahun", "belanja_id", "no_undangan", "tgl_surat", "sifat", "lampiran",
		"perihal", "kepada", "tgl", "jam_mulai", "jam_selesai", "tempat", "agenda", "pimpinan",
		"notulis", "jumlah_peserta", "pembukaan", "pembahasan", "keputusan", "tindak_lanjut",
		"no_bukti", "belanja_uraian", "belanja_total", "belanja_tgl", "komponen_kode", "komponen_nama"} {
		out = append(out, KV{k, d[k]})
	}
	notulen := d.S("pembahasan") != ""
	pdf := OM{{"undangan", "/dokumen/rapat/" + itoa(int(id)) + "/undangan.pdf"}}
	if notulen {
		pdf = append(pdf, KV{"notulen", "/dokumen/rapat/" + itoa(int(id)) + "/notulen.pdf"})
	}
	return append(out, KV{"ada_notulen", notulen}, KV{"pdf", pdf})
}

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
                        p.tempat, p.jam_mulai, p.jam_selesai, p.belanja_id, p.pembahasan,
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
				for _, r := range rows {
					r["ada_notulen"] = r.S("pembahasan") != ""
					delete(r, "pembahasan")
				}
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
				return rapatMcpDetail(id, int(row.I("tahun"))), nil
			},
		},
		{
			Nama:  "rapat_simpan",
			Akses: AksesTulis,
			Deskripsi: "Buat undangan & notulen rapat (tanpa id; belanja_id, tgl, dan agenda wajib) atau ubah sebagian (dengan id; isian yang tidak dikirim tetap). " +
				"Rapat wajib terikat pada satu belanja BKU dan satu belanja hanya boleh punya satu rapat; nomor undangan otomatis bila kosong.",
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
			}),
			Handler: func(a McpArgs) (any, error) {
				id := a.I("id")
				var lama Row
				if id != 0 {
					lama = RapatFind(id)
					if err := mcpWajibAda(lama, "Rapat", id); err != nil {
						return nil, err
					}
				}
				tahun, err := mcpRapatTahun(a, lama)
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
				return mcpOkID(id, rapatMcpDetail(id, tahun)), nil
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

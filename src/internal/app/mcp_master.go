package app

import (
	"fmt"
)

// Tool MCP untuk data master penerima dan toko. Validasi sama dengan form web.
func init() { RegisterMcp(2, mcpMasterDaftar) }

func mcpMasterDaftar() []McpTool {
	var out []McpTool
	for _, slug := range []string{"penerima", "toko"} {
		r, _ := MasterBySlug(slug)
		out = append(out, mcpMasterSimpan(r), mcpMasterHapus(r))
	}
	return out
}

// mcpMasterSkema membangun skema JSON dari definisi field master.
func mcpMasterSkema(r *MasterRes) OM {
	props := OM{{"id", mcpI("ID " + r.Singular + " untuk mengubah; kosongkan untuk menambah.")}}
	for _, f := range r.Fields {
		desc := f.Label + "."
		if f.Hint != "" {
			desc += " " + f.Hint
		}
		if f.Type == "digits" && f.Len > 0 {
			desc += fmt.Sprintf(" %d angka, boleh kosong.", f.Len)
		}
		var p OM
		switch f.Type {
		case "select":
			p = mcpEnum(desc, f.Options)
		case "money", "percent":
			p = mcpN(desc)
		case "int":
			p = mcpI(desc)
		case "bool":
			p = mcpB(desc)
		default:
			p = mcpS(desc)
		}
		props = append(props, KV{f.Name, p})
	}
	if r.Toggle {
		props = append(props, KV{"aktif", mcpB("Status aktif (bawaan aktif).")})
	}
	return mcpObj(props)
}

func mcpMasterSimpan(r *MasterRes) McpTool {
	return McpTool{
		Nama:  r.Slug + "_simpan",
		Akses: AksesTulis,
		Deskripsi: "Tambah (tanpa id) atau ubah (dengan id) data " + r.Singular + ". Saat ubah cukup kirim field yang berubah. " +
			r.Lead,
		Skema: mcpMasterSkema(r),
		Handler: func(a McpArgs) (any, error) {
			id := a.I("id")
			lama := Row{}
			if id != 0 {
				if lama = Q.One("SELECT * FROM "+r.Table+" WHERE id = ?", id); lama == nil {
					return nil, mcpError("%s dengan id %d tidak ditemukan.", ucfirst(r.Singular), id)
				}
			}
			form := Form{}
			for _, f := range r.Fields {
				switch {
				case a.Has(f.Name):
					form[f.Name] = a.S(f.Name)
				case id != 0:
					form[f.Name] = lama.S(f.Name)
				case f.Default != nil:
					form[f.Name] = toStr(f.Default)
				}
			}
			data, errs := MasterValidate(r, form, id)
			if len(errs) > 0 {
				e := NewValidationOrdered()
				for _, f := range r.Fields {
					if m, ok := errs[f.Name]; ok {
						e.Set(f.Name, m)
					}
				}
				return nil, e
			}
			if r.Toggle && a.Has("aktif") {
				data["aktif"] = 0
				if !phpEmpty(a["aktif"]) {
					data["aktif"] = 1
				}
			}
			if id != 0 {
				Q.Update(r.Table, data, id)
			} else {
				id = Q.Insert(r.Table, data)
			}
			return mcpOkID(id, Q.One("SELECT * FROM "+r.Table+" WHERE id = ?", id)), nil
		},
	}
}

func mcpMasterHapus(r *MasterRes) McpTool {
	return McpTool{
		Nama:      r.Slug + "_hapus",
		Akses:     AksesTulis,
		Deskripsi: "Hapus data " + r.Singular + " yang belum dipakai transaksi. Bila sudah dipakai, nonaktifkan lewat " + r.Slug + "_simpan (aktif=false).",
		Skema:     mcpHapusSkema(r.Singular),
		Handler: func(a McpArgs) (any, error) {
			id := a.I("id")
			if err := mcpWajibAda(Q.One("SELECT id FROM "+r.Table+" WHERE id = ?", id), ucfirst(r.Singular), id); err != nil {
				return nil, err
			}
			ok := true
			func() {
				defer func() {
					if recover() != nil {
						ok = false
					}
				}()
				Q.Run("DELETE FROM "+r.Table+" WHERE id = ?", id)
			}()
			if !ok {
				return nil, mcpError("%s sudah dipakai di transaksi sehingga tidak bisa dihapus. Nonaktifkan saja (%s_simpan dengan aktif=false).",
					ucfirst(r.Singular), r.Slug)
			}
			return mcpOkHapus(id), nil
		},
	}
}

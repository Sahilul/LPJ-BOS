package app

import (
	"fmt"
	"strings"
)

func optHas(opts []Opt, v string) bool {
	for _, o := range opts {
		if o.K == v {
			return true
		}
	}
	return false
}

// optText: label pilihan, atau nilai mentah bila tidak dikenal.
func optText(opts []Opt, v string) string {
	if l, ok := optLabel(opts, v); ok {
		return l
	}
	return v
}

// ucfirst meniru ucfirst() PHP untuk teks biasa.
func ucfirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func masterFind(c *Ctx, r *MasterRes) Row {
	row := Q.One("SELECT * FROM "+r.Table+" WHERE id = ?", c.ParamInt("id"))
	if row == nil {
		c.Abort(404, "Data tidak ditemukan.")
	}
	return row
}

// masterValidated menormalkan isian POST sesuai tipe field, lalu memvalidasi.
func masterValidated(c *Ctx, r *MasterRes, id int64, back string) Row {
	data, errs := MasterValidate(r, c.Form, id)
	if len(errs) > 0 {
		c.WithOld(c.Form, errs)
		c.Redirect(back)
	}
	return data
}

// MasterValidate menormalkan isian sesuai tipe field lalu memvalidasi (dipakai web dan MCP).
func MasterValidate(r *MasterRes, form Form, id int64) (Row, map[string]string) {
	data := Row{}
	errs := map[string]string{}
	for i := range r.Fields {
		f := &r.Fields[i]
		raw := form.Str(f.Name)
		var v any
		switch f.Type {
		case "bool":
			if raw == "" || raw == "0" {
				v = int64(0)
			} else {
				v = int64(1)
			}
		case "money":
			v = Uang(raw)
		case "int":
			v = toInt(raw)
		case "percent":
			v = toFloat(strings.ReplaceAll(raw, ",", "."))
		case "digits":
			v = Digits(raw)
		default:
			v = strings.TrimSpace(raw)
		}
		sv := toStr(v)
		if f.Required && sv == "" {
			errs[f.Name] = "Wajib diisi."
		}
		if f.Type == "select" && !optHas(f.Options, sv) {
			errs[f.Name] = "Pilihan tidak valid."
		}
		if f.Type == "digits" && sv != "" && f.Len > 0 && len(sv) != f.Len {
			errs[f.Name] = fmt.Sprintf("%s terdiri dari %d angka.", f.Label, f.Len)
		}
		if f.Type == "percent" && (toFloat(v) < 0 || toFloat(v) > 100) {
			errs[f.Name] = "Isi antara 0 dan 100."
		}
		if f.Unique && sv != "" {
			if Q.Int("SELECT id FROM "+r.Table+" WHERE "+f.Name+" = ? AND id <> ?", sv, id) != 0 {
				errs[f.Name] = "Sudah ada data dengan " + strings.ToLower(f.Label) + " ini."
			}
		}
		data[f.Name] = v
	}
	return data, errs
}

// masterDestroy menghapus satu baris; galat FK berarti data sudah dipakai transaksi.
func masterDestroy(c *Ctx, r *MasterRes) {
	row := masterFind(c, r)
	if r.Slug == "komponen" && Q.Int("SELECT COUNT(*) FROM rencana_belanja WHERE komponen_kode = ?", row.S("kode")) > 0 {
		c.Flash("danger", "Tidak bisa dihapus karena sudah dipakai di transaksi.")
		c.Redirect("/" + r.Slug)
	}
	ok := true
	func() {
		defer func() {
			if recover() != nil {
				ok = false
			}
		}()
		Q.Run("DELETE FROM "+r.Table+" WHERE id = ?", row.I("id"))
	}()
	if !ok {
		msg := ""
		if r.Toggle {
			msg = " Nonaktifkan saja agar riwayatnya tetap utuh."
		}
		c.Flash("danger", "Tidak bisa dihapus karena sudah dipakai di transaksi."+msg)
		c.Redirect("/" + r.Slug)
	}
	c.Flash("success", ucfirst(r.Singular)+" dihapus.")
	c.Redirect("/" + r.Slug)
}

func masterIndex(c *Ctx, r *MasterRes) {
	q := strings.TrimSpace(c.Q("q"))
	where := []string{}
	args := []any{}
	if q != "" {
		ors := make([]string, len(r.Search))
		for i, col := range r.Search {
			ors[i] = col + " LIKE ?"
			args = append(args, "%"+q+"%")
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}
	filterVal := ""
	if r.FilterCol != "" {
		filterVal = c.Q(r.FilterCol)
		if filterVal != "" && optHas(r.FilterOpts, filterVal) {
			where = append(where, r.FilterCol+" = ?")
			args = append(args, filterVal)
		}
	}
	status := c.Q("status")
	if r.Toggle && status != "" {
		where = append(where, "aktif = ?")
		if status == "nonaktif" {
			args = append(args, 0)
		} else {
			args = append(args, 1)
		}
	}
	sql := "SELECT * FROM " + r.Table
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	sql += " ORDER BY " + r.Order
	rows := Q.All(sql, args...)
	total := Q.Int("SELECT COUNT(*) FROM " + r.Table)

	headers := []D{}
	for _, col := range r.List {
		f := r.Field(col)
		lbl := r.LabelOf(col)
		if i := strings.Index(lbl, " ("); i > 0 {
			lbl = lbl[:i]
		}
		headers = append(headers, D{"label": lbl, "num": f != nil && isNumField(f)})
	}
	list := make([]masterRowView, 0, len(rows))
	for _, row := range rows {
		v := masterRowView{R: row, Label: MasterLabel(row), Off: r.Toggle && row.I("aktif") == 0}
		for _, col := range r.List {
			v.Cells = append(v.Cells, masterCellHTML(r.Field(col), row))
		}
		list = append(list, v)
	}
	c.View("crud/index", D{
		"title":     r.Title,
		"res":       r,
		"rows":      list,
		"total":     total,
		"q":         q,
		"headers":   headers,
		"filterVal": filterVal,
		"status":    status,
		"filtered":  q != "" || filterVal != "" || status != "",
		"searchPh":  "Cari " + strings.Join(r.SearchLabels(), ", "),
	})
}

func masterForm(c *Ctx, r *MasterRes, row Row) {
	edit := row != nil
	if row == nil {
		row = Row{}
		for i := range r.Fields {
			if r.Fields[i].Default != nil {
				row[r.Fields[i].Name] = r.Fields[i].Default
			}
		}
	}
	c.loadOld()
	fields := make([]masterFormField, 0, len(r.Fields))
	for i := range r.Fields {
		f := &r.Fields[i]
		typ := f.Type
		hint := f.Hint
		extra := ""
		switch f.Type {
		case "digits":
			typ = "text"
			extra = `inputmode="numeric"`
			if f.Len > 0 {
				extra += fmt.Sprintf(` maxlength="%d"`, f.Len+4)
			}
		case "int":
			typ = "number"
			extra = `inputmode="numeric"`
		}
		if hint == "" && f.Len > 0 {
			hint = fmt.Sprintf("%d angka.", f.Len)
		}
		fields = append(fields, masterFormField{
			Name: f.Name, Label: f.Label, Type: typ, Options: f.Options,
			Value:    c.OldVal(f.Name, row[f.Name]),
			Required: f.Required, Mono: f.Mono || f.Type == "digits",
			Span2: f.Span2, Hint: hint, Extra: extra,
		})
	}
	title := "Tambah " + r.Singular
	head := title
	if edit {
		title = "Ubah " + r.Singular
		head = MasterLabel(row)
	}
	c.View("crud/form", D{
		"title": title, "head": head, "res": r, "row": row, "edit": edit, "fields": fields,
	})
}

func init() {
	Register(20, func(rt *Router) {
		RegisterPenerimaExcel(rt)
		for _, res := range masterResAll {
			r := res
			base := "/" + r.Slug
			rt.Get(base, func(c *Ctx) { masterIndex(c, r) })
			rt.Get(base+"/baru", func(c *Ctx) { masterForm(c, r, nil) })
			rt.Post(base, func(c *Ctx) {
				data := masterValidated(c, r, 0, base+"/baru")
				id := Q.Insert(r.Table, data)
				c.Flash("success", ucfirst(r.Singular)+" “"+MasterLabel(data)+"” ditambahkan.")
				if c.Form.Truthy("_lagi") {
					c.Redirect(base + "/baru")
				}
				c.Redirect(fmt.Sprintf("%s#row-%d", base, id))
			})
			rt.Get(base+"/{id}/edit", func(c *Ctx) {
				masterForm(c, r, masterFind(c, r))
			})
			rt.Post(base+"/{id}", func(c *Ctx) {
				masterFind(c, r)
				id := c.ParamInt("id")
				data := masterValidated(c, r, id, fmt.Sprintf("%s/%d/edit", base, id))
				Q.Update(r.Table, data, id)
				c.Flash("success", "Perubahan disimpan.")
				c.Redirect(fmt.Sprintf("%s#row-%d", base, id))
			})
			rt.Post(base+"/{id}/hapus", func(c *Ctx) { masterDestroy(c, r) })
			if r.Toggle {
				rt.Post(base+"/{id}/aktif", func(c *Ctx) {
					row := masterFind(c, r)
					nw := int64(0)
					if row.I("aktif") == 0 {
						nw = 1
					}
					Q.Run("UPDATE "+r.Table+" SET aktif = ? WHERE id = ?", nw, row.I("id"))
					msg := " dinonaktifkan."
					if nw == 1 {
						msg = " diaktifkan kembali."
					}
					c.Flash("success", row.S("nama")+msg)
					c.Back(base)
				})
			}
		}
	})
}

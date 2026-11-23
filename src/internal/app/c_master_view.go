package app

import "html/template"

// masterRowView adalah satu baris tabel data master dengan sel siap tampil.
type masterRowView struct {
	R     Row
	Cells []template.HTML
	Off   bool
	Label string
}

// masterFormField adalah satu isian form data master.
type masterFormField struct {
	Name, Label, Type     string
	Options               []Opt
	Value                 any
	Required, Mono, Span2 bool
	Hint, Extra           string
}

// isNumField: kolom angka (rata kanan).
func isNumField(f *MasterField) bool {
	return f.Type == "money" || f.Type == "percent"
}

// masterCellHTML merender isi satu sel seperti crud/index.php (badge, mono, rupiah).
func masterCellHTML(f *MasterField, row Row) template.HTML {
	if f == nil {
		return ""
	}
	v := row[f.Name]
	s := toStr(v)
	esc := template.HTMLEscapeString
	muted := template.HTML(`<span class="muted">—</span>`)
	switch f.Type {
	case "money":
		if toFloat(v) > 0 {
			return template.HTML(`<span class="num">` + esc(Rupiah(v)) + `</span>`)
		}
		return muted
	case "percent":
		return template.HTML(`<span class="num">` + esc(AngkaD(v, 2)) + `%</span>`)
	case "bool":
		if toFloat(v) != 0 {
			return template.HTML(`<span class="badge badge-accent">Ya</span>`)
		}
		return template.HTML(`<span class="muted">Tidak</span>`)
	case "select":
		if s == "" {
			return muted
		}
		return template.HTML(esc(optText(f.Options, s)))
	default:
		if s == "" {
			return muted
		}
		if f.Mono || f.Type == "digits" {
			return template.HTML(`<span class="mono">` + esc(s) + `</span>`)
		}
		return template.HTML(esc(s))
	}
}

package app

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"math"
	"net/http"
	"sort"
	"strings"
)

//go:embed templates
var templatesFS embed.FS

var tmpl *template.Template

// D adalah data template.
type D map[string]any

func Icon(name string, size ...int) template.HTML {
	sz := 18
	if len(size) > 0 {
		sz = size[0]
	}
	p, ok := iconPaths[name]
	if !ok {
		p = iconPaths["file"]
	}
	return template.HTML(fmt.Sprintf(`<svg class="ic" width="%d" height="%d" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">%s</svg>`, sz, sz, p))
}

func dict(kv ...any) D {
	d := D{}
	for i := 0; i+1 < len(kv); i += 2 {
		d[toStr(kv[i])] = kv[i+1]
	}
	return d
}

// truthy meniru !empty() PHP.
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return x != "" && x != "0"
	case bool:
		return x
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	case []Row:
		return len(x) > 0
	case Row:
		return len(x) > 0
	case Form:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	case D:
		return len(x) > 0
	case []any:
		return len(x) > 0
	case []string:
		return len(x) > 0
	case template.HTML:
		return x != ""
	}
	return true
}

func isNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	var f float64
	_, err := fmt.Sscan(s, &f)
	return err == nil && strings.Trim(s, "0123456789.-+eE") == ""
}

// Field merender satu isian form (pengganti partials/field.php).
// Argumen pasangan kunci: name, label, value, type, options, required, hint, mono, span2, extra.
func (c *Ctx) Field(kv ...any) template.HTML {
	a := dict(kv...)
	name := toStr(a["name"])
	label := toStr(a["label"])
	typ := toStr(a["type"])
	if typ == "" {
		typ = "text"
	}
	value := a["value"]
	err := c.FieldError(name)
	id := "f-" + name
	esc := template.HTMLEscapeString
	attrs := ""
	if truthy(a["required"]) {
		attrs += " required"
	}
	if err != "" {
		attrs += ` aria-invalid="true" aria-describedby="` + id + `-err"`
	} else if truthy(a["hint"]) {
		attrs += ` aria-describedby="` + id + `-hint"`
	}
	if truthy(a["extra"]) {
		attrs += " " + toStr(a["extra"])
	}
	cls := "input"
	if truthy(a["mono"]) {
		cls += " mono"
	}
	var b strings.Builder
	b.WriteString(`<div class="field`)
	if truthy(a["span2"]) {
		b.WriteString(" span-2")
	}
	b.WriteString("\">\n")
	if typ == "bool" {
		b.WriteString(`<span class="label" aria-hidden="true">&nbsp;</span>`)
		fmt.Fprintf(&b, `<label class="check" for="%s"><input type="hidden" name="%s" value="0"><input type="checkbox" id="%s" name="%s" value="1"`, id, esc(name), id, esc(name))
		if truthy(value) {
			b.WriteString(" checked")
		}
		fmt.Fprintf(&b, `%s><span>%s</span></label>`, attrs, esc(label))
	} else {
		fmt.Fprintf(&b, `<label for="%s">%s`, id, esc(label))
		if truthy(a["required"]) {
			b.WriteString(`<span class="req" aria-hidden="true">*</span>`)
		}
		b.WriteString("</label>\n")
		sv := toStr(value)
		switch typ {
		case "textarea":
			fmt.Fprintf(&b, `<textarea class="%s" id="%s" name="%s" rows="3"%s>%s</textarea>`, cls, id, esc(name), attrs, esc(sv))
		case "select":
			fmt.Fprintf(&b, `<select class="%s" id="%s" name="%s"%s>`, cls, id, esc(name), attrs)
			for _, o := range toOptions(a["options"]) {
				sel := ""
				if o[0] == sv {
					sel = " selected"
				}
				fmt.Fprintf(&b, `<option value="%s"%s>%s</option>`, esc(o[0]), sel, esc(o[1]))
			}
			b.WriteString("</select>")
		case "money":
			var mv float64
			if isNumeric(sv) {
				mv = toFloat(sv)
			} else {
				mv = toFloat(Digits(sv))
			}
			val := ""
			if mv > 0 {
				val = Angka(mv)
			}
			fmt.Fprintf(&b, `<div class="input-group"><span class="addon">Rp</span><input class="input mono" id="%s" name="%s" value="%s" inputmode="numeric" data-money autocomplete="off"%s></div>`, id, esc(name), val, attrs)
		case "percent":
			val := strings.ReplaceAll(phpFloatStr(toFloat(value)), ".", ",")
			fmt.Fprintf(&b, `<div class="input-group"><input class="input mono" id="%s" name="%s" value="%s" inputmode="decimal" autocomplete="off"%s><span class="addon end">%%</span></div>`, id, esc(name), esc(val), attrs)
		default:
			fmt.Fprintf(&b, `<input class="%s" id="%s" name="%s" type="%s" value="%s"%s>`, cls, id, esc(name), esc(typ), esc(sv), attrs)
		}
	}
	if err != "" {
		fmt.Fprintf(&b, "\n"+`<span class="error" id="%s-err">%s</span>`, id, esc(err))
	} else if truthy(a["hint"]) {
		fmt.Fprintf(&b, "\n"+`<span class="hint" id="%s-hint">%s</span>`, id, esc(toStr(a["hint"])))
	}
	b.WriteString("\n</div>")
	return template.HTML(b.String())
}

// Opt adalah satu pilihan select dengan urutan terjaga.
type Opt struct{ K, V string }

func toOptions(v any) [][2]string {
	out := [][2]string{}
	switch x := v.(type) {
	case []Opt:
		for _, o := range x {
			out = append(out, [2]string{o.K, o.V})
		}
	case map[string]string:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, [2]string{k, x[k]})
		}
	}
	return out
}

func (c *Ctx) CSRFField() template.HTML {
	return template.HTML(`<input type="hidden" name="_csrf" value="` + template.HTMLEscapeString(c.CSRFToken()) + `">`)
}

var funcs = template.FuncMap{
	"icon":      Icon,
	"rupiah":    RupiahWeb,
	"rupiahC":   RupiahCell,
	"rupiahN":   func(v any) string { return RupiahP(v, false) },
	"angka":     Angka,
	"angkaD":    AngkaD,
	"persen":    Persen,
	"terbilang": Terbilang,
	"tglIndo":   TanggalIndo,
	"tglShort":  TanggalIndoShort,
	"bulanIndo": BulanIndo,
	"asset":     Asset,
	"uploadURL": UploadURL,
	"dict":      dict,
	"truthy":    truthy,
	"empty":     func(v any) bool { return !truthy(v) },
	"str":       toStr,
	"int":       toInt,
	"float":     toFloat,
	"digits":    Digits,
	"today":     Today,
	"safe":      func(s string) template.HTML { return template.HTML(s) },
	"safeAttr":  func(s string) template.HTMLAttr { return template.HTMLAttr(s) },
	"safeURL":   func(s string) template.URL { return template.URL(s) },
	"eq2":       func(a, b any) bool { return toStr(a) == toStr(b) },
	"add":       func(a, b any) float64 { return toFloat(a) + toFloat(b) },
	"sub":       func(a, b any) float64 { return toFloat(a) - toFloat(b) },
	"mul":       func(a, b any) float64 { return toFloat(a) * toFloat(b) },
	"div": func(a, b any) float64 {
		if toFloat(b) == 0 {
			return 0
		}
		return toFloat(a) / toFloat(b)
	},
	"addi":      func(a, b any) int64 { return toInt(a) + toInt(b) },
	"round":     func(a any) float64 { return math.Round(toFloat(a)) },
	"abs":       func(a any) float64 { return math.Abs(toFloat(a)) },
	"lt":        func(a, b any) bool { return toFloat(a) < toFloat(b) },
	"gt":        func(a, b any) bool { return toFloat(a) > toFloat(b) },
	"le":        func(a, b any) bool { return toFloat(a) <= toFloat(b) },
	"ge":        func(a, b any) bool { return toFloat(a) >= toFloat(b) },
	"upper":     strings.ToUpper,
	"lower":     strings.ToLower,
	"trim":      strings.TrimSpace,
	"contains":  strings.Contains,
	"hasPrefix": strings.HasPrefix,
	"join":      strings.Join,
	"substr": func(s string, a, b int) string {
		if a > len(s) {
			return ""
		}
		if b > len(s) {
			b = len(s)
		}
		return s[a:b]
	},
	"default": func(def, v any) any {
		if truthy(v) {
			return v
		}
		return def
	},
	"seq": func(a, b int) []int {
		out := []int{}
		for i := a; i <= b; i++ {
			out = append(out, i)
		}
		return out
	},
	"list": func(v ...any) []any { return v },
	"nl2br": func(s string) template.HTML {
		return template.HTML(strings.ReplaceAll(template.HTMLEscapeString(s), "\n", "<br>\n"))
	},
	"json": func(v any) template.JS { b, _ := jsonMarshal(v); return template.JS(b) },
}

// AddFuncs mendaftarkan fungsi template tambahan (dipanggil sebelum loadTemplates).
func AddFuncs(m template.FuncMap) {
	for k, v := range m {
		funcs[k] = v
	}
}

func loadTemplates() {
	t := template.New("").Funcs(funcs)
	err := fs.WalkDir(templatesFS, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return err
		}
		b, err := templatesFS.ReadFile(p)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(strings.TrimPrefix(p, "templates/"), ".html")
		_, err = t.New(name).Parse(string(b))
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	tmpl = t
}

// RenderString merender template tanpa layout.
func RenderString(name string, data D) string {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		panic(err)
	}
	return buf.String()
}

// View merender halaman: konten lalu layout (app/guest). layout "" = tanpa layout.
func (c *Ctx) View(name string, data D, layout ...string) {
	c.loadOld()
	if data == nil {
		data = D{}
	}
	data["C"] = c
	lay := "app"
	if len(layout) > 0 {
		lay = layout[0]
	}
	content := RenderString(name, data)
	out := content
	if lay != "" {
		data["content"] = template.HTML(content)
		if lay == "app" {
			data["nav"] = navGroups
		}
		out = RenderString("layouts/"+lay, data)
	}
	c.HTML(http.StatusOK, out)
}

func (c *Ctx) HTML(code int, body string) {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.saveSession()
	c.releaseLock()
	c.W.WriteHeader(code)
	_, _ = c.W.Write([]byte(body))
	c.Done()
}

func (c *Ctx) loadOld() {
	if c.Sess.Data.Old != nil || c.Sess.Data.Errors != nil {
		c.Old = normalizeForm(c.Sess.Data.Old)
		c.Errs = c.Sess.Data.Errors
		c.Sess.Data.Old, c.Sess.Data.Errors = nil, nil
	}
	if c.Old == nil {
		c.Old = Form{}
	}
	if c.Errs == nil {
		c.Errs = map[string]string{}
	}
}

func (c *Ctx) renderError(code int, msg string) {
	lay := "guest"
	if c.User() != nil {
		lay = "app"
	}
	c.loadOld()
	data := D{"code": code, "message": msg, "title": fmt.Sprintf("Galat %d", code), "C": c}
	content := RenderString("errors/error", data)
	data["content"] = template.HTML(content)
	data["nav"] = navGroups
	out := RenderString("layouts/"+lay, data)
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.saveSession()
	c.releaseLock()
	c.W.WriteHeader(code)
	_, _ = c.W.Write([]byte(out))
}

type NavLink struct{ Href, Icon, Label string }
type NavGroup struct {
	Label string
	Links []NavLink
}

var navGroups = []NavGroup{
	{"Utama", []NavLink{{"/", "home", "Beranda"}}},
	{"Sekolah", []NavLink{{"/sekolah", "school", "Profil & kop surat"}, {"/pejabat", "badge", "Pejabat & TTD"}}},
	{"Master data", []NavLink{{"/penerima", "users", "Penerima"}, {"/toko", "store", "Toko & penyedia"}, {"/komponen", "layers", "Komponen BOSP"}, {"/rekening", "hash", "Kode rekening"}, {"/pajak", "percent", "Tarif pajak"}}},
	{"Keuangan", []NavLink{{"/rencana", "list", "Rencana belanja"}, {"/penerimaan", "inbox", "Penerimaan dana"}, {"/bku", "book", "Buku kas umum"}, {"/mutasi", "swap", "Mutasi kas"}, {"/laporan", "chart", "Laporan & SPJ"}}},
	{"Sistem", []NavLink{{"/tahun", "calendar", "Tahun anggaran"}, {"/backup", "database", "Cadangkan data"}, {"/api", "plug", "Akses AI (MCP)"}, {"/akun", "key", "Akun"}}},
}

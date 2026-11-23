package app

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// Ctx adalah konteks satu request web.
type Ctx struct {
	W      http.ResponseWriter
	R      *http.Request
	Sess   *Session
	Path   string
	Params map[string]string
	Form   Form // isian POST (bersarang)
	Query  url.Values

	Old  Form
	Errs map[string]string

	user       Row
	userLoaded bool
	sekolah    Row
	written    bool
}

// Sinyal alur kontrol (ditangkap middleware), pengganti exit; di PHP.
type redirectSignal struct{ to string }
type abortSignal struct {
	code int
	msg  string
}
type doneSignal struct{}

func (c *Ctx) Param(k string) string   { return c.Params[k] }
func (c *Ctx) ParamInt(k string) int64 { return toInt(c.Params[k]) }

// Redirect menghentikan handler dan mengalihkan ke URL.
func (c *Ctx) Redirect(to string) { panic(redirectSignal{to}) }

func (c *Ctx) Abort(code int, msg string) { panic(abortSignal{code, msg}) }

// Done menghentikan handler setelah respons ditulis sendiri.
func (c *Ctx) Done() { panic(doneSignal{}) }

func (c *Ctx) Back(fallback string) {
	ref := c.R.Header.Get("Referer")
	if ref != "" {
		if u, err := url.Parse(ref); err == nil {
			host := c.R.Host
			if h, _, ok := strings.Cut(host, ":"); ok {
				host = h
			}
			if u.Hostname() == host {
				c.Redirect(ref)
			}
		}
	}
	c.Redirect(fallback)
}

func (c *Ctx) Flash(typ, msg string) {
	c.Sess.Data.Flash = append(c.Sess.Data.Flash, Flash{typ, msg})
}

func (c *Ctx) TakeFlash() []Flash {
	f := c.Sess.Data.Flash
	c.Sess.Data.Flash = nil
	return f
}

// WithOld menyimpan isian dan pesan galat untuk ditampilkan lagi setelah redirect.
func (c *Ctx) WithOld(input Form, errs map[string]string) {
	if input == nil {
		input = Form{}
	}
	c.Sess.Data.Old = map[string]any(input)
	if errs == nil {
		errs = map[string]string{}
	}
	c.Sess.Data.Errors = errs
}

// OldVal: nilai isian lama (string) atau default bila tidak ada.
func (c *Ctx) OldVal(k string, def any) any {
	if c.Old != nil {
		if v, ok := c.Old[k]; ok && v != nil {
			return v
		}
	}
	return def
}

func (c *Ctx) Posted() bool { return len(c.Old) > 0 }

func (c *Ctx) FieldError(k string) string {
	if c.Errs == nil {
		return ""
	}
	return c.Errs[k]
}

func (c *Ctx) CSRFToken() string {
	if c.Sess.Data.CSRF == "" {
		c.Sess.Data.CSRF = randomHex(32)
	}
	return c.Sess.Data.CSRF
}

func (c *Ctx) csrfValid() bool {
	sent := c.Form.Str("_csrf")
	if sent == "" {
		sent = c.R.Header.Get("X-CSRF-TOKEN")
	}
	tok := c.CSRFToken()
	return sent != "" && subtle.ConstantTimeCompare([]byte(sent), []byte(tok)) == 1
}

// User: pengguna yang login, atau nil.
func (c *Ctx) User() Row {
	if !c.userLoaded {
		c.userLoaded = true
		if c.Sess.Data.UserID != 0 {
			c.user = Q.One("SELECT id, username, nama FROM users WHERE id = ?", c.Sess.Data.UserID)
		}
	}
	return c.user
}

func (c *Ctx) Login(id int64) {
	c.Sess.Regenerate()
	c.Sess.Data.UserID = id
	c.user, c.userLoaded = nil, false
}

func (c *Ctx) Sekolah() Row {
	if c.sekolah == nil {
		c.sekolah = Sekolah()
	}
	return c.sekolah
}

// TahunAktif: tahun anggaran yang sedang dibuka (disimpan di sesi).
func (c *Ctx) TahunAktif() int {
	t := c.Sess.Data.Tahun
	if t != 0 && InTahunList(t) {
		return t
	}
	t = TahunDefault()
	c.Sess.Data.Tahun = t
	return t
}

func (c *Ctx) SetTahun(t int) { c.Sess.Data.Tahun = t }

// TglDefault: tanggal awal isian, hari ini bila tahun aktif = tahun berjalan.
func (c *Ctx) TglDefault() string {
	t := c.TahunAktif()
	if t == ThisYear() {
		return Today()
	}
	return fmt.Sprintf("%d-01-01", t)
}

func (c *Ctx) DalamTahunAktif(tgl string) bool {
	return len(tgl) >= 4 && tgl[:4] == fmt.Sprint(c.TahunAktif())
}

func (c *Ctx) IsActive(prefix string) bool {
	if prefix == "/" {
		return c.R.URL.Path == "/"
	}
	return strings.HasPrefix(c.R.URL.Path, prefix)
}

// File mengambil satu berkas upload; nil bila tidak ada.
func (c *Ctx) File(name string) (*multipart.FileHeader, bool) {
	if c.R.MultipartForm == nil || c.R.MultipartForm.File == nil {
		return nil, false
	}
	fs := c.R.MultipartForm.File[name]
	if len(fs) == 0 {
		return nil, false
	}
	return fs[0], true
}

func (c *Ctx) Files(name string) []*multipart.FileHeader {
	if c.R.MultipartForm == nil || c.R.MultipartForm.File == nil {
		return nil
	}
	return c.R.MultipartForm.File[name]
}

func (c *Ctx) Q(k string) string { return c.Query.Get(k) }

func (c *Ctx) JSON(code int, v any) {
	c.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.W.WriteHeader(code)
	_ = json.NewEncoder(c.W).Encode(v)
	c.Done()
}

// Download menulis berkas sebagai unduhan lalu menghentikan handler.
func (c *Ctx) Download(contentType, filename string, body []byte, inline bool) {
	disp := "attachment"
	if inline {
		disp = "inline"
	}
	h := c.W.Header()
	h.Set("Content-Type", contentType)
	h.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disp, strings.ReplaceAll(filename, `"`, "")))
	h.Set("Content-Length", fmt.Sprint(len(body)))
	h.Set("Cache-Control", "private, max-age=0, must-revalidate")
	c.saveSession()
	c.W.WriteHeader(200)
	_, _ = c.W.Write(body)
	c.Done()
}

func (c *Ctx) Stream(contentType, filename string, r io.Reader) {
	h := c.W.Header()
	h.Set("Content-Type", contentType)
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.saveSession()
	c.W.WriteHeader(200)
	_, _ = io.Copy(c.W, r)
	c.Done()
}

func (c *Ctx) saveSession() {
	if c.written {
		return
	}
	c.written = true
	c.Sess.save(c.W, c.R)
}

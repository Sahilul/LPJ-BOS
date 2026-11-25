package app

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

//go:embed assets
var assetsFS embed.FS

type HandlerFunc func(c *Ctx)

type route struct {
	method string
	re     *regexp.Regexp
	names  []string
	h      HandlerFunc
}

type Router struct{ routes []route }

var paramRe = regexp.MustCompile(`\{(\w+)\}`)

func (rt *Router) add(method, pattern string, h HandlerFunc) {
	p := strings.TrimRight(pattern, "/")
	if p == "" {
		p = "/"
	}
	names := []string{}
	for _, m := range paramRe.FindAllStringSubmatch(p, -1) {
		names = append(names, m[1])
	}
	quoted := regexp.QuoteMeta(p)
	quoted = regexp.MustCompile(`\\\{(\w+)\\\}`).ReplaceAllString(quoted, `([^/]+)`)
	rt.routes = append(rt.routes, route{method, regexp.MustCompile("^" + quoted + "$"), names, h})
}

func (rt *Router) Get(p string, h HandlerFunc)  { rt.add("GET", p, h) }
func (rt *Router) Post(p string, h HandlerFunc) { rt.add("POST", p, h) }

func (rt *Router) dispatch(c *Ctx) {
	allowed := false
	method := c.R.Method
	if method == "HEAD" {
		method = "GET"
	}
	for _, r := range rt.routes {
		m := r.re.FindStringSubmatch(c.Path)
		if m == nil {
			continue
		}
		if r.method != method {
			allowed = true
			continue
		}
		c.Params = map[string]string{}
		for i, n := range r.names {
			v, err := url.PathUnescape(m[i+1])
			if err != nil {
				v = m[i+1]
			}
			c.Params[n] = v
		}
		r.h(c)
		return
	}
	if allowed {
		c.Abort(405, "Metode tidak diizinkan.")
	}
	c.Abort(404, "Halaman tidak ditemukan.")
}

// Routes adalah router global; modul mendaftarkan rute lewat Register.
var Routes = &Router{}

// Register dipanggil dari init() modul untuk menambah rute (urutan = urutan registrasi).
func Register(order int, fn func(*Router)) {
	registrations = append(registrations, registration{order, fn})
}

type registration struct {
	order int
	fn    func(*Router)
}

var registrations []registration

// McpHandler diisi paket mcp.
var McpHandler http.Handler

// Init menyiapkan konfigurasi, folder data, database dan template.
func Init() error {
	setupTimezone()
	Cfg = loadConfig()
	for _, d := range []string{Cfg.DataDir, Cfg.UploadDir, Cfg.TmpDir} {
		if err := os.MkdirAll(d, 0o775); err != nil {
			return err
		}
	}
	d, err := openDB(Cfg.DBPath)
	if err != nil {
		return err
	}
	Q = d
	if err := Q.Migrate(); err != nil {
		return fmt.Errorf("migrasi: %w", err)
	}
	HitungUlangPPh21Sekali()
	loadTemplates()
	// urutkan registrasi menurut order (stabil)
	for i := 1; i < len(registrations); i++ {
		for j := i; j > 0 && registrations[j].order < registrations[j-1].order; j-- {
			registrations[j], registrations[j-1] = registrations[j-1], registrations[j]
		}
	}
	for _, r := range registrations {
		r.fn(Routes)
	}
	return nil
}

// ReopenDB menutup dan membuka ulang database (dipakai setelah pulihkan cadangan).
func ReopenDB() error {
	_ = Q.Close()
	d, err := openDB(Cfg.DBPath)
	if err != nil {
		return err
	}
	Q = d
	if err := Q.Migrate(); err != nil {
		return err
	}
	HitungUlangPPh21Sekali()
	return nil
}

func Handler() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(assetsFS, "assets")
	fileServer := http.StripPrefix("/assets/", http.FileServer(http.FS(sub)))
	mux.Handle("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fileServer.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/", serveApp)
	return securityHeaders(mux)
}

func securityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		h.ServeHTTP(w, r)
	})
}

const maxBody = 55 << 20

func serveApp(w http.ResponseWriter, r *http.Request) {
	path := "/" + strings.Trim(r.URL.Path, "/")
	if r.URL.Path == "/mcp" || path == "/mcp" {
		if McpHandler == nil {
			http.NotFound(w, r)
			return
		}
		McpHandler.ServeHTTP(w, r)
		return
	}
	c := &Ctx{W: w, R: r, Path: path, Query: r.URL.Query(), Form: Form{}}
	c.Sess = loadSession(r)
	// Kunci diambil setelah body terbaca agar upload lambat tidak menahan request lain.
	defer c.releaseLock()
	defer func() {
		if c.unlock == nil && !c.lockDone {
			c.unlock = lockApp(false)
		}
		if rec := recover(); rec != nil {
			switch s := rec.(type) {
			case redirectSignal:
				c.saveSession()
				c.releaseLock()
				http.Redirect(w, r, s.to, http.StatusFound)
			case abortSignal:
				func() {
					defer func() {
						if r2 := recover(); r2 != nil {
							if _, ok := r2.(doneSignal); !ok {
								log.Printf("galat render error: %v", r2)
							}
						}
					}()
					c.renderError(s.code, s.msg)
				}()
			case doneSignal:
			default:
				log.Printf("panic %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				if !c.written {
					func() {
						defer func() { _ = recover() }()
						c.renderError(500, "")
					}()
				}
			}
			return
		}
		c.saveSession()
	}()

	if r.Method == "POST" {
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		ct := r.Header.Get("Content-Type")
		var err error
		if strings.HasPrefix(ct, "multipart/form-data") {
			err = r.ParseMultipartForm(8 << 20)
		} else {
			err = r.ParseForm()
		}
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			c.Abort(413, "Berkas terlalu besar.")
		}
		if r.MultipartForm != nil {
			c.Form = ParseForm(url.Values(r.MultipartForm.Value))
		} else {
			c.Form = ParseForm(r.PostForm)
		}
		if !c.csrfValid() {
			c.Abort(419, "Sesi formulir sudah kedaluwarsa. Muat ulang halaman lalu coba lagi.")
		}
	}
	c.unlock = lockApp(r.Method != "GET" && r.Method != "HEAD")

	hasUser := Q.Int("SELECT COUNT(*) FROM users") > 0
	if !hasUser && path != "/setup" {
		c.Redirect("/setup")
	}
	if hasUser && c.User() == nil && path != "/login" && path != "/setup" {
		c.Redirect("/login")
	}
	Routes.dispatch(c)
}

// appMu menyerialkan akses database antar-request: Q.Tx mengganti koneksi
// global (Q.q) selama transaksi, dan pulihkan cadangan mengganti Q sendiri.
// Request baca (GET/HEAD) boleh berjalan bersamaan; request tulis eksklusif.
var appMu sync.RWMutex

func lockApp(write bool) func() {
	if write {
		appMu.Lock()
		return appMu.Unlock
	}
	appMu.RLock()
	return appMu.RUnlock
}

// Run menjalankan server HTTP sampai berhenti.
func Run(addr string) error {
	go func() {
		for {
			gcSessions()
			time.Sleep(6 * time.Hour)
		}
	}()
	srv := &http.Server{
		Addr:              addr,
		Handler:           Handler(),
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("LPJ BOS berjalan di %s (data: %s)", addr, Cfg.DataDir)
	return srv.ListenAndServe()
}

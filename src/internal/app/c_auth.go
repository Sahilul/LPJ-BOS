package app

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

func init() {
	Register(0, func(r *Router) {
		r.Get("/setup", setupForm)
		r.Post("/setup", setupStore)
		r.Get("/login", loginForm)
		r.Post("/login", loginStore)
		r.Post("/logout", logout)

		r.Get("/tahun", tahunIndex)
		r.Post("/tahun", tahunStore)
		r.Post("/tahun/aktif", tahunSwitch)
		r.Post("/tahun/{tahun}/default", tahunDefault)
		r.Post("/tahun/{tahun}/hapus", tahunDestroy)

		r.Get("/api", apiIndex)
		r.Post("/api", apiStore)
		r.Post("/api/{id}/hapus", apiDestroy)

		r.Get("/akun", akunForm)
		r.Post("/akun", akunStore)
		r.Get("/", dashboard)
		r.Get("/file/{name}", fileShow)
	})
}

// ---------- Auth ----------

func hashPassword(p string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

func verifyPassword(p, hash string) bool {
	return hash != "" && bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
}

var usernameRe = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)

func validateAccount(in Form, pass, pass2 string, checkPass bool) map[string]string {
	errs := map[string]string{}
	if in.Str("nama") == "" {
		errs["nama"] = "Isi nama Anda."
	}
	if !usernameRe.MatchString(in.Str("username")) {
		errs["username"] = "Gunakan 3–32 huruf kecil, angka, titik atau garis."
	}
	if checkPass {
		if len(pass) < 8 {
			errs["password"] = "Minimal 8 karakter."
		} else if pass != pass2 {
			errs["password2"] = "Ulangi kata sandi yang sama."
		}
	}
	return errs
}

func setupForm(c *Ctx) {
	if Q.Int("SELECT COUNT(*) FROM users") > 0 {
		c.Redirect("/login")
	}
	c.View("auth/setup", D{"title": "Siapkan akun"}, "guest")
}

func setupStore(c *Ctx) {
	if Q.Int("SELECT COUNT(*) FROM users") > 0 {
		c.Redirect("/login")
	}
	in := Form{
		"nama":     c.Form.Trim("nama"),
		"username": strings.ToLower(c.Form.Trim("username")),
		"sekolah":  c.Form.Trim("sekolah"),
	}
	pass := c.Form.Str("password")
	if errs := validateAccount(in, pass, c.Form.Str("password2"), true); len(errs) > 0 {
		c.WithOld(in, errs)
		c.Redirect("/setup")
	}
	id := Q.Insert("users", Row{"username": in["username"], "nama": in["nama"], "password_hash": hashPassword(pass)})
	if in.Str("sekolah") != "" {
		Q.Run("UPDATE sekolah SET nama = ? WHERE id = 1", in["sekolah"])
	}
	c.Login(id)
	c.SetTahun(TahunDefault())
	c.Flash("success", "Akun siap. Lengkapi profil sekolah lebih dulu.")
	c.Redirect("/sekolah")
}

func loginForm(c *Ctx) {
	if c.User() != nil {
		c.Redirect("/")
	}
	list := TahunList()
	def := TahunDefault()
	if len(list) == 0 {
		list = []int{def}
	}
	opts := []Opt{}
	for _, t := range list {
		label := strconv.Itoa(t)
		if t == def {
			label += " (default)"
		}
		opts = append(opts, Opt{strconv.Itoa(t), label})
	}
	c.View("auth/login", D{"title": "Masuk", "tahunOptions": opts, "tahunDefault": strconv.Itoa(def)}, "guest")
}

func loginStore(c *Ctx) {
	username := strings.ToLower(c.Form.Trim("username"))
	tahun := int(c.Form.Int("tahun"))
	if !InTahunList(tahun) {
		tahun = TahunDefault()
	}
	user := Q.One("SELECT * FROM users WHERE username = ?", username)
	if user == nil || !verifyPassword(c.Form.Str("password"), user.S("password_hash")) {
		time.Sleep(400 * time.Millisecond)
		c.WithOld(Form{"username": username, "tahun": strconv.Itoa(tahun)}, map[string]string{"password": "Nama pengguna atau kata sandi salah."})
		c.Redirect("/login")
	}
	c.Login(user.I("id"))
	c.SetTahun(tahun)
	c.Redirect("/")
}

func logout(c *Ctx) {
	c.Sess.Reset()
	c.Redirect("/login")
}

func akunForm(c *Ctx) {
	c.View("auth/akun", D{"title": "Akun", "user": c.User()})
}

func akunStore(c *Ctx) {
	user := c.User()
	row := Q.One("SELECT * FROM users WHERE id = ?", user.I("id"))
	in := Form{"nama": c.Form.Trim("nama"), "username": strings.ToLower(c.Form.Trim("username"))}
	pass := c.Form.Str("password")
	errs := validateAccount(in, pass, c.Form.Str("password2"), pass != "")
	if !verifyPassword(c.Form.Str("password_lama"), row.S("password_hash")) {
		errs["password_lama"] = "Kata sandi lama tidak cocok."
	}
	if Q.Int("SELECT id FROM users WHERE username = ? AND id <> ?", in["username"], user.I("id")) != 0 {
		errs["username"] = "Nama pengguna sudah dipakai."
	}
	if len(errs) > 0 {
		c.WithOld(in, errs)
		c.Redirect("/akun")
	}
	data := Row{"nama": in["nama"], "username": in["username"]}
	if pass != "" {
		data["password_hash"] = hashPassword(pass)
	}
	Q.Update("users", data, user.I("id"))
	c.Flash("success", "Akun diperbarui.")
	c.Redirect("/akun")
}

// ---------- Dashboard ----------

type step struct {
	Label, Href, Hint string
	Done              bool
}

func dashboard(c *Ctx) {
	s := c.Sekolah()
	kepala := PejabatAktif("kepala", "")
	bendahara := PejabatAktif("bendahara", "")
	steps := []step{
		{"Profil sekolah", "/sekolah", "Nama, NPSN, alamat dan rekening BOS", s.S("nama") != "" && s.S("npsn") != "" && s.S("kabupaten") != ""},
		{"Kop surat", "/sekolah#gambar", "Gambar kop yang sudah jadi", truthy(s["kop_path"])},
		{"Kepala sekolah", "/pejabat", "Nama, NIP dan tanda tangan", kepala != nil},
		{"Bendahara", "/pejabat", "Nama, NIP dan tanda tangan", bendahara != nil},
		{"Database penerima", "/penerima", "Guru dan tendik penerima honor", Q.Int("SELECT COUNT(*) FROM penerima") > 0},
		{"Toko langganan", "/toko", "Tempat belanja ATK, konsumsi, dll.", Q.Int("SELECT COUNT(*) FROM toko") > 0},
	}
	done := 0
	for _, st := range steps {
		if st.Done {
			done++
		}
	}
	pct := int(float64(done)/float64(len(steps))*100 + 0.5)

	tahun := c.TahunAktif()
	perTgl := Today()
	if a := fmt.Sprintf("%d-01-01", tahun); perTgl < a {
		perTgl = a
	}
	if z := fmt.Sprintf("%d-12-31", tahun); perTgl > z {
		perTgl = z
	}
	pagu := Q.Float("SELECT COALESCE(SUM(pagu), 0) FROM rencana_belanja WHERE tahun = ?", tahun)
	realisasi := Q.Float("SELECT COALESCE(SUM(b.total), 0) FROM belanja b JOIN rencana_belanja r ON r.id = b.rencana_id WHERE r.tahun = ?", tahun)
	real := 0
	if pagu > 0 {
		real = int(realisasi/pagu*100 + 0.5)
		if real > 100 {
			real = 100
		}
	}
	saldo := SaldoKas(perTgl, 0, 0)
	saldoLabel := "Saldo kas hari ini"
	if perTgl != Today() {
		saldoLabel = "Saldo kas per " + TanggalIndo(perTgl)
	}
	type stat struct {
		Href, Icon, Label string
		N                 int64
	}
	stats := []stat{
		{"/penerima", "users", "Penerima aktif", Q.Int("SELECT COUNT(*) FROM penerima WHERE aktif = 1")},
		{"/toko", "store", "Toko aktif", Q.Int("SELECT COUNT(*) FROM toko WHERE aktif = 1")},
		{"/komponen", "layers", "Komponen", Q.Int("SELECT COUNT(*) FROM komponen")},
		{"/rekening", "hash", "Kode rekening", Q.Int("SELECT COUNT(*) FROM kode_rekening")},
	}
	c.View("dashboard", D{
		"title": "Beranda", "s": s, "tahun": tahun, "kepala": kepala, "bendahara": bendahara,
		"steps": steps, "done": done, "pct": pct, "stats": stats,
		"pagu": pagu, "realisasi": realisasi, "real": real,
		"diterima":   Q.Float("SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan WHERE strftime('%Y', tgl) = ?", strconv.Itoa(tahun)),
		"saldo":      saldo.Total,
		"saldoLabel": saldoLabel,
	})
}

// ---------- File ----------

var safeFileRe = regexp.MustCompile(`(?i)^[a-z0-9_.-]+$`)

func fileShow(c *Ctx) {
	name := filepath.Base(c.Param("name"))
	path := UploadPath(name)
	st, err := os.Stat(path)
	if !safeFileRe.MatchString(name) || err != nil || st.IsDir() {
		c.Abort(404, "Berkas tidak ditemukan.")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		c.Abort(404, "Berkas tidak ditemukan.")
	}
	ctype := http.DetectContentType(body)
	if i := strings.Index(ctype, ";"); i >= 0 {
		ctype = ctype[:i]
	}
	if ctype == "text/xml" || ctype == "text/plain" {
		if t := mime.TypeByExtension(filepath.Ext(name)); t == "image/svg+xml" {
			ctype = t
		}
	}
	if !strings.HasPrefix(ctype, "image/") && ctype != "application/pdf" {
		ctype = "application/octet-stream"
	}
	h := c.W.Header()
	h.Set("Content-Type", ctype)
	h.Set("Content-Length", strconv.Itoa(len(body)))
	h.Set("Cache-Control", "private, max-age=31536000")
	c.saveSession()
	c.W.WriteHeader(200)
	if c.R.Method != "HEAD" {
		_, _ = c.W.Write(body)
	}
	c.Done()
}

// ---------- Tahun ----------

func tahunIndex(c *Ctx) {
	rows := Q.All(`SELECT t.*,
                (SELECT COUNT(*) FROM rencana_belanja r WHERE r.tahun = t.tahun) AS n_rencana,
                (SELECT COALESCE(SUM(pagu), 0) FROM rencana_belanja r WHERE r.tahun = t.tahun) AS pagu,
                (SELECT COUNT(*) FROM belanja b WHERE strftime('%Y', b.tgl) = CAST(t.tahun AS TEXT)) AS n_belanja,
                (SELECT COUNT(*) FROM penerimaan p WHERE strftime('%Y', p.tgl) = CAST(t.tahun AS TEXT)) AS n_penerimaan,
                (SELECT COUNT(*) FROM mutasi_kas m WHERE strftime('%Y', m.tgl) = CAST(t.tahun AS TEXT)) AS n_mutasi
             FROM tahun_anggaran t ORDER BY t.tahun DESC`)
	for _, r := range rows {
		r["isi"] = r.I("n_rencana") + r.I("n_belanja") + r.I("n_penerimaan") + r.I("n_mutasi")
	}
	list := TahunList()
	saran := ThisYear()
	if len(list) > 0 && list[0]+1 > saran {
		saran = list[0] + 1
	}
	c.View("tahun/index", D{"title": "Tahun anggaran", "rows": rows, "aktif": int64(c.TahunAktif()), "saran": strconv.Itoa(saran)})
}

func tahunFind(c *Ctx, s string) int {
	t, _ := strconv.Atoi(strings.TrimSpace(s))
	if !InTahunList(t) {
		c.Abort(404, "Tahun anggaran tidak terdaftar.")
	}
	return t
}

func tahunSetDefault(t int) {
	Q.Tx(func() {
		Q.Run("UPDATE tahun_anggaran SET is_default = 0")
		Q.Run("UPDATE tahun_anggaran SET is_default = 1 WHERE tahun = ?", t)
		Q.Run("UPDATE sekolah SET tahun_anggaran = ? WHERE id = 1", t)
	})
}

func tahunStore(c *Ctx) {
	tahun := int(c.Form.Int("tahun"))
	ket := c.Form.Trim("keterangan")
	errs := map[string]string{}
	if tahun < 2000 || tahun > 2100 {
		errs["tahun"] = fmt.Sprintf("Isi tahun empat angka, misalnya %d.", ThisYear())
	} else if InTahunList(tahun) {
		errs["tahun"] = fmt.Sprintf("Tahun %d sudah ada.", tahun)
	}
	if len(errs) > 0 {
		c.WithOld(c.Form, errs)
		c.Redirect("/tahun")
	}
	pertama := len(TahunList()) == 0
	isDef := 0
	if pertama {
		isDef = 1
	}
	Q.Insert("tahun_anggaran", Row{"tahun": tahun, "keterangan": ket, "is_default": isDef})
	if c.Form.Truthy("jadikan_default") {
		tahunSetDefault(tahun)
	}
	if c.Form.Truthy("pakai") {
		c.SetTahun(tahun)
	}
	c.Flash("success", fmt.Sprintf("Tahun anggaran %d ditambahkan.", tahun))
	c.Redirect("/tahun")
}

func tahunDefault(c *Ctx) {
	t := tahunFind(c, c.Param("tahun"))
	tahunSetDefault(t)
	c.Flash("success", fmt.Sprintf("Tahun %d sekarang terpilih otomatis saat masuk.", t))
	c.Redirect("/tahun")
}

func tahunSwitch(c *Ctx) {
	t := tahunFind(c, c.Form.Str("tahun"))
	c.SetTahun(t)
	c.Flash("success", fmt.Sprintf("Sekarang bekerja di tahun anggaran %d.", t))
	c.Back("/")
}

func tahunDestroy(c *Ctx) {
	t := tahunFind(c, c.Param("tahun"))
	row := Q.One("SELECT is_default FROM tahun_anggaran WHERE tahun = ?", t)
	y := strconv.Itoa(t)
	dipakai := Q.Int("SELECT COUNT(*) FROM rencana_belanja WHERE tahun = ?", t) +
		Q.Int("SELECT COUNT(*) FROM belanja WHERE strftime('%Y', tgl) = ?", y) +
		Q.Int("SELECT COUNT(*) FROM penerimaan WHERE strftime('%Y', tgl) = ?", y) +
		Q.Int("SELECT COUNT(*) FROM mutasi_kas WHERE strftime('%Y', tgl) = ?", y)
	switch {
	case row.I("is_default") != 0:
		c.Flash("danger", fmt.Sprintf("Tahun %d adalah tahun default. Jadikan tahun lain default dulu.", t))
	case dipakai > 0:
		c.Flash("danger", fmt.Sprintf("Tahun %d sudah berisi data, jadi tidak bisa dihapus.", t))
	default:
		Q.Run("DELETE FROM tahun_anggaran WHERE tahun = ?", t)
		if c.Sess.Data.Tahun == t {
			c.Sess.Data.Tahun = 0
		}
		c.Flash("success", fmt.Sprintf("Tahun %d dihapus.", t))
	}
	c.Redirect("/tahun")
}

// ---------- API token (MCP) ----------

const (
	AksesBaca  = "baca"
	AksesTulis = "tulis"
)

func TokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func mbSubstr(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// CreateToken membuat token baru; token asli hanya dikembalikan sekali.
func CreateToken(nama, akses string) (string, Row) {
	if akses != AksesTulis {
		akses = AksesBaca
	}
	nama = strings.TrimSpace(nama)
	if nama == "" {
		nama = "Token " + time.Now().Format("02/01/2006 15:04")
	}
	pb := make([]byte, 4)
	_, _ = rand.Read(pb)
	sb := make([]byte, 32)
	_, _ = rand.Read(sb)
	prefix := hex.EncodeToString(pb)
	token := "lpbos_" + prefix + "_" + base64.RawURLEncoding.EncodeToString(sb)
	id := Q.Insert("api_token", Row{"nama": mbSubstr(nama, 120), "prefix": prefix, "token_hash": TokenHash(token), "akses": akses})
	row := Q.One("SELECT * FROM api_token WHERE id = ?", id)
	if row == nil {
		row = Row{}
	}
	return token, row
}

var tokenPrefixRe = regexp.MustCompile(`^lpbos_([0-9a-f]{8})_`)

// VerifyToken mengembalikan baris token bila valid.
func VerifyToken(token string) Row {
	m := tokenPrefixRe.FindStringSubmatch(token)
	if m == nil {
		return nil
	}
	row := Q.One("SELECT * FROM api_token WHERE prefix = ?", m[1])
	if row == nil || subtle.ConstantTimeCompare([]byte(row.S("token_hash")), []byte(TokenHash(token))) != 1 {
		return nil
	}
	return row
}

func apiIndex(c *Ctx) {
	tokens := Q.All(`SELECT t.*, (SELECT COUNT(*) FROM api_log l WHERE l.token_id = t.id) AS n_panggil
             FROM api_token t ORDER BY t.id DESC`)
	logs := Q.All(`SELECT l.*, t.nama AS token_nama FROM api_log l LEFT JOIN api_token t ON t.id = l.token_id
             ORDER BY l.id DESC LIMIT 30`)
	baru := c.Sess.Data.ApiTokenBaru
	c.Sess.Data.ApiTokenBaru = nil
	scheme := "http"
	if IsHTTPS(c.R) {
		scheme = "https"
	}
	host := c.R.Host
	if host == "" {
		host = "localhost:8080"
	}
	url := scheme + "://" + host + "/mcp"
	tok := "<token>"
	if baru != nil {
		tok = baru["token"]
	}
	type srv struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	type wrap struct {
		McpServers struct {
			Lpjbos srv `json:"lpjbos"`
		} `json:"mcpServers"`
	}
	var cfgJSON wrap
	cfgJSON.McpServers.Lpjbos = srv{"http", url, map[string]string{"Authorization": "Bearer " + tok}}
	cfg := jsonPretty(cfgJSON)
	var baruD any
	if baru != nil {
		baruD = baru
	}
	c.View("api/index", D{"title": "Akses AI (MCP)", "tokens": tokens, "log": logs, "baru": baruD, "url": url, "cfg": cfg,
		"aksesOpts": []Opt{{"baca", "Baca saja"}, {"tulis", "Baca & tulis (bisa mencatat/menghapus)"}}})
}

func apiStore(c *Ctx) {
	nama := c.Form.Trim("nama")
	akses := AksesBaca
	if c.Form.Str("akses") == AksesTulis {
		akses = AksesTulis
	}
	if utf8.RuneCountInString(nama) > 120 {
		c.WithOld(c.Form, map[string]string{"nama": "Nama maksimal 120 karakter."})
		c.Redirect("/api")
	}
	token, row := CreateToken(nama, akses)
	n := row.S("nama")
	if n == "" {
		n = nama
	}
	c.Sess.Data.ApiTokenBaru = map[string]string{"token": token, "nama": n}
	c.Flash("success", "Token dibuat. Salin sekarang, token tidak akan ditampilkan lagi.")
	c.Redirect("/api")
}

func apiDestroy(c *Ctx) {
	row := Q.One("SELECT * FROM api_token WHERE id = ?", c.ParamInt("id"))
	if row == nil {
		c.Abort(404, "Token tidak ditemukan.")
	}
	Q.Run("DELETE FROM api_token WHERE id = ?", row.I("id"))
	c.Flash("success", "Token “"+row.S("nama")+"” dicabut. Aplikasi yang memakainya tidak bisa masuk lagi.")
	c.Redirect("/api")
}

var bearerRe = regexp.MustCompile(`(?i)^Bearer\s+(\S+)$`)

// TokenDariHeader membaca token dari Authorization: Bearer … atau X-API-Token.
func TokenDariHeader(r *http.Request) string {
	if m := bearerRe.FindStringSubmatch(r.Header.Get("Authorization")); m != nil {
		return m[1]
	}
	return strings.TrimSpace(r.Header.Get("X-API-Token"))
}

func SentuhToken(row Row) {
	Q.Run("UPDATE api_token SET last_used_at = datetime('now', 'localtime') WHERE id = ?", row.I("id"))
}

func emptyArg(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

// LogAPI mencatat satu panggilan tool MCP.
func LogAPI(token Row, tool string, args map[string]any, ok bool, pesan string) {
	clean := map[string]any{}
	for k, v := range args {
		if !emptyArg(v) {
			clean[k] = v
		}
	}
	var argStr string
	if len(clean) == 0 {
		argStr = "[]"
	} else if b, err := jsonMarshal(clean); err == nil {
		argStr = string(b)
	}
	var tid any
	if token != nil {
		tid = token.I("id")
	}
	okI := 0
	if ok {
		okI = 1
	}
	Q.Insert("api_log", Row{
		"token_id": tid, "tool": mbSubstr(tool, 120), "args": mbSubstr(argStr, 4000),
		"ok": okI, "pesan": mbSubstr(pesan, 500),
	})
}

func jsonPretty(v any) string {
	b, _ := jsonMarshalIndent(v)
	return string(b)
}

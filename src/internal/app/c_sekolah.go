package app

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

// SekolahFields: kolom profil sekolah yang bisa diubah (urutan seperti PHP).
var SekolahFields = []string{
	"nama", "npsn", "jenjang", "status", "alamat", "desa", "kecamatan", "kabupaten", "provinsi",
	"kode_pos", "telepon", "email", "npwp", "bank_nama", "bank_rekening", "bank_atas_nama",
}

// SekolahImage: satu jenis gambar sekolah (kop, logo, stempel).
type SekolahImage struct {
	Jenis string
	Col   string
	Label string
	Width int
	Hint  string
}

// SekolahImages: gambar sekolah berurutan.
var SekolahImages = []SekolahImage{
	{"kop", "kop_path", "Kop surat", 2400, "Gambar kop surat utuh (logo, nama, alamat, garis). Lebar disarankan ±2000 px."},
	{"logo", "logo_path", "Logo", 800, "Logo sekolah atau pemda, PNG transparan lebih baik."},
	{"stempel", "stempel_path", "Stempel", 800, "Stempel sekolah, PNG latar transparan agar menimpa tanda tangan dengan rapi."},
}

// SekolahImageByJenis mencari konfigurasi gambar; ok=false bila jenis tidak dikenal.
func SekolahImageByJenis(jenis string) (SekolahImage, bool) {
	for _, im := range SekolahImages {
		if im.Jenis == jenis {
			return im, true
		}
	}
	return SekolahImage{}, false
}

var skNpsnRe = regexp.MustCompile(`^\d{8}$`)

// ValidEmail meniru filter_var(FILTER_VALIDATE_EMAIL) secara longgar.
func ValidEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s {
		return false
	}
	at := strings.LastIndex(s, "@")
	return at > 0 && strings.Contains(s[at+1:], ".")
}

// SekolahValidasi memeriksa isian profil (sudah di-trim); urutan galat seperti PHP.
func SekolahValidasi(in map[string]string) *ValidationError {
	e := NewValidationOrdered()
	if in["nama"] == "" {
		e.Set("nama", "Isi nama sekolah.")
	}
	if in["npsn"] != "" && !skNpsnRe.MatchString(in["npsn"]) {
		e.Set("npsn", "NPSN terdiri dari 8 angka.")
	}
	if in["email"] != "" && !ValidEmail(in["email"]) {
		e.Set("email", "Alamat surel tidak valid.")
	}
	if len(e.Errors) > 0 {
		return e
	}
	return nil
}

// SekolahSimpan memvalidasi lalu menyimpan profil sekolah (semua SekolahFields).
func SekolahSimpan(in map[string]string) error {
	if ve := SekolahValidasi(in); ve != nil {
		return ve
	}
	data := Row{}
	for _, f := range SekolahFields {
		data[f] = in[f]
	}
	data["updated_at"] = time.Now().Format("2006-01-02 15:04:05")
	Q.Update("sekolah", data, 1)
	return nil
}

// SekolahGantiGambar menyimpan gambar baru untuk satu jenis dan menghapus yang lama.
func SekolahGantiGambar(im SekolahImage, data []byte) error {
	name, err := SimpanGambarBytes(data, im.Jenis, im.Width)
	if err != nil {
		return err
	}
	HapusGambar(Sekolah().S(im.Col))
	Q.Run("UPDATE sekolah SET "+im.Col+" = ? WHERE id = 1", name)
	return nil
}

// SekolahHapusGambar menghapus gambar satu jenis.
func SekolahHapusGambar(im SekolahImage) {
	HapusGambar(Sekolah().S(im.Col))
	Q.Run("UPDATE sekolah SET " + im.Col + " = NULL WHERE id = 1")
}

// ---- ImageUpload ----

// SimpanGambar menyimpan unggahan PNG/JPG ke folder uploads (lihat SimpanGambarBytes).
func SimpanGambar(fh *multipart.FileHeader, prefix string, maxWidth int) (string, error) {
	if fh == nil {
		return "", errors.New("Pilih berkas gambar lebih dulu.")
	}
	if fh.Size > Cfg.MaxImageBytes {
		return "", errors.New("Ukuran gambar maksimal 2 MB.")
	}
	f, err := fh.Open()
	if err != nil {
		return "", errors.New("Gagal menyimpan gambar.")
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return "", errors.New("Gagal menyimpan gambar.")
	}
	return SimpanGambarBytes(data, prefix, maxWidth)
}

// SimpanGambarBytes: simpan gambar PNG/JPG, perkecil bila lebih lebar dari maxWidth,
// kembalikan nama berkas di folder uploads.
func SimpanGambarBytes(data []byte, prefix string, maxWidth int) (string, error) {
	if len(data) == 0 {
		return "", errors.New("Pilih berkas gambar lebih dulu.")
	}
	if int64(len(data)) > Cfg.MaxImageBytes {
		return "", errors.New("Ukuran gambar maksimal 2 MB.")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		return "", errors.New("Gunakan gambar PNG atau JPG.")
	}
	ext := "jpg"
	if format == "png" {
		ext = "png"
	}
	rb := make([]byte, 6)
	_, _ = rand.Read(rb)
	name := prefix + "_" + hex.EncodeToString(rb) + "." + ext
	dest := UploadPath(name)
	fail := errors.New("Gagal menyimpan gambar.")

	if cfg.Width <= maxWidth {
		if os.WriteFile(dest, data, 0o664) != nil {
			return "", fail
		}
		return name, nil
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", errors.New("Gunakan gambar PNG atau JPG.")
	}
	nh := int(float64(cfg.Height)*float64(maxWidth)/float64(cfg.Width) + 0.5)
	if nh < 1 {
		nh = 1
	}
	var buf bytes.Buffer
	if format == "png" {
		dst := image.NewNRGBA(image.Rect(0, 0, maxWidth, nh))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
		err = png.Encode(&buf, dst)
	} else {
		dst := image.NewRGBA(image.Rect(0, 0, maxWidth, nh))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 88})
	}
	if err != nil || os.WriteFile(dest, buf.Bytes(), 0o664) != nil {
		return "", fail
	}
	return name, nil
}

// HapusGambar menghapus berkas unggahan beserta cache PDF-nya.
func HapusGambar(name string) {
	if name == "" {
		return
	}
	p := UploadPath(name)
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		_ = os.Remove(p)
	}
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	matches, _ := filepath.Glob(filepath.Join(Cfg.UploadDir, "pdf-cache", base+"-*"))
	for _, m := range matches {
		_ = os.Remove(m)
	}
}

// ---- Pejabat ----

// PejabatPeran: peran pejabat berurutan (kunci → label).
var PejabatPeran = []Opt{{"kepala", "Kepala Sekolah"}, {"bendahara", "Bendahara BOS"}}

// PejabatPeranLabel: label peran, atau "" bila tidak dikenal.
func PejabatPeranLabel(peran string) string {
	for _, o := range PejabatPeran {
		if o.K == peran {
			return o.V
		}
	}
	return ""
}

var pjTanggalRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// PejabatHapus menghapus pejabat beserta gambar tanda tangannya.
func PejabatHapus(id int64) bool {
	p := Q.One("SELECT * FROM pejabat WHERE id = ?", id)
	if p == nil {
		return false
	}
	HapusGambar(p.S("ttd_path"))
	Q.Run("DELETE FROM pejabat WHERE id = ?", id)
	return true
}

func pjFind(c *Ctx) Row {
	p := Q.One("SELECT * FROM pejabat WHERE id = ?", c.ParamInt("id"))
	if p == nil {
		c.Abort(404, "Pejabat tidak ditemukan.")
	}
	return p
}

func pjValidated(c *Ctx, back string) Row {
	in := Row{
		"peran":   c.Form.Str("peran"),
		"nama":    c.Form.Trim("nama"),
		"nip":     Digits(c.Form.Str("nip")),
		"pangkat": c.Form.Trim("pangkat"),
		"mulai":   c.Form.Str("mulai"),
		"selesai": nil,
	}
	if s := c.Form.Str("selesai"); s != "" && s != "0" {
		in["selesai"] = s
	}
	errs := map[string]string{}
	if PejabatPeranLabel(in.S("peran")) == "" {
		errs["peran"] = "Pilih peran."
	}
	if in.S("nama") == "" {
		errs["nama"] = "Isi nama lengkap beserta gelar."
	}
	if n := in.S("nip"); n != "" && len(n) != 18 {
		errs["nip"] = "NIP terdiri dari 18 angka."
	}
	if !pjTanggalRe.MatchString(in.S("mulai")) {
		errs["mulai"] = "Isi tanggal mulai menjabat."
	}
	if in["selesai"] != nil && in.S("selesai") < in.S("mulai") {
		errs["selesai"] = "Tanggal selesai tidak boleh sebelum tanggal mulai."
	}
	if len(errs) > 0 {
		old := Form{}
		for k, v := range in {
			if v == nil {
				old[k] = ""
			} else {
				old[k] = v
			}
		}
		c.WithOld(old, errs)
		c.Redirect(back)
	}
	return in
}

func pjStoreTtd(c *Ctx, current any, back string) any {
	fh, ok := c.File("ttd")
	if !ok || fh == nil {
		return current
	}
	name, err := SimpanGambar(fh, "ttd", 900)
	if err != nil {
		c.WithOld(c.Form, map[string]string{"ttd": err.Error()})
		c.Redirect(back)
	}
	return name
}

func init() {
	Register(10, func(r *Router) {
		r.Get("/sekolah", func(c *Ctx) {
			s := Sekolah()
			upd := s.S("updated_at")
			if len(upd) > 10 {
				upd = upd[:10]
			}
			c.View("sekolah/edit", D{
				"title":   "Profil sekolah",
				"s":       s,
				"updated": upd,
				"images":  SekolahImages,
				"maxMB":   int(Cfg.MaxImageBytes / 1048576),
				"jenjang": []Opt{{"", "—"}, {"SD", "SD"}, {"SMP", "SMP"}, {"SMA", "SMA"}, {"SMK", "SMK"}, {"SLB", "SLB"}, {"PAUD", "PAUD/TK"}},
				"status":  []Opt{{"Negeri", "Negeri"}, {"Swasta", "Swasta"}},
			})
		})
		r.Post("/sekolah", func(c *Ctx) {
			in := map[string]string{}
			for _, f := range SekolahFields {
				in[f] = c.Form.Trim(f)
			}
			if err := SekolahSimpan(in); err != nil {
				var ve *ValidationError
				if errors.As(err, &ve) {
					old := Form{}
					for k, v := range in {
						old[k] = v
					}
					c.WithOld(old, ve.Errors)
					c.Redirect("/sekolah")
				}
				panic(err)
			}
			c.Flash("success", "Profil sekolah disimpan.")
			c.Redirect("/sekolah")
		})
		r.Post("/sekolah/gambar/{jenis}", func(c *Ctx) {
			im, ok := SekolahImageByJenis(c.Param("jenis"))
			if !ok {
				c.Abort(404, "Halaman tidak ditemukan.")
			}
			fh, _ := c.File("gambar")
			name, err := SimpanGambar(fh, im.Jenis, im.Width)
			if err != nil {
				c.Flash("danger", im.Label+": "+err.Error())
				c.Redirect("/sekolah#gambar")
			}
			HapusGambar(Sekolah().S(im.Col))
			Q.Run("UPDATE sekolah SET "+im.Col+" = ? WHERE id = 1", name)
			c.Flash("success", im.Label+" diperbarui.")
			c.Redirect("/sekolah#gambar")
		})
		r.Post("/sekolah/gambar/{jenis}/hapus", func(c *Ctx) {
			im, ok := SekolahImageByJenis(c.Param("jenis"))
			if !ok {
				c.Abort(404, "Halaman tidak ditemukan.")
			}
			SekolahHapusGambar(im)
			c.Flash("success", im.Label+" dihapus.")
			c.Redirect("/sekolah#gambar")
		})

		r.Get("/pejabat", func(c *Ctx) {
			rows := Q.All("SELECT * FROM pejabat ORDER BY peran, mulai DESC")
			aktif := map[string]Row{}
			for _, o := range PejabatPeran {
				aktif[o.K] = PejabatAktif(o.K, "")
			}
			type kartu struct {
				Peran, Label string
				P            Row
			}
			kartus := []kartu{}
			for _, o := range PejabatPeran {
				kartus = append(kartus, kartu{o.K, o.V, aktif[o.K]})
			}
			type baris struct {
				R      Row
				Label  string
				Aktif  bool
			}
			list := []baris{}
			for _, r := range rows {
				lbl := PejabatPeranLabel(r.S("peran"))
				if lbl == "" {
					lbl = r.S("peran")
				}
				a := aktif[r.S("peran")]
				list = append(list, baris{r, lbl, a != nil && a.I("id") == r.I("id")})
			}
			c.View("pejabat/index", D{"title": "Pejabat & tanda tangan", "kartu": kartus, "rows": list})
		})
		r.Get("/pejabat/baru", func(c *Ctx) {
			peran := c.Q("peran")
			if _, ok := c.Query["peran"]; !ok {
				peran = "kepala"
			}
			p := Row{"peran": peran, "mulai": time.Now().Format("2006") + "-01-01"}
			c.View("pejabat/form", D{"title": "Tambah pejabat", "p": p, "edit": false, "peran": pjPeranOpts()})
		})
		r.Post("/pejabat", func(c *Ctx) {
			data := pjValidated(c, "/pejabat/baru")
			data["ttd_path"] = pjStoreTtd(c, nil, "/pejabat/baru")
			Q.Insert("pejabat", data)
			c.Flash("success", PejabatPeranLabel(data.S("peran"))+" ditambahkan.")
			c.Redirect("/pejabat")
		})
		r.Get("/pejabat/{id}/edit", func(c *Ctx) {
			p := pjFind(c)
			c.View("pejabat/form", D{"title": "Ubah pejabat", "p": p, "edit": true, "peran": pjPeranOpts()})
		})
		r.Post("/pejabat/{id}", func(c *Ctx) {
			p := pjFind(c)
			id := c.Param("id")
			back := "/pejabat/" + id + "/edit"
			data := pjValidated(c, back)
			cur := p["ttd_path"]
			if c.Form.Truthy("hapus_ttd") {
				HapusGambar(toStr(cur))
				data["ttd_path"] = nil
				cur = nil
			}
			nw := pjStoreTtd(c, cur, back)
			if toStr(nw) != toStr(cur) || (nw == nil) != (cur == nil) {
				HapusGambar(toStr(cur))
				data["ttd_path"] = nw
			}
			Q.Update("pejabat", data, c.ParamInt("id"))
			c.Flash("success", "Data pejabat disimpan.")
			c.Redirect("/pejabat")
		})
		r.Post("/pejabat/{id}/hapus", func(c *Ctx) {
			pjFind(c)
			PejabatHapus(c.ParamInt("id"))
			c.Flash("success", "Pejabat dihapus.")
			c.Redirect("/pejabat")
		})
	})
}

func pjPeranOpts() []Opt {
	return append([]Opt{{"", "Pilih peran"}}, PejabatPeran...)
}

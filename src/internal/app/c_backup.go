package app

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var backupSlugRe = regexp.MustCompile(`[^a-z0-9]+`)
var uploadNameRe = regexp.MustCompile(`^uploads/[a-z0-9_.-]+$`)

func backupSlug() string {
	s := backupSlugRe.ReplaceAllString(strings.ToLower(Sekolah().S("nama")), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "sekolah"
	}
	return s
}

// uploadFiles: berkas gambar/lampiran di folder uploads (tanpa subfolder).
func uploadFiles() []os.DirEntry {
	entries, err := os.ReadDir(Cfg.UploadDir)
	if err != nil {
		return nil
	}
	out := []os.DirEntry{}
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e)
		}
	}
	return out
}

func backupDownload(c *Ctx) {
	Q.Run("PRAGMA wal_checkpoint(TRUNCATE)")
	snap := filepath.Join(Cfg.TmpDir, "snapshot_"+randomHex(4)+".sqlite")
	Q.Run("VACUUM INTO ?", snap)
	defer func() { _ = os.Remove(snap) }()

	zipPath := filepath.Join(Cfg.TmpDir, "backup_"+randomHex(4)+".zip")
	f, err := os.Create(zipPath)
	if err != nil {
		c.Abort(500, "Gagal menyiapkan berkas cadangan.")
	}
	zw := zip.NewWriter(f)
	add := func(name, src string) {
		in, err := os.Open(src)
		if err != nil {
			return
		}
		defer in.Close()
		w, err := zw.Create(name)
		if err != nil {
			return
		}
		_, _ = io.Copy(w, in)
	}
	add("lpjbos.sqlite", snap)
	for _, e := range uploadFiles() {
		add("uploads/"+e.Name(), filepath.Join(Cfg.UploadDir, e.Name()))
	}
	meta, _ := json.MarshalIndent(map[string]any{
		"app": "lpjbos", "dibuat": time.Now().Format(time.RFC3339), "sekolah": Sekolah().S("nama"),
	}, "", "  ")
	if w, err := zw.Create("backup.json"); err == nil {
		_, _ = w.Write(meta)
	}
	_ = zw.Close()
	_ = f.Close()

	in, err := os.Open(zipPath)
	if err != nil {
		_ = os.Remove(zipPath)
		c.Abort(500, "Gagal menyiapkan berkas cadangan.")
	}
	defer func() {
		_ = in.Close()
		_ = os.Remove(zipPath)
	}()
	name := fmt.Sprintf("lpjbos-%s-%s.zip", backupSlug(), time.Now().Format("20060102-150405"))
	c.Stream("application/zip", name, in)
}

func rrmdir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			rrmdir(p)
		} else {
			_ = os.Remove(p)
		}
	}
	_ = os.Remove(dir)
}

// backupRestore mengganti database & uploads dengan isi berkas cadangan.
func backupRestore(c *Ctx) {
	fh, ok := c.File("berkas")
	if !ok || fh == nil {
		c.Flash("danger", "Pilih berkas cadangan (.zip) lebih dulu.")
		c.Redirect("/backup")
	}
	if c.Form.Trim("konfirmasi") != "PULIHKAN" {
		c.Flash("danger", "Ketik PULIHKAN untuk mengonfirmasi.")
		c.Redirect("/backup")
	}
	src, err := fh.Open()
	if err != nil {
		c.Flash("danger", "Berkas bukan cadangan LPJ BOS yang valid.")
		c.Redirect("/backup")
	}
	tmpZip := filepath.Join(Cfg.TmpDir, "unggah_"+randomHex(4)+".zip")
	out, err := os.Create(tmpZip)
	if err == nil {
		_, err = io.Copy(out, src)
		_ = out.Close()
	}
	_ = src.Close()
	defer func() { _ = os.Remove(tmpZip) }()
	if err != nil {
		c.Flash("danger", "Berkas bukan cadangan LPJ BOS yang valid.")
		c.Redirect("/backup")
	}

	zr, err := zip.OpenReader(tmpZip)
	if err != nil {
		c.Flash("danger", "Berkas bukan cadangan LPJ BOS yang valid.")
		c.Redirect("/backup")
	}
	defer zr.Close()

	hasDB := false
	for _, zf := range zr.File {
		if zf.Name == "lpjbos.sqlite" {
			hasDB = true
		}
	}
	if !hasDB {
		c.Flash("danger", "Berkas bukan cadangan LPJ BOS yang valid.")
		c.Redirect("/backup")
	}

	work := filepath.Join(Cfg.TmpDir, "restore_"+randomHex(4))
	if err := os.MkdirAll(filepath.Join(work, "uploads"), 0o775); err != nil {
		c.Abort(500, "Gagal menyiapkan folder pemulihan.")
	}
	defer rrmdir(work)
	for _, zf := range zr.File {
		if zf.Name != "lpjbos.sqlite" && !uploadNameRe.MatchString(zf.Name) {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			continue
		}
		dest := filepath.Join(work, filepath.FromSlash(zf.Name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o775); err != nil {
			_ = rc.Close()
			continue
		}
		w, err := os.Create(dest)
		if err == nil {
			_, err = io.Copy(w, rc)
			_ = w.Close()
		}
		_ = rc.Close()
		if err != nil {
			c.Flash("danger", "Berkas cadangan tidak bisa dibaca.")
			c.Redirect("/backup")
		}
	}

	newDB := filepath.Join(work, "lpjbos.sqlite")
	if !backupDBValid(newDB) {
		c.Flash("danger", "Database di dalam cadangan rusak atau bukan milik LPJ BOS.")
		c.Redirect("/backup")
	}

	_ = Q.Close()
	defer func() { _ = ReopenDB() }()
	backupCurrentDB()
	_ = os.Remove(Cfg.DBPath + "-wal")
	_ = os.Remove(Cfg.DBPath + "-shm")
	if err := os.Rename(newDB, Cfg.DBPath); err != nil {
		c.Flash("danger", "Gagal menyalin database dari cadangan.")
		c.Redirect("/backup")
	}
	for _, e := range uploadFiles() {
		_ = os.Remove(filepath.Join(Cfg.UploadDir, e.Name()))
	}
	rrmdir(filepath.Join(Cfg.UploadDir, "pdf-cache"))
	_ = os.MkdirAll(filepath.Join(Cfg.UploadDir, "pdf-cache"), 0o775)
	if entries, err := os.ReadDir(filepath.Join(work, "uploads")); err == nil {
		for _, e := range entries {
			_ = os.Rename(filepath.Join(work, "uploads", e.Name()), filepath.Join(Cfg.UploadDir, e.Name()))
		}
	}
	c.Sess.Reset()
	c.Flash("success", "Data berhasil dipulihkan. Silakan masuk dengan akun dari cadangan.")
	c.Redirect("/login")
}

// backupDBValid memastikan berkas SQLite berisi tabel inti aplikasi.
func backupDBValid(path string) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	d, err := openDB(path)
	if err != nil {
		return false
	}
	defer d.Close()
	return d.Int("SELECT COUNT(*) FROM sqlite_master WHERE name IN ('sekolah','users','penerima')") == 3
}

// backupCurrentDB menyalin database lama ke tmp sebagai jaring aman.
func backupCurrentDB() {
	src, err := os.Open(Cfg.DBPath)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.Create(filepath.Join(Cfg.TmpDir, "sebelum-pulihkan-"+time.Now().Format("20060102-150405")+".sqlite"))
	if err != nil {
		return
	}
	_, _ = io.Copy(dst, src)
	_ = dst.Close()
}

func init() {
	Register(90, func(rt *Router) {
		rt.Get("/backup", func(c *Ctx) {
			var dbSize int64
			if st, err := os.Stat(Cfg.DBPath); err == nil {
				dbSize = st.Size()
			}
			files := uploadFiles()
			var uploadSize int64
			for _, e := range files {
				if info, err := e.Info(); err == nil {
					uploadSize += info.Size()
				}
			}
			c.View("backup/index", D{
				"title": "Cadangkan & pulihkan", "dbSize": dbSize,
				"uploadCount": len(files), "uploadSize": uploadSize,
			})
		})
		rt.Post("/backup/unduh", backupDownload)
		rt.Post("/backup/pulihkan", backupRestore)
	})
}

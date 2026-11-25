package app

import (
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPdfImagePathMengecilkanDanCache(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "kop_x.jpg")
	f, _ := os.Create(src)
	_ = jpeg.Encode(f, image.NewRGBA(image.Rect(0, 0, 3000, 600)), nil)
	f.Close()

	out := pdfImagePath(src, 1600)
	if out == src || filepath.Dir(out) != filepath.Join(dir, "pdf-cache") {
		t.Fatalf("harus memakai salinan pdf-cache, dapat %s", out)
	}
	g, _ := os.Open(out)
	cfg, _, err := image.DecodeConfig(g)
	g.Close()
	if err != nil || cfg.Width != 1600 || cfg.Height != 320 {
		t.Fatalf("ukuran salah: %+v %v", cfg, err)
	}
	if again := pdfImagePath(src, 1600); again != out {
		t.Fatalf("cache tidak dipakai ulang: %s", again)
	}

	kecil := filepath.Join(dir, "kecil.jpg")
	f, _ = os.Create(kecil)
	_ = jpeg.Encode(f, image.NewRGBA(image.Rect(0, 0, 800, 200)), nil)
	f.Close()
	if got := pdfImagePath(kecil, 1600); got != kecil {
		t.Fatalf("gambar kecil harus dipakai apa adanya, dapat %s", got)
	}
}

// blockingWriter meniru klien lambat: Write memeriksa apakah appMu sudah bebas.
type blockingWriter struct {
	*httptest.ResponseRecorder
	lockFree bool
}

func (b *blockingWriter) Write(p []byte) (int, error) {
	if appMu.TryLock() {
		b.lockFree = true
		appMu.Unlock()
	}
	return b.ResponseRecorder.Write(p)
}

func TestDownloadMelepasKunciSebelumMenulis(t *testing.T) {
	w := &blockingWriter{ResponseRecorder: httptest.NewRecorder()}
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	c := &Ctx{W: w, R: r, Sess: &Session{Data: &SessionData{}}}
	c.unlock = lockApp(false)
	func() {
		defer func() { _ = recover() }()
		c.Download("application/pdf", "a.pdf", []byte("%PDF"), true)
	}()
	c.releaseLock()
	if !w.lockFree {
		t.Fatal("appMu masih dipegang saat body dikirim ke klien")
	}
}

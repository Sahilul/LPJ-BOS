package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Tulis (Q.Tx) dan baca bersamaan tidak boleh saling masuk transaksi atau macet.
func TestConcurrentRequests(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("DB_PATH", "")
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	defer Q.Close()
	Q.Insert("users", Row{"username": "admin", "nama": "Admin", "password_hash": hashPassword("rahasia123")})
	Q.Run("INSERT OR IGNORE INTO tahun_anggaran (tahun, keterangan, is_default) VALUES (2026, '', 0)")
	token, _ := CreateToken("uji", AksesTulis)
	h := Handler()

	var wg sync.WaitGroup
	errs := make(chan string, 100)
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"penerimaan_simpan","arguments":{"tgl":"2026-01-%02d","tahap":"Tahap 1","uraian":"Dana %d","jumlah":1000}}}`, i, i%28+1, i)
			req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != 200 || strings.Contains(rec.Body.String(), `"isError":true`) {
				errs <- fmt.Sprintf("mcp %d: %d %s", i, rec.Code, rec.Body.String())
			}
		}(i)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", "/login", nil))
			if rec.Code != http.StatusOK {
				errs <- fmt.Sprintf("login: %d", rec.Code)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	if n := Q.Int("SELECT COUNT(*) FROM penerimaan"); n != 20 {
		t.Fatalf("penerimaan = %d, want 20", n)
	}
	if n := Q.Int("SELECT COUNT(DISTINCT no_bukti) FROM penerimaan"); n != 20 {
		t.Fatalf("no_bukti unik = %d, want 20", n)
	}
}

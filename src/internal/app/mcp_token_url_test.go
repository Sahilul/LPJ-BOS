package app

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Klien seperti ChatGPT tidak bisa mengirim header kustom, jadi token boleh
// ditaruh di path: POST /mcp/<token>.
func TestMcpTokenDiURL(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("DB_PATH", "")
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	defer Q.Close()
	Q.Insert("users", Row{"username": "admin", "nama": "Admin", "password_hash": hashPassword("rahasia123")})
	token, _ := CreateToken("chatgpt", AksesBaca)
	h := Handler()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/mcp/"+token, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("token di URL: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"protocolVersion":"2025-06-18"`) {
		t.Fatalf("versi protokol tidak diikuti: %s", rec.Body.String())
	}

	// Token salah di path tetap ditolak.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/mcp/lpbos_salah", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("token salah: %d, want 401", rec.Code)
	}

	// Header tetap menang / tetap didukung.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("header bearer: %d %s", rec.Code, rec.Body.String())
	}
}

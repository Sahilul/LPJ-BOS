package app

import (
	"strings"
	"testing"
)

// Nomor otomatis tidak boleh bentrok setelah rapat lama dihapus.
func TestRapatNomorSetelahHapus(t *testing.T) {
	b1 := setupRapat(t)
	rencanaID := Q.Int("SELECT id FROM rencana_belanja ORDER BY id LIMIT 1")
	b2 := Q.Insert("belanja", Row{"no_bukti": "BKU-002", "tgl": "2026-04-05", "rencana_id": rencanaID,
		"jenis": "barang", "uraian": "Kedua", "metode": "tunai", "total": 1000})
	b3 := Q.Insert("belanja", Row{"no_bukti": "BKU-003", "tgl": "2026-04-06", "rencana_id": rencanaID,
		"jenis": "barang", "uraian": "Ketiga", "metode": "tunai", "total": 1000})
	buat := func(b int64) int64 {
		d, err := RapatValidate(formRapat(b, func(f Form) { f["tgl_surat"], f["tgl"] = "2026-04-06", "2026-04-06" }), 0, 2026)
		if err != nil {
			t.Fatal(err)
		}
		return RapatCreate(d)
	}
	id1 := buat(b1)
	buat(b2) // 002
	RapatDelete(RapatFind(id1))
	if _, err := RapatValidate(formRapat(b3, func(f Form) { f["tgl_surat"], f["tgl"] = "2026-04-06", "2026-04-06" }), 0, 2026); err != nil {
		t.Fatalf("nomor otomatis bentrok setelah hapus: %v", err)
	}
}

func TestBarisTeksAngkaRibuan(t *testing.T) {
	got := barisTeks("1. Evaluasi\n2) Rencana\n1.500 buku dibagikan\n- Penutup")
	mau := []string{"Evaluasi", "Rencana", "1.500 buku dibagikan", "Penutup"}
	if strings.Join(got, "|") != strings.Join(mau, "|") {
		t.Fatalf("barisTeks = %q, mau %q", got, mau)
	}
}

// Tahun rapat lewat MCP mengikuti belanja, dan detail memuat tautan PDF.
func TestMcpRapatTahunDanDetail(t *testing.T) {
	belanjaID := setupRapat(t)
	simpan := mcpCari("rapat_simpan")
	// Tanpa tgl: ditolak lewat validasi (bukan panik), tahun dari belanja.
	if _, err := simpan.Handler(McpArgs{"belanja_id": belanjaID, "agenda": "X"}); err == nil {
		t.Fatal("tanpa tgl seharusnya ditolak")
	}
	out, err := simpan.Handler(McpArgs{"belanja_id": belanjaID, "tgl": "2026-03-05", "agenda": "Evaluasi"})
	if err != nil {
		t.Fatal(err)
	}
	data := out.(OM)[2].V.(OM)
	var pdf OM
	for _, kv := range data {
		if kv.K == "pdf" {
			pdf = kv.V.(OM)
		}
		if kv.K == "sekolah" || kv.K == "ident" {
			t.Fatalf("detail MCP tidak boleh memuat %s", kv.K)
		}
	}
	if len(pdf) != 1 || !strings.HasSuffix(toStr(pdf[0].V), "/undangan.pdf") {
		t.Fatalf("tautan PDF salah: %v", pdf)
	}
}

package app

import (
	"strings"
	"testing"
)

// Tautan laporan harus membawa periode utuh (bukan "periode%3dT2"),
// agar Tahap II tidak jatuh ke bulan berjalan.
func TestLaporanTautanPeriode(t *testing.T) {
	setupBelanja(t, false, "")
	p, _ := LapPeriode("T2", 2026)
	html := RenderString("laporan/index", D{"p": p, "saldo": SaldoKas(p.S("akhir"), 0, 0), "jenisList": LapJenis, "C": &Ctx{}})
	if !strings.Contains(html, `href="/laporan/bku?periode=T2"`) {
		t.Fatalf("tautan BKU tidak membawa periode T2")
	}
	d := lapBKU(p)
	html = RenderString("laporan/show", D{"p": p, "jenis": "bku", "nama": "Buku kas umum", "data": d, "headers": lapHeadersOf(d), "rows": lapRowViews(d), "summary": lapSumViews(d), "C": &Ctx{}})
	if !strings.Contains(html, `href="/laporan?periode=T2"`) || strings.Contains(html, "%3d") {
		t.Fatalf("tautan kembali tidak membawa periode T2")
	}
}

package app

import (
	"strings"
	"testing"
)

// setupRapat menyiapkan satu belanja BKU tahun 2026 untuk diuji dengan rapat.
func setupRapat(t *testing.T) int64 {
	t.Helper()
	tokoID := setupBelanja(t, false, "")
	rencanaID := Q.Int("SELECT id FROM rencana_belanja ORDER BY id LIMIT 1")
	if rencanaID == 0 {
		t.Fatal("rencana belanja uji tidak terbentuk")
	}
	return Q.Insert("belanja", Row{
		"no_bukti": "BKU-001", "tgl": "2026-03-05", "rencana_id": rencanaID,
		"jenis": "barang", "toko_id": tokoID, "uraian": "Konsumsi rapat evaluasi",
		"metode": "tunai", "total": 500000,
	})
}

// formRapat membangun isian POST undangan & notulen.
func formRapat(belanjaID int64, ubah func(Form)) Form {
	f := Form{
		"belanja_id":  toStr(belanjaID),
		"tgl_surat":   "2026-03-02",
		"tgl":         "2026-03-05",
		"sifat":       "Penting",
		"perihal":     "Undangan Rapat Evaluasi",
		"kepada":      "Dewan Guru",
		"jam_mulai":   "09:00",
		"jam_selesai": "11:00",
		"tempat":      "Ruang Guru",
		"agenda":      "Evaluasi program",
		"pimpinan":    "Kepala Madrasah",
		"notulis":     "Wali Kelas",
	}
	if ubah != nil {
		ubah(f)
	}
	return f
}

func TestAngkaRomawi(t *testing.T) {
	kasus := map[int]string{1: "I", 4: "IV", 9: "IX", 11: "XI", 12: "XII", 2026: "MMXXVI"}
	for n, mau := range kasus {
		if got := angkaRomawi(n); got != mau {
			t.Errorf("angkaRomawi(%d) = %q, mau %q", n, got, mau)
		}
	}
}

func TestRapatNomorBerikut(t *testing.T) {
	setupRapat(t)
	tahun := 2026
	no := RapatNomorBerikut(tahun, "2026-03-02")
	if !strings.HasSuffix(no, "/III/2026") {
		t.Fatalf("nomor undangan %q tidak berakhiran /III/2026", no)
	}
	if !strings.HasPrefix(no, "001/UND/") {
		t.Fatalf("nomor undangan %q tidak dimulai 001/UND/", no)
	}
}

func TestRapatValidateBelanjaWajib(t *testing.T) {
	setupRapat(t)
	_, err := RapatValidate(formRapat(0, nil), 0, 2026)
	ve := errValidasi(t, err)
	if ve == nil || ve.Errors["belanja_id"] == "" {
		t.Fatalf("belanja kosong harus ditolak, dapat %v", err)
	}
}

func TestRapatValidateTanggal(t *testing.T) {
	setupRapat(t)
	id := Q.Int("SELECT id FROM belanja ORDER BY id LIMIT 1")
	// Tanggal rapat sebelum tanggal surat.
	_, err := RapatValidate(formRapat(id, func(f Form) { f["tgl"] = "2026-03-01" }), 0, 2026)
	ve := errValidasi(t, err)
	if ve == nil || ve.Errors["tgl"] == "" {
		t.Fatalf("tanggal rapat sebelum surat harus ditolak, dapat %v", err)
	}
	// Jam selesai lebih awal.
	_, err = RapatValidate(formRapat(id, func(f Form) { f["jam_selesai"] = "08:00" }), 0, 2026)
	ve = errValidasi(t, err)
	if ve == nil || ve.Errors["jam_selesai"] == "" {
		t.Fatalf("jam selesai lebih awal harus ditolak, dapat %v", err)
	}
	// Tanggal di luar tahun anggaran.
	_, err = RapatValidate(formRapat(id, func(f Form) {
		f["tgl_surat"] = "2025-12-01"
		f["tgl"] = "2025-12-05"
	}), 0, 2026)
	ve = errValidasi(t, err)
	if ve == nil || ve.Errors["tgl_surat"] == "" {
		t.Fatalf("tanggal di luar tahun anggaran harus ditolak, dapat %v", err)
	}
}

func TestRapatCreateSekaliPerBelanja(t *testing.T) {
	belanjaID := setupRapat(t)
	data, err := RapatValidate(formRapat(belanjaID, nil), 0, 2026)
	if err != nil {
		t.Fatal(err)
	}
	id := RapatCreate(data)
	row := RapatFind(id)
	if row == nil || row.I("belanja_id") != belanjaID {
		t.Fatalf("rapat tidak tersimpan: %v", row)
	}
	// Belanja yang sama tidak boleh dibuatkan rapat kedua.
	_, err = RapatValidate(formRapat(belanjaID, nil), 0, 2026)
	ve := errValidasi(t, err)
	if ve == nil || ve.Errors["belanja_id"] == "" {
		t.Fatalf("rapat ganda pada satu belanja harus ditolak, dapat %v", err)
	}
	// Setelah rapat pertama dihapus, belanja kembali menjadi kandidat.
	RapatDelete(row)
	if RapatFind(id) != nil {
		t.Fatal("rapat tidak terhapus")
	}
	if n := len(RapatBelanjaKandidat("", 2026, 0)); n != 1 {
		t.Fatalf("kandidat belanja setelah hapus = %d, mau 1", n)
	}
}

func TestRapatValidasiNomorGanda(t *testing.T) {
	belanjaID := setupRapat(t)
	data, err := RapatValidate(formRapat(belanjaID, func(f Form) { f["no_undangan"] = "001/UND/X/2026" }), 0, 2026)
	if err != nil {
		t.Fatal(err)
	}
	RapatCreate(data)
	// Belanja kedua dengan nomor undangan sama.
	rencanaID := Q.Int("SELECT id FROM rencana_belanja ORDER BY id LIMIT 1")
	kedua := Q.Insert("belanja", Row{
		"no_bukti": "BKU-002", "tgl": "2026-04-05", "rencana_id": rencanaID,
		"jenis": "barang", "uraian": "Belanja kedua", "metode": "tunai", "total": 250000,
	})
	_, err = RapatValidate(formRapat(kedua, func(f Form) {
		f["no_undangan"] = "001/UND/X/2026"
		f["tgl_surat"] = "2026-04-01"
		f["tgl"] = "2026-04-05"
	}), 0, 2026)
	ve := errValidasi(t, err)
	if ve == nil || ve.Errors["no_undangan"] == "" {
		t.Fatalf("nomor undangan ganda harus ditolak, dapat %v", err)
	}
}

func TestRapatPdf(t *testing.T) {
	belanjaID := setupRapat(t)
	data, err := RapatValidate(formRapat(belanjaID, func(f Form) {
		f["pembukaan"] = "Rapat dibuka pukul 09.00."
		f["pembahasan"] = "1. Evaluasi program\n2. Rencana tindak lanjut"
		f["keputusan"] = "Program dilanjutkan."
	}), 0, 2026)
	if err != nil {
		t.Fatal(err)
	}
	id := RapatCreate(data)
	d := rapatDoc(id, 2026)
	if d == nil {
		t.Fatal("rapatDoc kosong")
	}
	if pdf := rapatPdf(d, false); len(pdf) < 1000 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("PDF undangan tidak valid (%d byte)", len(pdf))
	}
	if pdf := rapatPdf(d, true); len(pdf) < 1000 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("PDF notulen tidak valid (%d byte)", len(pdf))
	}
}

func TestRapatTemplateRender(t *testing.T) {
	belanjaID := setupRapat(t)
	id := RapatCreate(mustRapat(t, belanjaID))
	rows := Q.All(`SELECT p.*, b.uraian AS belanja_uraian, b.total AS belanja_total FROM rapat p JOIN belanja b ON b.id = p.belanja_id`)
	for _, r := range rows {
		r["ada_notulen"] = false
		r["jam"] = r.S("jam_mulai")
	}
	c := &Ctx{Sess: &Session{Data: &SessionData{Tahun: 2026}}}
	html := RenderString("rapat/index", D{
		"rows": rows, "tahun": 2026, "komponen": "", "komponenOpts": rapatKomponenFilter(2026),
		"nKandidat": 0, "nNotulen": 0, "C": c,
	})
	for _, wajib := range []string{"Undangan", "notulen", "/rapat/" + toStr(id) + "/edit"} {
		if !strings.Contains(html, wajib) {
			t.Errorf("halaman index rapat tidak memuat %q", wajib)
		}
	}
	form := RenderString("rapat/form", D{
		"title": "Buat", "edit": false, "row": Row{}, "tahun": 2026,
		"belanjaOpts": []Opt{{"", "— Pilih belanja BKU —"}, {toStr(belanjaID), "Belanja uji"}},
		"komponen":    "", "komponenOpts": rapatKomponenFilter(2026), "sifatOpts": RapatSifat,
		"kandidat": 1, "belanjaVal": "", "vals": map[string]any{}, "C": c,
	})
	for _, wajib := range []string{`name="belanja_id"`, `name="pembahasan"`, `name="no_undangan"`} {
		if !strings.Contains(form, wajib) {
			t.Errorf("form rapat tidak memuat %q", wajib)
		}
	}
}

// TestSpjMemuatRapat memastikan undangan & notulen ikut terlampir pada PDF SPJ.
func TestSpjMemuatRapat(t *testing.T) {
	belanjaID := setupRapat(t)
	data := mustRapat(t, belanjaID)
	data["pembahasan"] = "Evaluasi program BOS"
	id := RapatCreate(data)

	p, ok := LapPeriode("Y", 2026)
	if !ok {
		t.Fatal("periode setahun tidak dikenali")
	}
	belanja := LapBelanjaPeriode(p)
	if len(belanja) != 1 {
		t.Fatalf("belanja periode = %d, mau 1", len(belanja))
	}
	if d := rapatUntukBelanja(belanja); len(d) != 1 || d[0].I("id") != id {
		t.Fatalf("rapatUntukBelanja = %d dokumen, mau 1", len(d))
	}
	pdf := lapPdf([]*LapData{LapBuild("bku", p)}, p, belanja, true)
	if len(pdf) < 2000 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("PDF SPJ tidak valid (%d byte)", len(pdf))
	}
	// SPJ tanpa rapat tetap bisa dibuat.
	Q.Run("DELETE FROM rapat")
	if pdf := lapPdf([]*LapData{LapBuild("bku", p)}, p, LapBelanjaPeriode(p), true); len(pdf) < 1000 {
		t.Fatalf("PDF SPJ tanpa rapat terlalu kecil (%d byte)", len(pdf))
	}
}

func mustRapat(t *testing.T, belanjaID int64) Row {
	t.Helper()
	data, err := RapatValidate(formRapat(belanjaID, nil), 0, 2026)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestBkuTombolRapat memastikan tombol undangan rapat tampil di BKU: buat baru
// bila belum ada, cetak PDF bila rapat sudah dibuat.
func TestBkuTombolRapat(t *testing.T) {
	belanjaID := setupRapat(t)
	c := &Ctx{Sess: &Session{Data: &SessionData{Tahun: 2026}}}
	data := func(rapatID int64, notulen ...bool) D {
		nNotulen, rn := 0, int64(0)
		nRapatTahun := 0
		if rapatID > 0 {
			nRapatTahun = 1
		}
		if len(notulen) > 0 && notulen[0] {
			nNotulen, rn = 1, 1
		}
		return D{
			"tahun": 2026, "bulan": 3, "periode": "3", "bulanNama": "Maret", "bulanan": true,
			"rows": []Row{{
				"src": "bel", "id": belanjaID, "tgl": "2026-03-05", "no_bukti": "BKU-001",
				"uraian": "Konsumsi rapat", "debit": 0.0, "kredit": 500000.0,
				"metode": "tunai", "jenis": "barang", "nilai": 500000.0, "saldo": 1000000.0,
				"lampiran": int64(0), "rapat_id": rapatID, "rapat_notulen": rn, "jenis_label": "Barang / jasa",
				"link": "/belanja/1/edit",
			}},
			"nKuitansi": 1, "nBukti": 0, "nRapat": 0, "nNotulen": nNotulen,
			"nRapatTahun": nRapatTahun, "nNotulenTahun": nNotulen,
			"saldoAwal": Saldo{}, "saldoAkhir": Saldo{}, "debit": 0.0, "kredit": 500000.0,
			"perBulan": map[int]int64{3: 1}, "C": c,
		}
	}
	html := RenderString("bku/index", data(0))
	if !strings.Contains(html, "/rapat/baru?belanja="+toStr(belanjaID)) {
		t.Error("tombol buat undangan rapat tidak muncul di BKU")
	}
	id := RapatCreate(mustRapat(t, belanjaID))
	html = RenderString("bku/index", data(id))
	if !strings.Contains(html, "/dokumen/rapat/"+toStr(id)+"/undangan.pdf") {
		t.Error("tombol cetak undangan rapat tidak muncul di BKU setelah rapat dibuat")
	}
	if strings.Contains(html, "notulen.pdf") || strings.Contains(html, "jenis=notulen") {
		t.Error("tombol notulen tidak boleh muncul sebelum notulen diisi")
	}
	html = RenderString("bku/index", data(id, true))
	if !strings.Contains(html, "/dokumen/rapat/"+toStr(id)+"/notulen.pdf") {
		t.Error("tombol cetak notulen per baris tidak muncul di BKU")
	}
	if !strings.Contains(html, "/dokumen/bku/rapat.pdf?periode=Y&amp;jenis=notulen") {
		t.Error("tombol cetak notulen tahunan tidak muncul di BKU")
	}
}

package app

import (
	"testing"
)

// setupBelanja menyiapkan database sementara berisi rencana, toko, dan penerima
// untuk menguji perhitungan pajak end-to-end lewat BelanjaValidate.
func setupBelanja(t *testing.T, pkp bool, npwpToko string) int64 {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("DB_PATH", "")
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Q.Close() })

	Q.Run(`INSERT OR IGNORE INTO tahun_anggaran (tahun, keterangan, is_default) VALUES (2026, '', 1)`)
	pkpInt := 0
	if pkp {
		pkpInt = 1
	}
	tokoID := Q.Insert("toko", Row{"nama": "Toko Uji", "npwp": npwpToko, "pkp": pkpInt, "aktif": 1})
	rencanaID := Q.Insert("rencana_belanja", Row{
		"tahun": 2026, "komponen_kode": "01", "uraian": "Paket Uji",
		"jenis": "barang", "pagu": 100000000, "frekuensi_bulan": 12, "toko_id": tokoID,
	})
	_ = rencanaID
	return tokoID
}

// formBelanja membangun isian POST belanja barang.
func formBelanja(tokoID int64, total float64, ppn, pph22, pph23 bool, danaBOS, siplah bool) Form {
	f := Form{
		"rencana_id":     toStr(Q.Int("SELECT id FROM rencana_belanja WHERE jenis = 'barang' ORDER BY id LIMIT 1")),
		"tgl":            "2026-03-05",
		"uraian":         "Belanja uji",
		"metode":         "tunai",
		"jumlah_periode": "1",
		"items": Form{"0": Form{
			"nama": "Barang uji", "volume": "1", "satuan": "unit", "harga": toStr(total),
		}},
	}
	if tokoID != 0 {
		f["toko_id"] = toStr(tokoID)
	}
	if ppn {
		f["pajak_ppn"] = "1"
	}
	if pph22 {
		f["pajak_pph22"] = "1"
	}
	if pph23 {
		f["pajak_pph23"] = "1"
	}
	if danaBOS {
		f["dana_bos"] = "1"
	}
	if siplah {
		f["via_siplah"] = "1"
	}
	return f
}

// errValidasi mengambil ValidationError dari hasil BelanjaValidate.
func errValidasi(t *testing.T, err error) *ValidationError {
	t.Helper()
	if err == nil {
		return nil
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("error bukan ValidationError: %T %v", err, err)
	}
	return ve
}

// pajakDari mengubah daftar pajak hasil validasi menjadi map jenis -> nilai.
func pajakDari(anak BelanjaAnak) map[string]float64 {
	m := map[string]float64{}
	for _, p := range anak.Pajak {
		m[p.S("jenis")] = p.F("nilai")
	}
	return m
}

func TestBelanjaPPNPakaiDPPNilaiLain(t *testing.T) {
	toko := setupBelanja(t, true, "0123456789012345")
	data, anak, err := BelanjaValidate(formBelanja(toko, 2775000, true, false, false, false, false), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	pj := pajakDari(anak)
	// 11/111 x Rp2.775.000 = Rp275.000 (contoh PMK 59/2022).
	if pj["PPN"] != 275000 {
		t.Errorf("PPN = %v, want 275000", pj["PPN"])
	}
	if data.F("total") != 2775000 {
		t.Errorf("total = %v, want 2775000", data.F("total"))
	}
}

func TestBelanjaPPNDiBawahBatasDitolak(t *testing.T) {
	toko := setupBelanja(t, true, "0123456789012345")
	// Nota Rp2.109.000 -> DPP Rp1.900.000, sehingga PPN tidak boleh dipungut.
	_, _, err := BelanjaValidate(formBelanja(toko, 2109000, true, false, false, false, false), 0, 2026)
	if err == nil {
		t.Fatal("PPN untuk DPP di bawah Rp2 juta seharusnya ditolak")
	}
	ve := errValidasi(t, err)
	if ve.Errors["pajak_ppn"] == "" {
		t.Errorf("pesan kesalahan pajak_ppn tidak ada: %v", ve.Errors)
	}
}

func TestBelanjaPPh22BOSDitolak(t *testing.T) {
	toko := setupBelanja(t, true, "0123456789012345")
	// Dana BOS: PPh 22 tidak boleh dipungut meski nilai besar.
	_, _, err := BelanjaValidate(formBelanja(toko, 10000000, false, true, false, true, false), 0, 2026)
	if err == nil {
		t.Fatal("PPh 22 untuk dana BOS seharusnya ditolak")
	}
	ve := errValidasi(t, err)
	if ve.Errors["pajak_pph22"] == "" {
		t.Errorf("pesan kesalahan pajak_pph22 tidak ada: %v", ve.Errors)
	}
}

func TestBelanjaPPh22BukanBOSDiterima(t *testing.T) {
	toko := setupBelanja(t, true, "0123456789012345")
	// Bukan dana BOS, nilai di atas Rp2 juta: PPh 22 1,5% x Rp10.000.000.
	_, anak, err := BelanjaValidate(formBelanja(toko, 10000000, false, true, false, false, false), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	if got := pajakDari(anak)["PPH22"]; got != 150000 {
		t.Errorf("PPh 22 = %v, want 150000", got)
	}
}

func TestBelanjaPPh23TanpaNPWPTarifGanda(t *testing.T) {
	toko := setupBelanja(t, true, "")
	_, anak, err := BelanjaValidate(formBelanja(toko, 10000000, false, false, true, false, false), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	// Tanpa NPWP: 4% x Rp10.000.000 = Rp400.000.
	if got := pajakDari(anak)["PPH23"]; got != 400000 {
		t.Errorf("PPh 23 tanpa NPWP = %v, want 400000", got)
	}
}

func TestBelanjaPPh23DenganNPWPTarifNormal(t *testing.T) {
	toko := setupBelanja(t, true, "0123456789012345")
	_, anak, err := BelanjaValidate(formBelanja(toko, 10000000, false, false, true, false, false), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	// Ber-NPWP: 2% x Rp10.000.000 = Rp200.000.
	if got := pajakDari(anak)["PPH23"]; got != 200000 {
		t.Errorf("PPh 23 ber-NPWP = %v, want 200000", got)
	}
}

// setupHonor menyiapkan rencana honor dan mengembalikan id penerima.
func setupHonor(t *testing.T, kategori, golongan, jenis string) int64 {
	t.Helper()
	data := Row{
		"nama": "Penerima Uji", "kategori": kategori, "golongan": golongan,
		"jenis_penerima": jenis, "ptkp": "A", "aktif": 1,
	}
	pid := Q.Insert("penerima", data)
	Q.Run(`INSERT OR IGNORE INTO tahun_anggaran (tahun, keterangan, is_default) VALUES (2026, '', 1)`)
	Q.Insert("rencana_belanja", Row{
		"tahun": 2026, "komponen_kode": "01", "uraian": "Honor Uji",
		"jenis": "honor", "pagu": 100000000, "frekuensi_bulan": 12,
	})
	return pid
}

func formHonor(pid int64, bruto float64, hari string) Form {
	f := Form{
		"rencana_id":     toStr(Q.Int("SELECT id FROM rencana_belanja WHERE jenis = 'honor' ORDER BY id LIMIT 1")),
		"tgl":            "2026-03-05",
		"uraian":         "Honor uji",
		"metode":         "tunai",
		"jumlah_periode": "1",
		"pen": Form{"0": Form{
			"penerima_id": toStr(pid), "volume": "1", "satuan": "OK",
			"tarif": toStr(bruto), "hari": hari,
		}},
	}
	return f
}

func TestBelanjaHonorNarasumber(t *testing.T) {
	setupBelanja(t, true, "")
	pid := setupHonor(t, "Luar", "", "bukan_pegawai")
	_, anak, err := BelanjaValidate(formHonor(pid, 3000000, "0"), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	if len(anak.Penerima) == 0 {
		t.Fatal("baris penerima tidak terbentuk")
	}
	// 50% x Rp3.000.000 = Rp1.500.000, tarif Pasal 17 5% = Rp75.000.
	if got := anak.Penerima[0].F("pph21"); got != 75000 {
		t.Errorf("PPh 21 narasumber = %v, want 75000", got)
	}
	if got := anak.Penerima[0].F("netto"); got != 2925000 {
		t.Errorf("netto = %v, want 2925000", got)
	}
}

func TestBelanjaHonorPNSGolonganIII(t *testing.T) {
	setupBelanja(t, true, "")
	pid := setupHonor(t, "PNS", "III/a", "tetap")
	_, anak, err := BelanjaValidate(formHonor(pid, 2000000, "0"), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	// Final 5% x Rp2.000.000 = Rp100.000.
	if got := anak.Penerima[0].F("pph21"); got != 100000 {
		t.Errorf("PPh 21 PNS gol III = %v, want 100000", got)
	}
}

func TestBelanjaHonorGTTDibayarHarian(t *testing.T) {
	setupBelanja(t, true, "")
	pid := setupHonor(t, "GTT", "", "tidak_tetap")
	// Rp4.500.000 untuk 10 hari = Rp450.000 sehari -> 0%.
	_, anak, err := BelanjaValidate(formHonor(pid, 4500000, "10"), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	if got := anak.Penerima[0].F("pph21"); got != 0 {
		t.Errorf("PPh 21 GTT Rp450.000/hari = %v, want 0", got)
	}
}

func TestBelanjaHonorGTTUpahSetengahJuta(t *testing.T) {
	setupBelanja(t, true, "")
	pid := setupHonor(t, "GTT", "", "tidak_tetap")
	// Rp5.000.000 untuk 10 hari = Rp500.000 sehari -> 0,5% x Rp5.000.000 = Rp25.000.
	_, anak, err := BelanjaValidate(formHonor(pid, 5000000, "10"), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	if got := anak.Penerima[0].F("pph21"); got != 25000 {
		t.Errorf("PPh 21 GTT Rp500.000/hari = %v, want 25000", got)
	}
}

func TestBelanjaHonorGTTBulananDiBawahTER(t *testing.T) {
	setupBelanja(t, true, "")
	pid := setupHonor(t, "GTT", "", "tidak_tetap")
	// Honor bulanan Rp544.000 (hari kosong): TER A 0% s.d. Rp5.400.000 sebulan.
	_, anak, err := BelanjaValidate(formHonor(pid, 544000, ""), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	if got := anak.Penerima[0].F("pph21"); got != 0 {
		t.Errorf("PPh 21 GTT bulanan Rp544.000 = %v, want 0", got)
	}
	if len(anak.Pajak) != 0 {
		t.Errorf("baris pajak = %v, want kosong", anak.Pajak)
	}
}

func TestBelanjaHonorGTTBulananDigabung(t *testing.T) {
	setupBelanja(t, true, "")
	pid := setupHonor(t, "GTT", "", "tidak_tetap")
	data, anak, err := BelanjaValidate(formHonor(pid, 5000000, ""), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	BelanjaCreate(data, anak)
	// Honor kedua bulan yang sama: total Rp5.500.000 -> TER A 0,25% = Rp13.750.
	_, anak, err = BelanjaValidate(formHonor(pid, 500000, ""), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	if got := anak.Penerima[0].F("pph21"); got != 13750 {
		t.Errorf("PPh 21 honor kedua = %v, want 13750", got)
	}
}

func TestBelanjaSwastaTolakPPNPPh22(t *testing.T) {
	toko := setupBelanja(t, true, "012345678901234")
	Q.Run("UPDATE sekolah SET status = 'Swasta' WHERE id = 1")
	_, _, err := BelanjaValidate(formBelanja(toko, 5000000, true, false, false, true, false), 0, 2026)
	if err == nil || errValidasi(t, err).Errors["pajak_ppn"] == "" {
		t.Errorf("PPN sekolah swasta harus ditolak, err = %v", err)
	}
	_, _, err = BelanjaValidate(formBelanja(toko, 5000000, false, true, false, false, false), 0, 2026)
	if err == nil || errValidasi(t, err).Errors["pajak_pph22"] == "" {
		t.Errorf("PPh 22 sekolah swasta harus ditolak, err = %v", err)
	}
}

func TestHitungUlangPPh21HonorBulananLama(t *testing.T) {
	setupBelanja(t, true, "")
	pid := setupHonor(t, "GTT", "", "tidak_tetap")
	data, anak, err := BelanjaValidate(formHonor(pid, 544000, ""), 0, 2026)
	if err != nil {
		t.Fatalf("validasi: %v", errValidasi(t, err).Errors)
	}
	bid := BelanjaCreate(data, anak)
	// Data lama: tercatat 1 hari dengan PPh 21 tarif lama 0,5%.
	Q.Run("UPDATE penerima_belanja SET hari = 1, pph21 = 2720, netto = bruto - 2720 WHERE belanja_id = ?", bid)
	Q.Insert("pajak", Row{"belanja_id": bid, "jenis": "PPH21", "dpp": 544000, "tarif": 0.5, "nilai": 2720})
	if n := HitungUlangPPh21(); n != 1 {
		t.Errorf("transaksi berubah = %d, want 1", n)
	}
	r := Q.One("SELECT * FROM penerima_belanja WHERE belanja_id = ?", bid)
	if r.F("pph21") != 0 || r.F("netto") != 544000 || r.I("hari") != 0 {
		t.Errorf("penerima setelah hitung ulang = %v", r)
	}
	if n := Q.Int("SELECT COUNT(*) FROM pajak WHERE belanja_id = ?", bid); n != 0 {
		t.Errorf("baris PPh 21 tersisa = %d, want 0", n)
	}
}

package app

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMigrasiPajak memastikan migrasi aturan pajak terbaru dapat diterapkan
// pada database yang sudah berisi data lama, dan data lama tetap utuh.
func TestMigrasiPajak(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "uji.sqlite")
	db, err := openDB(path)
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	// Terapkan migrasi awal sampai sebelum 010 agar menyerupai database lama.
	if _, err := db.sqlDB.Exec(`CREATE TABLE migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("buat tabel migrations: %v", err)
	}
	for _, name := range []string{
		"001_init.sql", "002_seed.sql", "003_tahun_anggaran.sql",
		"004_lampiran_jenis.sql", "005_rencana_paket.sql",
		"006_rencana_pendapatan.sql", "007_nomor_bukti_bku.sql",
		"008_api_token.sql", "009_nomor_bukti_per_jenis.sql",
	} {
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatalf("baca %s: %v", name, err)
		}
		if _, err := db.sqlDB.Exec(string(body)); err != nil {
			t.Fatalf("jalankan %s: %v", name, err)
		}
		if _, err := db.sqlDB.Exec(`INSERT INTO migrations (name, applied_at) VALUES (?, 'x')`, name); err != nil {
			t.Fatalf("catat %s: %v", name, err)
		}
	}

	// Data lama: satu penerima pihak luar dan satu PNS.
	db.Run(`INSERT INTO penerima (nama, kategori, golongan, npwp) VALUES ('Narasumber Lama', 'Luar', '', '')`)
	db.Run(`INSERT INTO penerima (nama, kategori, golongan, npwp) VALUES ('Guru Lama', 'PNS', 'III/a', '123')`)

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Kolom baru harus ada dan terisi hasil turunan kategori.
	luar := db.One(`SELECT jenis_penerima, ptkp FROM penerima WHERE nama = 'Narasumber Lama'`)
	if luar == nil {
		t.Fatal("data penerima lama hilang setelah migrasi")
	}
	if luar.S("jenis_penerima") != "bukan_pegawai" {
		t.Errorf("jenis_penerima pihak luar = %q, want bukan_pegawai", luar.S("jenis_penerima"))
	}
	if luar.S("ptkp") != "A" {
		t.Errorf("ptkp bawaan = %q, want A", luar.S("ptkp"))
	}
	pns := db.One(`SELECT jenis_penerima FROM penerima WHERE nama = 'Guru Lama'`)
	if pns.S("jenis_penerima") != "tetap" {
		t.Errorf("jenis_penerima PNS = %q, want tetap", pns.S("jenis_penerima"))
	}

	// Kolom transaksi baru tersedia dengan nilai bawaan yang aman.
	if got := db.Int(`SELECT COUNT(*) FROM belanja WHERE dana_bos = 1`); got != 0 {
		t.Errorf("belanja lama tidak boleh otomatis dianggap dana BOS, dapat %d", got)
	}

	// Tarif PPh 21 non-ASN yang lama sudah tidak dipakai lagi.
	if got := db.Int(`SELECT COUNT(*) FROM tarif_pajak WHERE kode IN ('PPH21_NPWP','PPH21_NON_NPWP')`); got != 0 {
		t.Errorf("tarif PPh 21 flat lama masih ada: %d baris", got)
	}
	if got := db.Float(`SELECT tarif FROM tarif_pajak WHERE kode = 'PPH21_BUKAN_PEGAWAI'`); got != 2.5 {
		t.Errorf("tarif bukan pegawai = %v, want 2.5", got)
	}

	// Migrasi harus idempoten: dijalankan dua kali tidak gagal.
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate kedua kali: %v", err)
	}
}

// TestMigrasiPajakDatabaseKosong memastikan instalfresh juga berhasil.
func TestMigrasiPajakDatabaseKosong(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baru.sqlite")
	db, err := openDB(path)
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if got := db.Int(`SELECT COUNT(*) FROM tarif_pajak`); got == 0 {
		t.Error("tabel tarif_pajak kosong setelah migrasi")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("berkas database tidak dibuat: %v", err)
	}
}

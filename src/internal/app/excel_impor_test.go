package app

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestImporPenerimaKolomBaru(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("DB_PATH", "")
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	defer Q.Close()

	f := excelize.NewFile()
	sheet := "Sheet1"
	heads := []string{"Nama", "Kategori", "Jenis Penerima (tetap/tidak_tetap/bukan_pegawai/peserta)", "Kategori PTKP (A/B/C)"}
	for i, h := range heads {
		c, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, c, h)
	}
	rows := [][]string{
		{"Narasumber A", "Luar", "bukan_pegawai", "B"},
		{"Guru B", "PNS", "NGAWUR", "Z"},
	}
	for r, row := range rows {
		for i, v := range row {
			c, _ := excelize.CoordinatesToCellName(i+1, r+2)
			_ = f.SetCellValue(sheet, c, v)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	rowsData, err := penerimaImport("uji.xlsx", buf.Bytes())
	if err != nil {
		t.Fatalf("baca: %v", err)
	}
	if _, _, err := penerimaImporBaris(rowsData); err != nil {
		t.Fatalf("impor: %v", err)
	}
	a := Q.One(`SELECT jenis_penerima, ptkp FROM penerima WHERE nama = 'Narasumber A'`)
	if a.S("jenis_penerima") != "bukan_pegawai" || a.S("ptkp") != "B" {
		t.Errorf("baris valid salah: %v", a)
	}
	// Nilai tidak dikenal harus jatuh ke turunan kategori, bukan tersimpan mentah.
	b := Q.One(`SELECT jenis_penerima, ptkp FROM penerima WHERE nama = 'Guru B'`)
	if b.S("jenis_penerima") != "" {
		t.Errorf("jenis_penerima tidak dikenal seharusnya dikosongkan, dapat %q", b.S("jenis_penerima"))
	}
	if b.S("ptkp") == "Z" {
		t.Error("ptkp tidak valid tidak boleh tersimpan")
	}
}

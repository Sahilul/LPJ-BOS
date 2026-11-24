package app

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var penerimaCols = []struct{ Key, Label string }{
	{"nama", "Nama"},
	{"nip", "NIP"},
	{"nik", "NIK"},
	{"npwp", "NPWP"},
	{"kategori", "Kategori (PNS/PPPK/GTT/PTT/Luar)"},
	{"jenis_penerima", "Jenis Penerima (tetap/tidak_tetap/bukan_pegawai/peserta)"},
	{"ptkp", "Kategori PTKP (A/B/C)"},
	{"golongan", "Golongan (I/II/III/IV/IX)"},
	{"jabatan", "Jabatan"},
	{"no_hp", "No HP"},
	{"alamat", "Alamat"},
	{"bank", "Bank"},
	{"no_rekening", "No Rekening"},
	{"atas_nama", "Atas Nama"},
	{"tarif_default", "Tarif Bawaan"},
}

var penerimaTextCols = map[string]bool{"nip": true, "nik": true, "npwp": true, "no_hp": true, "no_rekening": true}

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// penerimaSheet menyusun buku kerja Excel berisi satu baris per penerima.
func penerimaSheet(rows []Row) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Penerima"
	_ = f.SetSheetName("Sheet1", sheet)
	for i, col := range penerimaCols {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, col.Label)
	}
	last, _ := excelize.CoordinatesToCellName(len(penerimaCols), 1)
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8EDFC"}},
	})
	if err == nil {
		_ = f.SetCellStyle(sheet, "A1", last, style)
	}
	for r, row := range rows {
		for i, col := range penerimaCols {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
			v := row[col.Key]
			if penerimaTextCols[col.Key] {
				_ = f.SetCellStr(sheet, cell, toStr(v))
			} else {
				_ = f.SetCellValue(sheet, cell, v)
			}
		}
	}
	for i := range penerimaCols {
		letter, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, letter, letter, 18)
	}
	_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// penerimaImport membaca isi berkas (xlsx/xls/csv) menjadi baris data penerima.
func penerimaImport(nama string, data []byte) ([][]string, error) {
	if strings.EqualFold(filepath.Ext(nama), ".csv") {
		rd := csv.NewReader(bytes.NewReader(data))
		rd.FieldsPerRecord = -1
		rows, err := rd.ReadAll()
		if err != nil {
			return nil, err
		}
		return rows, nil
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("berkas kosong")
	}
	return f.GetRows(sheets[0])
}

// penerimaImporBaris menyimpan hasil impor: tambah baru / perbarui yang cocok NIP, NIK atau nama.
func penerimaImporBaris(rows [][]string) (added, updated int, err error) {
	if len(rows) < 2 {
		return 0, 0, fmt.Errorf("tidak ada baris data")
	}
	header := make([]string, len(rows[0]))
	for i, h := range rows[0] {
		header[i] = strings.ToLower(strings.TrimSpace(h))
	}
	find := func(label string) int {
		l := strings.ToLower(label)
		for i, h := range header {
			if h == l {
				return i
			}
		}
		short := l
		if j := strings.Index(l, " ("); j > 0 {
			short = l[:j]
		}
		for i, h := range header {
			if h == short {
				return i
			}
		}
		return -1
	}
	idx := map[string]int{}
	for _, col := range penerimaCols {
		if i := find(col.Label); i >= 0 {
			idx[col.Key] = i
		}
	}
	if _, ok := idx["nama"]; !ok {
		return 0, 0, fmt.Errorf("kolom \"Nama\" tidak ditemukan di baris pertama")
	}
	kategoriValid := []string{"PNS", "PPPK", "GTT", "PTT", "Luar"}
	Q.Tx(func() {
		for _, line := range rows[1:] {
			row := Row{}
			for key, i := range idx {
				if i < len(line) {
					row[key] = strings.TrimSpace(line[i])
				} else {
					row[key] = ""
				}
			}
			if row.S("nama") == "" {
				continue
			}
			for _, d := range []string{"nip", "nik", "no_rekening"} {
				if _, ok := row[d]; ok {
					row[d] = Digits(row[d])
				}
			}
			if _, ok := row["kategori"]; ok {
				k := strings.ToUpper(row.S("kategori"))
				if k == "LUAR" {
					k = "Luar"
				}
				if !contains(kategoriValid, k) {
					k = "GTT"
				}
				row["kategori"] = k
			}
			if _, ok := row["golongan"]; ok {
				g := strings.ToUpper(row.S("golongan"))
				if i := strings.IndexAny(g, "/."); i >= 0 {
					g = g[:i]
				}
				row["golongan"] = strings.TrimSpace(g)
			}
			// Jenis penerima dan PTKP menentukan rezim PPh 21; kosongkan agar
			// diturunkan dari kategori, dan tolak nilai yang tidak dikenal.
			if v := strings.ToLower(row.S("jenis_penerima")); v != "" {
				valid := false
				for _, o := range JenisPenerima {
					if o.K == v {
						valid = true
						break
					}
				}
				if valid {
					row["jenis_penerima"] = v
				} else {
					delete(row, "jenis_penerima")
				}
			} else {
				delete(row, "jenis_penerima")
			}
			if v := strings.ToUpper(row.S("ptkp")); v != "" {
				if v == "A" || v == "B" || v == "C" {
					row["ptkp"] = v
				} else {
					delete(row, "ptkp")
				}
			} else {
				delete(row, "ptkp")
			}
			if _, ok := row["tarif_default"]; ok {
				row["tarif_default"] = Uang(row.S("tarif_default"))
			}
			var existing int64
			if row.S("nip") != "" {
				existing = Q.Int("SELECT id FROM penerima WHERE nip = ?", row.S("nip"))
			}
			if existing == 0 && row.S("nik") != "" {
				existing = Q.Int("SELECT id FROM penerima WHERE nik = ?", row.S("nik"))
			}
			if existing == 0 {
				existing = Q.Int("SELECT id FROM penerima WHERE lower(nama) = lower(?)", row.S("nama"))
			}
			if existing != 0 {
				Q.Update("penerima", row, existing)
				updated++
			} else {
				Q.Insert("penerima", row)
				added++
			}
		}
	})
	return added, updated, nil
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// RegisterPenerimaExcel mendaftarkan rute template, ekspor dan impor penerima.
func RegisterPenerimaExcel(rt *Router) {
	rt.Get("/penerima/template", func(c *Ctx) {
		body, err := penerimaSheet([]Row{{
			"nama": "Siti Aminah, S.Pd.", "nik": "3501010101900001", "kategori": "GTT",
			"jabatan": "Guru Kelas 4", "bank": "BRI", "no_rekening": "012301000123456",
			"atas_nama": "Siti Aminah", "tarif_default": 500000,
		}})
		if err != nil {
			c.Abort(500, "Gagal menyiapkan template.")
		}
		c.Download(xlsxMime, "template-penerima.xlsx", body, false)
	})
	rt.Get("/penerima/ekspor", func(c *Ctx) {
		body, err := penerimaSheet(Q.All("SELECT * FROM penerima ORDER BY nama"))
		if err != nil {
			c.Abort(500, "Gagal menyiapkan berkas Excel.")
		}
		c.Download(xlsxMime, "penerima-"+time.Now().Format("20060102")+".xlsx", body, false)
	})
	rt.Post("/penerima/impor", func(c *Ctx) {
		fh, ok := c.File("berkas")
		if !ok || fh == nil {
			c.Flash("danger", "Pilih berkas Excel (.xlsx) lebih dulu.")
			c.Redirect("/penerima")
		}
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if ext != ".xlsx" && ext != ".xls" && ext != ".csv" {
			c.Flash("danger", "Format berkas harus .xlsx, .xls atau .csv.")
			c.Redirect("/penerima")
		}
		f, err := fh.Open()
		if err != nil {
			c.Flash("danger", "Berkas tidak bisa dibaca. Gunakan template dari tombol Unduh template.")
			c.Redirect("/penerima")
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			c.Flash("danger", "Berkas tidak bisa dibaca. Gunakan template dari tombol Unduh template.")
			c.Redirect("/penerima")
		}
		rows, err := penerimaImport(fh.Filename, data)
		if err != nil {
			c.Flash("danger", "Berkas tidak bisa dibaca. Gunakan template dari tombol Unduh template.")
			c.Redirect("/penerima")
		}
		added, updated, err := penerimaImporBaris(rows)
		if err != nil {
			c.Flash("danger", "Impor gagal: "+err.Error()+".")
			c.Redirect("/penerima")
		}
		c.Flash("success", fmt.Sprintf("Impor selesai: %d penerima baru, %d diperbarui.", added, updated))
		c.Redirect("/penerima")
	})
}

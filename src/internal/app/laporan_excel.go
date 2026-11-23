package app

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// lapSheetName: nama sheet aman (maks 31 karakter, unik).
func lapSheetName(nama string, dipakai map[string]bool) string {
	if len(nama) > 31 {
		nama = nama[:31]
	}
	nama = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\\`, r) {
			return '-'
		}
		return r
	}, nama)
	base := nama
	for i := 2; dipakai[nama]; i++ {
		suffix := fmt.Sprintf(" %d", i)
		if len(base)+len(suffix) > 31 {
			nama = base[:31-len(suffix)] + suffix
		} else {
			nama = base + suffix
		}
	}
	dipakai[nama] = true
	return nama
}

// lapXlsx menyusun buku kerja Excel berisi satu sheet per laporan.
func lapXlsx(parts []*LapData, p Row, belanja []Row) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	s := Sekolah()
	dipakai := map[string]bool{}
	headStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8EDFC"}},
	})
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	moneyStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 4}) // #,##0
	pctStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 2})   // 0.00

	for _, d := range parts {
		sheet := lapSheetName(d.Title, dipakai)
		if _, err := f.NewSheet(sheet); err != nil {
			return nil, err
		}
		last, _ := excelize.ColumnNumberToName(len(d.Cols))
		_ = f.SetCellValue(sheet, "A1", d.Title)
		_ = f.SetCellValue(sheet, "A2", s.S("nama"))
		_ = f.SetCellValue(sheet, "A3", "Periode: "+p.S("label"))
		for _, r := range []int{1, 2, 3} {
			_ = f.MergeCell(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("%s%d", last, r))
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("%s%d", last, r), boldStyle)
		}
		headRow := 5
		for i, c := range d.Cols {
			cell, _ := excelize.CoordinatesToCellName(i+1, headRow)
			_ = f.SetCellValue(sheet, cell, c.Label)
		}
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", headRow), fmt.Sprintf("%s%d", last, headRow), headStyle)
		r := headRow + 1
		writeRow := func(row Row, bold bool) {
			for i, c := range d.Cols {
				v := row[c.Key]
				if v == nil || toStr(v) == "" {
					continue
				}
				cell, _ := excelize.CoordinatesToCellName(i+1, r)
				switch c.Type {
				case "money":
					_ = f.SetCellValue(sheet, cell, toFloat(v))
					_ = f.SetCellStyle(sheet, cell, cell, moneyStyle)
				case "pct":
					_ = f.SetCellValue(sheet, cell, toFloat(v))
					_ = f.SetCellStyle(sheet, cell, cell, pctStyle)
				case "date":
					_ = f.SetCellStr(sheet, cell, TanggalIndoShort(toStr(v)))
				default:
					_ = f.SetCellStr(sheet, cell, toStr(v))
				}
			}
			if bold {
				_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("%s%d", last, r), boldStyle)
			}
			r++
		}
		for _, row := range d.Rows {
			cls := row.S("_class")
			writeRow(row, cls == "group" || cls == "sub")
		}
		if d.Foot != nil {
			writeRow(d.Foot, true)
		}
		r++
		for _, sum := range d.Summary {
			cellL, _ := excelize.CoordinatesToCellName(2, r)
			cellV, _ := excelize.CoordinatesToCellName(4, r)
			_ = f.SetCellValue(sheet, cellL, toStr(sum[0]))
			if fv, ok := sum[1].(float64); ok {
				_ = f.SetCellValue(sheet, cellV, fv)
				_ = f.SetCellStyle(sheet, cellV, cellV, moneyStyle)
			} else {
				_ = f.SetCellStr(sheet, cellV, toStr(sum[1]))
			}
			r++
		}
		for i := range d.Cols {
			letter, _ := excelize.ColumnNumberToName(i + 1)
			width := 16.0
			if d.Cols[i].Type == "money" || d.Cols[i].Type == "pct" {
				width = 15
			}
			if d.Cols[i].Key == "uraian" || d.Cols[i].Key == "nama" {
				width = 52
			}
			if d.Cols[i].Key == "tgl" {
				width = 14
			}
			_ = f.SetColWidth(sheet, letter, letter, width)
		}
		_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: headRow, TopLeftCell: fmt.Sprintf("A%d", headRow+1), ActivePane: "bottomLeft"})
	}

	if belanja != nil {
		sheet := lapSheetName("Rincian belanja", dipakai)
		if _, err := f.NewSheet(sheet); err != nil {
			return nil, err
		}
		head := []string{"Tanggal", "No. bukti", "Kode rekening", "Jenis", "Uraian", "Rincian / penerima", "Volume", "Satuan", "Harga / bruto", "Pajak", "Jumlah"}
		_ = f.SetCellValue(sheet, "A1", "RINCIAN BUKTI BELANJA")
		_ = f.SetCellValue(sheet, "A2", s.S("nama"))
		_ = f.SetCellValue(sheet, "A3", "Periode: "+p.S("label"))
		_ = f.MergeCell(sheet, "A1", "K1")
		_ = f.MergeCell(sheet, "A2", "K2")
		_ = f.MergeCell(sheet, "A3", "K3")
		for i, h := range head {
			cell, _ := excelize.CoordinatesToCellName(i+1, 5)
			_ = f.SetCellValue(sheet, cell, h)
		}
		_ = f.SetCellStyle(sheet, "A5", "K5", headStyle)
		r := 6
		for _, b := range belanja {
			pajak := 0.0
			for _, pj := range toRows(b["pajak"]) {
				pajak += pj.F("nilai")
			}
			rowVals := []any{TanggalIndoShort(b.S("tgl")), b.S("no_bukti"), b.S("rekening_kode"), b.S("jenis_label"), b.S("uraian"), b.S("toko_nama"), nil, nil, nil, pajak, b.F("total")}
			lapWriteExcelRow(f, sheet, r, rowVals, moneyStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("K%d", r), boldStyle)
			r++
			for _, it := range toRows(b["items"]) {
				lapWriteExcelRow(f, sheet, r, []any{nil, nil, nil, nil, nil, it.S("nama"), it.F("volume"), it.S("satuan"), it.F("harga"), nil, it.F("subtotal")}, moneyStyle)
				r++
			}
			for _, pn := range toRows(b["penerima"]) {
				label := pn.S("nama")
				if pn.S("nip") != "" {
					label += " / " + pn.S("nip")
				}
				lapWriteExcelRow(f, sheet, r, []any{nil, nil, nil, nil, nil, label, pn.F("volume"), pn.S("satuan"), pn.F("bruto"), pn.F("pph21"), pn.F("netto")}, moneyStyle)
				r++
			}
		}
		_ = f.SetColWidth(sheet, "A", "K", 15)
		_ = f.SetColWidth(sheet, "E", "F", 48)
		_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 5, TopLeftCell: "A6", ActivePane: "bottomLeft"})
	}

	_ = f.DeleteSheet("Sheet1")
	idx, _ := f.GetSheetIndex(f.GetSheetList()[0])
	f.SetActiveSheet(idx)
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func lapWriteExcelRow(f *excelize.File, sheet string, r int, vals []any, moneyStyle int) {
	for i, v := range vals {
		if v == nil {
			continue
		}
		cell, _ := excelize.CoordinatesToCellName(i+1, r)
		switch x := v.(type) {
		case float64:
			_ = f.SetCellValue(sheet, cell, x)
			_ = f.SetCellStyle(sheet, cell, cell, moneyStyle)
		case string:
			if x != "" {
				_ = f.SetCellStr(sheet, cell, x)
			}
		default:
			_ = f.SetCellValue(sheet, cell, x)
		}
	}
}

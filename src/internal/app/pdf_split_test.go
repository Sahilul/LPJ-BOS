package app

import (
	"strings"
	"testing"
)

func TestSplitTextUnicode(t *testing.T) {
	k := NewPdf(false, Row{})
	k.Pdf.SetFont("Times", "", 9)
	lines := k.splitText("Pencairan — dana BOS “tahap” II untuk kebutuhan belanja madrasah", 30)
	if len(lines) < 2 {
		t.Fatalf("teks harus terbungkus, dapat %d baris", len(lines))
	}
	if !strings.Contains(strings.Join(lines, " "), k.T("—")) {
		t.Fatalf("em dash hilang: %q", lines)
	}
}

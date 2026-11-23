package app

import (
	"net/url"
	"testing"
)

func TestTerbilang(t *testing.T) {
	cases := map[float64]string{
		0:          "nol",
		11:         "sebelas",
		15:         "lima belas",
		115:        "seratus lima belas",
		1000:       "seribu",
		1250000:    "satu juta dua ratus lima puluh ribu",
		2000001:    "dua juta satu",
		1000000000: "satu miliar",
	}
	for n, want := range cases {
		if got := Terbilang(n); got != want {
			t.Errorf("Terbilang(%v) = %q, want %q", n, got, want)
		}
	}
}

func TestNumberFormat(t *testing.T) {
	if got := Rupiah(1250000.5); got != "Rp1.250.001" {
		t.Errorf("Rupiah = %q", got)
	}
	if got := Angka(-1234567); got != "-1.234.567" {
		t.Errorf("Angka = %q", got)
	}
	if got := NumberFormat(-0.001, 2); got != "0,00" {
		t.Errorf("NumberFormat negatif nol = %q", got)
	}
	if got := NumberFormat(1234.5, 2); got != "1.234,50" {
		t.Errorf("NumberFormat desimal = %q", got)
	}
}

func TestParseFormNested(t *testing.T) {
	f := ParseForm(url.Values{
		"nama":            {"A"},
		"rincian[0][qty]": {"2"},
		"rincian[1][qty]": {"3"},
		"tag[]":           {"x", "y"},
		"meta[kode][sub]": {"z"},
	})
	if f.Str("nama") != "A" {
		t.Errorf("nama = %q", f.Str("nama"))
	}
	if l := f.List("rincian"); len(l) != 2 || l[1].Str("qty") != "3" {
		t.Errorf("rincian = %#v", l)
	}
	if s := f.Strings("tag"); len(s) != 2 || s[0] != "x" || s[1] != "y" {
		t.Errorf("tag = %#v", s)
	}
	if f.Sub("meta").Sub("kode").Str("sub") != "z" {
		t.Errorf("meta = %#v", f.Sub("meta"))
	}
}

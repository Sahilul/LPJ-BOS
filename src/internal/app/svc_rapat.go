package app

import (
	"fmt"
	"strings"
)

// Undangan & notulen rapat yang terkait satu belanja BKU. Satu belanja hanya
// boleh punya satu rapat (kolom belanja_id UNIQUE).

var RapatSifat = []Opt{
	{"Biasa", "Biasa"},
	{"Penting", "Penting"},
	{"Segera", "Segera"},
	{"Rahasia", "Rahasia"},
}

func RapatFind(id int64) Row {
	return Q.One("SELECT * FROM rapat WHERE id = ?", id)
}

func RapatFindBelanja(belanjaID int64) Row {
	return Q.One("SELECT * FROM rapat WHERE belanja_id = ?", belanjaID)
}

// RapatKomponenList: komponen yang punya belanja pada tahun anggaran.
func RapatKomponenList(tahun int) []Opt {
	out := []Opt{}
	rows := Q.All(`SELECT DISTINCT k.kode, k.nama FROM komponen k
        JOIN rencana_belanja r ON r.komponen_kode = k.kode
        JOIN belanja b ON b.rencana_id = r.id
        WHERE r.tahun = ? ORDER BY k.urutan, k.nama`, tahun)
	for _, r := range rows {
		out = append(out, Opt{r.S("kode"), r.S("nama")})
	}
	return out
}

// RapatBelanjaKandidat: belanja pada tahun anggaran (dan komponen tertentu)
// yang belum punya undangan/notulen. kecuali = id rapat yang sedang diubah.
func RapatBelanjaKandidat(komponenKode string, tahun int, kecuali int64) []Row {
	sql := `SELECT b.id, b.tgl, b.no_bukti, b.uraian, b.total,
            r.komponen_kode, k.nama AS komponen_nama
        FROM belanja b
        JOIN rencana_belanja r ON r.id = b.rencana_id
        LEFT JOIN komponen k ON k.kode = r.komponen_kode
        WHERE r.tahun = ?
          AND b.id NOT IN (SELECT belanja_id FROM rapat WHERE id != ?)`
	args := []any{tahun, kecuali}
	if komponenKode != "" {
		sql += " AND r.komponen_kode = ?"
		args = append(args, komponenKode)
	}
	sql += " ORDER BY b.tgl, b.id"
	return Q.All(sql, args...)
}

var romawiAngka = []struct {
	n int
	r string
}{
	{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"}, {100, "C"}, {90, "XC"},
	{50, "L"}, {40, "XL"}, {10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
}

func angkaRomawi(n int) string {
	if n <= 0 || n > 3999 {
		return ""
	}
	var b strings.Builder
	for _, p := range romawiAngka {
		for n >= p.n {
			b.WriteString(p.r)
			n -= p.n
		}
	}
	return b.String()
}

// RapatNomorBerikut: nomor undangan otomatis {urut}/UND/{NPSN}/{romawi bulan}/{tahun}.
func RapatNomorBerikut(tahun int, tglSurat string) string {
	urut := Q.Int("SELECT COUNT(*) FROM rapat WHERE tahun = ?", tahun) + 1
	bulan := 1
	if t, ok := ParseTime(tglSurat); ok {
		bulan = int(t.Month())
	}
	bagian := []string{fmt.Sprintf("%03d", urut), "UND"}
	// NPSN dilewati bila sekolah belum mengisinya.
	if npsn := strings.TrimSpace(Sekolah().S("npsn")); npsn != "" {
		bagian = append(bagian, npsn)
	}
	bagian = append(bagian, angkaRomawi(bulan), itoa(tahun))
	return strings.Join(bagian, "/")
}

// rapatTahunBelanja: tahun anggaran dari baris belanja/rencana.
func rapatTahunBelanja(b Row) int {
	if b == nil {
		return 0
	}
	if t := int(b.I("tahun")); t != 0 {
		return t
	}
	return int(toInt(keuanganTahunDari(b.S("tgl"))))
}

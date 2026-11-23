package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Rupiah(v any) string { return RupiahP(v, true) }

func RupiahP(v any, prefix bool) string {
	n := toFloat(v)
	s := NumberFormat(math.Abs(n), 0)
	if prefix {
		s = "Rp" + s
	}
	if n < 0 {
		return "-" + s
	}
	return s
}

func Angka(v any) string { return NumberFormat(toFloat(v), 0) }

func AngkaD(v any, dec int) string { return NumberFormat(toFloat(v), dec) }

// NumberFormat meniru number_format($n, $dec, ',', '.') milik PHP.
func NumberFormat(n float64, dec int) string {
	p := math.Pow(10, float64(dec))
	r := math.Round(n*p) / p
	neg := r < 0
	s := strconv.FormatFloat(math.Abs(r), 'f', dec, 64)
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	out := b.String()
	if dec > 0 {
		out += "," + frac
	}
	if neg && strings.Trim(out, "0.,") != "" {
		out = "-" + out
	}
	return out
}

func Persen(v any) string {
	s := NumberFormat(toFloat(v), 2)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ",")
	return s + "%"
}

var satuanKata = []string{"", "satu", "dua", "tiga", "empat", "lima", "enam", "tujuh", "delapan", "sembilan", "sepuluh", "sebelas"}

func terbilangRec(x int64) string {
	sfx := func(word string, div, rest int64) string {
		if rest != 0 {
			return terbilangRec(x/div) + " " + word + " " + terbilangRec(rest)
		}
		return terbilangRec(x/div) + " " + word
	}
	switch {
	case x < 12:
		return satuanKata[x]
	case x < 20:
		return terbilangRec(x-10) + " belas"
	case x < 100:
		return sfx("puluh", 10, x%10)
	case x < 200:
		if x-100 != 0 {
			return "seratus " + terbilangRec(x-100)
		}
		return "seratus"
	case x < 1000:
		return sfx("ratus", 100, x%100)
	case x < 2000:
		if x-1000 != 0 {
			return "seribu " + terbilangRec(x-1000)
		}
		return "seribu"
	case x < 1_000_000:
		return sfx("ribu", 1000, x%1000)
	case x < 1_000_000_000:
		return sfx("juta", 1_000_000, x%1_000_000)
	case x < 1_000_000_000_000:
		return sfx("miliar", 1_000_000_000, x%1_000_000_000)
	default:
		return sfx("triliun", 1_000_000_000_000, x%1_000_000_000_000)
	}
}

var multiSpace = regexp.MustCompile(`\s+`)

func Terbilang(v any) string {
	n := int64(math.Floor(math.Abs(toFloat(v))))
	if n == 0 {
		return "nol"
	}
	return strings.TrimSpace(multiSpace.ReplaceAllString(terbilangRec(n), " "))
}

var namaBulan = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func BulanIndo(m any) string {
	i := toInt(m)
	if i < 1 || i > 12 {
		return ""
	}
	return namaBulan[i]
}

var dateLayouts = []string{"2006-01-02", "2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006-01-02 15:04"}

// ParseTime mengurai tanggal seperti strtotime() untuk format yang dipakai aplikasi.
func ParseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, l := range dateLayouts {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func TanggalIndo(v any) string { return tanggalIndo(toStr(v), false) }

func TanggalIndoShort(v any) string { return tanggalIndo(toStr(v), true) }

func tanggalIndo(s string, short bool) string {
	if s == "" {
		return ""
	}
	t, ok := ParseTime(s)
	if !ok {
		return s
	}
	b := namaBulan[t.Month()]
	if short {
		b = b[:3]
	}
	return fmt.Sprintf("%d %s %d", t.Day(), b, t.Year())
}

func Today() string { return time.Now().Format("2006-01-02") }

func ThisYear() int { return time.Now().Year() }

func UploadPath(name string) string { return Cfg.UploadDir + "/" + filepath.Base(name) }

func UploadURL(v any) string {
	name := toStr(v)
	if name == "" {
		return ""
	}
	var mt int64
	if st, err := os.Stat(UploadPath(name)); err == nil {
		mt = st.ModTime().Unix()
	}
	return "/file/" + rawURLEncode(name) + "?v=" + strconv.FormatInt(mt, 10)
}

func rawURLEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func Asset(path string) string {
	return "/assets/" + strings.TrimLeft(path, "/") + "?v=" + strconv.FormatInt(startedAt, 10)
}

var nonDigit = regexp.MustCompile(`\D+`)

func Digits(v any) string { return nonDigit.ReplaceAllString(toStr(v), "") }

// Uang mengubah isian rupiah "1.200.000" menjadi angka.
func Uang(v any) float64 { return toFloat(Digits(v)) }

// Desimal mengubah isian "1,5" atau "1.000,5" menjadi angka.
func Desimal(v any) float64 {
	s := strings.NewReplacer(" ", "", ".", "").Replace(toStr(v))
	return toFloat(strings.ReplaceAll(s, ",", "."))
}

func ValidDate(s string) bool {
	t, err := time.Parse("2006-01-02", s)
	return err == nil && t.Format("2006-01-02") == s
}

func Sekolah() Row {
	r := Q.One("SELECT * FROM sekolah WHERE id = 1")
	if r == nil {
		return Row{}
	}
	return r
}

// TahunList: tahun anggaran terdaftar, terbaru dulu.
func TahunList() []int {
	out := []int{}
	for _, r := range Q.All("SELECT tahun FROM tahun_anggaran ORDER BY tahun DESC") {
		out = append(out, int(r.I("tahun")))
	}
	return out
}

func InTahunList(t int) bool {
	for _, x := range TahunList() {
		if x == t {
			return true
		}
	}
	return false
}

func TahunDefault() int {
	t := Q.Int("SELECT tahun FROM tahun_anggaran ORDER BY is_default DESC, tahun DESC LIMIT 1")
	if t == 0 {
		return ThisYear()
	}
	return int(t)
}

func PejabatAktif(peran string, tanggal string) Row {
	if tanggal == "" {
		tanggal = Today()
	}
	return Q.One(`SELECT * FROM pejabat WHERE peran = ? AND mulai <= ? AND (selesai IS NULL OR selesai = '' OR selesai >= ?)
         ORDER BY mulai DESC LIMIT 1`, peran, tanggal, tanggal)
}

// SisaRencana: pagu satu baris rencana dikurangi total belanja, tanpa belanja kecuali.
func SisaRencana(rencanaID, kecuali int64) float64 {
	r := Q.One("SELECT pagu FROM rencana_belanja WHERE id = ?", rencanaID)
	if r == nil {
		return 0
	}
	pakai := Q.Float("SELECT COALESCE(SUM(total), 0) FROM belanja WHERE rencana_id = ? AND id != ?", rencanaID, kecuali)
	return r.F("pagu") - pakai
}

func TarifPajak(kode string) float64 {
	return Q.Float("SELECT tarif FROM tarif_pajak WHERE kode = ?", kode)
}

// TarifPPh21 (persen) untuk penerima honor berdasarkan kategori, golongan, NPWP.
func TarifPPh21(p Row) float64 {
	if p.S("kategori") == "PNS" {
		gol := strings.ToUpper(strings.TrimSpace(p.S("golongan")))
		g := ""
		for _, cand := range []string{"IV", "III", "II", "I"} {
			if strings.HasPrefix(gol, cand) {
				next := gol[len(cand):]
				if next == "" || next[0] < 'A' || next[0] > 'Z' {
					g = cand
				}
				break
			}
		}
		if g == "" {
			return 0
		}
		return TarifPajak("PPH21_GOL_" + g)
	}
	if Digits(p.S("npwp")) != "" {
		return TarifPajak("PPH21_NPWP")
	}
	return TarifPajak("PPH21_NON_NPWP")
}

type Saldo struct {
	Tunai float64 `json:"tunai"`
	Bank  float64 `json:"bank"`
	Total float64 `json:"total"`
}

func (s Saldo) Get(metode string) float64 {
	if metode == "tunai" {
		return s.Tunai
	}
	return s.Bank
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// SaldoKas: saldo tunai dan bank sampai tgl (inklusif) sejak 1 Januari tahun yang sama.
func SaldoKas(tgl string, kecualiMutasi, kecualiBelanja int64) Saldo {
	awal := tgl[:4] + "-01-01"
	masuk := Q.Float("SELECT COALESCE(SUM(jumlah), 0) FROM penerimaan WHERE tgl BETWEEN ? AND ?", awal, tgl)
	mut := map[string]float64{}
	for _, r := range Q.All("SELECT jenis, COALESCE(SUM(jumlah), 0) AS j FROM mutasi_kas WHERE tgl BETWEEN ? AND ? AND id != ? GROUP BY jenis", awal, tgl, kecualiMutasi) {
		mut[r.S("jenis")] = r.F("j")
	}
	bel := map[string]float64{}
	for _, r := range Q.All("SELECT metode, COALESCE(SUM(total), 0) AS j FROM belanja WHERE tgl BETWEEN ? AND ? AND id != ? GROUP BY metode", awal, tgl, kecualiBelanja) {
		bel[r.S("metode")] = r.F("j")
	}
	tunai := mut["tarik"] - mut["setor"] - bel["tunai"]
	bank := masuk + mut["bunga"] - mut["admin_bank"] - mut["tarik"] + mut["setor"] - bel["bank"]
	return Saldo{Tunai: round2(tunai), Bank: round2(bank), Total: round2(tunai + bank)}
}

// TglLunas: tanggal belanja dianggap lunas (saldo metode tidak minus), "" bila sampai akhir tahun masih minus.
func TglLunas(tgl, metode string) string {
	if metode != "tunai" {
		metode = "bank"
	}
	if SaldoKas(tgl, 0, 0).Get(metode) > -0.005 {
		return tgl
	}
	akhir := tgl[:4] + "-12-31"
	rows := Q.All(`SELECT tgl FROM penerimaan WHERE tgl > ? AND tgl <= ?
         UNION SELECT tgl FROM mutasi_kas WHERE tgl > ? AND tgl <= ?
         ORDER BY tgl`, tgl, akhir, tgl, akhir)
	for _, r := range rows {
		d := r.S("tgl")
		if SaldoKas(d, 0, 0).Get(metode) > -0.005 {
			return d
		}
	}
	return ""
}

var NomorBuktiPrefix = map[string]string{"trm": "BKM", "bel": "BKK", "mut": "MUT"}
var NomorBuktiTabel = map[string]string{"trm": "penerimaan", "bel": "belanja", "mut": "mutasi_kas"}

// NoBuktiBerikut: nomor bukti berikutnya untuk jenis trm/bel/mut.
func NoBuktiBerikut(tgl, jenis string) string {
	tahun := tgl[:4]
	count := Q.Int("SELECT COUNT(*) FROM "+NomorBuktiTabel[jenis]+" WHERE strftime('%Y', tgl) = ?", tahun)
	return fmt.Sprintf("%s-%03d/%s/%s", NomorBuktiPrefix[jenis], count+1, tgl[5:7], tahun)
}

// SusunNomorBuktiBKU menyusun ulang nomor bukti satu tahun per jenis menurut tanggal.
func SusunNomorBuktiBKU(tahun int) {
	for _, jenis := range []string{"trm", "bel", "mut"} {
		tabel := NomorBuktiTabel[jenis]
		rows := Q.All("SELECT id, tgl FROM "+tabel+" WHERE strftime('%Y', tgl) = ? ORDER BY tgl, id", strconv.Itoa(tahun))
		for i, r := range rows {
			nomor := fmt.Sprintf("%s-%03d/%s/%d", NomorBuktiPrefix[jenis], i+1, r.S("tgl")[5:7], tahun)
			Q.Run("UPDATE "+tabel+" SET no_bukti = ? WHERE id = ?", nomor, r.I("id"))
		}
	}
}

// ValidationError: kesalahan isian per field, dipakai form web dan tool MCP.
type ValidationError struct {
	Errors map[string]string
	Msg    string
	// Order (opsional) menyimpan urutan field seperti array PHP; dipakai pesan MCP.
	Order []string
}

func (e *ValidationError) Error() string {
	if e.Msg == "" {
		return "Isian belum valid."
	}
	return e.Msg
}

func NewValidation(errs map[string]string) *ValidationError {
	return &ValidationError{Errors: errs, Msg: "Isian belum valid."}
}

// NewValidationOrdered membuat ValidationError dari pasangan field, pesan berurutan.
func NewValidationOrdered(pairs ...string) *ValidationError {
	e := NewValidation(map[string]string{})
	for i := 0; i+1 < len(pairs); i += 2 {
		if _, ok := e.Errors[pairs[i]]; !ok {
			e.Order = append(e.Order, pairs[i])
		}
		e.Errors[pairs[i]] = pairs[i+1]
	}
	return e
}

// Set menambah pesan field dengan menjaga urutan.
func (e *ValidationError) Set(field, pesan string) {
	if _, ok := e.Errors[field]; !ok {
		e.Order = append(e.Order, field)
	}
	e.Errors[field] = pesan
}

func validationMessages(e *ValidationError) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, k := range e.Order {
		if v, ok := e.Errors[k]; ok && !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	keys := make([]string, 0, len(e.Errors))
	for k := range e.Errors {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, e.Errors[k])
	}
	return out
}

// Fmt helpers yang umum dipakai: konversi nilai Row.
func S(v any) string            { return toStr(v) }
func I(v any) int64             { return toInt(v) }
func F(v any) float64           { return toFloat(v) }
func FloatStr(f float64) string { return phpFloatStr(f) }

// jsonMarshal tanpa escape HTML, seperti json_encode JSON_UNESCAPED_SLASHES|UNICODE.
func jsonMarshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func jsonMarshalIndent(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

package app

import "strings"

// MasterField mendeskripsikan satu isian data master.
type MasterField struct {
	Name     string
	Label    string
	Type     string // text, textarea, select, money, percent, bool, int, digits
	Options  []Opt
	Required bool
	Unique   bool
	Mono     bool
	Span2    bool
	Len      int
	Hint     string
	Default  any
}

// MasterRes mendeskripsikan satu resource data master (penerima, toko, dst).
type MasterRes struct {
	Slug       string
	Table      string
	Title      string
	Singular   string
	Lead       string
	Icon       string
	Order      string
	Search     []string
	Toggle     bool
	Excel      bool
	FilterCol  string
	FilterOpts []Opt
	List       []string
	Fields     []MasterField
}

// Field mencari definisi isian menurut nama.
func (r *MasterRes) Field(name string) *MasterField {
	for i := range r.Fields {
		if r.Fields[i].Name == name {
			return &r.Fields[i]
		}
	}
	return nil
}

// LabelOf mencari label isian, jatuh ke nama kolom bila tidak dikenal.
func (r *MasterRes) LabelOf(name string) string {
	if f := r.Field(name); f != nil {
		return f.Label
	}
	return name
}

// SearchLabels: label kolom pencarian untuk placeholder.
func (r *MasterRes) SearchLabels() []string {
	out := make([]string, 0, len(r.Search))
	for _, c := range r.Search {
		out = append(out, strings.ToLower(r.LabelOf(c)))
	}
	return out
}

var (
	masterKategori = []Opt{{"PNS", "PNS"}, {"PPPK", "PPPK"}, {"GTT", "GTT"}, {"PTT", "PTT"}, {"Luar", "Pihak luar"}}
	masterGolongan = []Opt{{"", "—"}, {"I", "I"}, {"II", "II"}, {"III", "III"}, {"IV", "IV"}, {"IX", "IX (PPPK)"}}
	masterJenisRek = []Opt{{"barang", "Barang"}, {"jasa", "Jasa"}, {"honor", "Honor"}, {"modal", "Modal"}, {"lainnya", "Lainnya"}}
)

var masterResAll = []*MasterRes{
	{
		Slug: "penerima", Table: "penerima", Title: "Penerima", Singular: "penerima",
		Lead: "Guru, tenaga kependidikan dan pihak luar yang menerima honor atau transport. Data dicatat ulang ke setiap transaksi, jadi mengubahnya tidak mengubah kuitansi lama.",
		Icon: "users", Order: "aktif DESC, nama",
		Search: []string{"nama", "nip", "nik", "jabatan"},
		Toggle: true, Excel: true,
		FilterCol: "kategori", FilterOpts: masterKategori,
		List: []string{"nama", "kategori", "golongan", "jabatan", "no_rekening", "tarif_default"},
		Fields: []MasterField{
			{Name: "nama", Label: "Nama lengkap", Required: true, Hint: "Tulis beserta gelar, seperti di kuitansi.", Span2: true},
			{Name: "kategori", Label: "Kategori", Type: "select", Options: masterKategori, Default: "GTT"},
			{Name: "jenis_penerima", Label: "Jenis penerima (PPh 21)", Type: "select", Options: JenisPenerima, Default: "tidak_tetap", Hint: "Menentukan cara pemotongan PPh 21 menurut PMK 168/2023. PPPK tidak memakai tarif final golongan PNS."},
			{Name: "ptkp", Label: "Kategori PTKP", Type: "select", Options: PtkpKategori, Default: "A", Hint: "Dipakai untuk tarif efektif bulanan pegawai tetap."},
			{Name: "golongan", Label: "Golongan", Type: "select", Options: masterGolongan, Hint: "Menentukan tarif final PPh 21 honorarium untuk PNS."},
			{Name: "nip", Label: "NIP", Type: "digits", Len: 18},
			{Name: "nik", Label: "NIK", Type: "digits", Len: 16},
			{Name: "npwp", Label: "NPWP"},
			{Name: "jabatan", Label: "Jabatan / tugas", Hint: "Contoh: Guru Kelas 4, Operator Dapodik."},
			{Name: "no_hp", Label: "Nomor HP"},
			{Name: "alamat", Label: "Alamat", Type: "textarea", Span2: true},
			{Name: "bank", Label: "Bank"},
			{Name: "no_rekening", Label: "Nomor rekening", Mono: true},
			{Name: "atas_nama", Label: "Atas nama rekening"},
			{Name: "tarif_default", Label: "Tarif bawaan", Type: "money", Hint: "Terisi otomatis saat memilih penerima di transaksi."},
		},
	},
	{
		Slug: "toko", Table: "toko", Title: "Toko & penyedia", Singular: "toko",
		Lead: "Toko dan penyedia jasa tempat sekolah berbelanja. Status PKP menentukan apakah PPN dipungut.",
		Icon: "store", Order: "aktif DESC, nama",
		Search: []string{"nama", "alamat", "npwp"},
		Toggle: true,
		List:   []string{"nama", "alamat", "npwp", "pkp"},
		Fields: []MasterField{
			{Name: "nama", Label: "Nama toko", Required: true, Span2: true},
			{Name: "alamat", Label: "Alamat", Type: "textarea", Span2: true},
			{Name: "npwp", Label: "NPWP"},
			{Name: "telepon", Label: "Telepon"},
			{Name: "pkp", Label: "Pengusaha Kena Pajak (PKP)", Type: "bool", Hint: "Centang bila toko memungut PPN."},
		},
	},
	{
		Slug: "komponen", Table: "komponen", Title: "Komponen BOSP", Singular: "komponen",
		Lead: "Komponen penggunaan dana BOSP Reguler. Kode dipakai di RKAS, BKU dan rekap.",
		Icon: "layers", Order: "urutan, kode",
		Search: []string{"kode", "nama"},
		List:   []string{"kode", "nama"},
		Fields: []MasterField{
			{Name: "kode", Label: "Kode", Required: true, Unique: true, Mono: true},
			{Name: "urutan", Label: "Urutan", Type: "int", Default: 0},
			{Name: "nama", Label: "Nama komponen", Required: true, Span2: true},
		},
	},
	{
		Slug: "rekening", Table: "kode_rekening", Title: "Kode rekening", Singular: "kode rekening",
		Lead: "Kode rekening belanja sesuai ARKAS. Jenis dipakai untuk memilih pajak yang biasa dipungut.",
		Icon: "hash", Order: "kode",
		Search: []string{"kode", "nama"},
		List:   []string{"kode", "nama", "jenis"},
		Fields: []MasterField{
			{Name: "kode", Label: "Kode rekening", Required: true, Unique: true, Mono: true},
			{Name: "jenis", Label: "Jenis", Type: "select", Options: masterJenisRek},
			{Name: "nama", Label: "Nama rekening", Required: true, Span2: true},
		},
	},
	{
		Slug: "pajak", Table: "tarif_pajak", Title: "Tarif pajak", Singular: "tarif pajak",
		Lead: "Tarif yang dipakai saat menghitung pajak transaksi. PPN 11%, PPh 22 1,5%, PPh 23 2%, dan PPh 21 sesuai PMK 168/2023. Sesuaikan bila ada aturan daerah yang berbeda.",
		Icon: "percent", Order: "kode",
		Search: []string{"kode", "nama"},
		List:   []string{"kode", "nama", "tarif", "keterangan"},
		Fields: []MasterField{
			{Name: "kode", Label: "Kode", Required: true, Unique: true, Mono: true},
			{Name: "tarif", Label: "Tarif (%)", Type: "percent", Default: 0},
			{Name: "nama", Label: "Nama pajak", Required: true, Span2: true},
			{Name: "keterangan", Label: "Keterangan", Type: "textarea", Span2: true},
		},
	},
}

// MasterBySlug mencari resource; ok=false bila tidak dikenal.
func MasterBySlug(slug string) (*MasterRes, bool) {
	for _, r := range masterResAll {
		if r.Slug == slug {
			return r, true
		}
	}
	return nil, false
}

// MasterLabel: label tampil sebuah baris (nama, atau kode bila tanpa nama).
func MasterLabel(row Row) string {
	if s := row.S("nama"); s != "" {
		return s
	}
	return row.S("kode")
}

package app

import "testing"

func TestMcpPenerimaSimpan(t *testing.T) {
	setupBelanja(t, false, "")
	tool := mcpCari("penerima_simpan")
	if tool == nil {
		t.Fatal("tool penerima_simpan tidak terdaftar")
	}
	out, err := tool.Handler(McpArgs{"nama": "Siti Aminah, S.Pd.", "nik": "3516010101900001", "ptkp": "B"})
	if err != nil {
		t.Fatal(err)
	}
	id := out.(OM)[1].V.(int64)
	p := Q.One("SELECT * FROM penerima WHERE id = ?", id)
	if p.S("kategori") != "GTT" || p.S("jenis_penerima") != "tidak_tetap" || p.S("ptkp") != "B" {
		t.Fatalf("nilai bawaan salah: %v", p)
	}
	if _, err := tool.Handler(McpArgs{"id": id, "jabatan": "Guru Fikih"}); err != nil {
		t.Fatal(err)
	}
	p = Q.One("SELECT * FROM penerima WHERE id = ?", id)
	if p.S("jabatan") != "Guru Fikih" || p.S("nik") != "3516010101900001" {
		t.Fatalf("ubah sebagian merusak data lama: %v", p)
	}
	if _, err := tool.Handler(McpArgs{"nama": "X", "nik": "123"}); err == nil {
		t.Fatal("NIK salah panjang seharusnya ditolak")
	}
	if _, err := mcpCari("penerima_hapus").Handler(McpArgs{"id": id}); err != nil {
		t.Fatal(err)
	}
}

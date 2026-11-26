package app

import (
	"strings"
	"testing"
)

// TestMcpRapat memastikan tool MCP rapat terdaftar dan bekerja end-to-end.
func TestMcpRapat(t *testing.T) {
	belanjaID := setupRapat(t)

	simpan := mcpCari("rapat_simpan")
	if simpan == nil {
		t.Fatal("tool rapat_simpan tidak terdaftar")
	}
	out, err := simpan.Handler(McpArgs{
		"belanja_id": belanjaID, "tgl": "2026-03-05", "agenda": "Evaluasi BOS",
		"jam_mulai": "09:00", "perihal": "Undangan Rapat",
	})
	if err != nil {
		t.Fatal(err)
	}
	id := out.(OM)[1].V.(int64)
	row := RapatFind(id)
	if row == nil || row.S("no_undangan") == "" {
		t.Fatalf("rapat tidak tersimpan atau nomor kosong: %v", row)
	}
	if row.S("tgl_surat") != "2026-03-05" {
		t.Errorf("tgl_surat = %q, mau 2026-03-05", row.S("tgl_surat"))
	}

	// Ubah sebagian: hanya isi notulen, isian lain harus tetap.
	if _, err := simpan.Handler(McpArgs{"id": id, "pembahasan": "1. Evaluasi\n2. Rencana"}); err != nil {
		t.Fatal(err)
	}
	row = RapatFind(id)
	if !strings.Contains(row.S("pembahasan"), "Evaluasi") || row.S("agenda") != "Evaluasi BOS" || row.S("belanja_id") == "" {
		t.Fatalf("ubah sebagian merusak data lama: %v", row)
	}

	detail := mcpCari("rapat_detail")
	if detail == nil {
		t.Fatal("tool rapat_detail tidak terdaftar")
	}
	if _, err := detail.Handler(McpArgs{"id": id}); err != nil {
		t.Fatal(err)
	}

	list := mcpCari("rapat_list")
	if list == nil {
		t.Fatal("tool rapat_list tidak terdaftar")
	}
	res, err := list.Handler(McpArgs{"tahun": 2026})
	if err != nil {
		t.Fatal(err)
	}
	if n := res.(OM)[1].V.(int); n != 1 {
		t.Fatalf("rapat_list jumlah = %d, mau 1", n)
	}

	hapus := mcpCari("rapat_hapus")
	if hapus == nil {
		t.Fatal("tool rapat_hapus tidak terdaftar")
	}
	if _, err := hapus.Handler(McpArgs{"id": id}); err != nil {
		t.Fatal(err)
	}
	if RapatFind(id) != nil {
		t.Fatal("rapat tidak terhapus lewat MCP")
	}
	// Belanja tetap ada setelah rapat dihapus.
	if BelanjaFind(belanjaID) == nil {
		t.Fatal("belanja ikut terhapus; seharusnya hanya rapat")
	}
}

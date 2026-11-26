# Berkontribusi

Terima kasih sudah tertarik membantu LPJ BOS.

## Melaporkan masalah

Buka issue dengan menyertakan:

- versi (commit/tag) dan cara menjalankan (Docker atau binary),
- langkah reproduksi yang singkat,
- hasil yang diharapkan vs yang terjadi,
- potongan log bila ada (hapus token, nama sekolah, dan data pribadi).

Untuk celah keamanan, jangan buka issue publik — hubungi pemelihara secara pribadi.

## Mengirim perubahan

1. Fork repo, buat branch dari `main` (`feat/…` atau `fix/…`).
2. Ikuti gaya kode yang ada: satu paket `internal/app`, handler `c_*.go`, logika `svc_*.go`, template di `templates/`.
3. Tambahkan test untuk perubahan logika.
4. Pastikan pemeriksaan lulus:

   ```bash
   cd src
   gofmt -l ./internal/app   # harus kosong
   go vet ./...
   go test ./...
   ```

5. Buka pull request dengan penjelasan singkat dan tangkapan layar bila mengubah tampilan.

## Gaya kode

- Komentar dan teks antarmuka memakai Bahasa Indonesia.
- Jangan menyisipkan string query utuh di template (`?{{.q}}`); tulis `?periode={{...}}` agar `=` tidak ter-escape.
- Migrasi database baru: tambahkan file berurutan di `src/internal/app/migrations/`, jangan ubah file lama.

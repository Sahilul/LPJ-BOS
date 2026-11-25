# Instruksi proyek LPJ BOS

- Balas dalam Bahasa Indonesia, ringkas.
- Setiap selesai mengerjakan perbaikan/fitur dan pengujian lulus (`cd src && go test ./...`), langsung commit dan push ke `main` (repo Sahilul/LPJ-BOS) tanpa menunggu diminta.
- Setelah push, ingatkan bahwa produksi (bosma.sabilillah.id) perlu redeploy di Dokploy.
- Di template html/template, jangan menyisipkan string query utuh (mis. `?{{.q}}`) karena `=` ter-escape menjadi `%3d`; tulis `?periode={{...}}`.

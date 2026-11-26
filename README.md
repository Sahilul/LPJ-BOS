# LPJ BOS

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![SQLite](https://img.shields.io/badge/SQLite-embedded-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Aplikasi web untuk menyusun **LPJ/SPJ dana BOSP** sekolah: rencana belanja (RKAS), penerimaan dana, transaksi belanja, mutasi kas, Buku Kas Umum, kuitansi/bukti terima, undangan & notulen rapat, sampai laporan PDF/Excel.

Satu binary Go tanpa CGO — template, aset, dan migrasi database tertanam di dalamnya. Cukup satu container dan satu file SQLite.

## Daftar isi

- [Fitur](#fitur)
- [Teknologi](#teknologi)
- [Mulai cepat](#mulai-cepat)
- [Konfigurasi](#konfigurasi)
- [Deploy](#deploy)
- [Akses AI (MCP)](#akses-ai-mcp)
- [Backup & pemulihan](#backup--pemulihan)
- [Pengembangan](#pengembangan)
- [Struktur proyek](#struktur-proyek)
- [Keamanan](#keamanan)
- [Lisensi](#lisensi)

## Fitur

**Data pokok**
- Setup akun pertama, login, ganti kata sandi, dan token API.
- Tahun anggaran (satu tahun aktif sebagai bawaan).
- Profil sekolah: nama, NPSN, alamat, kop surat, logo, TTD dan stempel pejabat.
- Master data: penerima, toko/rekanan, komponen belanja, rekening, aturan pajak — lengkap dengan impor Excel.

**Perencanaan & realisasi**
- RKAS: rencana belanja per paket/komponen dan rencana pendapatan.
- Penerimaan dana per tahap dengan nomor bukti otomatis.
- Belanja: pencatatan transaksi, pajak PPN, PPh 21 (TER bulanan), PPh 22, PPh 23.
- BKU (Buku Kas Umum) dan mutasi kas otomatis dari transaksi.
- Lampiran bukti (foto/PDF) per transaksi.

**Dokumen & laporan**
- Kuitansi dan bukti terima dalam PDF, siap cetak.
- Undangan rapat dan notulen rapat (PDF).
- Laporan PDF/Excel dan rekap SPJ.

**Lainnya**
- Backup & pemulihan database dari dalam aplikasi.
- Server MCP untuk asisten AI (ChatGPT, Claude, Copilot, Cursor).

## Teknologi

| Bagian | Pilihan |
| --- | --- |
| Bahasa | Go 1.26 (stdlib `net/http`, `html/template`) |
| Database | SQLite via `modernc.org/sqlite` (tanpa CGO) |
| PDF | `go-pdf/fpdf` |
| Excel | `excelize` |
| Aset | CSS/JS ditulis tangan, tertanam lewat `embed` |
| Deploy | Docker multi-stage (Alpine), cocok untuk Dokploy/VPS |

## Mulai cepat

### Docker (disarankan)

```bash
git clone https://github.com/Sahilul/LPJ-BOS.git
cd LPJ-BOS
docker compose up -d --build
```

Buka <http://localhost:8080>, lalu buat akun pertama di halaman setup.

### Tanpa Docker

Butuh Go 1.26+.

```bash
cd src
go run ./cmd/lpjbos
```

Bawaan: `PORT=8080`, `DATA_DIR=../data`. Database SQLite dan unggahan tersimpan di folder `data/` yang tidak ikut ke repository.

### Build binary

```bash
cd src
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o lpjbos ./cmd/lpjbos
DATA_DIR=./data PORT=8080 ./lpjbos
```

## Konfigurasi

Semua pengaturan lewat variabel lingkungan.

| Variabel | Bawaan | Keterangan |
| --- | --- | --- |
| `PORT` | `8080` | Port HTTP. Image Docker memakai `80`. |
| `ADDR` | kosong | Alamat bind, mis. `127.0.0.1`. Kosong = semua interface. |
| `DATA_DIR` | `../data` (dari `src/`) | Folder database, unggahan, dan cache PDF. |
| `DB_PATH` | `<DATA_DIR>/lpjbos.sqlite` | Lokasi file SQLite bila ingin dipisah. |
| `TZ` | `Asia/Jakarta` | Zona waktu untuk tanggal dan jam. |

Contoh:

```bash
DATA_DIR=/var/lib/lpjbos PORT=8080 TZ=Asia/Jakarta ./lpjbos
```

## Deploy

### Dokploy (disarankan)

1. **Create Project → Create Service → Compose**.
2. Provider **GitHub** (atau Git: `https://github.com/Sahilul/LPJ-BOS.git`), branch `main`.
3. **Compose Path**: `./docker-compose.yml` → **Deploy**. File `docker-compose.override.yml` hanya untuk dev lokal dan diabaikan Dokploy.
4. Tab **Domains** → **Add Domain**: host mis. `lpj.sekolah.sch.id`, Service Name `app`, Container Port `80`, HTTPS aktif (Let's Encrypt). Arahkan DNS (A record) ke IP server.
5. Buka domain, buat akun pertama.

Data tersimpan di volume Docker `lpjbos-data` sehingga aman saat redeploy. Setiap kali ada perubahan di `main`, klik **Deploy** (atau aktifkan auto-deploy) untuk membangun ulang image.

### VPS dengan Docker Compose + reverse proxy

```bash
git clone https://github.com/Sahilul/LPJ-BOS.git /opt/lpjbos
cd /opt/lpjbos
docker compose up -d --build
```

Arahkan reverse proxy (Nginx/Caddy/Traefik) ke `127.0.0.1:8080`. Contoh Nginx:

```nginx
server {
    listen 443 ssl;
    server_name lpj.sekolah.sch.id;
    client_max_body_size 60m;   # unggahan lampiran

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

> `client_max_body_size` perlu dinaikkan karena lampiran bisa besar (batas aplikasi 55 MB).

### Update

```bash
cd /opt/lpjbos
git pull
docker compose up -d --build
```

Migrasi database berjalan otomatis saat aplikasi start. Tetap cadangkan data sebelum update besar.

### Tanpa Docker (systemd)

```bash
cd /opt/lpjbos/src
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /usr/local/bin/lpjbos ./cmd/lpjbos
```

```ini
# /etc/systemd/system/lpjbos.service
[Unit]
Description=LPJ BOS
After=network.target

[Service]
Environment=DATA_DIR=/var/lib/lpjbos PORT=8080 TZ=Asia/Jakarta
ExecStart=/usr/local/bin/lpjbos
Restart=always
User=lpjbos

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now lpjbos
```

## Akses AI (MCP)

Aplikasi menjalankan server [Model Context Protocol](https://modelcontextprotocol.io) di `POST /mcp`, sehingga asisten AI bisa membaca dan mencatat data memakai aturan validasi yang sama dengan form aplikasi.

1. Buka menu **Sistem → Akses AI (MCP)**.
2. **Buat token**: beri nama, pilih akses **Baca saja** atau **Baca & tulis**.
3. Salin token (hanya ditampilkan sekali), lalu pasang di klien.

Token dikirim lewat header `Authorization: Bearer lpbos_…` atau `X-API-Token: lpbos_…`. Untuk klien yang tidak bisa mengirim header kustom, token boleh ditaruh di alamat: `https://domain-anda/mcp/lpbos_…`.

### VS Code (Copilot)

`.vscode/mcp.json`:

```json
{
  "servers": {
    "lpjbos": {
      "type": "http",
      "url": "https://lpj.sekolah.sch.id/mcp",
      "headers": { "Authorization": "Bearer lpbos_..." }
    }
  }
}
```

### Claude Desktop / Cursor

`claude_desktop_config.json` atau `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "lpjbos": {
      "type": "http",
      "url": "https://lpj.sekolah.sch.id/mcp",
      "headers": { "Authorization": "Bearer lpbos_..." }
    }
  }
}
```

### Claude.ai (konektor kustom)

- URL server: `https://lpj.sekolah.sch.id/mcp`
- Autentikasi: **Tanpa masuk** (bukan OAuth).
- Header permintaan, pilih salah satu:
  - `X-API-Token` = token saja (tanpa kata “Bearer”), atau
  - `Authorization` = `Bearer <token>`.
- Centang **Wajib**, lalu tambahkan.

### ChatGPT (konektor kustom)

ChatGPT belum bisa mengirim header kustom, jadi token ditaruh di URL. Butuh paket **Plus/Pro/Business** dan domain **HTTPS publik** (localhost tidak bisa).

1. **Settings → Apps & Connectors → Advanced**, nyalakan **Developer mode**.
2. **Create connector**:
   - Nama: `LPJ BOS`
   - MCP Server URL: `https://lpj.sekolah.sch.id/mcp/lpbos_...`
   - Authentication: **No authentication**
   - Centang **I trust this application**, lalu **Create**.
3. Di percakapan baru: **+** → **Developer mode** → pilih **LPJ BOS**, lalu minta mis. “Tampilkan BKU bulan ini”.

> URL tersebut memuat token rahasia — jangan dibagikan. Untuk mencatat/menghapus data pakai token **Baca & tulis**; ChatGPT meminta konfirmasi sebelum menjalankan aksi tulis.

### Tool yang tersedia

Sekitar 38 tool: baca master data, RKAS, penerimaan, belanja, BKU, mutasi kas, rapat/notulen, dan laporan; serta tool tulis untuk mencatat transaksi, master data, dan rapat. Semua panggilan tercatat di tabel log pada halaman **Akses AI (MCP)**.

## Backup & pemulihan

- **Dari aplikasi**: menu **Cadangkan data** menghasilkan file database SQLite. Simpan di luar server.
- **Dari Docker**: cadangkan volume `lpjbos-data`, atau pakai fitur Volume Backups di Dokploy.
- **Pemulihan**: hentikan aplikasi, ganti file `lpjbos.sqlite` (beserta `-wal`/`-shm` bila ada) dengan hasil cadangan, lalu jalankan lagi.

```bash
docker compose stop app
docker run --rm -v lpjbos-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/lpjbos-$(date +%F).tar.gz -C /data .
docker compose start app
```

## Pengembangan

```bash
cd src
gofmt -l ./internal/app     # harus kosong
go vet ./...
go test ./...
```

Untuk dev lokal dengan kode yang di-mount:

```bash
docker compose up --build   # memakai docker-compose.override.yml (port 8080, ./data)
```

## Struktur proyek

```
.
├── Dockerfile                  # build multi-stage → image Alpine
├── docker-compose.yml          # produksi (Dokploy)
├── docker-compose.override.yml # dev lokal
└── src/
    ├── cmd/lpjbos/main.go      # entry point
    └── internal/app/
        ├── server.go           # router, middleware, sesi
        ├── c_*.go              # handler per halaman
        ├── svc_*.go            # logika bisnis & dokumen
        ├── mcp*.go             # server MCP
        ├── migrations/         # skema SQL berurutan
        ├── templates/          # html/template
        └── assets/             # CSS & JS
```

## Keamanan

- Ganti kata sandi bawaan setelah setup; pakai HTTPS di produksi.
- Token MCP setara kata sandi. Beri akses **Baca saja** bila hanya untuk bertanya, dan hapus token yang tidak dipakai.
- Jangan menaruh token di URL yang dibagikan/di-log; bentuk `/mcp/<token>` hanya untuk klien yang tidak mendukung header.
- Batas unggahan 55 MB per permintaan; naikkan `client_max_body_size` di reverse proxy bila perlu.
- Laporkan celah keamanan lewat issue privat/email pemelihara, bukan issue publik.

## Lisensi

Dirilis di bawah lisensi [MIT](LICENSE).

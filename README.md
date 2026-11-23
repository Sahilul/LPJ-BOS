# LPJ BOS

Aplikasi web untuk menyusun LPJ/SPJ dana BOSP sekolah: rencana belanja (RKAS), penerimaan dana, transaksi belanja, mutasi kas, Buku Kas Umum, kuitansi/bukti terima, dan laporan PDF/Excel.

Dibangun dengan PHP 8.3 + SQLite, dijalankan lewat Docker.

## Menjalankan

```bash
docker compose up -d --build
```

Buka http://localhost:8080 lalu buat akun pertama di halaman setup. Data (database SQLite dan unggahan) tersimpan di folder `data/` yang tidak ikut ke repository.

## Deploy di Dokploy

1. **Create Project → Create Service → Compose**.
2. Provider **GitHub** (atau Git: `https://github.com/Sahilul/LPJ-BOS.git`), branch `main`.
3. **Compose Path**: `./docker-compose.yml` (bawaan) → **Deploy**. File `docker-compose.override.yml` hanya untuk dev lokal dan diabaikan Dokploy.
4. Tab **Domains** → Add Domain: host mis. `lpj.sekolah.sch.id`, Service Name `app`, Container Port `80`, HTTPS aktif (Let's Encrypt). Arahkan DNS domain (A record) ke IP server.
5. Buka domain, buat akun pertama.

Data tersimpan di volume Docker `lpjbos-data`, tetap aman saat redeploy. Cadangkan rutin dari menu **Cadangkan data** atau fitur Volume Backups di Dokploy.

## Akses AI (MCP)

Aplikasi menyediakan server [Model Context Protocol](https://modelcontextprotocol.io) di `POST /mcp` agar asisten AI (Claude, Copilot, Cursor) bisa membaca dan mencatat data. Buat token di menu **Sistem → Akses AI (MCP)**, lalu tambahkan ke konfigurasi klien:

```json
{
  "servers": {
    "lpjbos": {
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "headers": { "Authorization": "Bearer lpbos_..." }
    }
  }
}
```

Rincian rancangan dan progres ada di [PLAN.md](PLAN.md).

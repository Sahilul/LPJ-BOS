<div class="error-page">
  <div class="code">Galat <?= (int) $code ?></div>
  <h1><?= $code === 404 ? 'Halaman tidak ditemukan' : ($code === 419 ? 'Sesi kedaluwarsa' : 'Terjadi kesalahan') ?></h1>
  <p><?= e($message ?: ($code === 404 ? 'Alamat yang Anda buka tidak ada atau sudah dihapus.' : 'Muat ulang halaman lalu coba lagi.')) ?></p>
  <a class="btn btn-ghost" href="/"><?= icon('home', 16) ?>Kembali ke beranda</a>
</div>

<?php
$cfg = json_encode(['mcpServers' => ['lpjbos' => [
    'type' => 'http',
    'url' => $url,
    'headers' => ['Authorization' => 'Bearer ' . ($baru['token'] ?? '<token>')],
]]], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
?>
<div class="page-head">
  <div>
    <h1>Akses AI (MCP)</h1>
    <p class="lead">Hubungkan asisten AI (Claude, Copilot, Cursor) ke data LPJ BOS lewat Model Context Protocol. Asisten memakai aturan validasi yang sama dengan form di aplikasi ini.</p>
  </div>
</div>

<?php if ($baru): ?>
<div class="card card-pad stack" style="--gap:12px;margin-bottom:24px;border-color:var(--p-accent)">
  <h2 style="font-size:18px">Token “<?= e($baru['nama']) ?>”</h2>
  <p class="muted small">Salin sekarang. Token ini tidak akan ditampilkan lagi; bila hilang, cabut lalu buat yang baru.</p>
  <input class="input mono" readonly value="<?= e($baru['token']) ?>" onclick="this.select()" aria-label="Token API">
</div>
<?php endif; ?>

<div class="grid grid-tahun">
  <div class="stack" style="--gap:24px">
    <div class="card">
      <div class="table-wrap">
        <table class="table">
          <thead><tr><th>Nama</th><th>Akses</th><th>Terakhir dipakai</th><th class="num">Panggilan</th><th><span class="sr-only">Aksi</span></th></tr></thead>
          <tbody>
            <?php foreach ($tokens as $t): ?>
            <tr>
              <td class="primary"><?= e($t['nama']) ?><span class="muted small mono" style="display:block">lpbos_<?= e($t['prefix']) ?>_…</span></td>
              <td><?= $t['akses'] === 'tulis' ? '<span class="badge badge-accent">Baca & tulis</span>' : '<span class="badge badge-off">Baca saja</span>' ?></td>
              <td class="small muted"><?= $t['last_used_at'] ? e($t['last_used_at']) : 'Belum pernah' ?></td>
              <td class="num"><?= (int) $t['n_panggil'] ?></td>
              <td>
                <div class="row-actions">
                  <form method="post" action="/api/<?= (int) $t['id'] ?>/hapus" data-confirm="Cabut token “<?= e($t['nama']) ?>”?">
                    <?= csrf_field() ?>
                    <button class="btn btn-danger btn-icon" type="submit" aria-label="Cabut token <?= e($t['nama']) ?>"><?= icon('trash', 16) ?></button>
                  </form>
                </div>
              </td>
            </tr>
            <?php endforeach; ?>
            <?php if (!$tokens): ?><tr><td colspan="5" class="muted">Belum ada token.</td></tr><?php endif; ?>
          </tbody>
        </table>
      </div>
      <div class="table-foot"><?= count($tokens) ?> token</div>
    </div>

    <div class="card card-pad stack" style="--gap:12px">
      <h2 style="font-size:18px">Cara menghubungkan</h2>
      <p class="small">Alamat server: <code class="mono"><?= e($url) ?></code>. Kirim token di header <code class="mono">Authorization: Bearer …</code>. Contoh konfigurasi (VS Code <code>.vscode/mcp.json</code> memakai kunci <code>servers</code>, Claude/Cursor memakai <code>mcpServers</code>):</p>
      <pre class="mono small" style="white-space:pre-wrap;margin:0;padding:12px;background:var(--p-surface);border-radius:8px"><?= e($cfg) ?></pre>
    </div>

    <div class="card">
      <div class="table-wrap">
        <table class="table">
          <thead><tr><th>Waktu</th><th>Token</th><th>Tool</th><th>Hasil</th></tr></thead>
          <tbody>
            <?php foreach ($log as $l): ?>
            <tr>
              <td class="small muted mono"><?= e($l['created_at']) ?></td>
              <td class="small"><?= e($l['token_nama'] ?? '(dicabut)') ?></td>
              <td class="mono small" title="<?= e($l['args']) ?>"><?= e($l['tool']) ?></td>
              <td class="small"><?= $l['ok'] ? '<span class="badge badge-ok">OK</span>' : '<span class="badge badge-warn">Gagal</span> ' . e($l['pesan']) ?></td>
            </tr>
            <?php endforeach; ?>
            <?php if (!$log): ?><tr><td colspan="4" class="muted">Belum ada aktivitas.</td></tr><?php endif; ?>
          </tbody>
        </table>
      </div>
      <div class="table-foot">30 aktivitas terakhir</div>
    </div>
  </div>

  <form class="card card-pad stack" method="post" action="/api" novalidate style="--gap:16px;align-self:start">
    <?= csrf_field() ?>
    <h2 style="font-size:18px">Buat token</h2>
    <?php partial('partials/field', ['name' => 'nama', 'label' => 'Nama', 'value' => old('nama'), 'hint' => 'Mis. “Claude di laptop bendahara”.']); ?>
    <?php partial('partials/field', ['name' => 'akses', 'label' => 'Akses', 'type' => 'select', 'value' => old('akses', 'baca'), 'options' => ['baca' => 'Baca saja', 'tulis' => 'Baca & tulis (bisa mencatat/menghapus)']]); ?>
    <button class="btn" type="submit"><?= icon('plus', 16) ?>Buat token</button>
  </form>
</div>

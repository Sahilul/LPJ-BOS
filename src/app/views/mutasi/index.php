<?php
use App\Controllers\MutasiController;
$badge = ['tarik' => 'badge-accent', 'setor' => 'badge-ok', 'admin_bank' => 'badge-warn', 'bunga' => 'badge-ok'];
$label = ['tarik' => 'Tarik tunai', 'setor' => 'Setor ke bank', 'admin_bank' => 'Biaya admin', 'bunga' => 'Bunga bank'];
$saldo = saldo_kas($tahun . '-12-31');
?>
<div class="page-head">
  <div>
    <h1>Mutasi kas</h1>
    <p class="lead">Perpindahan uang antara rekening dan kas tunai, serta bunga dan biaya administrasi bank.</p>
  </div>
  <div class="actions">
    <a class="btn btn-ghost" href="/mutasi/baru?jenis=setor">Setor ke bank</a>
    <a class="btn" href="/mutasi/baru?jenis=tarik"><?= icon('plus', 16) ?>Tarik tunai</a>
  </div>
</div>

<div class="grid grid-3" style="margin-bottom:18px">
  <div class="card stat"><span class="label"><?= icon('wallet', 16) ?>Kas tunai</span><span class="value mono" style="display:block"><?= rupiah($saldo['tunai']) ?></span><span class="foot">Saldo akhir tahun <?= (int) $tahun ?></span></div>
  <div class="card stat"><span class="label"><?= icon('swap', 16) ?>Rekening bank</span><span class="value mono" style="display:block"><?= rupiah($saldo['bank']) ?></span><span class="foot">Saldo akhir tahun <?= (int) $tahun ?></span></div>
  <div class="card stat"><span class="label"><?= icon('book', 16) ?>Total saldo</span><span class="value mono" style="display:block"><?= rupiah($saldo['total']) ?></span><span class="foot">Sama dengan saldo BKU</span></div>
</div>

<div class="card">
  <form class="toolbar" method="get" action="/mutasi">
    <?php partial('partials/tahun-select', ['tahun' => $tahun]); ?>
    <span class="muted small" style="margin-left:auto">Catat juga <a href="/mutasi/baru?jenis=bunga">bunga</a> atau <a href="/mutasi/baru?jenis=admin_bank">biaya admin</a></span>
  </form>
  <?php if (!$rows): ?>
    <div class="empty">
      <div class="ic-wrap"><?= icon('swap', 22) ?></div>
      <h3>Belum ada mutasi tahun <?= (int) $tahun ?></h3>
      <p>Catat penarikan tunai sebelum membayar belanja secara tunai.</p>
      <a class="btn" href="/mutasi/baru?jenis=tarik"><?= icon('plus', 16) ?>Tarik tunai</a>
    </div>
  <?php else: ?>
    <div class="table-wrap">
      <table class="table">
        <thead><tr><th>Tanggal</th><th>Jenis</th><th>Uraian</th><th class="num">Jumlah</th><th><span class="sr-only">Aksi</span></th></tr></thead>
        <tbody>
          <?php foreach ($rows as $r): ?>
          <tr id="row-<?= (int) $r['id'] ?>">
            <td class="mono"><?= e(tanggal_indo($r['tgl'], true)) ?></td>
            <td><span class="badge <?= $badge[$r['jenis']] ?? 'badge-off' ?>"><?= e($label[$r['jenis']] ?? $r['jenis']) ?></span></td>
            <td class="primary"><a href="/mutasi/<?= (int) $r['id'] ?>/edit"><?= e($r['uraian']) ?></a><?php if ($r['no_bukti'] !== ''): ?><div class="muted small mono"><?= e($r['no_bukti']) ?></div><?php endif; ?></td>
            <td class="num"><?= rupiah((float) $r['jumlah']) ?></td>
            <td>
              <div class="row-actions">
                <a class="btn btn-ghost btn-icon" href="/mutasi/<?= (int) $r['id'] ?>/edit" aria-label="Ubah mutasi"><?= icon('edit', 16) ?></a>
                <form method="post" action="/mutasi/<?= (int) $r['id'] ?>/hapus" data-confirm="Hapus mutasi ini?">
                  <?= csrf_field() ?>
                  <button class="btn btn-danger btn-icon" type="submit" aria-label="Hapus mutasi"><?= icon('trash', 16) ?></button>
                </form>
              </div>
            </td>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <div class="table-foot"><?= count($rows) ?> mutasi</div>
  <?php endif; ?>
</div>

<?php $total = array_sum(array_column($rows, 'jumlah')); ?>
<div class="page-head">
  <div>
    <h1>Penerimaan dana</h1>
    <p class="lead">Dana BOSP yang masuk ke rekening sekolah tahun <?= (int) $tahun ?>. Setiap penerimaan tercatat di debit BKU dan menambah saldo bank.</p>
  </div>
  <div class="actions">
    <a class="btn btn-ghost" href="/rencana/pendapatan"><?= icon('list', 16) ?>Rencana pendapatan</a>
    <a class="btn" href="/penerimaan/baru"><?= icon('plus', 16) ?>Catat penerimaan</a>
  </div>
</div>

<div class="card">
  <form class="toolbar" method="get" action="/penerimaan">
    <?php partial('partials/tahun-select', ['tahun' => $tahun]); ?>
    <span class="muted small" style="margin-left:auto">Total diterima <strong class="mono" style="color:var(--p-ink)"><?= rupiah($total) ?></strong></span>
  </form>
  <?php if (!$rows): ?>
    <div class="empty">
      <div class="ic-wrap"><?= icon('inbox', 22) ?></div>
      <h3>Belum ada penerimaan tahun <?= (int) $tahun ?></h3>
      <p>Catat dana Tahap 1 begitu masuk ke rekening, supaya saldo BKU dan belanja bisa berjalan.</p>
      <a class="btn" href="/penerimaan/baru"><?= icon('plus', 16) ?>Catat penerimaan</a>
    </div>
  <?php else: ?>
    <div class="table-wrap">
      <table class="table">
        <thead><tr><th>Tanggal</th><th>No. bukti</th><th>Uraian</th><th>Tahap</th><th>Rencana pendapatan</th><th class="num">Jumlah</th><th><span class="sr-only">Aksi</span></th></tr></thead>
        <tbody>
          <?php foreach ($rows as $r): ?>
          <tr id="row-<?= (int) $r['id'] ?>">
            <td class="mono"><?= e(tanggal_indo($r['tgl'], true)) ?></td>
            <td class="mono"><?= e($r['no_bukti']) ?></td>
            <td class="primary"><a href="/penerimaan/<?= (int) $r['id'] ?>/edit"><?= e($r['uraian']) ?></a></td>
            <td><span class="badge badge-accent"><?= e($r['tahap']) ?></span></td>
            <td><?= $r['rencana_uraian'] ? e($r['rencana_uraian']) : '<span class="muted">Belum dipasangkan</span>' ?></td>
            <td class="num"><?= rupiah((float) $r['jumlah']) ?></td>
            <td>
              <div class="row-actions">
                <a class="btn btn-ghost btn-icon" href="/penerimaan/<?= (int) $r['id'] ?>/edit" aria-label="Ubah <?= e($r['uraian']) ?>"><?= icon('edit', 16) ?></a>
                <form method="post" action="/penerimaan/<?= (int) $r['id'] ?>/hapus" data-confirm="Hapus penerimaan <?= e(rupiah((float) $r['jumlah'])) ?>?">
                  <?= csrf_field() ?>
                  <button class="btn btn-danger btn-icon" type="submit" aria-label="Hapus <?= e($r['uraian']) ?>"><?= icon('trash', 16) ?></button>
                </form>
              </div>
            </td>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <div class="table-foot"><?= count($rows) ?> penerimaan · total <?= rupiah($total) ?></div>
  <?php endif; ?>
</div>

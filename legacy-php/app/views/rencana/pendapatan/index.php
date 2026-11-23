<div class="page-head">
  <div>
    <div class="crumb"><a href="/rencana">Rencana</a><span aria-hidden="true">/</span><span>Pendapatan</span></div>
    <h1>Rencana pendapatan</h1>
    <p class="lead">Catat target dana per tahap, lalu pasangkan penerimaan aktual untuk melihat realisasi dan selisih tahun <?= (int) $tahun ?>.</p>
  </div>
  <div class="actions">
    <a class="btn btn-ghost" href="/rencana"><?= icon('list', 16) ?>Rencana belanja</a>
    <a class="btn" href="/rencana/pendapatan/baru"><?= icon('plus', 16) ?>Tambah target</a>
  </div>
</div>

<div class="grid grid-3" style="margin-bottom:18px">
  <div class="card stat">
    <span class="label"><?= icon('list', 16) ?>Total target</span>
    <span class="value mono" style="display:block"><?= rupiah($totalTarget) ?></span>
    <span class="foot"><?= count($rows) ?> rencana pendapatan</span>
  </div>
  <div class="card stat">
    <span class="label"><?= icon('inbox', 16) ?>Aktual terpasang</span>
    <span class="value mono" style="display:block"><?= rupiah($totalRealisasi) ?></span>
    <span class="foot">Dipasangkan dengan target</span>
  </div>
  <div class="card stat">
    <span class="label"><?= icon('swap', 16) ?>Belum dipasangkan</span>
    <span class="value mono" style="display:block"><?= rupiah($belumDipetakan) ?></span>
    <span class="foot"><a href="/penerimaan">Buka daftar penerimaan</a></span>
  </div>
</div>

<div class="card">
  <?php if (!$rows): ?>
    <div class="empty">
      <div class="ic-wrap"><?= icon('inbox', 22) ?></div>
      <h3>Belum ada target pendapatan</h3>
      <p>Tambahkan rencana dana, misalnya BOSP Tahap 1 dan Tahap 2. Penerimaan aktual bisa dipasangkan saat dicatat.</p>
      <a class="btn" href="/rencana/pendapatan/baru"><?= icon('plus', 16) ?>Tambah target pendapatan</a>
    </div>
  <?php else: ?>
    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr><th>Target</th><th>Bulan</th><th class="num">Direncanakan</th><th class="num">Aktual</th><th class="num">Selisih</th><th><span class="sr-only">Aksi</span></th></tr>
        </thead>
        <tbody>
          <?php foreach ($rows as $r):
              $selisih = (float) $r['target_jumlah'] - (float) $r['realisasi'];
              $persen = $r['target_jumlah'] > 0 ? min(100, (int) round($r['realisasi'] / $r['target_jumlah'] * 100)) : 0;
          ?>
          <tr id="row-<?= (int) $r['id'] ?>">
            <td class="primary">
              <a href="/rencana/pendapatan/<?= (int) $r['id'] ?>/edit"><?= e($r['uraian']) ?></a>
              <div class="muted small">
                <span class="badge badge-accent"><?= e(\App\Controllers\RencanaPendapatanController::TAHAP[$r['tahap']] ?? $r['tahap']) ?></span>
                <?= (int) $r['n_penerimaan'] ?> penerimaan
              </div>
            </td>
            <td><?= $r['target_bulan'] ? e(bulan_indo((int) $r['target_bulan'])) : '<span class="muted">Belum ditentukan</span>' ?></td>
            <td class="num mono"><?= rupiah((float) $r['target_jumlah']) ?></td>
            <td class="num mono">
              <?= rupiah((float) $r['realisasi']) ?>
              <div class="progress" role="progressbar" aria-valuenow="<?= $persen ?>" aria-valuemin="0" aria-valuemax="100" aria-label="Realisasi <?= e($r['uraian']) ?>"><span style="width:<?= $persen ?>%"></span></div>
            </td>
            <td class="num mono"><strong<?= $selisih < 0 ? ' style="color:var(--bad)"' : '' ?>><?= rupiah($selisih) ?></strong></td>
            <td>
              <div class="row-actions">
                <a class="btn btn-ghost btn-icon" href="/penerimaan/baru?rencana_pendapatan=<?= (int) $r['id'] ?>" title="Catat penerimaan aktual" aria-label="Catat penerimaan aktual untuk <?= e($r['uraian']) ?>"><?= icon('plus', 16) ?></a>
                <a class="btn btn-ghost btn-icon" href="/rencana/pendapatan/<?= (int) $r['id'] ?>/edit" title="Ubah" aria-label="Ubah <?= e($r['uraian']) ?>"><?= icon('edit', 16) ?></a>
                <form method="post" action="/rencana/pendapatan/<?= (int) $r['id'] ?>/hapus" data-confirm="Hapus rencana pendapatan “<?= e($r['uraian']) ?>”?">
                  <?= csrf_field() ?>
                  <button class="btn btn-danger btn-icon" type="submit" title="Hapus" aria-label="Hapus <?= e($r['uraian']) ?>"<?= $r['n_penerimaan'] ? ' disabled' : '' ?>><?= icon('trash', 16) ?></button>
                </form>
              </div>
            </td>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <div class="table-foot">Selisih = target − aktual. Penerimaan yang belum dipasangkan tetap tercatat di BKU dan tercantum pada total belum dipasangkan.</div>
  <?php endif; ?>
</div>

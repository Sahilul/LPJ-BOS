<?php
$totPagu = array_sum(array_column($rows, 'pagu'));
$totReal = array_sum(array_column($rows, 'realisasi'));
$totSisa = $totPagu - $totReal;
$totPct = $totPagu > 0 ? min(100, (int) round($totReal / $totPagu * 100)) : 0;
$filtered = $q !== '' || $komp !== '';
?>
<div class="page-head">
  <div>
    <h1>Rencana belanja</h1>
    <p class="lead">Pagu per kegiatan untuk tahun <?= $tahun ?>. Setiap transaksi di BKU memotong sisa pagu baris yang dipilih.</p>
  </div>
  <div class="actions">
    <a class="btn btn-ghost" href="/rencana/pendapatan"><?= icon('inbox', 16) ?>Rencana pendapatan</a>
    <a class="btn" href="/rencana/baru"><?= icon('plus', 16) ?>Tambah rencana</a>
  </div>
</div>

<div class="grid grid-3" style="margin-bottom:18px">
  <div class="card stat">
    <span class="label"><?= icon('list', 16) ?>Total pagu</span>
    <span class="value mono" style="display:block"><?= rupiah($totPagu) ?></span>
    <span class="foot"><?= count($rows) ?> baris rencana</span>
  </div>
  <div class="card stat">
    <span class="label"><?= icon('cart', 16) ?>Sudah dibelanjakan</span>
    <span class="value mono" style="display:block"><?= rupiah($totReal) ?></span>
    <span class="foot"><span class="progress" style="flex:1" role="progressbar" aria-valuenow="<?= $totPct ?>" aria-valuemin="0" aria-valuemax="100" aria-label="Realisasi"><span style="width:<?= $totPct ?>%"></span></span><span class="mono"><?= $totPct ?>%</span></span>
  </div>
  <div class="card stat">
    <span class="label"><?= icon('wallet', 16) ?>Sisa anggaran</span>
    <span class="value mono" style="display:block"><?= rupiah($totSisa) ?></span>
    <span class="foot">Belum dibelanjakan</span>
  </div>
</div>

<div class="card">
  <form class="toolbar" method="get" action="/rencana" role="search">
    <label class="search">
      <span class="sr-only">Cari rencana</span>
      <?= icon('search', 16) ?>
      <input class="input" type="search" name="q" value="<?= e($q) ?>" placeholder="Cari uraian, kegiatan, rekening">
    </label>
    <label class="sr-only" for="flt-komponen">Komponen</label>
    <select class="input" id="flt-komponen" name="komponen" data-autosubmit style="width:auto;max-width:260px">
      <option value="">Semua komponen</option>
      <?php foreach ($komponen as $k => $v): ?><option value="<?= e($k) ?>"<?= $komp === (string) $k ? ' selected' : '' ?>><?= e($v) ?></option><?php endforeach; ?>
    </select>
    <button class="btn btn-ghost" type="submit">Cari</button>
    <?php if ($filtered): ?><a class="btn btn-ghost" href="/rencana"><?= icon('x', 16) ?>Hapus filter</a><?php endif; ?>
  </form>

  <?php if (!$rows): ?>
    <div class="empty">
      <div class="ic-wrap"><?= icon('list', 22) ?></div>
      <h3><?= $filtered ? 'Tidak ada yang cocok' : 'Belum ada rencana belanja' ?></h3>
      <p><?= $filtered ? 'Coba kata kunci lain atau hapus filter.' : 'Masukkan pagu dari RKAS, misalnya Belanja ATK Rp200.000 × 6 bulan. Sisanya akan berkurang otomatis setiap kali dicatat di BKU.' ?></p>
      <?php if (!$filtered): ?><a class="btn" href="/rencana/baru"><?= icon('plus', 16) ?>Tambah rencana</a><?php endif; ?>
    </div>
  <?php else: ?>
    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Uraian</th>
            <th>Jenis</th>
            <th class="num">Pagu</th>
            <th class="num">Terpakai</th>
            <th class="num">Sisa</th>
            <th style="min-width:120px">Realisasi</th>
            <th><span class="sr-only">Aksi</span></th>
          </tr>
        </thead>
        <tbody>
          <?php $lastKomp = null; foreach ($rows as $r):
              $sisa = (float) $r['pagu'] - (float) $r['realisasi'];
              $pct = $r['pagu'] > 0 ? min(100, (int) round($r['realisasi'] / $r['pagu'] * 100)) : 0;
              if ($r['komponen_kode'] !== $lastKomp): $lastKomp = $r['komponen_kode']; ?>
            <tr class="group-row"><td colspan="7"><span class="mono"><?= e($r['komponen_kode']) ?></span> <?= e($r['komponen_nama'] ?? '') ?></td></tr>
          <?php endif; ?>
          <tr id="row-<?= (int) $r['id'] ?>">
            <td class="primary">
              <a href="/rencana/<?= (int) $r['id'] ?>/edit"><?= e($r['uraian']) ?></a>
              <div class="muted small">
                <?= (int) $r['frekuensi_bulan'] ?> bulan × <?= rupiah((float) $r['harga_satuan']) ?> per bulan
                <?php if ($r['rekening_kode'] !== ''): ?> · <span class="mono"><?= e($r['rekening_kode']) ?></span><?php endif; ?>
                <?php if ($r['toko_nama']): ?> · Toko: <?= e($r['toko_nama']) ?><?php endif; ?>
              </div>
              <?php $detail = $paket[$r['id']] ?? []; ?>
              <?php if ($r['jenis'] === 'barang' && !empty($detail['items'])): ?>
                <div class="muted small"><?= e(implode(' · ', array_map(fn ($item) => $item['nama'] . ' ' . angka((float) $item['volume'], 2) . ' ' . $item['satuan'], $detail['items']))) ?></div>
              <?php elseif (!empty($detail['penerima'])): ?>
                <div class="muted small"><?= e('Penerima: ' . implode(', ', array_column($detail['penerima'], 'nama'))) ?></div>
              <?php endif; ?>
            </td>
            <td><span class="badge <?= $r['jenis'] === 'barang' ? 'badge-off' : 'badge-accent' ?>"><?= e(\App\Controllers\RencanaController::JENIS[$r['jenis']] ?? $r['jenis']) ?></span></td>
            <td class="num mono"><?= rupiah((float) $r['pagu']) ?></td>
            <td class="num mono"><?= $r['realisasi'] > 0 ? rupiah((float) $r['realisasi']) : '<span class="muted">—</span>' ?></td>
            <td class="num mono"><strong<?= $sisa <= 0 ? ' class="muted"' : '' ?>><?= rupiah($sisa) ?></strong></td>
            <td>
              <div class="progress<?= $pct >= 100 ? ' is-full' : '' ?>" role="progressbar" aria-valuenow="<?= $pct ?>" aria-valuemin="0" aria-valuemax="100" aria-label="Realisasi <?= e($r['uraian']) ?>"><span style="width:<?= $pct ?>%"></span></div>
              <div class="muted small mono" style="margin-top:4px"><?= $pct ?>%<?= $r['n_belanja'] ? ' · ' . (int) $r['n_belanja'] . ' transaksi' : '' ?></div>
            </td>
            <td>
              <div class="row-actions">
                <?php if ($sisa > 0 && (int) $r['periode_terpakai'] < (int) $r['frekuensi_bulan']): ?>
                  <a class="btn btn-ghost btn-icon" href="/belanja/baru?rencana=<?= (int) $r['id'] ?>" title="Catat belanja" aria-label="Catat belanja dari <?= e($r['uraian']) ?>"><?= icon('cart', 16) ?></a>
                <?php endif; ?>
                <a class="btn btn-ghost btn-icon" href="/rencana/<?= (int) $r['id'] ?>/edit" title="Ubah" aria-label="Ubah <?= e($r['uraian']) ?>"><?= icon('edit', 16) ?></a>
                <form method="post" action="/rencana/<?= (int) $r['id'] ?>/hapus" data-confirm="Hapus rencana “<?= e($r['uraian']) ?>”?">
                  <?= csrf_field() ?>
                  <button class="btn btn-danger btn-icon" type="submit" title="Hapus" aria-label="Hapus <?= e($r['uraian']) ?>"<?= $r['n_belanja'] ? ' disabled' : '' ?>><?= icon('trash', 16) ?></button>
                </form>
              </div>
            </td>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <div class="table-foot"><?= count($rows) ?> baris · baris yang sudah dipakai di BKU tidak bisa dihapus.</div>
  <?php endif; ?>
</div>

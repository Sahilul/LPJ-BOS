<?php
$jenisLabel = ['barang' => 'Barang / jasa', 'honor' => 'Honor', 'transport' => 'Transport', 'tarik' => 'Tarik tunai', 'setor' => 'Setor bank', 'bunga' => 'Bunga bank', 'admin_bank' => 'Biaya admin'];
$link = fn (array $r) => match ($r['src']) {
    'trm' => '/penerimaan/' . $r['id'] . '/edit',
    'bel' => '/belanja/' . $r['id'] . '/edit',
    default => '/mutasi/' . $r['id'] . '/edit',
};
?>
<div class="page-head">
  <div>
    <h1>Buku kas umum</h1>
    <p class="lead"><?= bulan_indo($bulan) ?> <?= (int) $tahun ?>. Klik baris untuk mengubah; saldo dihitung ulang otomatis.</p>
  </div>
  <div class="actions">
    <a class="btn btn-ghost" href="/penerimaan/baru"><?= icon('inbox', 16) ?>Penerimaan</a>
    <a class="btn btn-ghost" href="/mutasi/baru"><?= icon('swap', 16) ?>Mutasi kas</a>
    <a class="btn" href="/belanja/baru"><?= icon('plus', 16) ?>Catat belanja</a>
  </div>
</div>

<nav class="month-tabs" aria-label="Pilih bulan">
  <?php for ($m = 1; $m <= 12; $m++): ?>
    <a href="/bku?tahun=<?= (int) $tahun ?>&amp;bulan=<?= $m ?>"<?= $m === $bulan ? ' aria-current="page"' : '' ?>>
      <?= e(substr(bulan_indo($m), 0, 3)) ?><?php if (!empty($perBulan[$m])): ?><span class="dot" aria-label="<?= (int) $perBulan[$m] ?> transaksi"></span><?php endif; ?>
    </a>
  <?php endfor; ?>
</nav>

<div class="grid grid-4" style="margin-bottom:18px">
  <div class="card stat"><span class="label">Saldo awal bulan</span><span class="value mono" style="display:block"><?= rupiah($saldoAwal['total']) ?></span><span class="foot">Per <?= e(tanggal_indo(sprintf('%04d-%02d-01', $tahun, $bulan), true)) ?></span></div>
  <div class="card stat"><span class="label">Debit (masuk)</span><span class="value mono" style="display:block"><?= rupiah($debit) ?></span><span class="foot">Penerimaan &amp; bunga</span></div>
  <div class="card stat"><span class="label">Kredit (keluar)</span><span class="value mono" style="display:block"><?= rupiah($kredit) ?></span><span class="foot">Belanja &amp; biaya bank</span></div>
  <div class="card stat"><span class="label">Saldo akhir bulan</span><span class="value mono" style="display:block"><?= rupiah($saldoAkhir['total']) ?></span><span class="foot">Tunai <?= rupiah($saldoAkhir['tunai']) ?> · Bank <?= rupiah($saldoAkhir['bank']) ?></span></div>
</div>

<div class="card">
  <form class="toolbar" method="get" action="/bku">
    <?php partial('partials/tahun-select', ['tahun' => $tahun]); ?>
    <span class="muted small" style="margin-left:auto"><?= count($rows) ?> transaksi bulan ini</span>
  </form>
  <?php if (!$rows): ?>
    <div class="empty">
      <div class="ic-wrap"><?= icon('book', 22) ?></div>
      <h3>Belum ada transaksi di <?= bulan_indo($bulan) ?></h3>
      <p>Catat penerimaan dana, tarik tunai, lalu belanja. Semuanya otomatis tersusun di sini dengan saldo berjalan.</p>
      <a class="btn" href="/belanja/baru"><?= icon('plus', 16) ?>Catat belanja</a>
    </div>
  <?php else: ?>
    <div class="table-wrap">
      <table class="table bku-table">
        <thead><tr><th>Tanggal</th><th>No. bukti</th><th>Uraian</th><th class="num">Debit</th><th class="num">Kredit</th><th class="num">Saldo</th><th><span class="sr-only">Aksi</span></th></tr></thead>
        <tbody>
          <tr class="group-row"><td colspan="3">Saldo awal <?= bulan_indo($bulan) ?></td><td></td><td></td><td class="num" data-label="Saldo"><?= rupiah($saldoAwal['total']) ?></td><td></td></tr>
          <?php foreach ($rows as $r): $info = $r['debit'] == 0 && $r['kredit'] == 0; ?>
          <tr id="<?= $r['src'] ?>-<?= (int) $r['id'] ?>"<?= $info ? ' class="is-off"' : '' ?>>
            <td class="mono bku-tgl">
              <form class="bku-date-form" method="post" action="/bku/<?= e($r['src']) ?>/<?= (int) $r['id'] ?>/tanggal">
                <?= csrf_field() ?>
                <input type="date" name="tgl" value="<?= e($r['tgl']) ?>" aria-label="Ubah tanggal <?= e($r['uraian']) ?>" title="Ubah tanggal; nomor bukti dan saldo akan disusun ulang otomatis" required data-bku-date>
              </form>
            </td>
            <td class="mono bku-no"><?= e($r['no_bukti'] ?: '—') ?></td>
            <td class="primary">
              <a href="<?= $link($r) ?>"><?= e($r['uraian']) ?></a>
              <div class="sub">
                <?php if ($r['src'] === 'trm'): ?><span class="badge badge-ok">Penerimaan</span>
                <?php elseif ($r['src'] === 'bel'): ?><span class="badge badge-accent"><?= e($jenisLabel[$r['jenis']] ?? $r['jenis']) ?></span><span class="badge badge-off"><?= $r['metode'] === 'bank' ? 'Bank' : 'Tunai' ?></span>
                <?php else: ?><span class="badge <?= $info ? 'badge-off' : 'badge-warn' ?>"><?= e($jenisLabel[$r['jenis']] ?? $r['jenis']) ?></span><?php if ($info): ?><span class="muted small"><?= rupiah((float) $r['nilai']) ?>, tidak mengubah saldo</span><?php endif; ?>
                <?php endif; ?>
              </div>
            </td>
            <td class="num" data-label="Debit"><?= $r['debit'] > 0 ? rupiah((float) $r['debit']) : '' ?></td>
            <td class="num" data-label="Kredit"><?= $r['kredit'] > 0 ? rupiah((float) $r['kredit']) : '' ?></td>
            <td class="num" data-label="Saldo"><?= rupiah((float) $r['saldo']) ?></td>
            <td><div class="row-actions">
              <?php if ($r['src'] === 'bel'): ?>
                <a class="btn btn-ghost btn-sm" href="/dokumen/belanja/<?= (int) $r['id'] ?>/kuitansi.pdf" target="_blank" rel="noopener">Cetak kuitansi</a>
                <?php if (in_array($r['jenis'], ['honor', 'transport'], true)): ?>
                  <a class="btn btn-ghost btn-sm" href="/dokumen/belanja/<?= (int) $r['id'] ?>/bukti-terima.pdf">Unduh bukti terima</a>
                <?php endif; ?>
                <?php if ((int) $r['lampiran'] > 0): ?>
                  <a class="btn btn-ghost btn-sm" href="/belanja/<?= (int) $r['id'] ?>/edit#lampiran">Lampiran (<?= (int) $r['lampiran'] ?>)</a>
                <?php endif; ?>
              <?php endif; ?>
              <a class="btn btn-ghost btn-icon" href="<?= $link($r) ?>" aria-label="Ubah <?= e($r['uraian']) ?>"><?= icon('edit', 16) ?></a>
            </div></td>
          </tr>
          <?php endforeach; ?>
        </tbody>
        <tfoot><tr><td colspan="3">Jumlah <?= bulan_indo($bulan) ?></td><td class="num" data-label="Debit"><?= rupiah($debit) ?></td><td class="num" data-label="Kredit"><?= rupiah($kredit) ?></td><td class="num" data-label="Saldo akhir"><?= rupiah($saldoAkhir['total']) ?></td><td></td></tr></tfoot>
      </table>
    </div>
    <div class="table-foot">Kuitansi tersedia untuk transaksi belanja; daftar bukti terima tersedia untuk honor dan transport.</div>
  <?php endif; ?>
</div>

<?php
use App\Services\Laporan;

$q = 'periode=' . urlencode($p['kode']);
$pilihan = [];
for ($m = 1; $m <= 12; $m++) {
    $pilihan[(string) $m] = substr(bulan_indo($m), 0, 3);
}
$pilihan += ['T1' => 'Tahap I', 'T2' => 'Tahap II', 'Y' => 'Setahun'];
?>
<div class="page-head">
  <div>
    <h1>Laporan &amp; SPJ</h1>
    <p class="lead"><?= e($p['label']) ?>. Semua laporan disusun otomatis dari BKU, siap dicetak atau diekspor ke Excel.</p>
  </div>
  <div class="actions">
    <a class="btn btn-ghost" href="/laporan/spj?<?= $q ?>&amp;format=xlsx"><?= icon('download', 16) ?>SPJ Excel</a>
    <a class="btn" href="/laporan/spj?<?= $q ?>" target="_blank" rel="noopener"><?= icon('file', 16) ?>Cetak SPJ lengkap</a>
  </div>
</div>

<nav class="month-tabs" aria-label="Pilih periode laporan">
  <?php foreach ($pilihan as $kode => $label): ?>
    <a href="/laporan?periode=<?= e((string) $kode) ?>"<?= (string) $kode === $p['kode'] ? ' aria-current="page"' : '' ?>><?= e($label) ?></a>
  <?php endforeach; ?>
</nav>

<div class="grid grid-4" style="margin-bottom:18px">
  <div class="card stat"><span class="label">Transaksi periode ini</span><span class="value mono" style="display:block"><?= (int) $nTransaksi ?></span><span class="foot"><?= (int) $nBelanja ?> bukti belanja</span></div>
  <div class="card stat"><span class="label">Saldo kas akhir</span><span class="value mono" style="display:block"><?= rupiah($saldo['total']) ?></span><span class="foot">Per <?= e(tanggal_indo($p['akhir'], true)) ?></span></div>
  <div class="card stat"><span class="label">Kas tunai</span><span class="value mono" style="display:block"><?= rupiah($saldo['tunai']) ?></span><span class="foot">Di brankas bendahara</span></div>
  <div class="card stat"><span class="label">Saldo bank</span><span class="value mono" style="display:block"><?= rupiah($saldo['bank']) ?></span><span class="foot">Rekening BOS</span></div>
</div>

<div class="card card-pad" style="margin-bottom:18px">
  <h2 style="margin:0 0 4px;font-size:1.05rem">SPJ lengkap <?= e($p['label']) ?></h2>
  <p class="muted" style="margin:0">Satu PDF berisi BKU, buku pembantu tunai, bank, dan pajak, realisasi vs RKAS, register penutupan kas dengan BA pemeriksaan kas, serta rincian setiap bukti belanja. Kop, tanda tangan, dan stempel diambil dari profil sekolah dan pejabat.</p>
</div>

<div class="grid grid-2">
  <?php foreach (Laporan::JENIS as $jenis => [$nama, $desc, $ic]): ?>
    <div class="card card-pad" style="display:flex;flex-direction:column;gap:12px">
      <div style="display:flex;gap:12px;align-items:flex-start">
        <span class="muted" style="flex:none;margin-top:2px"><?= icon($ic, 20) ?></span>
        <div>
          <h3 style="margin:0 0 2px;font-size:1rem"><?= e($nama) ?></h3>
          <p class="muted small" style="margin:0"><?= e($desc) ?></p>
        </div>
      </div>
      <div class="row-actions" style="justify-content:flex-start;margin-top:auto">
        <a class="btn btn-sm" href="/laporan/<?= $jenis ?>?<?= $q ?>"><?= icon('eye', 14) ?>Lihat</a>
        <a class="btn btn-ghost btn-sm" href="/laporan/<?= $jenis ?>?<?= $q ?>&amp;format=pdf" target="_blank" rel="noopener"><?= icon('file', 14) ?>PDF</a>
        <a class="btn btn-ghost btn-sm" href="/laporan/<?= $jenis ?>?<?= $q ?>&amp;format=xlsx"><?= icon('download', 14) ?>Excel</a>
      </div>
    </div>
  <?php endforeach; ?>
</div>

<?php
$done = count(array_filter($steps, fn ($x) => $x['done']));
$pct = (int) round($done / count($steps) * 100);
?>
<section class="band" aria-labelledby="judul">
  <div>
    <p class="muted small">Tahun anggaran <span class="mono"><?= $tahun ?></span> · <a href="/tahun">Ganti</a></p>
    <h1 id="judul" style="margin-top:8px"><?= e($s['nama'] ?: 'Sekolah Anda') ?></h1>
    <p class="muted" style="margin-top:10px">
      <?= $s['npsn'] ? 'NPSN <span class="mono">' . e($s['npsn']) . '</span>' : 'NPSN belum diisi' ?>
      <?= $s['kabupaten'] ? ' · ' . e($s['kabupaten']) : '' ?>
    </p>
  </div>
  <dl class="kv">
    <dt>Kepala sekolah</dt><dd><?= e($kepala['nama'] ?? '—') ?></dd>
    <dt>Bendahara</dt><dd><?= e($bendahara['nama'] ?? '—') ?></dd>
  </dl>
</section>

<div class="split" style="margin-top:18px">
  <section class="card" aria-labelledby="persiapan">
    <div class="card-head">
      <div>
        <h2 id="persiapan">Persiapan sebelum mencatat</h2>
        <p><?= $done ?> dari <?= count($steps) ?> selesai. Data ini dipakai di setiap kuitansi dan laporan.</p>
      </div>
      <span class="badge <?= $pct === 100 ? 'badge-ok' : 'badge-accent' ?> mono"><?= $pct ?>%</span>
    </div>
    <div style="padding:0 24px 4px"><div class="progress" role="progressbar" aria-valuenow="<?= $pct ?>" aria-valuemin="0" aria-valuemax="100" aria-label="Kemajuan persiapan" style="margin-top:16px"><span style="width:<?= $pct ?>%"></span></div></div>
    <ol class="steps" style="margin-top:8px">
      <?php foreach ($steps as $i => $st): ?>
        <li>
          <a class="step<?= $st['done'] ? ' done' : '' ?>" href="<?= e($st['href']) ?>">
            <span class="dot" aria-hidden="true"><?= $st['done'] ? icon('check', 14) : $i + 1 ?></span>
            <span class="t"><?= e($st['label']) ?><span><?= e($st['hint']) ?></span></span>
            <span class="sr-only"><?= $st['done'] ? 'Selesai' : 'Belum' ?></span>
            <span class="go"><?= icon('arrow', 16) ?></span>
          </a>
        </li>
      <?php endforeach; ?>
    </ol>
  </section>

  <div class="stack">
    <div class="grid grid-2">
      <?php foreach ([
          ['/penerima', 'users', 'Penerima aktif', $counts['penerima']],
          ['/toko', 'store', 'Toko aktif', $counts['toko']],
          ['/komponen', 'layers', 'Komponen', $counts['komponen']],
          ['/rekening', 'hash', 'Kode rekening', $counts['rekening']],
      ] as [$href, $ic, $label, $n]): ?>
        <a class="card stat" href="<?= $href ?>">
          <span class="label"><?= icon($ic, 16) ?><?= e($label) ?></span>
          <span class="value mono" style="display:block"><?= angka($n) ?></span>
          <span class="foot">Kelola <?= icon('arrow', 14) ?></span>
        </a>
      <?php endforeach; ?>
    </div>
    <section class="card card-pad" aria-labelledby="keu">
      <div class="split-head">
        <h3 id="keu">Anggaran <?= $tahun ?></h3>
        <a class="btn btn-sm" href="/belanja/baru"><?= icon('plus', 14) ?>Catat belanja</a>
      </div>
      <?php $real = $keu['pagu'] > 0 ? min(100, (int) round($keu['realisasi'] / $keu['pagu'] * 100)) : 0; ?>
      <div class="progress" role="progressbar" aria-valuenow="<?= $real ?>" aria-valuemin="0" aria-valuemax="100" aria-label="Realisasi belanja" style="margin-top:14px"><span style="width:<?= $real ?>%"></span></div>
      <dl class="kv-list kv-end" style="margin-top:14px">
        <dt>Rencana belanja</dt><dd class="mono"><a href="/rencana"><?= rupiah($keu['pagu']) ?></a></dd>
        <dt>Realisasi (<?= $real ?>%)</dt><dd class="mono"><a href="/bku"><?= rupiah($keu['realisasi']) ?></a></dd>
        <dt>Dana diterima</dt><dd class="mono"><a href="/penerimaan"><?= rupiah($keu['diterima']) ?></a></dd>
        <dt><?= $keu['per'] === date('Y-m-d') ? 'Saldo kas hari ini' : 'Saldo kas per ' . e(tanggal_indo($keu['per'])) ?></dt><dd class="mono"><a href="/mutasi"><?= rupiah($keu['saldo']['total']) ?></a></dd>
      </dl>
    </section>
  </div>
</div>

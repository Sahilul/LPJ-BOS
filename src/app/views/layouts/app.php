<?php
$user = auth_user();
$sek = sekolah();
$nav = [
    'Utama' => [
        ['/', 'home', 'Beranda'],
    ],
    'Sekolah' => [
        ['/sekolah', 'school', 'Profil & kop surat'],
        ['/pejabat', 'badge', 'Pejabat & TTD'],
    ],
    'Master data' => [
        ['/penerima', 'users', 'Penerima'],
        ['/toko', 'store', 'Toko & penyedia'],
        ['/komponen', 'layers', 'Komponen BOSP'],
        ['/rekening', 'hash', 'Kode rekening'],
        ['/pajak', 'percent', 'Tarif pajak'],
    ],
    'Keuangan' => [
        ['/rencana', 'list', 'Rencana belanja'],
        ['/penerimaan', 'inbox', 'Penerimaan dana'],
        ['/bku', 'book', 'Buku kas umum'],
        ['/mutasi', 'swap', 'Mutasi kas'],
        ['/laporan', 'chart', 'Laporan & SPJ'],
    ],
    'Sistem' => [
        ['/tahun', 'calendar', 'Tahun anggaran'],
        ['/backup', 'database', 'Cadangkan data'],
        ['/api', 'plug', 'Akses AI (MCP)'],
        ['/akun', 'key', 'Akun'],
    ],
];
?><!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title><?= e(($title ?? '') !== '' ? $title . ' · ' : '') ?>LPJ BOS</title>
<meta name="theme-color" content="#111111">
<link rel="preload" href="<?= asset('fonts/epilogue.woff2') ?>" as="font" type="font/woff2" crossorigin>
<link rel="stylesheet" href="<?= asset('css/app.css') ?>">
<link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%3E%3Crect width='32' height='32' rx='8' fill='%231a4be0'/%3E%3Ctext x='16' y='21.5' font-family='Arial' font-weight='700' font-size='14' fill='white' text-anchor='middle'%3EBOS%3C/text%3E%3C/svg%3E">
<script src="<?= asset('js/app.js') ?>" defer></script>
</head>
<body>
<a class="skip" href="#isi">Lewati ke konten</a>
<div class="shell">
  <aside class="side" id="menu" aria-label="Menu utama">
    <a class="brand" href="/">
      <span class="brand-mark" aria-hidden="true">B</span>
      <span class="brand-name">LPJ BOS<small><?= e($sek['nama'] ?: 'Sekolah belum diisi') ?></small></span>
    </a>
    <nav>
      <?php foreach ($nav as $group => $links): ?>
        <div class="nav-group">
          <div class="nav-label"><?= e($group) ?></div>
          <?php foreach ($links as [$href, $ic, $label]): ?>
            <?php if ($href === null): ?>
              <span class="nav-link is-soon" aria-disabled="true"><?= icon($ic) ?><?= e($label) ?><span class="soon">segera</span></span>
            <?php else: ?>
              <a class="nav-link" href="<?= e($href) ?>"<?= is_active($href) ? ' aria-current="page"' : '' ?>><?= icon($ic) ?><?= e($label) ?></a>
            <?php endif; ?>
          <?php endforeach; ?>
        </div>
      <?php endforeach; ?>
    </nav>
    <div class="side-foot">
      <div class="who"><strong><?= e($user['nama'] ?? '') ?></strong><span class="year-switch">Tahun anggaran <span class="mono"><?= tahun_aktif() ?></span> · <a href="/tahun">Ganti</a></span></div>
      <form method="post" action="/logout">
        <?= csrf_field() ?>
        <button class="nav-link" type="submit"><?= icon('logout') ?>Keluar</button>
      </form>
    </div>
  </aside>

  <div class="main">
    <header class="topbar">
      <button class="btn btn-ghost btn-icon" type="button" data-nav-toggle aria-controls="menu" aria-expanded="false" aria-label="Buka menu"><?= icon('menu') ?></button>
      <span class="title"><?= e($title ?? 'LPJ BOS') ?></span>
      <a class="year-pill" href="/tahun" aria-label="Tahun anggaran <?= tahun_aktif() ?>, ganti tahun"><?= icon('calendar', 14) ?><?= tahun_aktif() ?></a>
    </header>
    <main class="page" id="isi" tabindex="-1">
      <?php $flashes = take_flash(); if ($flashes): ?>
        <div class="flashes" role="status">
          <?php foreach ($flashes as $f): ?>
            <div class="alert alert-<?= e($f['type']) ?>">
              <?= icon($f['type'] === 'success' ? 'check' : 'x') ?>
              <div class="msg"><?= e($f['message']) ?></div>
              <button class="close" type="button" data-dismiss aria-label="Tutup pesan"><?= icon('x', 16) ?></button>
            </div>
          <?php endforeach; ?>
        </div>
      <?php endif; ?>
      <?= $content ?>
    </main>
  </div>
</div>
</body>
</html>

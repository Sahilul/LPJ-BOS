<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<style>
@page { margin: 18mm 15mm; }
body { font: 9pt "DejaVu Sans", sans-serif; color: #171717; line-height: 1.4; }
.kop { width: 100%; max-height: 32mm; object-fit: contain; margin-bottom: 7mm; }
.school { text-align: center; padding-bottom: 4mm; border-bottom: 2px solid #222; margin-bottom: 6mm; }
h1 { font-size: 14pt; text-align: center; margin: 0 0 2mm; letter-spacing: 1px; }
.sub { text-align: center; margin: 0 0 7mm; }
.meta { width: 100%; margin-bottom: 5mm; }
.meta td { padding: 1mm 0; vertical-align: top; }
table.list { width: 100%; border-collapse: collapse; }
.list th, .list td { border: 1px solid #555; padding: 2mm 1.5mm; vertical-align: top; }
.list th { background: #eee; text-align: left; }
.list td.num, .list th.num { text-align: right; white-space: nowrap; }
.list thead { display: table-header-group; }
.list tr { page-break-inside: avoid; }
.total { text-align: right; margin-top: 4mm; font-weight: bold; }
.statement { margin-top: 7mm; }
.signatures { width: 100%; text-align: center; margin-top: 12mm; page-break-inside: avoid; }
.signatures td { width: 50%; vertical-align: top; }
.sig { height: 21mm; }
.sig img { max-width: 30mm; max-height: 20mm; }
.name { font-weight: bold; text-decoration: underline; }
.nip { font-size: 8pt; }
</style>
</head>
<body>
<?php if ($kop): ?>
  <img class="kop" src="<?= e($kop) ?>" alt="Kop surat <?= e($sekolah['nama'] ?? '') ?>">
<?php else: ?>
  <div class="school">
    <strong><?= e($sekolah['nama'] ?? '') ?></strong><br>
    <?= e(implode(', ', array_filter([$sekolah['alamat'] ?? '', $sekolah['desa'] ?? '', $sekolah['kecamatan'] ?? '', $sekolah['kabupaten'] ?? '']))) ?>
  </div>
<?php endif; ?>
<h1>DAFTAR BUKTI PENERIMAAN <?= e(strtoupper($jenisLabel)) ?></h1>
<p class="sub">Tahun Anggaran <?= (int) substr($belanja['tgl'], 0, 4) ?></p>
<table class="meta">
  <tr><td style="width:34mm">Nomor bukti</td><td>: <?= e($belanja['no_bukti']) ?></td></tr>
  <tr><td>Tanggal</td><td>: <?= e($tanggal) ?></td></tr>
  <tr><td>Uraian</td><td>: <?= e($belanja['uraian']) ?></td></tr>
</table>
<table class="list">
  <thead>
    <tr><th style="width:7mm">No.</th><th>Nama / identitas penerima</th><th class="num">Bruto</th><th class="num">PPh 21</th><th class="num">Diterima</th><th style="width:30mm">Tanda tangan</th></tr>
  </thead>
  <tbody>
    <?php $totalBruto = $totalPajak = $totalNetto = 0.0; ?>
    <?php foreach ($penerima as $i => $p): ?>
      <?php $totalBruto += (float) $p['bruto']; $totalPajak += (float) $p['pph21']; $totalNetto += (float) $p['netto']; ?>
      <tr>
        <td><?= $i + 1 ?></td>
        <td><strong><?= e($p['nama']) ?></strong>
          <?php if ($p['nip']): ?><br>NIP/NIK: <?= e($p['nip']) ?><?php endif; ?>
          <?php if ($p['jabatan']): ?><br><?= e($p['jabatan']) ?><?php endif; ?>
          <br><?= e(angka((float) $p['volume'], 2)) ?> <?= e($p['satuan']) ?> × <?= e(rupiah((float) $p['tarif'])) ?>
        </td>
        <td class="num"><?= e(rupiah((float) $p['bruto'])) ?></td>
        <td class="num"><?= e(rupiah((float) $p['pph21'])) ?></td>
        <td class="num"><strong><?= e(rupiah((float) $p['netto'])) ?></strong></td>
        <td></td>
      </tr>
    <?php endforeach; ?>
  </tbody>
</table>
<div class="total">Total bruto <?= e(rupiah($totalBruto)) ?> · PPh 21 <?= e(rupiah($totalPajak)) ?> · Total diterima <?= e(rupiah($totalNetto)) ?></div>
<p class="statement">Dengan ini penerima menyatakan telah menerima pembayaran sesuai jumlah bersih yang tercantum pada daftar di atas.</p>
<div style="text-align:right"><?= e($lokasiTanggal) ?></div>
<table class="signatures">
  <tr>
    <td>Mengetahui,<br>Kepala Sekolah<div class="sig"><?php if ($kepala['ttd']): ?><img src="<?= e($kepala['ttd']) ?>" alt="Tanda tangan kepala sekolah"><?php endif; ?></div>
      <div class="name"><?= e($kepala['nama'] ?: '................................') ?></div>
      <?php if ($kepala['nip']): ?><div class="nip">NIP. <?= e($kepala['nip']) ?></div><?php endif; ?>
    </td>
    <td>Bendahara BOS<div class="sig"><?php if ($stempel): ?><img src="<?= e($stempel) ?>" alt="Stempel sekolah"><?php endif; ?><?php if ($bendahara['ttd']): ?><img src="<?= e($bendahara['ttd']) ?>" alt="Tanda tangan bendahara"><?php endif; ?></div>
      <div class="name"><?= e($bendahara['nama'] ?: '................................') ?></div>
      <?php if ($bendahara['nip']): ?><div class="nip">NIP. <?= e($bendahara['nip']) ?></div><?php endif; ?>
    </td>
  </tr>
</table>
</body>
</html>

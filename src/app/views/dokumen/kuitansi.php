<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<style>
@page { margin: 12mm 16mm; }
body { font: 11pt "DejaVu Sans", sans-serif; color: #171717; line-height: 1.5; }
.kop { width: 100%; max-height: 24mm; object-fit: contain; margin-bottom: 5mm; }
.school { text-align: center; padding-bottom: 3mm; border-bottom: 2px solid #222; margin-bottom: 5mm; }
h1 { font-size: 15pt; letter-spacing: 2px; text-align: center; margin: 0 0 1mm; }
.sub { text-align: center; font-size: 10pt; margin: 0 0 4mm; }
.meta { width: 100%; border-collapse: collapse; margin-bottom: 7mm; }
.meta td { padding: .5mm 0; vertical-align: top; }
.meta td:first-child { width: 34mm; }
.receipt { border: 1px solid #333; padding: 4mm; margin: 3mm 0; }
.amount { font-size: 14pt; font-weight: bold; }
.words { font-style: italic; }
.detail { margin: 3mm 0 4mm; }
.signatures { width: 100%; text-align: center; margin-top: 5mm; page-break-inside: avoid; }
.signatures td { width: 33.33%; vertical-align: top; }
.sig { height: 12mm; }
.sig img { max-width: 26mm; max-height: 11mm; }
.stamp { max-width: 13mm; max-height: 11mm; }
.name { font-weight: bold; text-decoration: underline; }
.nip { font-size: 9pt; }
.page-break { page-break-after: always; }
.page-break:last-child { page-break-after: auto; }
</style>
</head>
<body>
<?php foreach ($receivers as $i => $p): ?>
  <?php if ($kop): ?>
    <img class="kop" src="<?= e($kop) ?>" alt="Kop surat <?= e($sekolah['nama'] ?? '') ?>">
  <?php else: ?>
    <div class="school">
      <strong><?= e($sekolah['nama'] ?? '') ?></strong><br>
      <?= e(implode(', ', array_filter([$sekolah['alamat'] ?? '', $sekolah['desa'] ?? '', $sekolah['kecamatan'] ?? '', $sekolah['kabupaten'] ?? '']))) ?>
    </div>
  <?php endif; ?>
  <h1>KUITANSI</h1>
  <p class="sub">Tahun Anggaran <?= (int) substr($belanja['tgl'], 0, 4) ?></p>
  <table class="meta">
    <tr><td>Nomor</td><td>: <?= e($belanja['no_bukti']) ?></td></tr>
    <tr><td>Sudah terima dari</td><td>: Bendahara BOS <?= e($sekolah['nama'] ?? '') ?></td></tr>
    <tr><td>Uang sejumlah</td><td>: <span class="amount"><?= e(rupiah((float) $p['netto'])) ?></span></td></tr>
    <tr><td>Terbilang</td><td>: <span class="words"><?= e(ucfirst(terbilang((int) $p['netto']))) ?> rupiah</span></td></tr>
  </table>
  <div class="receipt">
    <strong>Untuk pembayaran</strong>
    <div class="detail"><?= e($belanja['uraian']) ?></div>
    <?php if ($belanja['jenis'] !== 'barang'): ?>
      <div>Penerima: <strong><?= e($p['nama']) ?></strong><?= $receiverCount > 1 ? ' (' . $receiverCount . ' orang; rincian pada daftar bukti terima)' : (!empty($p['jabatan']) ? ' · ' . e($p['jabatan']) : '') ?></div>
      <?php if ((float) $p['pph21'] > 0): ?><div>Nilai bruto <?= e(rupiah((float) $p['bruto'])) ?> · PPh 21 <?= e(rupiah((float) $p['pph21'])) ?></div><?php endif; ?>
    <?php elseif ($items): ?>
      <table class="meta">
        <?php foreach ($items as $item): ?>
          <tr><td><?= e($item['nama']) ?></td><td class="amount"><?= e(rupiah((float) $item['subtotal'])) ?></td></tr>
        <?php endforeach; ?>
      </table>
      <?php if ($belanja['toko_nama']): ?><div>Penyedia: <?= e($belanja['toko_nama']) ?></div><?php endif; ?>
    <?php endif; ?>
  </div>
  <div style="text-align:right"><?= e($lokasiTanggal) ?></div>
  <table class="signatures">
    <tr>
      <td>Mengetahui,<br>Kepala Sekolah<div class="sig"><?php if ($kepala['ttd']): ?><img src="<?= e($kepala['ttd']) ?>" alt="Tanda tangan kepala sekolah"><?php endif; ?></div>
        <div class="name"><?= e($kepala['nama'] ?: '................................') ?></div>
        <?php if ($kepala['nip']): ?><div class="nip">NIP. <?= e($kepala['nip']) ?></div><?php endif; ?>
      </td>
      <td>Bendahara BOS<div class="sig"><?php if ($stempel): ?><img class="stamp" src="<?= e($stempel) ?>" alt="Stempel sekolah"><?php endif; ?><?php if ($bendahara['ttd']): ?><img src="<?= e($bendahara['ttd']) ?>" alt="Tanda tangan bendahara"><?php endif; ?></div>
        <div class="name"><?= e($bendahara['nama'] ?: '................................') ?></div>
        <?php if ($bendahara['nip']): ?><div class="nip">NIP. <?= e($bendahara['nip']) ?></div><?php endif; ?>
      </td>
      <td><?= $belanja['jenis'] === 'barang' ? 'Penyedia' : 'Penerima' ?><div class="sig"></div><div class="name"><?= e($p['nama']) ?></div>
        <?php if (!empty($p['nip'])): ?><div class="nip">NIP. <?= e($p['nip']) ?></div><?php endif; ?>
      </td>
    </tr>
  </table>
  <?php if ($i < count($receivers) - 1): ?><div class="page-break"></div><?php endif; ?>
<?php endforeach; ?>
</body>
</html>

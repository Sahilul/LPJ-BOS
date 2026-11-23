<?php
$namaSekolah = (string) ($sekolah['nama'] ?? '');
$lembaga = preg_match('/^(MA|MTs|MI|RA|Madrasah)\b/i', $namaSekolah) || stripos((string) ($sekolah['jenjang'] ?? ''), 'M') === 0 ? 'Madrasah' : 'Sekolah';
$tahun = (int) substr($belanja['tgl'], 0, 4);
$tahapNo = trim(preg_replace('/^Tahap\s*/i', '', $tahap ?? ''));
$tglPendek = function (string $d): string {
    $ts = strtotime($d);
    return date('j', $ts) . ' ' . substr(bulan_indo((int) date('n', $ts)), 0, 3) . ' ' . date('Y', $ts);
};
$uang = fn (float $n): string => 'Rp. ' . number_format($n, 2, ',', '.');
$desaKec = implode(' / ', array_filter([$sekolah['desa'] ?? '', $sekolah['kecamatan'] ?? '']));
$komponen = trim(($belanja['komponen_kode'] ?? '') . ' — ' . ($belanja['komponen_nama'] ?? ''), ' —');
$sumber = 'Dana BOS' . ($tahap ? ' ' . $tahap : '') . ' Tahun ' . $tahun;
$kota = (string) ($sekolah['kabupaten'] ?? '');
?><!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<style>
@page { margin: 12mm 14mm; }
body { font-family: "Times New Roman", Times, serif; font-size: 12pt; color: #111; line-height: 1.35; }
.lembar { border: 1.2pt solid #222; padding: 6mm 8mm 5mm; }
.kop { width: 100%; max-height: 24mm; margin-bottom: 4mm; }
h1 { font-size: 15pt; text-align: center; text-decoration: underline; letter-spacing: .5pt; margin: 0 0 6mm; }
.data { width: 100%; border-collapse: collapse; }
.data td { padding: .6mm 0; vertical-align: top; }
.data td.l { width: 42mm; }
.data td.c { width: 6mm; }
.data td.v { font-weight: bold; }
.jumlah { display: inline-block; border: 1.4pt solid #222; border-radius: 3mm; padding: 1.5mm 5mm; font-size: 14pt; font-weight: bold; }
.terbilang { font-style: italic; font-weight: bold; }
.ket { font-weight: normal; }
.ttd { width: 100%; border-collapse: collapse; margin-top: 6mm; }
.ttd td { vertical-align: top; text-align: center; }
.ruang { height: 17mm; position: relative; }
.ruang img.sig { max-height: 17mm; max-width: 40mm; }
.stempel { position: absolute; left: -6mm; top: -8mm; width: 30mm; opacity: .85; }
.nama { font-weight: bold; text-decoration: underline; text-transform: uppercase; }
.nama-p { font-weight: bold; text-decoration: underline; }
.nip { font-size: 10pt; }
.tgl { text-align: right; margin: 3mm 0 2mm; }
.lunas { border: 1.4pt solid #b4232a; border-radius: 3mm; color: #b4232a; text-align: center; font-weight: bold; font-size: 13pt; padding: 1.5mm; margin-top: 4mm; }
.page-break { page-break-after: always; }
</style>
</head>
<body>
<?php foreach ($receivers as $i => $p): ?>
<?php if ($kop): ?><img class="kop" src="<?= e($kop) ?>" alt="Kop surat"><?php endif; ?>
<div class="lembar">
  <h1>KWITANSI / BUKTI PEMBAYARAN</h1>
  <table class="data">
    <tr><td class="l">Tahun Anggaran</td><td class="c">:</td><td class="v"><?= $tahun ?></td></tr>
    <?php if ($tahapNo !== ''): ?><tr><td class="l">Tahap</td><td class="c">:</td><td class="v"><?= e($tahapNo) ?></td></tr><?php endif; ?>
    <tr><td class="l">Nomor Bukti</td><td class="c">:</td><td class="v"><?= e($belanja['no_bukti']) ?></td></tr>
    <tr><td class="l">Sudah terima dari</td><td class="c">:</td><td class="v">Kepala <?= $lembaga ?></td></tr>
    <tr><td class="l"><?= $lembaga ?></td><td class="c">:</td><td class="v"><?= e($namaSekolah) ?></td></tr>
    <?php if ($desaKec !== ''): ?><tr><td class="l">Desa / Kecamatan</td><td class="c">:</td><td class="v"><?= e($desaKec) ?></td></tr><?php endif; ?>
    <?php if ($kota !== ''): ?><tr><td class="l">Kabupaten</td><td class="c">:</td><td class="v"><?= e($kota) ?></td></tr><?php endif; ?>
    <?php if (!empty($sekolah['provinsi'])): ?><tr><td class="l">Provinsi</td><td class="c">:</td><td class="v"><?= e($sekolah['provinsi']) ?></td></tr><?php endif; ?>
    <tr><td class="l" style="padding-top:3mm">Jumlah Uang</td><td class="c" style="padding-top:3mm">:</td><td style="padding-top:1.5mm"><span class="jumlah"><?= e($uang((float) $p['netto'])) ?></span></td></tr>
    <tr><td class="l" style="padding-top:2mm">Terbilang</td><td class="c" style="padding-top:2mm">:</td><td class="terbilang" style="padding-top:2mm"><?= e(ucfirst(terbilang((int) $p['netto']))) ?> rupiah</td></tr>
    <tr><td class="l" style="padding-top:2mm">Untuk pembayaran</td><td class="c" style="padding-top:2mm">:</td><td class="v" style="padding-top:2mm">
      <?= e($belanja['uraian']) ?>
      <?php if ($komponen !== ''): ?><div class="ket">(<?= e($komponen) ?>)</div><?php endif; ?>
      <?php if ($belanja['jenis'] !== 'barang' && $receiverCount > 1): ?><div class="ket"><?= $receiverCount ?> orang; rincian pada daftar bukti terima.</div><?php endif; ?>
      <?php if ((float) $p['pph21'] > 0): ?><div class="ket">Bruto <?= e(rupiah((float) $p['bruto'])) ?>, PPh 21 <?= e(rupiah((float) $p['pph21'])) ?></div><?php endif; ?>
    </td></tr>
    <tr><td class="l">Sumber Dana</td><td class="c">:</td><td class="v"><?= e($sumber) ?></td></tr>
  </table>

  <table class="ttd">
    <tr>
      <td style="width:50%"></td>
      <td><?= $belanja['jenis'] === 'barang' ? 'Penyedia,' : 'Penerima Uang,' ?><div class="ruang"></div>
        <div class="nama-p"><?= e($p['nama']) ?></div>
        <?php if (!empty($p['nip'])): ?><div class="nip">NIP. <?= e($p['nip']) ?></div><?php endif; ?>
      </td>
    </tr>
  </table>
  <div class="tgl"><?= e(trim($kota . ', ' . $tglPendek($belanja['tgl']), ', ')) ?></div>
  <table class="ttd" style="margin-top:0">
    <tr>
      <td style="width:50%">Mengetahui,<br>Kepala <?= $lembaga ?>
        <div class="ruang"><?php if ($stempel): ?><img class="stempel" src="<?= e($stempel) ?>" alt="Stempel"><?php endif; ?><?php if ($kepala['ttd']): ?><img class="sig" src="<?= e($kepala['ttd']) ?>" alt="TTD kepala"><?php endif; ?></div>
        <div class="nama"><?= e($kepala['nama'] ?: '................................') ?></div>
        <?php if ($kepala['nip']): ?><div class="nip">NIP. <?= e($kepala['nip']) ?></div><?php endif; ?>
      </td>
      <td><br>Bendahara <?= $lembaga ?>
        <div class="ruang"><?php if ($bendahara['ttd']): ?><img class="sig" src="<?= e($bendahara['ttd']) ?>" alt="TTD bendahara"><?php endif; ?></div>
        <div class="nama"><?= e($bendahara['nama'] ?: '................................') ?></div>
        <?php if ($bendahara['nip']): ?><div class="nip">NIP. <?= e($bendahara['nip']) ?></div><?php endif; ?>
      </td>
    </tr>
  </table>
  <div class="lunas">LUNAS DIBAYAR — <?= $tglLunas ? e(tanggal_indo($tglLunas)) : '.......................' ?></div>
</div>
<?php if ($i < count($receivers) - 1): ?><div class="page-break"></div><?php endif; ?>
<?php endforeach; ?>
</body>
</html>

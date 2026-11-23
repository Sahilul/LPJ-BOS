<?php
use App\Services\Laporan;

$cell = function (array $row, string $key, string $type): string {
    $v = $row[$key] ?? null;
    if ($v === null || $v === '') {
        return '';
    }
    return match ($type) {
        'money' => e(angka((float) $v)),
        'pct' => e(persen((float) $v)),
        'date' => e(tanggal_indo((string) $v, true)),
        default => e((string) $v),
    };
};
$alamat = implode(', ', array_filter([$sekolah['alamat'] ?? '', $sekolah['desa'] ?? '', $sekolah['kecamatan'] ?? '', $sekolah['kabupaten'] ?? '']));
$kopHtml = function () use ($kop, $sekolah, $alamat): void { ?>
  <?php if ($kop): ?>
    <img class="kop" src="<?= e($kop) ?>" alt="Kop surat">
  <?php else: ?>
    <div class="school"><strong><?= e($sekolah['nama'] ?? '') ?></strong><?php if (!empty($sekolah['npsn'])): ?> · NPSN <?= e($sekolah['npsn']) ?><?php endif; ?><br><?= e($alamat) ?></div>
  <?php endif; ?>
<?php };
$ttd = function (string $kiri = 'Mengetahui,<br>Kepala Sekolah', string $kanan = 'Bendahara BOS') use ($kepala, $bendahara, $stempel, $lokasiTanggal): void { ?>
  <table class="signatures">
    <tr><td></td><td><?= e($lokasiTanggal) ?></td></tr>
    <tr>
      <td><?= $kiri ?><div class="sig"><?php if ($kepala['ttd']): ?><img src="<?= e($kepala['ttd']) ?>" alt="Tanda tangan kepala sekolah"><?php endif; ?></div>
        <div class="name"><?= e($kepala['nama'] ?: '................................') ?></div>
        <?php if ($kepala['nip']): ?><div class="nip">NIP. <?= e($kepala['nip']) ?></div><?php endif; ?>
      </td>
      <td><?= $kanan ?><div class="sig"><?php if ($stempel): ?><img src="<?= e($stempel) ?>" alt="Stempel sekolah"><?php endif; ?><?php if ($bendahara['ttd']): ?><img src="<?= e($bendahara['ttd']) ?>" alt="Tanda tangan bendahara"><?php endif; ?></div>
        <div class="name"><?= e($bendahara['nama'] ?: '................................') ?></div>
        <?php if ($bendahara['nip']): ?><div class="nip">NIP. <?= e($bendahara['nip']) ?></div><?php endif; ?>
      </td>
    </tr>
  </table>
<?php };
$first = true;
$pageBreak = function () use (&$first): string {
    $html = $first ? '' : ' style="page-break-before: always"';
    $first = false;
    return $html;
};
?>
<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<style>
@page { margin: 14mm 12mm; }
body { font: 8pt "DejaVu Sans", sans-serif; color: #171717; line-height: 1.35; }
.kop { width: 100%; max-height: 22mm; object-fit: contain; margin-bottom: 4mm; }
.school { text-align: center; padding-bottom: 3mm; border-bottom: 2px solid #222; margin-bottom: 5mm; font-size: 9pt; }
h1 { font-size: 12pt; text-align: center; margin: 0 0 1mm; letter-spacing: .5px; }
h2 { font-size: 10pt; margin: 6mm 0 2mm; }
.sub { text-align: center; margin: 0 0 5mm; }
table.list { width: 100%; border-collapse: collapse; }
.list th, .list td { border: 1px solid #555; padding: 1.2mm 1.3mm; vertical-align: top; }
.list th { background: #eee; text-align: left; }
.list .num { text-align: right; white-space: nowrap; }
.list .nowrap { white-space: nowrap; }
.list thead { display: table-header-group; }
.list tr { page-break-inside: avoid; }
.list tr.group td, .list tr.foot td { font-weight: bold; background: #f4f4f4; }
.list tr.sub td { font-weight: bold; }
.list tr.info td { color: #777; font-style: italic; }
table.summary { margin-top: 4mm; border-collapse: collapse; }
.summary td { padding: .8mm 3mm .8mm 0; }
.summary td.num { text-align: right; font-weight: bold; }
.signatures { width: 100%; text-align: center; margin-top: 5mm; page-break-inside: avoid; }
.signatures td { width: 50%; vertical-align: top; }
.sig { height: 17mm; }
.sig img { max-width: 30mm; max-height: 16mm; }
.name { font-weight: bold; text-decoration: underline; }
.nip { font-size: 7.5pt; }
.ba p { margin: 0 0 3mm; text-align: justify; font-size: 9pt; }
.cover { text-align: center; padding-top: 30mm; }
.cover h1 { font-size: 20pt; margin-bottom: 4mm; }
.cover .big { font-size: 12pt; margin: 2mm 0; }
.meta td { padding: .6mm 3mm .6mm 0; vertical-align: top; }
.bukti { page-break-inside: avoid; margin-bottom: 6mm; }
.muted { color: #666; }
</style>
</head>
<body>

<?php if ($spj): ?>
  <div<?= $pageBreak() ?> class="cover">
    <?php $kopHtml(); ?>
    <h1>SURAT PERTANGGUNGJAWABAN (SPJ)</h1>
    <div class="big">Dana Bantuan Operasional Satuan Pendidikan (BOSP)</div>
    <div class="big"><strong><?= e($p['label']) ?></strong></div>
    <div class="big"><?= e($sekolah['nama'] ?? '') ?><?php if (!empty($sekolah['npsn'])): ?> · NPSN <?= e($sekolah['npsn']) ?><?php endif; ?></div>
    <table class="list" style="width:60%;margin:12mm auto 0">
      <thead><tr><th style="width:8mm">No.</th><th>Isi dokumen</th></tr></thead>
      <tbody>
        <?php $no = 0; foreach ($parts as $part): ?>
          <tr><td><?= ++$no ?></td><td><?= e(Laporan::JENIS[$part['jenis']][0]) ?></td></tr>
        <?php endforeach; ?>
        <?php if ($belanja): ?><tr><td><?= ++$no ?></td><td>Rincian bukti belanja (<?= count($belanja) ?> bukti)</td></tr><?php endif; ?>
      </tbody>
    </table>
  </div>
<?php endif; ?>

<?php foreach ($parts as $part): $d = $part['data']; ?>
  <div<?= $pageBreak() ?>>
    <?php $kopHtml(); ?>
    <h1><?= e($d['title']) ?></h1>
    <p class="sub"><?= e($p['label']) ?></p>
    <table class="list">
      <thead><tr>
        <?php if ($part['jenis'] !== 'kas'): ?><th style="width:6mm">No.</th><?php endif; ?>
        <?php foreach ($d['columns'] as [$label, $type]): ?>
          <th<?= in_array($type, ['money', 'pct'], true) ? ' class="num"' : '' ?>><?= e($label) ?></th>
        <?php endforeach; ?>
      </tr></thead>
      <tbody>
        <?php if (!$d['rows']): ?>
          <tr><td colspan="<?= count($d['columns']) + 1 ?>" class="muted">Tidak ada data pada periode ini.</td></tr>
        <?php endif; ?>
        <?php $no = 0; foreach ($d['rows'] as $row): $c = $row['_class'] ?? ''; ?>
          <tr<?= $c ? ' class="' . e($c) . '"' : '' ?>>
            <?php if ($part['jenis'] !== 'kas'): ?><td><?= $c === '' ? ++$no : '' ?></td><?php endif; ?>
            <?php foreach ($d['columns'] as $key => [$label, $type]): ?>
              <td class="<?= in_array($type, ['money', 'pct'], true) ? 'num' : (in_array($type, ['date'], true) || $key === 'no_bukti' ? 'nowrap' : '') ?>"><?= $cell($row, $key, $type) ?></td>
            <?php endforeach; ?>
          </tr>
        <?php endforeach; ?>
        <?php if ($d['foot']): ?>
          <tr class="foot">
            <?php if ($part['jenis'] !== 'kas'): ?><td></td><?php endif; ?>
            <?php foreach ($d['columns'] as $key => [$label, $type]): ?>
              <td class="<?= in_array($type, ['money', 'pct'], true) ? 'num' : '' ?>"><?= $cell($d['foot'], $key, $type) ?></td>
            <?php endforeach; ?>
          </tr>
        <?php endif; ?>
      </tbody>
    </table>

    <?php if ($d['summary']): ?>
      <table class="summary">
        <?php foreach ($d['summary'] as [$label, $value]): ?>
          <tr><td><?= e($label) ?></td><td>:</td><td class="num"><?= is_string($value) ? e($value) : e(rupiah((float) $value)) ?></td></tr>
        <?php endforeach; ?>
      </table>
    <?php endif; ?>

    <?php if ($part['jenis'] === 'kas'): $k = $d['kas']; ?>
      <?php $selisih = $k['saldo']['total'] - ($k['saldo']['tunai'] + $k['saldo']['bank']); ?>
      <div class="ba" style="margin-top:8mm">
        <h1 style="margin-bottom:5mm">BERITA ACARA PEMERIKSAAN KAS</h1>
        <p>Pada hari ini, <?= e(['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'][(int) date('w', strtotime($k['tgl']))]) ?> tanggal <?= e(tanggal_indo($k['tgl'])) ?>, yang bertanda tangan di bawah ini, Kepala <?= e($sekolah['nama'] ?? 'Sekolah') ?>, telah melakukan pemeriksaan kas terhadap Bendahara BOS, <?= e($bendahara['nama'] ?: '................................') ?>, dengan hasil sebagai berikut:</p>
        <table class="meta" style="margin:0 0 3mm 6mm">
          <tr><td>a.</td><td>Saldo menurut Buku Kas Umum</td><td>: <?= e(rupiah($k['saldo']['total'])) ?></td></tr>
          <tr><td>b.</td><td>Uang tunai di brankas</td><td>: <?= e(rupiah($k['saldo']['tunai'])) ?></td></tr>
          <tr><td>c.</td><td>Saldo rekening bank</td><td>: <?= e(rupiah($k['saldo']['bank'])) ?></td></tr>
          <tr><td>d.</td><td>Jumlah kas (b + c)</td><td>: <?= e(rupiah($k['saldo']['tunai'] + $k['saldo']['bank'])) ?></td></tr>
          <tr><td>e.</td><td>Selisih (a − d)</td><td>: <?= e(rupiah($selisih)) ?></td></tr>
          <tr><td>f.</td><td>Bukti belanja terakhir</td><td>: <?= e($k['bukti_terakhir']) ?></td></tr>
        </table>
        <p>Terbilang saldo kas: <em><?= e(ucfirst(trim(terbilang((float) $k['saldo']['total'])))) ?> rupiah</em>.</p>
        <p>Demikian berita acara ini dibuat dengan sebenarnya untuk dapat dipergunakan sebagaimana mestinya.</p>
      </div>
      <?php $ttd('Yang memeriksa,<br>Kepala Sekolah', 'Yang diperiksa,<br>Bendahara BOS'); ?>
    <?php else: ?>
      <?php $ttd(); ?>
    <?php endif; ?>
  </div>
<?php endforeach; ?>

<?php if ($belanja !== null): ?>
  <div<?= $pageBreak() ?>>
    <?php $kopHtml(); ?>
    <h1>RINCIAN BUKTI BELANJA</h1>
    <p class="sub"><?= e($p['label']) ?></p>
    <?php if (!$belanja): ?><p class="muted">Tidak ada belanja pada periode ini.</p><?php endif; ?>
    <?php foreach ($belanja as $b): ?>
      <div class="bukti">
        <table class="meta">
          <tr><td><strong><?= e($b['no_bukti']) ?></strong></td><td><?= e(tanggal_indo($b['tgl'])) ?></td><td><?= e($b['jenis_label']) ?> · <?= $b['metode'] === 'bank' ? 'Bank' : 'Tunai' ?></td><td><?= e($b['rekening_kode'] ?? '') ?></td></tr>
          <tr><td colspan="4"><?= e($b['uraian']) ?><?php if ($b['toko_nama']): ?> — <?= e($b['toko_nama']) ?><?php endif; ?></td></tr>
        </table>
        <?php if ($b['items']): ?>
          <table class="list">
            <thead><tr><th style="width:6mm">No.</th><th>Barang / jasa</th><th class="num">Volume</th><th>Satuan</th><th class="num">Harga</th><th class="num">Jumlah</th></tr></thead>
            <tbody>
              <?php foreach ($b['items'] as $i => $it): ?>
                <tr><td><?= $i + 1 ?></td><td><?= e($it['nama']) ?></td><td class="num"><?= e(angka((float) $it['volume'], 2)) ?></td><td><?= e($it['satuan']) ?></td><td class="num"><?= e(angka((float) $it['harga'])) ?></td><td class="num"><?= e(angka((float) $it['subtotal'])) ?></td></tr>
              <?php endforeach; ?>
            </tbody>
          </table>
        <?php endif; ?>
        <?php if ($b['penerima']): ?>
          <table class="list">
            <thead><tr><th style="width:6mm">No.</th><th>Penerima</th><th class="num">Volume</th><th class="num">Tarif</th><th class="num">Bruto</th><th class="num">PPh 21</th><th class="num">Diterima</th></tr></thead>
            <tbody>
              <?php foreach ($b['penerima'] as $i => $pn): ?>
                <tr><td><?= $i + 1 ?></td><td><?= e($pn['nama']) ?><?php if ($pn['nip']): ?> · <?= e($pn['nip']) ?><?php endif; ?></td><td class="num"><?= e(angka((float) $pn['volume'], 2)) ?> <?= e($pn['satuan']) ?></td><td class="num"><?= e(angka((float) $pn['tarif'])) ?></td><td class="num"><?= e(angka((float) $pn['bruto'])) ?></td><td class="num"><?= e(angka((float) $pn['pph21'])) ?></td><td class="num"><?= e(angka((float) $pn['netto'])) ?></td></tr>
              <?php endforeach; ?>
            </tbody>
          </table>
        <?php endif; ?>
        <table class="meta" style="margin-top:1mm">
          <tr><td>Total belanja: <strong><?= e(rupiah((float) $b['total'])) ?></strong></td>
            <?php foreach ($b['pajak'] as $pj): ?>
              <td><?= e(Laporan::jenisPajak($pj['jenis'])) ?>: <?= e(rupiah((float) $pj['nilai'])) ?><?= $pj['ntpn'] ? ' · NTPN ' . e($pj['ntpn']) : '' ?></td>
            <?php endforeach; ?>
          </tr>
        </table>
      </div>
    <?php endforeach; ?>
    <?php $ttd(); ?>
  </div>
<?php endif; ?>

</body>
</html>

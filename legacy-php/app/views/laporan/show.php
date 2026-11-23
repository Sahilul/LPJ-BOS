<?php
$q = 'periode=' . urlencode($p['kode']);
$cell = function (array $row, string $key, string $type): string {
    $v = $row[$key] ?? null;
    if ($v === null || $v === '') {
        return '';
    }
    return match ($type) {
        'money' => rupiah((float) $v),
        'pct' => e(persen((float) $v)),
        'date' => e(tanggal_indo((string) $v, true)),
        default => e((string) $v),
    };
};
$cls = fn (array $row) => match ($row['_class'] ?? '') {
    'group', 'sub' => ' class="group-row"',
    'info' => ' class="is-off"',
    default => '',
};
$tdClass = fn (string $key, string $type) => in_array($type, ['money', 'pct'], true) ? 'num' : (in_array($key, ['tgl', 'no_bukti', 'kode', 'rekening'], true) ? 'mono' : '');
?>
<div class="page-head">
  <div>
    <h1><?= e($nama) ?></h1>
    <p class="lead"><?= e($p['label']) ?> · <a href="/laporan?<?= $q ?>">Kembali ke daftar laporan</a></p>
  </div>
  <div class="actions">
    <a class="btn btn-ghost" href="/laporan/<?= e($jenis) ?>?<?= $q ?>&amp;format=xlsx"><?= icon('download', 16) ?>Excel</a>
    <a class="btn" href="/laporan/<?= e($jenis) ?>?<?= $q ?>&amp;format=pdf" target="_blank" rel="noopener"><?= icon('file', 16) ?>Cetak PDF</a>
  </div>
</div>

<?php if ($data['summary']): ?>
  <div class="grid grid-4" style="margin-bottom:18px">
    <?php foreach ($data['summary'] as [$label, $value]): ?>
      <div class="card stat"><span class="label"><?= e($label) ?></span><span class="value mono" style="display:block"><?= is_string($value) ? e($value) : rupiah((float) $value) ?></span></div>
    <?php endforeach; ?>
  </div>
<?php endif; ?>

<div class="card">
  <div class="table-wrap">
    <table class="table">
      <thead><tr>
        <?php foreach ($data['columns'] as [$label, $type]): ?>
          <th<?= in_array($type, ['money', 'pct'], true) ? ' class="num"' : '' ?>><?= e($label) ?></th>
        <?php endforeach; ?>
      </tr></thead>
      <tbody>
        <?php if (!$data['rows']): ?>
          <tr><td colspan="<?= count($data['columns']) ?>" class="muted">Tidak ada data pada periode ini.</td></tr>
        <?php endif; ?>
        <?php foreach ($data['rows'] as $row): ?>
          <tr<?= $cls($row) ?>>
            <?php foreach ($data['columns'] as $key => [$label, $type]): ?>
              <td class="<?= $tdClass($key, $type) ?>" data-label="<?= e($label) ?>"><?= $cell($row, $key, $type) ?></td>
            <?php endforeach; ?>
          </tr>
        <?php endforeach; ?>
        <?php if ($data['foot']): ?>
          <tr class="group-row">
            <?php foreach ($data['columns'] as $key => [$label, $type]): ?>
              <td class="<?= $tdClass($key, $type) ?>" data-label="<?= e($label) ?>"><strong><?= $cell($data['foot'], $key, $type) ?></strong></td>
            <?php endforeach; ?>
          </tr>
        <?php endif; ?>
      </tbody>
    </table>
  </div>
</div>

<?php if ($jenis === 'kas'): ?>
  <p class="muted small" style="margin-top:12px">Versi PDF menyertakan Berita Acara Pemeriksaan Kas dengan tanda tangan kepala sekolah dan bendahara.</p>
<?php endif; ?>

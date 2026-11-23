<?php
$fields = $res['fields'];
$toggle = !empty($res['toggle']);
$filtered = $q !== '' || array_filter(array_map(fn ($c) => $_GET[$c] ?? '', array_keys($res['filter'] ?? []))) || ($_GET['status'] ?? '') !== '';
$cell = function (string $k, array $row) use ($fields): string {
    $f = $fields[$k];
    $v = $row[$k] ?? '';
    return match ($f['type'] ?? 'text') {
        'money' => (float) $v > 0 ? '<span class="num">' . e(rupiah((float) $v)) . '</span>' : '<span class="muted">—</span>',
        'percent' => '<span class="num">' . e(angka((float) $v, 2)) . '%</span>',
        'bool' => $v ? '<span class="badge badge-accent">Ya</span>' : '<span class="muted">Tidak</span>',
        'select' => $v === '' ? '<span class="muted">—</span>' : e($f['options'][$v] ?? $v),
        default => $v === '' ? '<span class="muted">—</span>' : (!empty($f['mono']) || ($f['type'] ?? '') === 'digits' ? '<span class="mono">' . e($v) . '</span>' : e($v)),
    };
};
$label = fn (array $row) => $row['nama'] ?? $row['kode'];
?>
<div class="page-head">
  <div>
    <h1><?= e($res['title']) ?></h1>
    <p class="lead"><?= e($res['lead']) ?></p>
  </div>
  <div class="actions">
    <?php if (!empty($res['excel'])): ?>
      <a class="btn btn-ghost" href="/<?= $slug ?>/ekspor"><?= icon('download', 16) ?>Ekspor Excel</a>
    <?php endif; ?>
    <a class="btn" href="/<?= $slug ?>/baru"><?= icon('plus', 16) ?>Tambah <?= e($res['singular']) ?></a>
  </div>
</div>

<?php if (!empty($res['excel'])): ?>
  <details class="more card" style="margin-bottom:18px"<?= field_error('berkas') ? ' open' : '' ?>>
    <summary><?= icon('upload', 16) ?>Impor banyak <?= e($res['singular']) ?> sekaligus dari Excel</summary>
    <div class="card-body" style="padding-top:0">
      <ol class="muted small" style="margin:0 0 14px;padding-left:18px;line-height:1.8">
        <li>Unduh <a href="/<?= $slug ?>/template">template Excel</a>, isi satu baris per orang.</li>
        <li>Unggah kembali di sini. Baris dengan NIP/NIK yang sudah ada akan diperbarui, sisanya ditambahkan.</li>
      </ol>
      <form method="post" action="/<?= $slug ?>/impor" enctype="multipart/form-data" style="display:flex;gap:10px;flex-wrap:wrap;align-items:stretch">
        <?= csrf_field() ?>
        <label class="drop" style="flex:1;min-width:240px">
          <?= icon('file') ?>
          <span>Pilih berkas Excel<br><span class="name" data-empty="atau seret ke sini">atau seret ke sini</span></span>
          <input type="file" name="berkas" accept=".xlsx,.xls,.csv" required aria-label="Berkas Excel penerima">
        </label>
        <button class="btn" type="submit"><?= icon('upload', 16) ?>Impor data</button>
      </form>
      <?php if ($err = field_error('berkas')): ?><p class="error" style="margin-top:8px"><?= e($err) ?></p><?php endif; ?>
    </div>
  </details>
<?php endif; ?>

<div class="card">
  <form class="toolbar" method="get" action="/<?= $slug ?>" role="search">
    <label class="search">
      <span class="sr-only">Cari <?= e($res['singular']) ?></span>
      <?= icon('search', 16) ?>
      <input class="input" type="search" name="q" value="<?= e($q) ?>" placeholder="Cari <?= e(implode(', ', array_map(fn ($c) => strtolower($fields[$c]['label'] ?? $c), $res['search']))) ?>">
    </label>
    <?php foreach ($res['filter'] ?? [] as $col => $opts): ?>
      <label class="sr-only" for="flt-<?= $col ?>"><?= e($fields[$col]['label']) ?></label>
      <select class="input" id="flt-<?= $col ?>" name="<?= $col ?>" data-autosubmit style="width:auto">
        <option value="">Semua <?= e(strtolower($fields[$col]['label'])) ?></option>
        <?php foreach ($opts as $k => $v): ?><option value="<?= e($k) ?>"<?= ($_GET[$col] ?? '') === (string) $k ? ' selected' : '' ?>><?= e($v) ?></option><?php endforeach; ?>
      </select>
    <?php endforeach; ?>
    <?php if ($toggle): ?>
      <label class="sr-only" for="flt-status">Status</label>
      <select class="input" id="flt-status" name="status" data-autosubmit style="width:auto">
        <?php foreach (['' => 'Semua status', 'aktif' => 'Aktif', 'nonaktif' => 'Nonaktif'] as $k => $v): ?>
          <option value="<?= $k ?>"<?= ($_GET['status'] ?? '') === $k ? ' selected' : '' ?>><?= $v ?></option>
        <?php endforeach; ?>
      </select>
    <?php endif; ?>
    <button class="btn btn-ghost" type="submit">Cari</button>
    <?php if ($filtered): ?><a class="btn btn-ghost" href="/<?= $slug ?>"><?= icon('x', 16) ?>Hapus filter</a><?php endif; ?>
  </form>

  <?php if (!$rows): ?>
    <div class="empty">
      <span class="ic-wrap"><?= icon($res['icon'], 22) ?></span>
      <?php if ($filtered): ?>
        <h3>Tidak ada yang cocok</h3>
        <p>Coba kata kunci lain atau hapus filter.</p>
      <?php else: ?>
        <h3>Belum ada <?= e($res['singular']) ?></h3>
        <p>Data ini dipilih saat mencatat transaksi, jadi cukup diisi sekali.</p>
        <a class="btn" href="/<?= $slug ?>/baru"><?= icon('plus', 16) ?>Tambah <?= e($res['singular']) ?></a>
      <?php endif; ?>
    </div>
  <?php else: ?>
    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <?php foreach ($res['list'] as $k): ?>
              <th<?= in_array($fields[$k]['type'] ?? '', ['money', 'percent'], true) ? ' class="num"' : '' ?>><?= e(preg_replace('/ \(.*\)$/', '', $fields[$k]['label'])) ?></th>
            <?php endforeach; ?>
            <?php if ($toggle): ?><th>Status</th><?php endif; ?>
            <th><span class="sr-only">Aksi</span></th>
          </tr>
        </thead>
        <tbody>
          <?php foreach ($rows as $row): $off = $toggle && !$row['aktif']; ?>
            <tr id="row-<?= (int) $row['id'] ?>"<?= $off ? ' class="is-off"' : '' ?>>
              <?php foreach ($res['list'] as $i => $k): ?>
                <td<?= $i === 0 ? ' class="primary"' : (in_array($fields[$k]['type'] ?? '', ['money', 'percent'], true) ? ' class="num"' : '') ?>><?= $cell($k, $row) ?></td>
              <?php endforeach; ?>
              <?php if ($toggle): ?>
                <td><?= $off ? '<span class="badge badge-off">Nonaktif</span>' : '<span class="badge badge-ok">Aktif</span>' ?></td>
              <?php endif; ?>
              <td>
                <div class="row-actions">
                  <a class="btn btn-ghost btn-icon" href="/<?= $slug ?>/<?= (int) $row['id'] ?>/edit" aria-label="Ubah <?= e($label($row)) ?>" title="Ubah"><?= icon('edit', 16) ?></a>
                  <?php if ($toggle): ?>
                    <form method="post" action="/<?= $slug ?>/<?= (int) $row['id'] ?>/aktif">
                      <?= csrf_field() ?>
                      <button class="btn btn-ghost btn-icon" type="submit" aria-label="<?= $off ? 'Aktifkan' : 'Nonaktifkan' ?> <?= e($label($row)) ?>" title="<?= $off ? 'Aktifkan' : 'Nonaktifkan' ?>"><?= icon('power', 16) ?></button>
                    </form>
                  <?php endif; ?>
                  <form method="post" action="/<?= $slug ?>/<?= (int) $row['id'] ?>/hapus" data-confirm="Hapus “<?= e($label($row)) ?>”?">
                    <?= csrf_field() ?>
                    <button class="btn btn-danger btn-icon" type="submit" aria-label="Hapus <?= e($label($row)) ?>" title="Hapus"><?= icon('trash', 16) ?></button>
                  </form>
                </div>
              </td>
            </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <div class="table-foot">
      Menampilkan <span class="num"><?= count($rows) ?></span> dari <span class="num"><?= $total ?></span> <?= e($res['singular']) ?>
    </div>
  <?php endif; ?>
</div>

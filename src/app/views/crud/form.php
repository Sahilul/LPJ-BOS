<?php
$edit = !empty($row['id']);
?>
<div class="page-head">
  <div>
    <div class="crumb"><a href="/<?= $slug ?>"><?= e($res['title']) ?></a><span aria-hidden="true">/</span><span><?= $edit ? 'Ubah' : 'Tambah' ?></span></div>
    <h1><?= $edit ? e($row['nama'] ?? $row['kode']) : 'Tambah ' . e($res['singular']) ?></h1>
  </div>
</div>

<form class="card" method="post" action="/<?= $slug ?><?= $edit ? '/' . (int) $row['id'] : '' ?>" novalidate style="max-width:860px">
  <?= csrf_field() ?>
  <div class="card-body form-grid">
    <?php foreach ($res['fields'] as $k => $f):
        $type = $f['type'] ?? 'text';
        $extra = match ($type) {
            'digits' => 'inputmode="numeric"' . (isset($f['len']) ? ' maxlength="' . ($f['len'] + 4) . '"' : ''),
            'int' => 'inputmode="numeric"',
            default => '',
        };
        partial('partials/field', [
            'name' => $k,
            'label' => $f['label'],
            'value' => old($k, $row[$k] ?? ''),
            'type' => match ($type) { 'digits' => 'text', 'int' => 'number', default => $type },
            'options' => $f['options'] ?? [],
            'required' => !empty($f['required']),
            'hint' => $f['hint'] ?? (isset($f['len']) ? $f['len'] . ' angka.' : null),
            'mono' => !empty($f['mono']) || $type === 'digits',
            'span2' => ($f['span'] ?? 1) === 2,
            'extra' => $extra,
        ]);
    endforeach; ?>
  </div>
  <div class="form-foot">
    <a class="btn btn-ghost left" href="/<?= $slug ?>">Batal</a>
    <?php if (!$edit): ?>
      <button class="btn btn-ghost" type="submit" name="_lagi" value="1"><?= icon('plus', 16) ?>Simpan & tambah lagi</button>
    <?php endif; ?>
    <button class="btn" type="submit"><?= icon('check', 16) ?><?= $edit ? 'Simpan perubahan' : 'Simpan ' . e($res['singular']) ?></button>
  </div>
</form>

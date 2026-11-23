<?php
use App\Controllers\MutasiController;
$edit = !empty($row['id']);
$saldo = saldo_kas($row['tgl'] ?? tgl_default(), $edit ? (int) $row['id'] : null);
?>
<div class="page-head">
  <div>
    <div class="crumb"><a href="/mutasi">Mutasi kas</a><span aria-hidden="true">/</span><span><?= $edit ? 'Ubah' : 'Baru' ?></span></div>
    <h1><?= $edit ? 'Ubah mutasi kas' : 'Catat mutasi kas' ?></h1>
    <p class="lead">Saldo per <?= e(tanggal_indo($row['tgl'] ?? tgl_default())) ?>: tunai <strong class="mono"><?= rupiah($saldo['tunai']) ?></strong>, bank <strong class="mono"><?= rupiah($saldo['bank']) ?></strong>.</p>
  </div>
</div>

<form class="card" method="post" action="/mutasi<?= $edit ? '/' . (int) $row['id'] : '' ?>" novalidate style="max-width:860px" data-mutasi='<?= e(json_encode(MutasiController::URAIAN)) ?>'>
  <?= csrf_field() ?>
  <div class="card-body form-grid">
    <?php
    partial('partials/field', ['name' => 'jenis', 'label' => 'Jenis mutasi', 'type' => 'select', 'options' => MutasiController::JENIS, 'value' => old('jenis', $row['jenis'] ?? 'tarik'), 'required' => true, 'span2' => true]);
    partial('partials/field', ['name' => 'tgl', 'label' => 'Tanggal', 'type' => 'date', 'value' => old('tgl', $row['tgl'] ?? ''), 'required' => true]);
    partial('partials/field', ['name' => 'no_bukti', 'label' => 'Nomor bukti BKU', 'value' => !empty($row['id']) ? $row['no_bukti'] : '', 'mono' => true, 'extra' => 'readonly', 'hint' => 'Dibuat otomatis mengikuti urutan tanggal transaksi saat disimpan.']);
    partial('partials/field', ['name' => 'uraian', 'label' => 'Uraian', 'value' => old('uraian', $row['uraian'] ?? ''), 'span2' => true]);
    partial('partials/field', ['name' => 'jumlah', 'label' => 'Jumlah', 'type' => 'money', 'value' => old('jumlah', $row['jumlah'] ?? ''), 'required' => true]);
    ?>
  </div>
  <div class="form-foot">
    <a class="btn btn-ghost left" href="/mutasi">Batal</a>
    <button class="btn" type="submit"><?= icon('check', 16) ?><?= $edit ? 'Simpan perubahan' : 'Simpan mutasi' ?></button>
  </div>
</form>

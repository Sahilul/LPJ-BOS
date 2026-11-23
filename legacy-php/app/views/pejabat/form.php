<?php
use App\Controllers\PejabatController;

$edit = !empty($p['id']);
$f = fn (string $k) => old($k, $p[$k] ?? '');
?>
<div class="page-head">
  <div>
    <div class="crumb"><a href="/pejabat">Pejabat</a><span aria-hidden="true">/</span><span><?= $edit ? "Ubah" : "Tambah" ?></span></div>
    <h1><?= $edit ? 'Ubah pejabat' : 'Tambah pejabat' ?></h1>
    <p class="lead">Tulis nama lengkap beserta gelar persis seperti di dokumen resmi.</p>
  </div>
</div>

<form class="card" method="post" action="/pejabat<?= $edit ? '/' . (int) $p['id'] : '' ?>" enctype="multipart/form-data" novalidate style="max-width:860px">
  <?= csrf_field() ?>
  <div class="card-body form-grid">
    <?php partial('partials/field', ['name' => 'peran', 'label' => 'Peran', 'value' => $f('peran'), 'type' => 'select', 'options' => ['' => 'Pilih peran'] + PejabatController::PERAN, 'required' => true]) ?>
    <?php partial('partials/field', ['name' => 'pangkat', 'label' => 'Pangkat / golongan', 'value' => $f('pangkat'), 'hint' => 'Contoh: Pembina, IV/a. Kosongkan bila bukan PNS.']) ?>
    <?php partial('partials/field', ['name' => 'nama', 'label' => 'Nama lengkap', 'value' => $f('nama'), 'required' => true, 'span2' => true]) ?>
    <?php partial('partials/field', ['name' => 'nip', 'label' => 'NIP', 'value' => $f('nip'), 'mono' => true, 'span2' => true, 'hint' => '18 angka. Kosongkan bila tidak punya NIP.', 'extra' => 'inputmode="numeric" maxlength="21"']) ?>
    <?php partial('partials/field', ['name' => 'mulai', 'label' => 'Mulai menjabat', 'value' => $f('mulai'), 'type' => 'date', 'required' => true]) ?>
    <?php partial('partials/field', ['name' => 'selesai', 'label' => 'Selesai menjabat', 'value' => $f('selesai'), 'type' => 'date', 'hint' => 'Kosongkan bila masih menjabat.']) ?>

    <div class="fieldset-title">Tanda tangan</div>
    <div class="field">
      <div class="ttd" id="prev-ttd" style="height:130px">
        <?php if (!empty($p['ttd_path'])): ?>
          <img src="<?= e(upload_url($p['ttd_path'])) ?>" alt="Tanda tangan saat ini">
        <?php else: ?>
          <span class="placeholder"><?= icon('pen', 18) ?><br>Belum ada tanda tangan</span>
        <?php endif; ?>
      </div>
      <?php if (!empty($p['ttd_path'])): ?>
        <label class="check" style="margin-top:6px"><input type="checkbox" name="hapus_ttd" value="1"><span>Hapus tanda tangan ini</span></label>
      <?php endif; ?>
    </div>
    <div class="field">
      <label class="drop">
        <?= icon('upload') ?>
        <span><?= !empty($p['ttd_path']) ? 'Ganti tanda tangan' : 'Pilih gambar tanda tangan' ?><br><span class="name" data-empty="PNG atau JPG">PNG atau JPG</span></span>
        <input type="file" name="ttd" accept="image/png,image/jpeg" data-preview="#prev-ttd" aria-label="Unggah tanda tangan"<?= field_error('ttd') ? ' aria-invalid="true"' : '' ?>>
      </label>
      <?php if ($err = field_error('ttd')): ?><span class="error"><?= e($err) ?></span><?php else: ?>
      <span class="hint">Tanda tangan di kertas putih, difoto atau dipindai. PNG berlatar transparan paling rapi di kuitansi.</span><?php endif; ?>
    </div>
  </div>
  <div class="form-foot">
    <a class="btn btn-ghost left" href="/pejabat">Batal</a>
    <button class="btn" type="submit"><?= icon('check', 16) ?><?= $edit ? 'Simpan perubahan' : 'Simpan pejabat' ?></button>
  </div>
</form>

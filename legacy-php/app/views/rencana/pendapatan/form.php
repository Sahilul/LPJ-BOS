<?php
$edit = !empty($row['id']);
$bulanOptions = ['' => '— Belum ditentukan —'];
for ($m = 1; $m <= 12; $m++) {
    $bulanOptions[(string) $m] = bulan_indo($m);
}
?>
<div class="page-head">
  <div>
    <div class="crumb"><a href="/rencana/pendapatan">Rencana pendapatan</a><span aria-hidden="true">/</span><span><?= $edit ? 'Ubah target' : 'Tambah target' ?></span></div>
    <h1><?= $edit ? e($row['uraian']) : 'Tambah target pendapatan' ?></h1>
    <p class="lead">Masukkan nominal yang diperkirakan diterima. Saat dana benar-benar masuk, pasangkan penerimaan aktual ke target ini.</p>
  </div>
</div>

<form class="card" method="post" action="/rencana/pendapatan<?= $edit ? '/' . (int) $row['id'] : '' ?>" novalidate style="max-width:760px">
  <?= csrf_field() ?>
  <div class="card-body form-grid">
    <?php
    partial('partials/field', [
        'name' => 'tahun_tampil',
        'label' => 'Tahun anggaran',
        'value' => (string) ($row['tahun'] ?? tahun_aktif()),
        'mono' => true,
        'extra' => 'readonly',
        'hint' => $edit ? null : 'Mengikuti tahun anggaran yang sedang dibuka.',
    ]);
    partial('partials/field', [
        'name' => 'tahap',
        'label' => 'Sumber / tahap',
        'type' => 'select',
        'options' => \App\Controllers\RencanaPendapatanController::TAHAP,
        'value' => old('tahap', $row['tahap'] ?? ''),
        'required' => true,
    ]);
    partial('partials/field', [
        'name' => 'uraian',
        'label' => 'Nama rencana pendapatan',
        'value' => old('uraian', $row['uraian'] ?? ''),
        'required' => true,
        'span2' => true,
        'extra' => 'placeholder="Dana BOSP Tahap 1"',
    ]);
    partial('partials/field', [
        'name' => 'target_bulan',
        'label' => 'Perkiraan bulan diterima',
        'type' => 'select',
        'options' => $bulanOptions,
        'value' => old('target_bulan', $row['target_bulan'] ?? ''),
        'hint' => 'Opsional. Isi jika jadwal penerimaan sudah diketahui.',
    ]);
    partial('partials/field', [
        'name' => 'target_jumlah',
        'label' => 'Target pendapatan',
        'type' => 'money',
        'value' => old('target_jumlah', $row['target_jumlah'] ?? ''),
        'required' => true,
    ]);
    ?>
  </div>
  <div class="form-foot">
    <a class="btn btn-ghost left" href="/rencana/pendapatan">Batal</a>
    <button class="btn" type="submit"><?= icon('check', 16) ?><?= $edit ? 'Simpan perubahan' : 'Simpan target' ?></button>
  </div>
</form>

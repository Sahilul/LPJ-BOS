<?php
use App\Controllers\SekolahController;

$f = fn (string $k) => old($k, $s[$k] ?? '');
$imgHint = [
    'kop' => 'Gambar kop surat utuh (logo, nama, alamat, garis). Lebar disarankan ±2000 px.',
    'logo' => 'Logo sekolah atau pemda, PNG transparan lebih baik.',
    'stempel' => 'Stempel sekolah, PNG latar transparan agar menimpa tanda tangan dengan rapi.',
];
?>
<div class="page-head">
  <div>
    <h1>Profil sekolah</h1>
    <p class="lead">Identitas ini muncul di kuitansi, BKU dan semua laporan. Kop surat, logo dan stempel cukup diunggah sebagai gambar.</p>
  </div>
  <div class="actions"><a class="btn btn-ghost" href="#gambar"><?= icon('image', 16) ?>Kop & stempel</a></div>
</div>

<form class="card" method="post" action="/sekolah" novalidate>
  <?= csrf_field() ?>
  <div class="card-body form-grid">
    <div class="fieldset-title">Identitas</div>
    <?php partial('partials/field', ['name' => 'nama', 'label' => 'Nama sekolah', 'value' => $f('nama'), 'required' => true, 'span2' => true]) ?>
    <?php partial('partials/field', ['name' => 'npsn', 'label' => 'NPSN', 'value' => $f('npsn'), 'mono' => true, 'extra' => 'inputmode="numeric" maxlength="8"']) ?>
    <?php partial('partials/field', ['name' => 'jenjang', 'label' => 'Jenjang', 'value' => $f('jenjang'), 'type' => 'select', 'options' => ['' => '—', 'SD' => 'SD', 'SMP' => 'SMP', 'SMA' => 'SMA', 'SMK' => 'SMK', 'SLB' => 'SLB', 'PAUD' => 'PAUD/TK']]) ?>
    <?php partial('partials/field', ['name' => 'status', 'label' => 'Status', 'value' => $f('status'), 'type' => 'select', 'options' => ['Negeri' => 'Negeri', 'Swasta' => 'Swasta']]) ?>
    <?php partial('partials/field', ['name' => 'npwp', 'label' => 'NPWP bendahara/sekolah', 'value' => $f('npwp'), 'mono' => true]) ?>
    <?php partial('partials/field', ['name' => 'email', 'label' => 'Surel', 'value' => $f('email'), 'type' => 'email']) ?>

    <div class="fieldset-title">Alamat</div>
    <?php partial('partials/field', ['name' => 'alamat', 'label' => 'Jalan', 'value' => $f('alamat'), 'span2' => true]) ?>
    <?php partial('partials/field', ['name' => 'desa', 'label' => 'Desa / kelurahan', 'value' => $f('desa')]) ?>
    <?php partial('partials/field', ['name' => 'kecamatan', 'label' => 'Kecamatan', 'value' => $f('kecamatan')]) ?>
    <?php partial('partials/field', ['name' => 'kabupaten', 'label' => 'Kabupaten / kota', 'value' => $f('kabupaten'), 'hint' => 'Dipakai sebagai tempat tanda tangan, mis. “Sleman, 5 Januari 2026”.']) ?>
    <?php partial('partials/field', ['name' => 'provinsi', 'label' => 'Provinsi', 'value' => $f('provinsi')]) ?>
    <?php partial('partials/field', ['name' => 'kode_pos', 'label' => 'Kode pos', 'value' => $f('kode_pos'), 'mono' => true]) ?>
    <?php partial('partials/field', ['name' => 'telepon', 'label' => 'Telepon', 'value' => $f('telepon')]) ?>

    <div class="fieldset-title">Rekening BOS</div>
    <?php partial('partials/field', ['name' => 'bank_nama', 'label' => 'Bank', 'value' => $f('bank_nama')]) ?>
    <?php partial('partials/field', ['name' => 'bank_rekening', 'label' => 'Nomor rekening', 'value' => $f('bank_rekening'), 'mono' => true]) ?>
    <?php partial('partials/field', ['name' => 'bank_atas_nama', 'label' => 'Atas nama', 'value' => $f('bank_atas_nama'), 'span2' => true]) ?>
  </div>
  <div class="form-foot">
    <?php if (!empty($s['updated_at'])): ?><span class="left muted small">Terakhir disimpan <?= e(tanggal_indo(substr($s['updated_at'], 0, 10))) ?></span><?php endif; ?>
    <button class="btn" type="submit"><?= icon('check', 16) ?>Simpan profil</button>
  </div>
</form>

<section id="gambar" style="margin-top:34px;scroll-margin-top:24px" aria-labelledby="judul-gambar">
  <div class="page-head" style="margin-bottom:18px">
    <div>
      <h2 id="judul-gambar">Kop surat, logo & stempel</h2>
      <p class="lead">PNG atau JPG, paling besar <?= (int) (config('max_image_bytes') / 1048576) ?> MB. Gambar langsung tersimpan setelah dipilih.</p>
    </div>
  </div>
  <div class="img-grid">
    <?php foreach (SekolahController::IMAGES as $jenis => $cfg): $path = $s[$cfg['col']] ?? null; ?>
      <div class="card card-pad img-slot">
        <div style="display:flex;justify-content:space-between;align-items:center;gap:10px">
          <h3><?= e($cfg['label']) ?></h3>
          <?= $path ? '<span class="badge badge-ok">' . icon('check', 12) . 'Terpasang</span>' : '<span class="badge badge-off">Belum ada</span>' ?>
        </div>
        <div class="img-frame<?= $jenis === 'kop' ? ' wide' : '' ?>" id="prev-<?= $jenis ?>">
          <?php if ($path): ?>
            <img src="<?= e(upload_url($path)) ?>" alt="<?= e($cfg['label']) ?> sekolah">
          <?php else: ?>
            <span class="placeholder"><?= icon('image', 26) ?>Belum ada gambar</span>
          <?php endif; ?>
        </div>
        <p class="hint"><?= e($imgHint[$jenis]) ?></p>
        <?php if ($err = field_error('gambar_' . $jenis)): ?><span class="error"><?= e($err) ?></span><?php endif; ?>
        <form method="post" action="/sekolah/gambar/<?= $jenis ?>" enctype="multipart/form-data">
          <?= csrf_field() ?>
          <label class="drop">
            <?= icon('upload') ?>
            <span><?= $path ? 'Ganti gambar' : 'Pilih gambar' ?><br><span class="name" data-empty="atau seret ke sini">atau seret ke sini</span></span>
            <input type="file" name="gambar" accept="image/png,image/jpeg" data-preview="#prev-<?= $jenis ?>" data-autosubmit aria-label="Unggah <?= e(strtolower($cfg['label'])) ?>">
          </label>
          <noscript><button class="btn btn-sm" type="submit" style="margin-top:8px">Unggah</button></noscript>
        </form>
        <?php if ($path): ?>
          <form method="post" action="/sekolah/gambar/<?= $jenis ?>/hapus" data-confirm="Hapus <?= e(strtolower($cfg['label'])) ?>?">
            <?= csrf_field() ?>
            <button class="btn btn-danger btn-sm" type="submit"><?= icon('trash', 14) ?>Hapus <?= e(strtolower($cfg['label'])) ?></button>
          </form>
        <?php endif; ?>
      </div>
    <?php endforeach; ?>
  </div>
</section>

<h1>Masuk</h1>
<p class="lead">Pilih tahun anggaran yang mau dikerjakan, lalu masuk.</p>
<form method="post" action="/login" novalidate>
  <?= csrf_field() ?>
  <?php partial('partials/field', ['name' => 'tahun', 'label' => 'Tahun anggaran', 'type' => 'select', 'options' => $tahunOptions, 'value' => old('tahun', (string) tahun_default()), 'required' => true, 'mono' => true, 'hint' => 'Tahun lain bisa dibuka nanti dari menu Tahun anggaran.']) ?>
  <?php partial('partials/field', ['name' => 'username', 'label' => 'Nama pengguna', 'value' => old('username'), 'required' => true, 'extra' => 'autocomplete="username" autocapitalize="none" autofocus']) ?>
  <?php partial('partials/field', ['name' => 'password', 'label' => 'Kata sandi', 'type' => 'password', 'required' => true, 'extra' => 'autocomplete="current-password"']) ?>
  <button class="btn btn-block" type="submit">Masuk <?= icon('arrow', 16) ?></button>
</form>

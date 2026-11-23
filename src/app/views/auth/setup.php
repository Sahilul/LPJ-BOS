<h1>Siapkan aplikasi</h1>
<p class="lead">Sekali saja. Akun ini dipakai bendahara untuk masuk di komputer ini.</p>
<form method="post" action="/setup" novalidate>
  <?= csrf_field() ?>
  <?php partial('partials/field', ['name' => 'sekolah', 'label' => 'Nama sekolah', 'value' => old('sekolah'), 'required' => true, 'extra' => 'autocomplete="organization"']) ?>
  <?php partial('partials/field', ['name' => 'nama', 'label' => 'Nama Anda', 'value' => old('nama'), 'required' => true, 'extra' => 'autocomplete="name"']) ?>
  <?php partial('partials/field', ['name' => 'username', 'label' => 'Nama pengguna', 'value' => old('username'), 'required' => true, 'hint' => 'Huruf kecil dan angka, tanpa spasi.', 'extra' => 'autocomplete="username" autocapitalize="none"']) ?>
  <?php partial('partials/field', ['name' => 'password', 'label' => 'Kata sandi', 'type' => 'password', 'required' => true, 'hint' => 'Minimal 6 karakter.', 'extra' => 'autocomplete="new-password"']) ?>
  <?php partial('partials/field', ['name' => 'password2', 'label' => 'Ulangi kata sandi', 'type' => 'password', 'required' => true, 'extra' => 'autocomplete="new-password"']) ?>
  <button class="btn btn-block" type="submit">Buat akun & mulai <?= icon('arrow', 16) ?></button>
</form>

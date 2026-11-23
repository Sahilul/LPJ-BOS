<div class="page-head">
  <div>
    <h1>Akun</h1>
    <p class="lead">Nama dan kata sandi untuk masuk ke aplikasi ini.</p>
  </div>
</div>

<form class="card" method="post" action="/akun" novalidate style="max-width:760px">
  <?= csrf_field() ?>
  <div class="card-body form-grid">
    <?php partial('partials/field', ['name' => 'nama', 'label' => 'Nama', 'value' => old('nama', $user['nama']), 'required' => true]) ?>
    <?php partial('partials/field', ['name' => 'username', 'label' => 'Nama pengguna', 'value' => old('username', $user['username']), 'required' => true, 'extra' => 'autocomplete="username"']) ?>
    <div class="fieldset-title">Ganti kata sandi <span style="text-transform:none;letter-spacing:0">— kosongkan bila tidak diganti</span></div>
    <?php partial('partials/field', ['name' => 'password_lama', 'label' => 'Kata sandi sekarang', 'type' => 'password', 'span2' => true, 'extra' => 'autocomplete="current-password"']) ?>
    <?php partial('partials/field', ['name' => 'password', 'label' => 'Kata sandi baru', 'type' => 'password', 'hint' => 'Minimal 6 karakter.', 'extra' => 'autocomplete="new-password"']) ?>
    <?php partial('partials/field', ['name' => 'password2', 'label' => 'Ulangi kata sandi baru', 'type' => 'password', 'extra' => 'autocomplete="new-password"']) ?>
  </div>
  <div class="form-foot">
    <button class="btn" type="submit"><?= icon('check', 16) ?>Simpan akun</button>
  </div>
</form>

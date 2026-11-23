<?php $mb = fn (int $b) => angka($b / 1048576, 2) . ' MB'; ?>
<div class="page-head">
  <div>
    <h1>Cadangkan & pulihkan</h1>
    <p class="lead">Semua data tersimpan di komputer ini. Unduh cadangan secara rutin, misalnya setiap selesai tutup buku bulanan, dan simpan di flashdisk atau Google Drive.</p>
  </div>
</div>

<div class="grid grid-2">
  <section class="card card-pad" aria-labelledby="unduh">
    <h2 id="unduh" style="font-size:18px">Unduh cadangan</h2>
    <p class="muted small" style="margin:6px 0 18px">Satu berkas .zip berisi database dan semua gambar.</p>
    <dl class="kv-list">
      <dt>Database</dt><dd class="num"><?= $mb($dbSize) ?></dd>
      <dt>Gambar</dt><dd class="num"><?= angka($uploadCount) ?> berkas · <?= $mb($uploadSize) ?></dd>
    </dl>
    <form method="post" action="/backup/unduh" style="margin-top:22px">
      <?= csrf_field() ?>
      <button class="btn" type="submit"><?= icon('download', 16) ?>Unduh cadangan</button>
    </form>
  </section>

  <section class="card card-pad danger-zone" aria-labelledby="pulihkan">
    <h2 id="pulihkan" style="font-size:18px">Pulihkan dari cadangan</h2>
    <p class="muted small" style="margin:6px 0 18px">Semua data saat ini <strong>diganti</strong> dengan isi cadangan. Unduh cadangan terbaru dulu bila ragu.</p>
    <form method="post" action="/backup/pulihkan" enctype="multipart/form-data" data-confirm="Data saat ini akan diganti seluruhnya. Lanjutkan?" class="stack" style="--gap:14px">
      <?= csrf_field() ?>
      <label class="drop">
        <?= icon('upload') ?>
        <span>Pilih berkas cadangan<br><span class="name" data-empty="berkas .zip">berkas .zip</span></span>
        <input type="file" name="berkas" accept=".zip,application/zip" required aria-label="Berkas cadangan">
      </label>
      <div class="field">
        <label for="konfirmasi">Ketik <span class="mono">PULIHKAN</span> untuk mengonfirmasi</label>
        <input class="input mono" id="konfirmasi" name="konfirmasi" autocomplete="off" required pattern="PULIHKAN">
      </div>
      <button class="btn btn-danger" type="submit"><?= icon('upload', 16) ?>Pulihkan data</button>
    </form>
  </section>
</div>

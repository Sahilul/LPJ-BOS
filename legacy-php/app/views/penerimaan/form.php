<?php $edit = !empty($row['id']); ?>
<div class="page-head">
  <div>
    <div class="crumb"><a href="/penerimaan">Penerimaan dana</a><span aria-hidden="true">/</span><span><?= $edit ? 'Ubah' : 'Baru' ?></span></div>
    <h1><?= $edit ? 'Ubah penerimaan' : 'Catat penerimaan' ?></h1>
    <p class="lead">Isi sesuai tanggal dan nominal di rekening koran.</p>
  </div>
</div>

<form class="card" method="post" action="/penerimaan<?= $edit ? '/' . (int) $row['id'] : '' ?>" novalidate style="max-width:860px">
  <?= csrf_field() ?>
  <div class="card-body form-grid">
    <?php
    partial('partials/field', ['name' => 'tgl', 'label' => 'Tanggal diterima', 'type' => 'date', 'value' => old('tgl', $row['tgl'] ?? ''), 'required' => true]);
    partial('partials/field', ['name' => 'no_bukti', 'label' => 'Nomor bukti BKU', 'value' => !empty($row['id']) ? $row['no_bukti'] : '', 'mono' => true, 'extra' => 'readonly', 'hint' => 'Dibuat otomatis (BKM = Bukti Kas Masuk) mengikuti urutan tanggal penerimaan saat disimpan.']);
    ?>
    <div class="field span-2">
      <label for="f-rencana_pendapatan_id">Pasangkan dengan rencana pendapatan <span class="muted small">(opsional)</span></label>
      <?php $planValue = (string) old('rencana_pendapatan_id', $row['rencana_pendapatan_id'] ?? ''); ?>
      <select class="input" id="f-rencana_pendapatan_id" name="rencana_pendapatan_id" data-rencana-pendapatan<?= field_error('rencana_pendapatan_id') ? ' aria-invalid="true" aria-describedby="f-rencana_pendapatan_id-err"' : '' ?>>
        <option value="">— Belum dipasangkan —</option>
        <?php foreach ($rencanaPendapatan as $plan): ?>
          <option value="<?= (int) $plan['id'] ?>"
                  data-tahap="<?= e($plan['tahap']) ?>"
                  data-uraian="<?= e($plan['uraian']) ?>"
                  data-target="<?= (float) $plan['target_jumlah'] ?>"
                  data-realisasi="<?= (float) $plan['realisasi'] ?>"
                  <?= (string) $plan['id'] === $planValue ? 'selected' : '' ?>>
            <?= e(\App\Controllers\RencanaPendapatanController::TAHAP[$plan['tahap']] ?? $plan['tahap']) ?>
            · <?= e($plan['uraian']) ?> · target <?= rupiah((float) $plan['target_jumlah']) ?>
          </option>
        <?php endforeach; ?>
      </select>
      <?php if ($e = field_error('rencana_pendapatan_id')): ?>
        <span class="error" id="f-rencana_pendapatan_id-err"><?= e($e) ?></span>
      <?php elseif (!$rencanaPendapatan): ?>
        <span class="hint">Belum ada target pendapatan. <a href="/rencana/pendapatan/baru">Tambah rencana pendapatan</a> atau lanjutkan tanpa pasangan.</span>
      <?php else: ?>
        <span class="hint">Satu target dapat dipasangkan dengan beberapa penerimaan aktual.</span>
      <?php endif; ?>
    </div>
    <?php
    partial('partials/field', ['name' => 'uraian', 'label' => 'Uraian', 'value' => old('uraian', $row['uraian'] ?? ''), 'required' => true, 'span2' => true]);
    partial('partials/field', ['name' => 'tahap', 'label' => 'Tahap', 'type' => 'select', 'options' => \App\Controllers\PenerimaanController::TAHAP, 'value' => old('tahap', $row['tahap'] ?? ''), 'required' => true]);
    partial('partials/field', ['name' => 'jumlah', 'label' => 'Jumlah', 'type' => 'money', 'value' => old('jumlah', $row['jumlah'] ?? ''), 'required' => true]);
    ?>
  </div>
  <div class="form-foot">
    <a class="btn btn-ghost left" href="/penerimaan">Batal</a>
    <button class="btn" type="submit"><?= icon('check', 16) ?><?= $edit ? 'Simpan perubahan' : 'Simpan penerimaan' ?></button>
  </div>
</form>

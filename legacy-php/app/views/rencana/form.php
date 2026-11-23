<?php
$edit = !empty($row['id']);
$posted = !empty($GLOBALS['_old']);
$jadwal = array_map('intval', $posted ? (array) old('jadwal', []) : array_filter(explode(',', (string) ($row['jadwal_bulan'] ?? ''))));
$rekVal = (string) old('rekening_kode', $row['rekening_kode'] ?? '');
$jenis = (string) old('jenis', $row['jenis'] ?? 'barang');
$frekuensi = (int) old('frekuensi_bulan', $row['frekuensi_bulan'] ?? 1);
$items = $posted ? array_values((array) old('items', [])) : $row['items'];
$recipients = $posted ? array_values((array) old('penerima', [])) : $row['penerima'];
if (!$items) {
    $items = [['nama' => '', 'volume' => '1', 'satuan' => '', 'harga' => '']];
}
if (!$recipients) {
    $recipients = [['penerima_id' => '', 'volume' => '1', 'satuan' => '', 'tarif' => '']];
}
$fmtVol = fn ($v) => str_replace('.', ',', (string) (float) ($v === '' ? 1 : $v));
$fmtMoney = fn ($v) => is_numeric($v) ? ((float) $v > 0 ? angka((float) $v) : '') : (string) $v;
$penerimaOptions = function (string $selected) use ($penerimaList): string {
    $html = '<option value="">— Pilih penerima —</option>';
    foreach ($penerimaList as $p) {
        $html .= '<option value="' . (int) $p['id'] . '"' . ((string) $p['id'] === $selected ? ' selected' : '') . '>'
            . e($p['nama']) . ($p['nip'] !== '' ? ' · ' . e($p['nip']) : '') . '</option>';
    }
    return $html;
};
?>
<div class="page-head">
  <div>
    <div class="crumb"><a href="/rencana">Rencana belanja</a><span aria-hidden="true">/</span><span><?= $edit ? 'Ubah paket' : 'Tambah paket' ?></span></div>
    <h1><?= $edit ? e($row['uraian']) : 'Tambah paket rencana' ?></h1>
    <p class="lead">Satu paket dapat berisi banyak barang atau penerima. Atur jumlah per bulan dan frekuensi; BKU akan mengalikan rinciannya.</p>
  </div>
</div>

<form class="card" method="post" action="/rencana<?= $edit ? '/' . (int) $row['id'] : '' ?>" novalidate style="max-width:980px" data-rencana-paket>
  <?= csrf_field() ?>
  <div class="card-body form-grid">
    <?php
    partial('partials/field', ['name' => 'tahun_tampil', 'label' => 'Tahun anggaran', 'value' => (string) ($row['tahun'] ?? tahun_aktif()), 'mono' => true, 'extra' => 'readonly', 'hint' => empty($row['id']) ? 'Mengikuti tahun yang sedang dibuka.' : null]);
    partial('partials/field', ['name' => 'komponen_kode', 'label' => 'Komponen BOSP', 'type' => 'select', 'options' => ['' => '— Pilih komponen —'] + $komponen, 'value' => old('komponen_kode', $row['komponen_kode'] ?? ''), 'required' => true]);
    ?>
    <div class="field span-2">
      <label for="f-rekening_kode">Kode rekening</label>
      <select class="input" id="f-rekening_kode" name="rekening_kode" data-rekening>
        <option value="">— Tanpa kode rekening —</option>
        <?php foreach ($rekening as $rk): ?>
          <option value="<?= e($rk['kode']) ?>" data-jenis="<?= e($rk['jenis']) ?>"<?= $rk['kode'] === $rekVal ? ' selected' : '' ?>><?= e($rk['kode'] . ' · ' . $rk['nama']) ?></option>
        <?php endforeach; ?>
      </select>
      <span class="hint">Kelola daftar rekening di <a href="/rekening">Kode rekening</a>.</span>
    </div>
    <?php
    partial('partials/field', ['name' => 'kegiatan', 'label' => 'Kegiatan', 'value' => old('kegiatan', $row['kegiatan'] ?? ''), 'hint' => 'Opsional, sesuai RKAS.', 'span2' => true]);
    partial('partials/field', ['name' => 'uraian', 'label' => 'Nama paket', 'value' => old('uraian', $row['uraian'] ?? ''), 'required' => true, 'span2' => true, 'extra' => 'placeholder="Belanja ATK"']);
    partial('partials/field', ['name' => 'jenis', 'label' => 'Jenis paket', 'type' => 'select', 'options' => \App\Controllers\RencanaController::JENIS, 'value' => $jenis, 'required' => true, 'extra' => 'data-jenis-target']);
    partial('partials/field', ['name' => 'frekuensi_bulan', 'label' => 'Frekuensi (bulan)', 'type' => 'number', 'value' => (string) $frekuensi, 'required' => true, 'hint' => 'Contoh: 6 berarti paket dapat dipakai untuk enam bulan.', 'extra' => 'min="1" max="12" step="1" inputmode="numeric" data-frekuensi']);
    ?>

    <div class="field span-2" data-paket-only="barang">
      <label for="f-toko_id">Toko / penyedia bawaan</label>
      <select class="input" id="f-toko_id" name="toko_id">
        <option value="">— Pilih nanti di BKU —</option>
        <?php foreach ($toko as $t): ?>
          <option value="<?= (int) $t['id'] ?>"<?= (string) $t['id'] === (string) old('toko_id', $row['toko_id'] ?? '') ? ' selected' : '' ?>><?= e($t['nama']) ?></option>
        <?php endforeach; ?>
      </select>
      <span class="hint">Bisa diganti saat mencatat nota di BKU.</span>
    </div>

    <section class="span-2" data-paket-only="barang" aria-labelledby="h-rencana-items">
      <div class="row-between" style="margin-bottom:10px">
        <h2 id="h-rencana-items" style="margin:0;font-size:1rem">Barang dalam satu bulan</h2>
        <span class="muted small">Jumlah dan harga satuan per bulan</span>
      </div>
      <div class="rows-editor" data-package-rows="items">
        <table class="table">
          <thead><tr><th>Nama barang</th><th style="width:100px">Jumlah</th><th style="width:120px">Satuan</th><th style="width:180px">Harga satuan</th><th class="num" style="width:150px">Subtotal</th><th style="width:52px"><span class="sr-only">Hapus</span></th></tr></thead>
          <tbody>
            <?php foreach ($items as $i => $item): ?>
            <tr>
              <td data-label="Nama barang"><input class="input" name="items[<?= $i ?>][nama]" value="<?= e($item['nama'] ?? '') ?>" aria-label="Nama barang" placeholder="Buku tulis" data-item-nama></td>
              <td data-label="Jumlah"><input class="input mono" name="items[<?= $i ?>][volume]" value="<?= e($fmtVol($item['volume'] ?? 1)) ?>" inputmode="decimal" aria-label="Jumlah per bulan" data-vol autocomplete="off"></td>
              <td data-label="Satuan"><input class="input" name="items[<?= $i ?>][satuan]" value="<?= e($item['satuan'] ?? '') ?>" aria-label="Satuan" placeholder="buah"></td>
              <td data-label="Harga satuan"><input class="input mono" name="items[<?= $i ?>][harga]" value="<?= e($fmtMoney($item['harga'] ?? $item['harga_satuan'] ?? '')) ?>" inputmode="numeric" aria-label="Harga satuan" data-money data-harga autocomplete="off"></td>
              <td class="num mono" data-label="Subtotal" data-sub>Rp0</td>
              <td><button class="btn btn-ghost btn-icon" type="button" data-row-remove aria-label="Hapus barang"><?= icon('trash', 16) ?></button></td>
            </tr>
            <?php endforeach; ?>
          </tbody>
        </table>
        <template>
          <tr>
            <td data-label="Nama barang"><input class="input" name="items[__i__][nama]" aria-label="Nama barang" placeholder="Buku tulis" data-item-nama></td>
            <td data-label="Jumlah"><input class="input mono" name="items[__i__][volume]" value="1" inputmode="decimal" aria-label="Jumlah per bulan" data-vol autocomplete="off"></td>
            <td data-label="Satuan"><input class="input" name="items[__i__][satuan]" aria-label="Satuan" placeholder="buah"></td>
            <td data-label="Harga satuan"><input class="input mono" name="items[__i__][harga]" inputmode="numeric" aria-label="Harga satuan" data-money data-harga autocomplete="off"></td>
            <td class="num mono" data-label="Subtotal" data-sub>Rp0</td>
            <td><button class="btn btn-ghost btn-icon" type="button" data-row-remove aria-label="Hapus barang"><?= icon('trash', 16) ?></button></td>
          </tr>
        </template>
        <div class="rows-foot">
          <button class="btn btn-ghost" type="button" data-row-add><?= icon('plus', 16) ?>Tambah barang</button>
          <?php if ($e = field_error('items')): ?><span class="error"><?= e($e) ?></span><?php endif; ?>
        </div>
      </div>
    </section>

    <section class="span-2" data-paket-only="orang" aria-labelledby="h-rencana-penerima">
      <div class="row-between" style="margin-bottom:10px">
        <h2 id="h-rencana-penerima" style="margin:0;font-size:1rem">Penerima dalam satu periode</h2>
        <a class="btn btn-ghost" href="/penerima/baru" target="_blank" rel="noopener"><?= icon('plus', 16) ?>Penerima baru</a>
      </div>
      <div class="rows-editor" data-package-rows="penerima">
        <?php if (!$penerimaList): ?><p class="alert alert-warn">Database penerima masih kosong. <a href="/penerima/baru">Tambahkan penerima</a> dulu.</p><?php endif; ?>
        <table class="table">
          <thead><tr><th>Nama penerima</th><th style="width:100px">Volume</th><th style="width:120px">Satuan</th><th style="width:180px">Tarif</th><th class="num" style="width:150px">Subtotal</th><th style="width:52px"><span class="sr-only">Hapus</span></th></tr></thead>
          <tbody>
            <?php foreach ($recipients as $i => $person): ?>
            <tr>
              <td data-label="Nama penerima"><select class="input" name="penerima[<?= $i ?>][penerima_id]" aria-label="Penerima" data-package-person><?= $penerimaOptions((string) ($person['penerima_id'] ?? '')) ?></select></td>
              <td data-label="Volume"><input class="input mono" name="penerima[<?= $i ?>][volume]" value="<?= e($fmtVol($person['volume'] ?? 1)) ?>" inputmode="decimal" aria-label="Volume" data-vol autocomplete="off"></td>
              <td data-label="Satuan"><input class="input" name="penerima[<?= $i ?>][satuan]" value="<?= e($person['satuan'] ?? '') ?>" aria-label="Satuan" placeholder="OH / OJ"></td>
              <td data-label="Tarif"><input class="input mono" name="penerima[<?= $i ?>][tarif]" value="<?= e($fmtMoney($person['tarif'] ?? '')) ?>" inputmode="numeric" aria-label="Tarif" data-money data-harga autocomplete="off"></td>
              <td class="num mono" data-label="Subtotal" data-sub>Rp0</td>
              <td><button class="btn btn-ghost btn-icon" type="button" data-row-remove aria-label="Hapus penerima"><?= icon('trash', 16) ?></button></td>
            </tr>
            <?php endforeach; ?>
          </tbody>
        </table>
        <template>
          <tr>
            <td data-label="Nama penerima"><select class="input" name="penerima[__i__][penerima_id]" aria-label="Penerima" data-package-person><?= $penerimaOptions('') ?></select></td>
            <td data-label="Volume"><input class="input mono" name="penerima[__i__][volume]" value="1" inputmode="decimal" aria-label="Volume" data-vol autocomplete="off"></td>
            <td data-label="Satuan"><input class="input" name="penerima[__i__][satuan]" aria-label="Satuan" placeholder="OH / OJ"></td>
            <td data-label="Tarif"><input class="input mono" name="penerima[__i__][tarif]" inputmode="numeric" aria-label="Tarif" data-money data-harga autocomplete="off"></td>
            <td class="num mono" data-label="Subtotal" data-sub>Rp0</td>
            <td><button class="btn btn-ghost btn-icon" type="button" data-row-remove aria-label="Hapus penerima"><?= icon('trash', 16) ?></button></td>
          </tr>
        </template>
        <div class="rows-foot">
          <button class="btn btn-ghost" type="button" data-row-add><?= icon('plus', 16) ?>Tambah penerima</button>
          <?php if ($e = field_error('penerima')): ?><span class="error"><?= e($e) ?></span><?php endif; ?>
        </div>
      </div>
    </section>

    <div class="field span-2">
      <span class="label">Pagu keseluruhan</span>
      <output class="pagu-out mono" data-package-total aria-live="polite">Rp0</output>
      <span class="hint">Subtotal satu periode × frekuensi. Perubahan paket tidak mengubah rincian transaksi BKU yang sudah tersimpan.</span>
      <?php if ($edit && $terpakai > 0): ?><span class="hint">Sudah dibelanjakan <?= rupiah($terpakai) ?>; pagu paket tidak boleh lebih kecil dari realisasi.</span><?php endif; ?>
    </div>
    <fieldset class="span-2 months">
      <legend>Jadwal bulan <span class="muted small">(opsional)</span></legend>
      <div class="months-grid">
        <?php for ($m = 1; $m <= 12; $m++): ?>
          <label class="month-chip">
            <input type="checkbox" name="jadwal[]" value="<?= $m ?>"<?= in_array($m, $jadwal, true) ? ' checked' : '' ?>>
            <span><?= e(mb_substr(bulan_indo($m), 0, 3)) ?></span>
          </label>
        <?php endfor; ?>
      </div>
    </fieldset>
  </div>
  <div class="form-foot">
    <a class="btn btn-ghost left" href="/rencana">Batal</a>
    <?php if (!$edit): ?><button class="btn btn-ghost" type="submit" name="_lagi" value="1"><?= icon('plus', 16) ?>Simpan & tambah lagi</button><?php endif; ?>
    <button class="btn" type="submit"><?= icon('check', 16) ?><?= $edit ? 'Simpan perubahan' : 'Simpan paket' ?></button>
  </div>
</form>

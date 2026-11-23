<?php
$edit = !empty($row['id']);
$posted = !empty($GLOBALS['_old']);
$rid = (string) old('rencana_id', $row['rencana_id'] ?? '');
$sel = null;
foreach ($rencana as $r) {
    if ((string) $r['id'] === $rid) {
        $sel = $r;
    }
}
$jenis = $sel['jenis'] ?? 'barang';
$satDefault = $row['satuan_default'] ?? ($sel['satuan'] ?? '');
$periode = (int) old('jumlah_periode', $row['jumlah_periode'] ?? 1);

$items = $posted ? array_values((array) old('items', [])) : $row['items'];
if (!$items) {
    $items = [['nama' => '', 'volume' => 1, 'satuan' => '', 'harga' => '']];
}
$pens = $posted ? array_values((array) old('pen', [])) : $row['penerima'];
if (!$pens) {
    $pens = [['penerima_id' => '', 'volume' => 1, 'satuan' => $satDefault, 'tarif' => '']];
}
$pj = $row['pajak'] ?? [];
$chk = fn (string $k, string $jenisPajak) => $posted ? !empty(old($k)) : isset($pj[$jenisPajak]);

$fmtVol = fn ($v) => str_replace('.', ',', (string) (float) ($v === '' ? 1 : $v));
$fmtMoney = fn ($v) => is_numeric($v) ? ((float) $v > 0 ? angka((float) $v) : '') : (string) $v;

$penOptions = function (string $selected) use ($penerimaList): string {
    $h = '<option value="">— Pilih penerima —</option>';
    foreach ($penerimaList as $p) {
        $h .= '<option value="' . $p['id'] . '" data-tarif="' . $p['tarif'] . '" data-pph="' . $p['pph'] . '"'
            . ((string) $p['id'] === $selected ? ' selected' : '') . '>' . e($p['nama']) . ($p['ket'] !== '' ? ' — ' . e($p['ket']) : '') . '</option>';
    }
    return $h;
};
?>
<div class="page-head">
  <div>
    <div class="crumb"><a href="/bku">Buku kas umum</a><span aria-hidden="true">/</span><span><?= $edit ? e($row['no_bukti']) : 'Catat belanja' ?></span></div>
    <h1><?= $edit ? 'Ubah belanja' : 'Catat belanja' ?></h1>
    <p class="lead">Pilih paket dan jumlah bulan. Kuantitas dikalikan otomatis; sesuaikan harga atau rincian dengan nota aktual.</p>
  </div>
</div>

<form class="belanja-form" method="post" action="/belanja<?= $edit ? '/' . (int) $row['id'] : '' ?>" novalidate data-belanja<?= $edit ? ' data-edit="1"' : '' ?>
      data-saldo-tunai="<?= $saldo['tunai'] ?>" data-saldo-bank="<?= $saldo['bank'] ?>"
      data-tarif-ppn="<?= $tarif['PPN'] ?>" data-tarif-pph22="<?= $tarif['PPH22'] ?>" data-tarif-pph23="<?= $tarif['PPH23'] ?>" data-batas="<?= $batas ?>">
  <?= csrf_field() ?>

  <div class="belanja-layout">
    <div class="stack">
      <section class="card" aria-labelledby="h-umum">
        <div class="card-head"><h2 id="h-umum">Transaksi</h2></div>
        <div class="card-body form-grid">
          <div class="field span-2">
            <label for="f-rencana_id">Rencana belanja<span class="req" aria-hidden="true">*</span></label>
            <select class="input" id="f-rencana_id" name="rencana_id" required data-rencana<?= field_error('rencana_id') ? ' aria-invalid="true" aria-describedby="f-rencana_id-err"' : '' ?>>
              <option value="">— Pilih baris rencana —</option>
              <?php $lk = null; foreach ($rencana as $r):
                  if ($r['komponen_kode'] !== $lk): if ($lk !== null) echo '</optgroup>'; $lk = $r['komponen_kode']; ?>
                <optgroup label="Komponen <?= e($r['komponen_kode']) ?>">
              <?php endif; ?>
                <?php
                  $packageJson = json_encode($paket[$r['id']] ?? ['items' => [], 'penerima' => []], JSON_HEX_TAG | JSON_HEX_APOS | JSON_HEX_AMP | JSON_HEX_QUOT);
                ?>
                <option value="<?= (int) $r['id'] ?>" data-jenis="<?= e($r['jenis']) ?>" data-sisa="<?= (float) $r['sisa'] ?>" data-pagu="<?= (float) $r['pagu'] ?>" data-satuan="<?= e($paket[$r['id']]['satuan'] ?? '') ?>" data-uraian="<?= e($r['uraian']) ?>" data-frekuensi="<?= (int) $r['frekuensi_bulan'] ?>" data-periode-sisa="<?= max(0, (int) $r['periode_sisa']) ?>" data-toko="<?= (int) ($r['toko_id'] ?? 0) ?>" data-paket="<?= e($packageJson ?: '{}') ?>"<?= (string) $r['id'] === $rid ? ' selected' : '' ?><?= ($r['sisa'] <= 0 || $r['periode_sisa'] <= 0) && (string) $r['id'] !== $rid ? ' disabled' : '' ?>>
                  <?= e($r['uraian']) ?> · sisa <?= rupiah((float) $r['sisa']) ?>
                </option>
              <?php endforeach; if ($lk !== null) echo '</optgroup>'; ?>
            </select>
            <?php if ($e = field_error('rencana_id')): ?><span class="error" id="f-rencana_id-err"><?= e($e) ?></span>
            <?php elseif (!$rencana): ?><span class="hint">Belum ada rencana untuk tahun ini. <a href="/rencana/baru">Tambah rencana belanja</a> dulu.</span><?php endif; ?>
          </div>
          <?php
          partial('partials/field', ['name' => 'tgl', 'label' => 'Tanggal', 'type' => 'date', 'value' => old('tgl', $row['tgl'] ?? ''), 'required' => true]);
          partial('partials/field', ['name' => 'no_bukti', 'label' => 'Nomor bukti BKU', 'value' => !empty($row['id']) ? $row['no_bukti'] : '', 'mono' => true, 'hint' => 'Dibuat otomatis (BKK = Bukti Kas Keluar) mengikuti urutan tanggal belanja saat disimpan.', 'extra' => 'readonly']);
          ?>
          <div class="field">
            <label for="f-jumlah_periode">Jumlah periode (bulan)<span class="req" aria-hidden="true">*</span></label>
            <input class="input mono" id="f-jumlah_periode" type="number" name="jumlah_periode" value="<?= $periode ?>" min="1" max="12" step="1" required inputmode="numeric" data-periode>
            <span class="hint" data-periode-hint><?= $sel ? 'Maksimal ' . max(0, (int) $sel['periode_sisa']) . ' bulan tersisa.' : 'Pilih jumlah bulan yang dibelanjakan dalam nota ini.' ?></span>
            <?php if ($e = field_error('jumlah_periode')): ?><span class="error"><?= e($e) ?></span><?php endif; ?>
          </div>
          <?php
          partial('partials/field', ['name' => 'uraian', 'label' => 'Uraian', 'value' => old('uraian', $row['uraian'] ?? ''), 'required' => true, 'span2' => true, 'extra' => 'data-uraian placeholder="Belanja ATK bulan Januari"']);
          partial('partials/field', ['name' => 'metode', 'label' => 'Dibayar dari', 'type' => 'select', 'options' => \App\Controllers\BelanjaController::METODE, 'value' => old('metode', $row['metode'] ?? 'tunai'), 'extra' => 'data-metode']);
          ?>
          <div class="field" data-only="barang">
            <label for="f-toko_id">Toko / penyedia</label>
            <select class="input" id="f-toko_id" name="toko_id" data-toko>
              <option value="" data-pkp="0">— Tanpa toko —</option>
              <?php $tv = (string) old('toko_id', $row['toko_id'] ?? ''); foreach ($toko as $t): ?>
                <option value="<?= (int) $t['id'] ?>" data-pkp="<?= (int) $t['pkp'] ?>"<?= (string) $t['id'] === $tv ? ' selected' : '' ?>><?= e($t['nama']) ?><?= $t['pkp'] ? ' (PKP)' : '' ?></option>
              <?php endforeach; ?>
            </select>
          </div>
        </div>
      </section>

      <section class="card" data-only="barang" aria-labelledby="h-item">
        <div class="card-head"><h2 id="h-item">Rincian barang / jasa</h2></div>
        <div class="rows-editor" data-rows="items">
          <table class="table">
            <thead><tr><th>Nama barang</th><th style="width:96px">Volume</th><th style="width:110px">Satuan</th><th style="width:170px">Harga satuan</th><th class="num" style="width:140px">Jumlah</th><th style="width:52px"><span class="sr-only">Hapus</span></th></tr></thead>
            <tbody>
              <?php foreach ($items as $i => $it): ?>
              <tr>
                <td data-label="Nama barang"><input class="input" name="items[<?= $i ?>][nama]" value="<?= e($it['nama'] ?? '') ?>" aria-label="Nama barang" placeholder="Kertas HVS A4"></td>
                <td data-label="Volume"><input class="input mono" name="items[<?= $i ?>][volume]" value="<?= e($fmtVol($it['volume'] ?? 1)) ?>"<?= isset($it['package_volume']) ? ' data-package-volume="' . e($fmtVol($it['package_volume'])) . '"' : '' ?> inputmode="decimal" aria-label="Volume" data-vol autocomplete="off"></td>
                <td data-label="Satuan"><input class="input" name="items[<?= $i ?>][satuan]" value="<?= e($it['satuan'] ?? '') ?>" aria-label="Satuan" placeholder="rim"></td>
                <td data-label="Harga satuan"><input class="input mono" name="items[<?= $i ?>][harga]" value="<?= e($fmtMoney($it['harga'] ?? '')) ?>" inputmode="numeric" aria-label="Harga satuan" data-money data-harga autocomplete="off"></td>
                <td class="num mono" data-label="Jumlah" data-sub>0</td>
                <td><button class="btn btn-ghost btn-icon" type="button" data-row-remove aria-label="Hapus baris"><?= icon('trash', 16) ?></button></td>
              </tr>
              <?php endforeach; ?>
            </tbody>
          </table>
          <template>
            <tr>
              <td data-label="Nama barang"><input class="input" name="items[__i__][nama]" aria-label="Nama barang"></td>
              <td data-label="Volume"><input class="input mono" name="items[__i__][volume]" value="1" inputmode="decimal" aria-label="Volume" data-vol autocomplete="off"></td>
              <td data-label="Satuan"><input class="input" name="items[__i__][satuan]" aria-label="Satuan"></td>
              <td data-label="Harga satuan"><input class="input mono" name="items[__i__][harga]" inputmode="numeric" aria-label="Harga satuan" data-money data-harga autocomplete="off"></td>
              <td class="num mono" data-label="Jumlah" data-sub>0</td>
              <td><button class="btn btn-ghost btn-icon" type="button" data-row-remove aria-label="Hapus baris"><?= icon('trash', 16) ?></button></td>
            </tr>
          </template>
          <div class="rows-foot">
            <button class="btn btn-ghost" type="button" data-row-add><?= icon('plus', 16) ?>Tambah barang</button>
            <?php if ($e = field_error('items')): ?><span class="error"><?= e($e) ?></span><?php endif; ?>
          </div>
        </div>
      </section>

      <section class="card" data-only="barang" aria-labelledby="h-pajak">
        <div class="card-head"><h2 id="h-pajak">Pajak dipungut</h2></div>
        <div class="card-body">
          <p class="muted small" style="margin:0 0 12px" data-pajak-saran>PPN dan PPh 22 berlaku bila toko PKP dan nilai transaksi di atas <?= rupiah($batas) ?>.</p>
          <div class="tax-list">
            <label class="check"><input type="checkbox" name="pajak_ppn" value="1" data-pajak="ppn"<?= $chk('pajak_ppn', 'PPN') ? ' checked' : '' ?>><span>PPN <?= persen($tarif['PPN']) ?></span><output class="mono" data-pajak-out="ppn">Rp0</output></label>
            <label class="check"><input type="checkbox" name="pajak_pph22" value="1" data-pajak="pph22"<?= $chk('pajak_pph22', 'PPH22') ? ' checked' : '' ?>><span>PPh 22 <?= persen($tarif['PPH22']) ?></span><output class="mono" data-pajak-out="pph22">Rp0</output></label>
            <label class="check"><input type="checkbox" name="pajak_pph23" value="1" data-pajak="pph23"<?= $chk('pajak_pph23', 'PPH23') ? ' checked' : '' ?>><span>PPh 23 <?= persen($tarif['PPH23']) ?> (jasa)</span><output class="mono" data-pajak-out="pph23">Rp0</output></label>
          </div>
        </div>
      </section>

      <section class="card" data-only="orang" aria-labelledby="h-pen">
        <div class="card-head">
          <h2 id="h-pen">Daftar penerima</h2>
          <a class="btn btn-ghost" href="/penerima/baru" target="_blank" rel="noopener"><?= icon('plus', 16) ?>Penerima baru</a>
        </div>
        <?php if (!$penerimaList): ?>
          <div class="card-body"><p class="alert alert-warn" style="margin:0">Database penerima masih kosong. <a href="/penerima/baru">Tambahkan penerima</a> atau impor dari Excel dulu.</p></div>
        <?php endif; ?>
        <div class="rows-editor" data-rows="pen">
          <table class="table">
            <thead><tr><th>Penerima</th><th style="width:88px">Volume</th><th style="width:100px">Satuan</th><th style="width:160px">Tarif</th><th class="num" style="width:120px">Bruto</th><th class="num" style="width:110px" data-only="honor">PPh 21</th><th class="num" style="width:120px">Diterima</th><th style="width:52px"><span class="sr-only">Hapus</span></th></tr></thead>
            <tbody>
              <?php foreach ($pens as $i => $pn): ?>
              <tr>
                <td data-label="Penerima"><select class="input" name="pen[<?= $i ?>][penerima_id]" aria-label="Penerima" data-pen><?= $penOptions((string) ($pn['penerima_id'] ?? '')) ?></select></td>
                <td data-label="Volume"><input class="input mono" name="pen[<?= $i ?>][volume]" value="<?= e($fmtVol($pn['volume'] ?? 1)) ?>"<?= isset($pn['package_volume']) ? ' data-package-volume="' . e($fmtVol($pn['package_volume'])) . '"' : '' ?> inputmode="decimal" aria-label="Volume" data-vol autocomplete="off"></td>
                <td data-label="Satuan"><input class="input" name="pen[<?= $i ?>][satuan]" value="<?= e($pn['satuan'] ?? $satDefault) ?>" aria-label="Satuan" placeholder="OJ, OK, OH" data-satuan></td>
                <td data-label="Tarif"><input class="input mono" name="pen[<?= $i ?>][tarif]" value="<?= e($fmtMoney($pn['tarif'] ?? '')) ?>" inputmode="numeric" aria-label="Tarif" data-money data-harga autocomplete="off"></td>
                <td class="num mono" data-label="Bruto" data-sub>0</td>
                <td class="num mono" data-label="PPh 21" data-only="honor" data-pph>0</td>
                <td class="num mono" data-label="Diterima" data-netto>0</td>
                <td><button class="btn btn-ghost btn-icon" type="button" data-row-remove aria-label="Hapus penerima"><?= icon('trash', 16) ?></button></td>
              </tr>
              <?php endforeach; ?>
            </tbody>
          </table>
          <template>
            <tr>
              <td data-label="Penerima"><select class="input" name="pen[__i__][penerima_id]" aria-label="Penerima" data-pen><?= $penOptions('') ?></select></td>
              <td data-label="Volume"><input class="input mono" name="pen[__i__][volume]" value="1" inputmode="decimal" aria-label="Volume" data-vol autocomplete="off"></td>
              <td data-label="Satuan"><input class="input" name="pen[__i__][satuan]" aria-label="Satuan" data-satuan></td>
              <td data-label="Tarif"><input class="input mono" name="pen[__i__][tarif]" inputmode="numeric" aria-label="Tarif" data-money data-harga autocomplete="off"></td>
              <td class="num mono" data-label="Bruto" data-sub>0</td>
              <td class="num mono" data-label="PPh 21" data-only="honor" data-pph>0</td>
              <td class="num mono" data-label="Diterima" data-netto>0</td>
              <td><button class="btn btn-ghost btn-icon" type="button" data-row-remove aria-label="Hapus penerima"><?= icon('trash', 16) ?></button></td>
            </tr>
          </template>
          <div class="rows-foot">
            <button class="btn btn-ghost" type="button" data-row-add><?= icon('plus', 16) ?>Tambah penerima</button>
            <?php if ($e = field_error('pen')): ?><span class="error"><?= e($e) ?></span><?php endif; ?>
          </div>
        </div>
      </section>
    </div>

    <aside class="belanja-side">
      <div class="card summary" aria-live="polite">
        <div class="kv"><span>Sisa pagu</span><strong class="mono" data-out="sisa">—</strong></div>
        <div class="kv"><span>Transaksi ini</span><strong class="mono" data-out="total">Rp0</strong></div>
        <div class="progress summary-bar"><span data-out="bar" style="width:0%"></span></div>
        <div class="kv"><span>Sisa setelah transaksi</span><strong class="mono" data-out="after">—</strong></div>
        <div class="kv" data-only="pajak"><span>Pajak dipungut</span><strong class="mono" data-out="pajak">Rp0</strong></div>
        <div class="kv"><span>Saldo <span data-out="metode">kas tunai</span></span><strong class="mono" data-out="saldo">—</strong></div>
        <p class="summary-msg" data-out="msg" hidden></p>
        <?php if ($e = field_error('metode')): ?><p class="summary-msg is-bad"><?= e($e) ?></p><?php endif; ?>
      </div>
      <div class="side-actions">
        <button class="btn" type="submit" data-submit><?= icon('check', 16) ?><?= $edit ? 'Simpan perubahan' : 'Simpan belanja' ?></button>
        <?php if (!$edit): ?><button class="btn btn-ghost" type="submit" name="_lagi" value="1"><?= icon('plus', 16) ?>Simpan & catat lagi</button><?php endif; ?>
        <a class="btn btn-ghost" href="/bku">Batal</a>
      </div>
    </aside>
  </div>
</form>

<?php if ($edit): ?>
<section class="card" id="lampiran" style="margin-top:18px">
  <div class="card-head"><h2>Lampiran transaksi</h2></div>
  <div class="card-body stack" style="--gap:14px">
    <p class="muted small" style="margin:0">Unggah nota, BAST, atau dokumen pendukung dalam format PDF, JPG, atau PNG (maksimal 2 MB per berkas).</p>
    <?php if ($lampiran): ?>
      <div class="stack" style="--gap:8px">
        <?php foreach ($lampiran as $file): ?>
          <div class="row-between">
            <div>
              <span class="badge badge-off"><?= e(['nota' => 'Nota', 'bast' => 'BAST', 'lainnya' => 'Lainnya'][$file['jenis']] ?? 'Lampiran') ?></span>
              <a href="/file/<?= rawurlencode($file['path']) ?>" target="_blank" rel="noopener"><?= e($file['nama_asli']) ?></a>
            </div>
            <form method="post" action="/belanja/<?= (int) $row['id'] ?>/lampiran/<?= (int) $file['id'] ?>/hapus" data-confirm="Hapus lampiran ini?">
              <?= csrf_field() ?>
              <button class="btn btn-ghost btn-icon" type="submit" aria-label="Hapus lampiran <?= e($file['nama_asli']) ?>"><?= icon('trash', 16) ?></button>
            </form>
          </div>
        <?php endforeach; ?>
      </div>
    <?php endif; ?>
    <form class="form-grid" method="post" action="/belanja/<?= (int) $row['id'] ?>/lampiran" enctype="multipart/form-data">
      <?= csrf_field() ?>
      <div class="field">
        <label for="lampiran-jenis">Jenis dokumen</label>
        <select class="input" id="lampiran-jenis" name="jenis">
          <option value="nota">Nota</option>
          <option value="bast">BAST</option>
          <option value="lainnya">Dokumen lain</option>
        </select>
      </div>
      <div class="field">
        <label for="lampiran-berkas">Berkas</label>
        <input class="input" id="lampiran-berkas" type="file" name="berkas" accept="application/pdf,image/jpeg,image/png" required>
      </div>
      <div class="form-foot" style="grid-column:1/-1;padding:0">
        <button class="btn btn-ghost" type="submit"><?= icon('upload', 16) ?>Unggah lampiran</button>
      </div>
    </form>
  </div>
</section>
<form method="post" action="/belanja/<?= (int) $row['id'] ?>/hapus" data-confirm="Hapus belanja <?= e($row['no_bukti']) ?>? Pagunya akan kembali ke rencana." class="danger-zone">
  <?= csrf_field() ?>
  <button class="btn btn-danger" type="submit"><?= icon('trash', 16) ?>Hapus transaksi ini</button>
</form>
<?php endif; ?>

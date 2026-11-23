<div class="page-head">
  <div>
    <h1>Tahun anggaran</h1>
    <p class="lead">Setiap tahun punya rencana, penerimaan dan BKU sendiri. Tahun default terpilih otomatis di halaman masuk; tahun lain tetap bisa dibuka kapan saja.</p>
  </div>
</div>

<div class="grid grid-tahun">
  <div class="card">
    <div class="table-wrap">
      <table class="table">
        <thead><tr><th>Tahun</th><th>Status</th><th class="num">Pagu rencana</th><th>Isi data</th><th><span class="sr-only">Aksi</span></th></tr></thead>
        <tbody>
          <?php foreach ($rows as $r): $t = (int) $r['tahun']; $isi = $r['n_rencana'] + $r['n_belanja'] + $r['n_penerimaan'] + $r['n_mutasi']; ?>
          <tr id="tahun-<?= $t ?>">
            <td class="primary mono" style="font-size:16px"><?= $t ?><?php if ($r['keterangan'] !== ''): ?><span class="muted small" style="display:block;font-family:var(--font)"><?= e($r['keterangan']) ?></span><?php endif; ?></td>
            <td>
              <span class="tags">
                <?php if ($r['is_default']): ?><span class="badge badge-accent">Default</span><?php endif; ?>
                <?php if ($t === $aktif): ?><span class="badge badge-ok"><?= icon('check', 12) ?>Sedang dibuka</span><?php endif; ?>
                <?php if (!$r['is_default'] && $t !== $aktif): ?><span class="muted small">—</span><?php endif; ?>
              </span>
            </td>
            <td class="num"><?= rupiah((float) $r['pagu']) ?></td>
            <td class="small muted"><?= $isi ? (int) $r['n_rencana'] . ' rencana · ' . (int) $r['n_belanja'] . ' belanja · ' . (int) $r['n_penerimaan'] . ' penerimaan' : 'Masih kosong' ?></td>
            <td>
              <div class="row-actions">
                <?php if ($t !== $aktif): ?>
                <form method="post" action="/tahun/aktif">
                  <?= csrf_field() ?><input type="hidden" name="tahun" value="<?= $t ?>">
                  <button class="btn btn-ghost btn-sm" type="submit">Buka <?= $t ?></button>
                </form>
                <?php endif; ?>
                <?php if (!$r['is_default']): ?>
                <form method="post" action="/tahun/<?= $t ?>/default">
                  <?= csrf_field() ?>
                  <button class="btn btn-ghost btn-sm" type="submit">Jadikan default</button>
                </form>
                <?php if (!$isi): ?>
                <form method="post" action="/tahun/<?= $t ?>/hapus" data-confirm="Hapus tahun anggaran <?= $t ?>?">
                  <?= csrf_field() ?>
                  <button class="btn btn-danger btn-icon" type="submit" aria-label="Hapus tahun <?= $t ?>"><?= icon('trash', 16) ?></button>
                </form>
                <?php endif; ?>
                <?php endif; ?>
              </div>
            </td>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <div class="table-foot"><?= count($rows) ?> tahun terdaftar · tahun yang sudah berisi data tidak bisa dihapus</div>
  </div>

  <form class="card card-pad stack" method="post" action="/tahun" novalidate style="--gap:16px;align-self:start">
    <?= csrf_field() ?>
    <h2 style="font-size:18px">Tambah tahun</h2>
    <?php partial('partials/field', ['name' => 'tahun', 'label' => 'Tahun', 'type' => 'number', 'value' => old('tahun', (string) $saran), 'required' => true, 'mono' => true, 'extra' => 'min="2000" max="2100" inputmode="numeric"']); ?>
    <?php partial('partials/field', ['name' => 'keterangan', 'label' => 'Catatan', 'value' => old('keterangan'), 'hint' => 'Opsional, mis. “BOSP Reguler”.']); ?>
    <?php partial('partials/field', ['name' => 'jadikan_default', 'label' => 'Jadikan default saat masuk', 'type' => 'bool', 'value' => old('jadikan_default', '1')]); ?>
    <?php partial('partials/field', ['name' => 'pakai', 'label' => 'Langsung buka tahun ini', 'type' => 'bool', 'value' => old('pakai', '1')]); ?>
    <button class="btn" type="submit"><?= icon('plus', 16) ?>Tambah tahun</button>
  </form>
</div>

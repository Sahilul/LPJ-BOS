<?php use App\Controllers\PejabatController; ?>
<div class="page-head">
  <div>
    <h1>Pejabat & tanda tangan</h1>
    <p class="lead">Kepala sekolah dan bendahara yang menandatangani kuitansi dan laporan. Saat pejabat berganti, tambahkan yang baru; dokumen lama tetap memakai pejabat pada tanggalnya.</p>
  </div>
  <div class="actions"><a class="btn" href="/pejabat/baru"><?= icon('plus', 16) ?>Tambah pejabat</a></div>
</div>

<div class="grid grid-2">
  <?php foreach (PejabatController::PERAN as $peran => $label): $p = $aktif[$peran]; ?>
    <div class="card card-pad person">
      <div>
        <div class="role"><?= e($label) ?> · aktif</div>
        <?php if ($p): ?>
          <div class="name"><?= e($p['nama']) ?></div>
          <div class="meta"><?= $p['nip'] ? 'NIP <span class="mono">' . e($p['nip']) . '</span>' : 'Tanpa NIP' ?><?= $p['pangkat'] ? ' · ' . e($p['pangkat']) : '' ?></div>
          <div class="meta">Sejak <?= e(tanggal_indo($p['mulai'])) ?></div>
          <a class="btn btn-ghost btn-sm" style="margin-top:14px" href="/pejabat/<?= (int) $p['id'] ?>/edit"><?= icon('edit', 14) ?>Ubah data</a>
        <?php else: ?>
          <div class="name muted">Belum diisi</div>
          <div class="meta">Dibutuhkan untuk tanda tangan kuitansi.</div>
          <a class="btn btn-sm" style="margin-top:14px" href="/pejabat/baru?peran=<?= $peran ?>"><?= icon('plus', 14) ?>Isi <?= e(strtolower($label)) ?></a>
        <?php endif; ?>
      </div>
      <div class="ttd">
        <?php if ($p && $p['ttd_path']): ?>
          <img src="<?= e(upload_url($p['ttd_path'])) ?>" alt="Tanda tangan <?= e($p['nama']) ?>">
        <?php else: ?>
          <span class="placeholder"><?= icon('pen', 18) ?><br>Tanda tangan<br>belum diunggah</span>
        <?php endif; ?>
      </div>
    </div>
  <?php endforeach; ?>
</div>

<section class="card" style="margin-top:22px" aria-labelledby="riwayat">
  <div class="card-head"><div><h2 id="riwayat" style="font-size:18px">Riwayat pejabat</h2><p>Semua kepala dan bendahara yang pernah tercatat.</p></div></div>
  <?php if (!$rows): ?>
    <div class="empty"><span class="ic-wrap"><?= icon('badge', 22) ?></span><p>Belum ada pejabat. Mulai dari kepala sekolah.</p></div>
  <?php else: ?>
    <div class="table-wrap">
      <table class="table">
        <thead><tr><th>Peran</th><th>Nama</th><th>NIP</th><th>Masa jabatan</th><th>TTD</th><th><span class="sr-only">Aksi</span></th></tr></thead>
        <tbody>
          <?php foreach ($rows as $r): $isAktif = ($aktif[$r['peran']]['id'] ?? null) === $r['id']; ?>
            <tr id="p<?= (int) $r['id'] ?>" class="<?= $isAktif ? '' : 'is-off' ?>">
              <td><?= e(PejabatController::PERAN[$r['peran']] ?? $r['peran']) ?></td>
              <td class="primary"><?= e($r['nama']) ?> <?= $isAktif ? '<span class="badge badge-ok">Aktif</span>' : '' ?></td>
              <td class="mono"><?= e($r['nip'] ?: '—') ?></td>
              <td><?= e(tanggal_indo($r['mulai'], true)) ?> – <?= $r['selesai'] ? e(tanggal_indo($r['selesai'], true)) : 'sekarang' ?></td>
              <td><?= $r['ttd_path'] ? '<span class="badge badge-ok">Ada</span>' : '<span class="badge badge-warn">Belum</span>' ?></td>
              <td>
                <div class="row-actions">
                  <a class="btn btn-ghost btn-icon" href="/pejabat/<?= (int) $r['id'] ?>/edit" aria-label="Ubah <?= e($r['nama']) ?>" title="Ubah"><?= icon('edit', 16) ?></a>
                  <form method="post" action="/pejabat/<?= (int) $r['id'] ?>/hapus" data-confirm="Hapus <?= e($r['nama']) ?> dari daftar pejabat?">
                    <?= csrf_field() ?>
                    <button class="btn btn-danger btn-icon" type="submit" aria-label="Hapus <?= e($r['nama']) ?>" title="Hapus"><?= icon('trash', 16) ?></button>
                  </form>
                </div>
              </td>
            </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
  <?php endif; ?>
</section>

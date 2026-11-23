<?php

declare(strict_types=1);

require __DIR__ . '/../app/bootstrap.php';

use App\Controllers\AuthController;
use App\Controllers\BackupController;
use App\Controllers\BelanjaController;
use App\Controllers\BkuController;
use App\Controllers\DashboardController;
use App\Controllers\DokumenController;
use App\Controllers\LampiranController;
use App\Controllers\LaporanController;
use App\Controllers\FileController;
use App\Controllers\MasterController;
use App\Controllers\MutasiController;
use App\Controllers\PejabatController;
use App\Controllers\PenerimaController;
use App\Controllers\PenerimaanController;
use App\Controllers\RencanaController;
use App\Controllers\RencanaPendapatanController;
use App\Controllers\SekolahController;
use App\Controllers\TahunController;
use App\Router;

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$uri = $_SERVER['REQUEST_URI'] ?? '/';
$path = parse_url($uri, PHP_URL_PATH) ?: '/';

if ($path === '/mcp') {
    App\Mcp\Http::handle($method);
    exit;
}

if ($method === 'POST' && !csrf_valid()) {
    abort(419, 'Sesi formulir sudah kedaluwarsa. Muat ulang halaman lalu coba lagi.');
}

$hasUser = (bool) App\Database::value('SELECT COUNT(*) FROM users');
$public = ['/login', '/setup'];
if (!$hasUser && $path !== '/setup') {
    redirect('/setup');
}
if ($hasUser && !auth_user() && !in_array($path, $public, true)) {
    redirect('/login');
}

$r = new Router();

$r->get('/setup', [AuthController::class, 'setupForm']);
$r->post('/setup', [AuthController::class, 'setup']);
$r->get('/login', [AuthController::class, 'loginForm']);
$r->post('/login', [AuthController::class, 'login']);
$r->post('/logout', [AuthController::class, 'logout']);
$r->get('/tahun', [TahunController::class, 'index']);
$r->post('/tahun', [TahunController::class, 'store']);
$r->post('/tahun/aktif', [TahunController::class, 'switch']);
$r->post('/tahun/{tahun}/default', [TahunController::class, 'makeDefault']);
$r->post('/tahun/{tahun}/hapus', [TahunController::class, 'destroy']);
$r->get('/api', [App\Controllers\ApiTokenController::class, 'index']);
$r->post('/api', [App\Controllers\ApiTokenController::class, 'store']);
$r->post('/api/{id}/hapus', [App\Controllers\ApiTokenController::class, 'destroy']);
$r->get('/akun', [AuthController::class, 'akunForm']);
$r->post('/akun', [AuthController::class, 'akun']);

$r->get('/', [DashboardController::class, 'index']);
$r->get('/file/{name}', [FileController::class, 'show']);

$r->get('/sekolah', [SekolahController::class, 'edit']);
$r->post('/sekolah', [SekolahController::class, 'update']);
$r->post('/sekolah/gambar/{jenis}', [SekolahController::class, 'upload']);
$r->post('/sekolah/gambar/{jenis}/hapus', [SekolahController::class, 'removeImage']);

$r->get('/pejabat', [PejabatController::class, 'index']);
$r->get('/pejabat/baru', [PejabatController::class, 'create']);
$r->post('/pejabat', [PejabatController::class, 'store']);
$r->get('/pejabat/{id}/edit', [PejabatController::class, 'edit']);
$r->post('/pejabat/{id}', [PejabatController::class, 'update']);
$r->post('/pejabat/{id}/hapus', [PejabatController::class, 'destroy']);

$r->get('/penerima/template', [PenerimaController::class, 'template']);
$r->get('/penerima/ekspor', [PenerimaController::class, 'export']);
$r->post('/penerima/impor', [PenerimaController::class, 'import']);

foreach (MasterController::resources() as $slug => $_) {
    $r->get("/$slug", fn () => (new MasterController($slug))->index());
    $r->get("/$slug/baru", fn () => (new MasterController($slug))->create());
    $r->post("/$slug", fn () => (new MasterController($slug))->store());
    $r->get("/$slug/{id}/edit", fn ($id) => (new MasterController($slug))->edit($id));
    $r->post("/$slug/{id}", fn ($id) => (new MasterController($slug))->update($id));
    $r->post("/$slug/{id}/hapus", fn ($id) => (new MasterController($slug))->destroy($id));
    $r->post("/$slug/{id}/aktif", fn ($id) => (new MasterController($slug))->toggle($id));
}

$r->get('/rencana/pendapatan', [RencanaPendapatanController::class, 'index']);
$r->get('/rencana/pendapatan/baru', [RencanaPendapatanController::class, 'create']);
$r->post('/rencana/pendapatan', [RencanaPendapatanController::class, 'store']);
$r->get('/rencana/pendapatan/{id}/edit', [RencanaPendapatanController::class, 'edit']);
$r->post('/rencana/pendapatan/{id}', [RencanaPendapatanController::class, 'update']);
$r->post('/rencana/pendapatan/{id}/hapus', [RencanaPendapatanController::class, 'destroy']);

$crud = [
    'rencana' => RencanaController::class,
    'penerimaan' => PenerimaanController::class,
    'mutasi' => MutasiController::class,
    'belanja' => BelanjaController::class,
];
foreach ($crud as $slug => $ctl) {
    if (method_exists($ctl, 'index')) {
        $r->get("/$slug", [$ctl, 'index']);
    }
    $r->get("/$slug/baru", [$ctl, 'create']);
    $r->post("/$slug", [$ctl, 'store']);
    $r->get("/$slug/{id}/edit", [$ctl, 'edit']);
    $r->post("/$slug/{id}", [$ctl, 'update']);
    $r->post("/$slug/{id}/hapus", [$ctl, 'destroy']);
}
$r->get('/belanja', fn () => redirect('/bku'));
$r->post('/belanja/{id}/lampiran', [LampiranController::class, 'store']);
$r->post('/belanja/{id}/lampiran/{lampiranId}/hapus', [LampiranController::class, 'destroy']);
$r->get('/dokumen/belanja/{id}/kuitansi.pdf', [DokumenController::class, 'kuitansi']);
$r->get('/dokumen/belanja/{id}/bukti-terima.pdf', [DokumenController::class, 'buktiTerima']);
$r->get('/bku', [BkuController::class, 'index']);
$r->post('/bku/{src}/{id}/tanggal', [BkuController::class, 'updateDate']);
$r->get('/laporan', [LaporanController::class, 'index']);
$r->get('/laporan/spj', [LaporanController::class, 'spj']);
$r->get('/laporan/{jenis}', [LaporanController::class, 'show']);

$r->get('/backup', [BackupController::class, 'index']);
$r->post('/backup/unduh', [BackupController::class, 'download']);
$r->post('/backup/pulihkan', [BackupController::class, 'restore']);

$r->dispatch($method, $uri);

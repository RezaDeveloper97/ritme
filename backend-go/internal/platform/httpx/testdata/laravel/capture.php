<?php
// Captures Laravel framework error bodies + paginator JSON for backend-go httpx tests.
// Run from the repo root (no DB needed; APP_DEBUG=false like production):
//   php backend-go/internal/platform/httpx/testdata/laravel/capture.php backend-go/internal/platform/httpx/testdata/laravel
// Captured 2026-09-23 with PHP 8.4.6 / Laravel 12 (backend/composer.lock).
$base = dirname(__DIR__, 6).'/backend';
putenv('APP_ENV=production'); $_ENV['APP_ENV']='production'; $_SERVER['APP_ENV']='production';
putenv('APP_DEBUG=false'); $_ENV['APP_DEBUG']='false'; $_SERVER['APP_DEBUG']='false';
require $base.'/vendor/autoload.php';
$app = require $base.'/bootstrap/app.php';
$kernel = $app->make(Illuminate\Contracts\Http\Kernel::class);
$out = $argv[1];
function dump($name, $resp) {
    global $out;
    $h = [];
    foreach (['Content-Type','Retry-After','X-RateLimit-Limit','X-RateLimit-Remaining','X-RateLimit-Reset','Allow'] as $k) {
        if ($resp->headers->has($k)) $h[$k] = $resp->headers->get($k);
    }
    file_put_contents("$out/$name.body", $resp->getContent());
    file_put_contents("$out/$name.meta.json", json_encode(['status'=>$resp->getStatusCode(),'headers'=>$h], JSON_PRETTY_PRINT|JSON_UNESCAPED_SLASHES)."\n");
    echo "$name: {$resp->getStatusCode()}\n";
}
$srv = ['HTTP_ACCEPT'=>'application/json','HTTP_HOST'=>'api.ritme.app','HTTPS'=>'on'];
foreach ([
  ['route_404', 'GET', '/api/v1/does-not-exist'],
  ['route_404_nested', 'GET', '/api/v1/foo/bar/baz'],
  ['method_405_single', 'GET', '/api/v1/auth/send-otp'],
  ['method_405_multi', 'GET', '/api/v1/reminders/5'],
  ['method_405_post_on_get', 'POST', '/api/v1/languages'],
  ['route_404_encoded', 'GET', '/api/v1/%D8%B3%D9%84%D8%A7%D9%85/x?y=1'],
  ['route_404_trailing_slash', 'GET', '/api/v1/nope/'],
  ['route_404_root_level', 'GET', '/nope'],
] as [$n,$m,$p]) {
  $req = Illuminate\Http\Request::create($p, $m, [], [], [], $srv);
  $resp = $kernel->handle($req); dump($n, $resp); $kernel->terminate($req, $resp);
}
// Exceptions rendered through the handler directly (no route/DB needed).
$handler = $app->make(Illuminate\Contracts\Debug\ExceptionHandler::class);
$req = Illuminate\Http\Request::create('/api/v1/x', 'GET', [], [], [], $srv);
$app->instance('request', $req);
dump('throttle_429', $handler->render($req, new Illuminate\Http\Exceptions\ThrottleRequestsException('Too Many Attempts.', null, ['Retry-After'=>42,'X-RateLimit-Limit'=>5,'X-RateLimit-Remaining'=>0,'X-RateLimit-Reset'=>1790000042])));
dump('server_500', $handler->render($req, new RuntimeException('boom')));
dump('maintenance_503', $handler->render($req, new Symfony\Component\HttpKernel\Exception\HttpException(503, 'Service Unavailable')));
dump('abort_404', $handler->render($req, new Symfony\Component\HttpKernel\Exception\NotFoundHttpException('')));
dump('model_404', $handler->render($req, (new Illuminate\Database\Eloquent\ModelNotFoundException)->setModel('App\\Models\\TaskTemplate', [99])));
dump('model_404_noid', $handler->render($req, (new Illuminate\Database\Eloquent\ModelNotFoundException)->setModel('App\\Models\\TaskTemplate')));
// Validation summary
$v = Illuminate\Support\Facades\Validator::make(['a'=>null,'b'=>null,'c'=>null], ['a'=>'required','b'=>'required','c'=>'required']);
dump('validation_422', $handler->render($req, new Illuminate\Validation\ValidationException($v)));
$v = Illuminate\Support\Facades\Validator::make(['a'=>null,'b'=>null], ['a'=>'required','b'=>'required']);
dump('validation_422_one_more', $handler->render($req, new Illuminate\Validation\ValidationException($v)));
app()->setLocale('fa');
$v = Illuminate\Support\Facades\Validator::make(['a'=>null,'b'=>null,'c'=>null], ['a'=>'required','b'=>'required','c'=>'required']);
dump('validation_422_fa', $handler->render($req, new Illuminate\Validation\ValidationException($v)));
app()->setLocale('en');
// Paginator (raw)
$req2 = Illuminate\Http\Request::create('/api/v1/health-logs?from_date=2026-01-01&page=2', 'GET', [], [], [], $srv);
$app->instance('request', $req2);
Illuminate\Pagination\Paginator::currentPathResolver(fn() => $req2->url());
Illuminate\Pagination\Paginator::currentPageResolver(fn($n='page') => (int)$req2->query($n, 1));
foreach ([[2, 95, 30, 'paginator_p2'], [1, 0, 30, 'paginator_empty'], [5, 400, 30, 'paginator_many'], [1, 45, 30, 'paginator_p1'], [7, 95, 30, 'paginator_beyond'], [20, 1500, 30, 'paginator_slider'], [48, 1500, 30, 'paginator_near_end'], [1, 5, 2, 'paginator_small']] as [$page,$total,$per,$n]) {
  $items = [];
  $cnt = max(0, min($per, $total - ($page-1)*$per));
  for ($i=0;$i<$cnt;$i++) $items[] = ['id'=>($page-1)*$per+$i+1];
  $p = new Illuminate\Pagination\LengthAwarePaginator($items, $total, $per, $page, ['path'=>$req2->url(), 'pageName'=>'page']);
  file_put_contents("$out/$n.json", json_encode($p));
  echo "$n\n";
}

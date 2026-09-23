<?php
// Records HtmlSanitizer::clean / ::toPlainText for every input of corpus.json, run in the
// production PHP image (libxml2 2.9.14) — the contract stack's Laravel container:
//
//   docker cp dump_sanitizer.php ritme-contract-contract-laravel-1:/tmp/ &&
//   docker exec -i ritme-contract-contract-laravel-1 php /tmp/dump_sanitizer.php \
//     < corpus.json > php_sanitizer.json
//   (same with fuzz_corpus.json > php_fuzz.json)
require '/var/www/html/app/Services/Content/HtmlSanitizer.php';

use App\Services\Content\HtmlSanitizer;

$out = [];
foreach (json_decode(stream_get_contents(STDIN), true) as $input) {
    $out[] = [
        'input' => $input,
        'clean' => HtmlSanitizer::clean($input),
        'plain' => HtmlSanitizer::toPlainText($input),
    ];
}
echo json_encode($out, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_INVALID_UTF8_SUBSTITUTE), "\n";

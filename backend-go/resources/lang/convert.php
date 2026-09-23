<?php
// Converts Laravel's translation files to the JSON the Go translator embeds
// (internal/i18n/lang). Run once from the repo root after a lang/ change:
//
//   php backend-go/resources/lang/convert.php
//
// Sources, merged exactly like Illuminate\Translation\FileLoader::loadPaths
// (framework lang path first, then the app's lang/ with array_replace_recursive):
//   - backend/vendor/laravel/framework/src/Illuminate/Translation/lang/<locale>/validation.php
//   - backend/lang/<locale>/<group>.php
// Output: backend-go/resources/lang/<locale>/<group>.json (key order kept).
$root = dirname(__DIR__, 3);
$framework = $root.'/backend/vendor/laravel/framework/src/Illuminate/Translation/lang';
$app = $root.'/backend/lang';
$out = __DIR__;

$groups = [];
foreach (glob($app.'/*/*.php') as $file) {
    $groups[basename(dirname($file))][basename($file, '.php')] = true;
}
// Laravel's built-in English validation lines are the fallback for every locale.
foreach (glob($framework.'/*/validation.php') as $file) {
    $groups[basename(dirname($file))]['validation'] = true;
}
ksort($groups);

foreach ($groups as $locale => $names) {
    ksort($names);
    foreach (array_keys($names) as $group) {
        $lines = [];
        foreach ([$framework, $app] as $base) {
            $path = "$base/$locale/$group.php";
            if (is_file($path)) {
                $lines = array_replace_recursive($lines, require $path);
            }
        }
        @mkdir("$out/$locale", 0755, true);
        file_put_contents(
            "$out/$locale/$group.json",
            json_encode($lines, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)."\n"
        );
        echo "$locale/$group.json\n";
    }
}

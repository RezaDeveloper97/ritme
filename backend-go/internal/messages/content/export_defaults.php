<?php

/**
 * Exports every smart-message class's contentDefaults() (the MessageContentSeeder providers, in
 * seeder order) to one JSON document: group → item_key → locale → payload. The output is
 * committed as defaults.json next to this script and embedded by the Go port (go:embed); the
 * content test re-runs this script and requires byte equality.
 *
 *   php internal/messages/content/export_defaults.php ../backend > internal/messages/content/defaults.json
 *
 * contentDefaults() is static and needs no booted application, only the Composer autoloader.
 */

$backend = $argv[1] ?? __DIR__.'/../../../../backend';
require $backend.'/vendor/autoload.php';

// database/seeders/MessageContentSeeder.php PROVIDERS, same order.
$providers = [
    App\Services\MessageSystem\Engines\CycleMessageEngine::class,
    App\Services\MessageSystem\Engines\PregnancyMessageEngine::class,
    App\Services\MessageSystem\Modules\NutritionModule::class,
    App\Services\MessageSystem\Modules\SleepModule::class,
    App\Services\MessageSystem\Modules\ExerciseModule::class,
    App\Services\MessageSystem\Layers\CorrelationLayer::class,
    App\Services\MessageSystem\Layers\PatternLayer::class,
    App\Services\BmiService::class,
];

$out = [];
foreach ($providers as $provider) {
    foreach ($provider::contentDefaults() as $group => $items) {
        if (array_key_exists($group, $out)) {
            fwrite(STDERR, "duplicate group {$group}\n");
            exit(1);
        }
        foreach ($items as $itemKey => $locales) {
            // Item keys are strings in message_contents ((string) $itemKey in the seeder); an
            // object keeps "1", "2", … as keys instead of becoming a JSON list.
            $out[$group][(string) $itemKey] = (object) $locales;
        }
        $out[$group] = (object) $out[$group];
    }
}

echo json_encode((object) $out, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_THROW_ON_ERROR), "\n";

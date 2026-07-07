<?php

namespace Database\Seeders;

use App\Models\MessageContent;
use App\Services\MessageSystem\Engines\CycleMessageEngine;
use App\Services\MessageSystem\Engines\PregnancyMessageEngine;
use App\Services\MessageSystem\Layers\CorrelationLayer;
use App\Services\MessageSystem\Layers\PatternLayer;
use App\Services\MessageSystem\Modules\ExerciseModule;
use App\Services\MessageSystem\Modules\NutritionModule;
use App\Services\MessageSystem\Modules\SleepModule;
use Illuminate\Database\Seeder;

/**
 * Seeds message_contents from each provider's contentDefaults(). Idempotent and
 * NON-destructive: firstOrCreate never overwrites an existing (admin-edited) row.
 */
class MessageContentSeeder extends Seeder
{
    /** Every class that owns editable smart-message text. */
    private const PROVIDERS = [
        CycleMessageEngine::class,
        PregnancyMessageEngine::class,
        NutritionModule::class,
        SleepModule::class,
        ExerciseModule::class,
        CorrelationLayer::class,
        PatternLayer::class,
    ];

    public function run(): void
    {
        foreach (self::PROVIDERS as $provider) {
            if (! method_exists($provider, 'contentDefaults')) {
                continue;
            }

            foreach ($provider::contentDefaults() as $group => $items) {
                foreach ($items as $itemKey => $locales) {
                    foreach ($locales as $locale => $payload) {
                        MessageContent::firstOrCreate(
                            [
                                'group' => $group,
                                'item_key' => (string) $itemKey,
                                'locale' => $locale,
                            ],
                            [
                                'label' => "{$group} / {$itemKey}",
                                'payload' => $payload,
                                'is_active' => true,
                                'is_approved' => true,
                            ]
                        );
                    }
                }
            }
        }
    }
}

<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 5 — "توصیه‌های امروز": phase & symptom driven daily tips coming from
 * the HealthDataEngine (daily_tips), each mapped to a display icon.
 */
class RecommendationsSection extends AbstractHomeSection
{
    private const TYPE_ICONS = [
        'nutrition' => 'apple',
        'hydration' => 'water-glass',
        'warmth' => 'heat',
        'rest' => 'bed',
        'energy' => 'bolt',
        'exercise' => 'walking',
        'fertility' => 'heart',
        'pms' => 'flower',
        'mood' => 'smile',
        'mental_health' => 'brain',
        'pain_relief' => 'bandage',
        'sleep' => 'moon',
        'digestion' => 'stomach',
    ];

    private const TYPE_LABELS = [
        'nutrition' => ['fa' => 'تغذیه', 'en' => 'Nutrition'],
        'hydration' => ['fa' => 'آب‌رسانی', 'en' => 'Hydration'],
        'warmth' => ['fa' => 'گرما درمانی', 'en' => 'Warmth'],
        'rest' => ['fa' => 'استراحت', 'en' => 'Rest'],
        'energy' => ['fa' => 'انرژی', 'en' => 'Energy'],
        'exercise' => ['fa' => 'ورزش', 'en' => 'Exercise'],
        'fertility' => ['fa' => 'باروری', 'en' => 'Fertility'],
        'pms' => ['fa' => 'پی‌ام‌اس', 'en' => 'PMS'],
        'mood' => ['fa' => 'خلق‌وخو', 'en' => 'Mood'],
        'mental_health' => ['fa' => 'سلامت روان', 'en' => 'Mental health'],
        'pain_relief' => ['fa' => 'تسکین درد', 'en' => 'Pain relief'],
        'sleep' => ['fa' => 'خواب', 'en' => 'Sleep'],
        'digestion' => ['fa' => 'گوارش', 'en' => 'Digestion'],
    ];

    public function key(): string
    {
        return 'recommendations';
    }

    public function order(): int
    {
        return 50;
    }

    public function supports(HomeContext $context): bool
    {
        return $context->isCycleMode();
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $tips = $context->cycleData()['daily_tips'] ?? [];
        if (empty($tips)) {
            return null;
        }

        $items = [];
        $tagTypes = [];

        foreach (array_slice($tips, 0, 6) as $tip) {
            $type = $tip['type'] ?? 'general';
            $text = $this->pick($tip, $context->locale);
            if (!$text) {
                continue;
            }

            $tagTypes[$type] = true;
            $items[] = [
                'type' => $type,
                'icon' => self::TYPE_ICONS[$type] ?? 'sparkle',
                'title' => $this->pick(self::TYPE_LABELS[$type] ?? null, $context->locale)
                    ?? $context->t('توصیه', 'Tip'),
                'text' => $text,
            ];
        }

        if (empty($items)) {
            return null;
        }

        $tags = array_map(fn ($type) => [
            'type' => $type,
            'icon' => self::TYPE_ICONS[$type] ?? 'sparkle',
        ], array_keys($tagTypes));

        return new HomeSection(
            key: $this->key(),
            type: 'recommendations',
            title: $context->t('توصیه‌های امروز', "Today's recommendations"),
            data: [
                'tags' => $tags,
                'items' => $items,
            ],
            order: $this->order(),
        );
    }
}

<?php

namespace App\Services\HomePage\Sections;

use App\Enums\RecommendationType;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 5 — "توصیه‌های امروز": the admin-managed daily recommendations the
 * HealthDataEngine resolved for today (`daily_tips`), each carrying its category
 * icon and title.
 *
 * Icons and default titles come from {@see RecommendationType} — the same table
 * the admin form and the cycle-calculation payload read, so a category is
 * described in exactly one place.
 */
class RecommendationsSection extends AbstractHomeSection
{
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
            $type = $tip['type'] ?? RecommendationType::GENERAL->value;
            $text = $this->pick($tip, $context->locale);
            if (! $text) {
                continue;
            }

            $tagTypes[$type] = true;
            $items[] = [
                'type' => $type,
                'icon' => RecommendationType::iconFor($type),
                // An admin-set title wins; otherwise the category's own label.
                'title' => $this->pick($tip['title'] ?? null, $context->locale)
                    ?? RecommendationType::labelFor($type, $context->locale),
                'text' => $text,
            ];
        }

        if (empty($items)) {
            return null;
        }

        $tags = array_map(fn ($type) => [
            'type' => $type,
            'icon' => RecommendationType::iconFor($type),
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

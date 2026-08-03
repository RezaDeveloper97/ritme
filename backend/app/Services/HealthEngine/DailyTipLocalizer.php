<?php

namespace App\Services\HealthEngine;

use App\Enums\RecommendationType;

/**
 * Collapses the engine's bilingual `daily_tips` blobs into render-ready rows
 * for one locale.
 *
 * The engine keeps `{type, fa, en, title: {fa, en}}` so a single calculation can
 * serve either locale; clients get `{type, title, icon, text}`. The title falls
 * back to the category label, so an admin's wording change reaches every client
 * without a release.
 *
 * `icon` is this API's own vocabulary and the web client keeps its own map keyed
 * on `type` — it is here for clients that would rather render what the server
 * names than ship a category table (the Android app takes its recommendations
 * from /messages/daily and reads neither).
 */
class DailyTipLocalizer
{
    /**
     * @param  array<int, mixed>  $tips
     * @return array<int, array{type: string, title: string, icon: string, text: string}>
     */
    public function localize(array $tips, string $locale): array
    {
        $localized = [];

        foreach ($tips as $tip) {
            if (! is_array($tip)) {
                continue;
            }

            $text = $tip[$locale] ?? $tip['en'] ?? null;
            if (! is_string($text) || $text === '') {
                continue;
            }

            $type = is_string($tip['type'] ?? null) ? $tip['type'] : RecommendationType::GENERAL->value;
            $title = $tip['title'][$locale] ?? $tip['title']['en'] ?? null;

            $localized[] = [
                'type' => $type,
                'title' => is_string($title) && $title !== ''
                    ? $title
                    : RecommendationType::labelFor($type, $locale),
                'icon' => RecommendationType::iconFor($type),
                'text' => $text,
            ];
        }

        return $localized;
    }
}

<?php

namespace App\Services\MessageSystem\Support;

use App\Models\MessageContent;

/**
 * Resolves editable smart-message text from the message_contents table, keyed
 * by (group, item_key, locale). Results are memoised per request. Callers pass
 * a code fallback so the app keeps working if a row is missing or unapproved.
 */
class MessageContentRepository
{
    /** @var array<string, ?array> */
    private array $cache = [];

    /** The raw stored payload, or null when no live row exists. */
    public function payload(string $group, string $itemKey, string $locale): ?array
    {
        $cacheKey = "{$group}|{$itemKey}|{$locale}";

        if (! array_key_exists($cacheKey, $this->cache)) {
            $this->cache[$cacheKey] = MessageContent::query()
                ->live()
                ->where('group', $group)
                ->where('item_key', $itemKey)
                ->where('locale', $locale)
                ->value('payload');
        }

        return $this->cache[$cacheKey];
    }

    /**
     * DB payload when present, otherwise the supplied code fallback.
     *
     * @param  array<string, mixed>  $fallback
     * @return array<string, mixed>
     */
    public function resolve(string $group, string $itemKey, string $locale, array $fallback): array
    {
        return $this->payload($group, $itemKey, $locale) ?? $fallback;
    }
}

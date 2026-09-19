<?php

namespace App\Services\MessageSystem\Support;

use App\Models\MessageContent;

/**
 * Resolves editable smart-message text from the message_contents table, keyed
 * by (group, item_key, locale). Callers pass a code fallback so the app keeps
 * working if a row is missing or unapproved.
 *
 * The first lookup in a locale loads every live row of that locale in one query
 * (a few dozen small rows) and later lookups, in any group, are answered from
 * memory. A daily message reads four groups, so this is one query where it used
 * to be four. Nothing outlives the request (the class is a request singleton),
 * so an admin save is visible on the very next request with no invalidation
 * step. A cross-request cache would need either a save hook on the model or a
 * freshness probe that reads the table anyway, which costs what this load does.
 */
class MessageContentRepository
{
    /** @var array<string, array<string, array<string, string>>> raw JSON payloads: locale → "group|item_key" → payload */
    private array $rowsByLocale = [];

    /** @var array<string, ?array> decoded payloads, memoised per "locale|group|item_key" */
    private array $decoded = [];

    /** The raw stored payload, or null when no live row exists. */
    public function payload(string $group, string $itemKey, string $locale): ?array
    {
        $memoKey = "{$locale}|{$group}|{$itemKey}";

        if (! array_key_exists($memoKey, $this->decoded)) {
            $raw = $this->rows($locale)["{$group}|{$itemKey}"] ?? null;
            $this->decoded[$memoKey] = $raw === null ? null : json_decode($raw, true);
        }

        return $this->decoded[$memoKey];
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

    /**
     * Every live row of a locale, keyed "group|item_key" (the table's unique key).
     *
     * @return array<string, string>
     */
    private function rows(string $locale): array
    {
        return $this->rowsByLocale[$locale] ??= MessageContent::query()
            ->live()
            ->where('locale', $locale)
            ->toBase()
            ->get(['group', 'item_key', 'payload'])
            ->mapWithKeys(fn (object $row): array => ["{$row->group}|{$row->item_key}" => $row->payload])
            ->all();
    }
}

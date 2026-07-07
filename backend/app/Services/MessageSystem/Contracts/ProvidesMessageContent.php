<?php

namespace App\Services\MessageSystem\Contracts;

/**
 * Implemented by every engine/layer/module that owns editable smart-message
 * text. contentDefaults() is the single seed/fallback source: the seeder writes
 * it into the message_contents table, and the runtime falls back to it when a
 * row is missing. At runtime the text is read from the DB (admin-editable).
 */
interface ProvidesMessageContent
{
    /**
     * @return array<string, array<string, array<string, mixed>>>
     *         group => item_key => locale => payload
     */
    public static function contentDefaults(): array;
}

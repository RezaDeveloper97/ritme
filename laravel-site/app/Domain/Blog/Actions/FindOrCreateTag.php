<?php

declare(strict_types=1);

namespace App\Domain\Blog\Actions;

use App\Domain\Blog\Models\Tag;

/**
 * Inline tag creation from the post editor: an existing tag with the same name (whitespace-normalised) is reused,
 * otherwise a new one is created (TaxonomyObserver fills the slug).
 */
final class FindOrCreateTag
{
    public function handle(string $name): Tag
    {
        $name = trim((string) preg_replace('/\s+/u', ' ', $name));

        return Tag::query()->firstOrCreate(['name' => mb_substr($name, 0, 191)]);
    }
}

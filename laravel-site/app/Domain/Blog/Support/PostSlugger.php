<?php

declare(strict_types=1);

namespace App\Domain\Blog\Support;

use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\PostSlug;
use App\Support\Html\TextSlug;

/**
 * Post slugs: normalised (Persian kept, TextSlug), unique against current slugs AND other posts' history (an old URL
 * keeps redirecting to its post), and a changed slug of a post that was ever published goes into history so the old
 * URL 301s (L4-03). Re-using one of the post's own old slugs removes it from history.
 */
final class PostSlugger
{
    public const MAX_LENGTH = 120;

    /**
     * Called while saving: fills an empty slug from the title, normalises and de-duplicates a new/changed one.
     */
    public function assign(Post $post): void
    {
        $current = trim((string) $post->getAttribute('slug'));

        if ($post->exists && $current !== '' && ! $post->isDirty('slug')) {
            return;
        }

        $base = TextSlug::make($current !== '' ? $current : $post->title, self::MAX_LENGTH);
        $post->slug = $this->unique($base === '' ? 'post' : $base, $post->exists ? $post->id : null);
    }

    /**
     * Called after an update: keeps the previous slug as history.
     */
    public function recordChange(Post $post): void
    {
        if (! $post->wasChanged('slug')) {
            return;
        }

        PostSlug::query()->where('post_id', $post->id)->where('slug', $post->slug)->delete();

        $previous = (string) $post->getOriginal('slug');
        if ($previous === '' || $post->getOriginal('published_at') === null) {
            return; // a draft that was never public has no URL worth redirecting
        }

        if (! PostSlug::query()->where('slug', $previous)->exists()) {
            PostSlug::query()->create(['post_id' => $post->id, 'slug' => $previous]);
        }
    }

    private function unique(string $base, ?int $ignoreId): string
    {
        $candidate = $base;
        for ($n = 2; $this->taken($candidate, $ignoreId); $n++) {
            $suffix = '-'.$n;
            $candidate = mb_substr($base, 0, self::MAX_LENGTH - strlen($suffix), 'UTF-8').$suffix;
        }

        return $candidate;
    }

    private function taken(string $slug, ?int $ignoreId): bool
    {
        $live = Post::query()->where('slug', $slug)->when($ignoreId !== null, static fn ($q) => $q->whereKeyNot($ignoreId))->exists();

        return $live || PostSlug::query()->where('slug', $slug)->when($ignoreId !== null, static fn ($q) => $q->where('post_id', '!=', $ignoreId))->exists();
    }
}

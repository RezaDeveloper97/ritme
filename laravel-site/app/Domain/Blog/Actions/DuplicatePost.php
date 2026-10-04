<?php

declare(strict_types=1);

namespace App\Domain\Blog\Actions;

use App\Domain\Blog\Enums\PostStatus;
use App\Domain\Blog\Models\Post;
use Illuminate\Database\ConnectionInterface;

/**
 * Copies a post as a new draft: content, taxonomy, covers, tags and SEO overrides (without the canonical override,
 * which would point the copy at the original). The copy gets «(کپی)» in the title and a fresh slug; dates, views
 * and the featured flag are not copied.
 */
final class DuplicatePost
{
    public const TITLE_SUFFIX = ' (کپی)';

    public function __construct(
        private readonly SyncPostTags $syncTags,
        private readonly ConnectionInterface $db,
    ) {}

    public function handle(Post $source): Post
    {
        $source->loadMissing(['tags', 'seoMeta']);

        return $this->db->transaction(function () use ($source): Post {
            $copy = $source->replicate(['slug', 'views', 'published_at', 'updated_content_at', 'reviewed_at', 'word_count', 'reading_time']);
            $copy->title = mb_substr($source->title, 0, 255 - mb_strlen(self::TITLE_SUFFIX)).self::TITLE_SUFFIX;
            $copy->slug = '';
            $copy->status = PostStatus::Draft;
            $copy->is_featured = false;
            $copy->save();

            $this->syncTags->handle($copy, $source->tags->modelKeys());

            if ($source->seoMeta !== null) {
                $seo = $source->seoMeta->replicate(['seoable_type', 'seoable_id', 'route_name', 'canonical_url']);
                $copy->seoMeta()->save($seo);
            }

            return $copy;
        });
    }
}

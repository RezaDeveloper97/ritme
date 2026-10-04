<?php

declare(strict_types=1);

namespace App\Domain\Blog\Repositories;

use App\Domain\Blog\Contracts\TagRepository;
use App\Domain\Blog\Data\TagData;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use Illuminate\Database\Eloquent\Builder;

final class EloquentTagRepository implements TagRepository
{
    public function findBySlug(string $slug): ?TagData
    {
        $tag = Tag::query()
            ->where('slug', $slug)
            ->withCount(['posts as published_posts_count' => static function (Builder $q): void {
                /** @var Builder<Post> $q */
                $q->published();
            }])
            ->first();

        return $tag === null ? null : TagData::fromModel($tag, (int) $tag->getAttribute('published_posts_count'));
    }
}

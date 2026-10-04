<?php

declare(strict_types=1);

namespace App\Domain\Blog\Queries;

use App\Domain\Blog\Data\PostPage;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Models\Post;
use Illuminate\Database\Eloquent\Builder;

/**
 * Newest published posts, optionally narrowed to a life stage (post stage, or its category's stage), a tag or an
 * author.
 */
final class LatestPosts
{
    use PaginatesPosts;

    public function __construct(
        private readonly int $page = 1,
        private readonly int $perPage = 12,
        private readonly ?LifeStage $stage = null,
        private readonly ?int $tagId = null,
        private readonly ?int $authorId = null,
    ) {}

    public function get(): PostPage
    {
        $query = Post::query()->published();

        if ($this->stage !== null) {
            $stage = $this->stage->value;
            $query->where(static function (Builder $q) use ($stage): void {
                $q->where('life_stage', $stage)
                    ->orWhere(static function (Builder $q) use ($stage): void {
                        $q->whereNull('life_stage')->whereHas('category', static fn (Builder $c) => $c->where('life_stage', $stage));
                    });
            });
        }

        if ($this->tagId !== null) {
            $tagId = $this->tagId;
            $query->whereHas('tags', static fn (Builder $q) => $q->whereKey($tagId));
        }

        if ($this->authorId !== null) {
            $query->where('author_id', $this->authorId);
        }

        return $this->paginate($query, $this->page, $this->perPage);
    }
}

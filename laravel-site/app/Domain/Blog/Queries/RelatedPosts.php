<?php

declare(strict_types=1);

namespace App\Domain\Blog\Queries;

use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Models\Post;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Support\Carbon;

/**
 * Related posts for an article. Candidates are the newest `$pool` published posts that share a tag or the category;
 * each is scored
 *
 *     3 × shared tags  +  2 (same category)  +  1 (same life stage)  +  2 / (1 + age in days / 90)
 *
 * so topical overlap wins and recency breaks ties (a fresh post with one shared tag beats an old one). When fewer than
 * `$limit` candidates exist, the list is topped up with the latest posts. Portable SQL (SQLite + MySQL); scoring in PHP.
 */
final class RelatedPosts
{
    public function __construct(
        private readonly int $postId,
        private readonly int $limit = 3,
        private readonly int $pool = 60,
    ) {}

    /**
     * @return list<PostCardData>
     */
    public function get(): array
    {
        if ($this->limit < 1) {
            return [];
        }

        $post = Post::query()->with('tags:blog_tags.id')->find($this->postId, ['id', 'category_id', 'life_stage']);
        if ($post === null) {
            return [];
        }

        /** @var list<int> $tagIds */
        $tagIds = $post->tags->pluck('id')->map(intval(...))->values()->all();
        $categoryId = $post->category_id;

        $candidates = $categoryId === null && $tagIds === [] ? new Collection : Post::query()
            ->published()
            ->whereKeyNot($post->id)
            ->where(static function (Builder $q) use ($categoryId, $tagIds): void {
                if ($categoryId !== null) {
                    $q->orWhere('category_id', $categoryId);
                }
                if ($tagIds !== []) {
                    $q->orWhereHas('tags', static fn (Builder $t) => $t->whereIn('blog_tags.id', $tagIds));
                }
            })
            ->withCount(['tags as shared_tags' => static fn (Builder $t) => $t->whereIn('blog_tags.id', $tagIds === [] ? [0] : $tagIds)])
            ->with('category')
            ->orderByDesc('published_at')
            ->orderByDesc('id')
            ->limit($this->pool)
            ->get();

        $now = Carbon::now();
        $scored = $candidates
            ->map(static function (Post $candidate) use ($post, $now): array {
                $ageDays = max(0.0, (float) ($candidate->published_at?->diffInSeconds($now) ?? 0) / 86400);
                $score = 3 * (int) $candidate->getAttribute('shared_tags')
                    + ($post->category_id !== null && $candidate->category_id === $post->category_id ? 2 : 0)
                    + ($post->life_stage !== null && $candidate->life_stage === $post->life_stage ? 1 : 0)
                    + 2 / (1 + $ageDays / 90);

                return ['post' => $candidate, 'score' => $score];
            })
            ->sort(static fn (array $a, array $b): int => [$b['score'], $b['post']->published_at, $b['post']->id] <=> [$a['score'], $a['post']->published_at, $a['post']->id])
            ->take($this->limit)
            ->map(static fn (array $row): PostCardData => PostCardData::fromModel($row['post']))
            ->values()
            ->all();

        $missing = $this->limit - count($scored);
        if ($missing > 0) {
            $exclude = [$post->id, ...array_map(static fn (PostCardData $card): int => $card->id, $scored)];
            $fill = Post::query()
                ->published()
                ->whereNotIn('id', $exclude)
                ->with('category')
                ->orderByDesc('published_at')
                ->orderByDesc('id')
                ->limit($missing)
                ->get()
                ->map(PostCardData::fromModel(...))
                ->all();
            $scored = [...$scored, ...$fill];
        }

        return array_values($scored);
    }
}

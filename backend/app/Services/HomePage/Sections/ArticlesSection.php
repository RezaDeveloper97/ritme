<?php

namespace App\Services\HomePage\Sections;

use App\Models\Article;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 15 — "بر اساس سیکل فعلی شما": educational articles relevant to the
 * user's current cycle phase.
 */
class ArticlesSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'articles';
    }

    public function order(): int
    {
        return 150;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $articles = Article::query()
            ->published()
            ->forPhase($context->phase())
            ->limit(6)
            ->get();

        if ($articles->isEmpty()) {
            return null;
        }

        $items = $articles->map(fn (Article $article) => [
            'id' => $article->id,
            'slug' => $article->slug,
            'title' => $article->localized('title', $context->locale),
            'excerpt' => $article->localized('excerpt', $context->locale),
            'read_time_minutes' => $article->read_time_minutes,
            'image_url' => $article->image_url,
            'category' => $article->category,
            'cycle_phase' => $article->cycle_phase,
        ])->all();

        return new HomeSection(
            key: $this->key(),
            type: 'articles',
            title: $context->t('بر اساس سیکل فعلی شما', 'Based on your current cycle'),
            data: [
                'items' => $items,
            ],
            order: $this->order(),
            action: $this->action('view_more', $context->t('مشاهده بیشتر', 'View more')),
        );
    }
}

<?php

namespace App\Services\HomePage\Sections;

use App\Enums\CycleSubphase;
use App\Models\Article;
use App\Services\Content\HtmlSanitizer;
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
            ->forPhase($this->phaseCandidates($context))
            ->limit(6)
            ->get();

        if ($articles->isEmpty()) {
            return null;
        }

        // Excerpts are authored in a rich-text editor but this row renders card
        // summaries, so they cross as plain text; the full HTML stays on the
        // article itself (GET /articles/{slug}).
        $items = $articles->map(fn (Article $article) => [
            'id' => $article->id,
            'slug' => $article->slug,
            'title' => $article->localized('title', $context->locale),
            'excerpt' => HtmlSanitizer::toPlainText($article->localized('excerpt', $context->locale)),
            'read_time_minutes' => $article->read_time_minutes,
            'image_url' => $article->image_url,
            'category' => $article->category,
            'cycle_phases' => $article->cycle_phases ?? [],
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

    /**
     * Every phase key an article may carry that should surface for this reader:
     * the sub-phase the engine resolved, its canonical alias (the key the admin
     * panel offers), and the coarse main phase older rows were tagged with.
     *
     * @return array<int, string|null>
     */
    private function phaseCandidates(HomeContext $context): array
    {
        $subphase = $context->subphase();

        return [
            $subphase,
            $subphase === null ? null : CycleSubphase::tryFrom($subphase)?->canonical()->value,
            $context->phase(),
        ];
    }
}

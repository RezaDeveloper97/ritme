<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages\Data;

use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Seo\Schema\Data\FaqItem;
use App\Domain\Settings\Data\AppLinksSettings;

/**
 * Everything `pages/stages/show.blade.php` renders — built by StagePageBuilder, no lookups left for the view.
 */
final readonly class StagePageData
{
    /**
     * @param  list<StageNavItemData>  $nav
     * @param  list<StageFeatureData>  $features
     * @param  list<StageCardData>  $tools
     * @param  list<StageCardData>  $help  empty = no «وقتی کمک بیشتری لازم داری» block
     * @param  list<PostCardData>  $readings
     * @param  list<FaqItem>  $faq
     * @param  array<string, string>  $copy  shared block headings (tools/help/readings/faq eyebrows and titles)
     */
    public function __construct(
        public LifeStage $stage,
        public string $name,
        public string $seoTitle,
        public string $seoDescription,
        public string $navLabel,
        public string $stagesLabel,
        public array $nav,
        public StageHeroData $hero,
        public array $features,
        public array $tools,
        public array $help,
        public string $emergencyText,
        public string $emergencyNumber,
        public array $readings,
        public string $readingsMoreUrl,
        public string $faqGroup,
        public string $faqTitle,
        public array $faq,
        public string $appCtaTitle,
        public ?string $appCtaLead,
        public AppLinksSettings $appLinks,
        public ?string $qrUrl,
        public array $copy,
    ) {}

    /**
     * @return list<array<string, string|null>>
     */
    public function toolProps(): array
    {
        return array_map(static fn (StageCardData $card): array => $card->toProps(), $this->tools);
    }

    /**
     * @return list<array<string, string|null>>
     */
    public function helpProps(): array
    {
        return array_map(static fn (StageCardData $card): array => $card->toProps(), $this->help);
    }

    /**
     * FAQ items as x-ui.accordion props.
     *
     * @return list<array{question: string, answer: string}>
     */
    public function faqProps(): array
    {
        return array_map(static fn (FaqItem $item): array => ['question' => $item->question, 'answer' => $item->answer], $this->faq);
    }
}

<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Content\Enums\StaticPage;

/**
 * One life-stage page (cycle, ttc, pregnancy, postpartum, menopause, teen) rendered by the shared template
 * `pages/stages/show.blade.php`. A subclass declares the page STRUCTURE — icons, colours, link targets, mock
 * screens, which optional blocks exist; every word lives in `lang/fa/stages/<slug>.php` under the same keys.
 *
 * Adding a stage (L3-04 / L3-05): create `App\Domain\Content\Stages\<Studly slug>` extending this class and
 * `lang/fa/stages/<slug>.php`; StageRegistry discovers the class by name and StagePageController serves the
 * route as soon as it exists (until then the route stays a noindex placeholder). Section partials that differ
 * from the shared ones go in `resources/views/pages/stages/partials/<slug>/` and are named by `featurePartial()`.
 */
abstract class StageDefinition
{
    abstract public function stage(): LifeStage;

    /**
     * Feature splits in page order; the first gets `id="how"` (target of the hero «چطور کار می‌کند؟»).
     *
     * @return list<FeatureSpec>
     */
    abstract public function features(): array;

    /**
     * «کارهای کوچک، آمادگی بیشتر» cards.
     *
     * @return list<CardSpec>
     */
    abstract public function tools(): array;

    /**
     * «وقتی کمک بیشتری لازم داری» cards; an empty list drops the block (teen has none — AUDIT §1).
     *
     * @return list<CardSpec>
     */
    public function help(): array
    {
        return [];
    }

    /** Partial in resources/views/pages/stages/mock/screens shown in the hero phone. */
    public function heroScreen(): string
    {
        return 'cycle-today';
    }

    /**
     * @return list<FloatCardSpec>
     */
    public function floatCards(): array
    {
        return [];
    }

    /** Sprite icon of the hero eyebrow pill. */
    public function eyebrowIcon(): string
    {
        return StageNavigation::icon($this->stage());
    }

    /**
     * View used for one feature split: the shared `pages.stages.partials.feature`, or a stage-specific partial
     * (`pages.stages.partials.<slug>.<key>`) when the design differs for that section.
     */
    public function featurePartial(FeatureSpec $feature): string
    {
        return 'pages.stages.partials.feature';
    }

    /** FAQ group in the Faq context (L3-09 seeds `stage-<slug>`; copy is in the lang file until then). */
    public function faqGroup(): string
    {
        return 'stage-'.$this->stage()->value;
    }

    public function slug(): string
    {
        return $this->stage()->value;
    }

    public function page(): StaticPage
    {
        return StaticPage::from('stage.'.$this->slug());
    }

    /** Translation group of the stage copy (`lang/fa/stages/<slug>.php`). */
    public function langGroup(): string
    {
        return 'stages/'.$this->slug();
    }
}

<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Content\Stages\Data\StageNavItemData;

/**
 * The row of six stage pills on every stage page (AUDIT §2.1 `x-layout.stage-nav`): icon + label, the current
 * stage highlighted in its colour. Labels come from the StaticPage registry, so nav, footer and pills agree.
 */
final class StageNavigation
{
    public function __construct(private readonly SiteNavigation $navigation) {}

    /**
     * @return list<StageNavItemData>
     */
    public function items(LifeStage $active): array
    {
        return array_map(function (LifeStage $stage) use ($active): StageNavItemData {
            $page = StaticPage::from('stage.'.$stage->value);

            return new StageNavItemData(
                label: $page->label(),
                url: $this->navigation->url($page),
                icon: self::icon($stage),
                color: $stage->value,
                active: $stage === $active,
            );
        }, LifeStage::cases());
    }

    public static function icon(LifeStage $stage): string
    {
        return match ($stage) {
            LifeStage::Cycle => 'drop',
            LifeStage::Ttc => 'target',
            LifeStage::Pregnancy => 'egg',
            LifeStage::Postpartum => 'person',
            LifeStage::Menopause => 'moon',
            LifeStage::Teen => 'sprout',
        };
    }
}

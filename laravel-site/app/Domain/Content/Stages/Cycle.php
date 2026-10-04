<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;

/**
 * /cycle — «پیگیری چرخه» (design/html/cycle.html). Copy: lang/fa/stages/cycle.php.
 */
final class Cycle extends StageDefinition
{
    public function stage(): LifeStage
    {
        return LifeStage::Cycle;
    }

    public function features(): array
    {
        return [
            new FeatureSpec('prediction', screen: 'cycle-today'),
            new FeatureSpec('logging', screen: 'checklist', tint: 'period'),
            new FeatureSpec('insights', screen: 'checklist', tint: 'period'),
        ];
    }

    public function tools(): array
    {
        return [
            new CardSpec('fertility', 'calculator', 'ttc', 'tools', 'fertility', CardSpec::ON_SITE),
            new CardSpec('pain-diary', 'flame', 'cycle', 'tools', where: CardSpec::IN_APP),
            new CardSpec('pill-reminder', 'pill', 'postpartum', 'tools', where: CardSpec::IN_APP),
        ];
    }

    public function help(): array
    {
        return [
            new CardSpec('clinicians', 'stethoscope', 'postpartum', 'services'),
            new CardSpec('assistant', 'sparkle', 'primary', 'services'),
            new CardSpec('shop', 'store', 'muted', 'shop.index'),
        ];
    }

    public function heroScreen(): string
    {
        return 'cycle-today';
    }

    public function floatCards(): array
    {
        return [
            new FloatCardSpec('next', 'drop', 'period', FloatCardSpec::TOP_START),
            new FloatCardSpec('voice', 'mic', 'lilac', FloatCardSpec::BOTTOM_END),
        ];
    }

    public function eyebrowIcon(): string
    {
        return 'sprout';
    }
}

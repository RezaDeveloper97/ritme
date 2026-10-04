<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;

/**
 * /pregnancy — «بارداری» (design/html/pregnancy.html): week by week, appointments, hospital bag / birth plan /
 * sisemoni. Copy: lang/fa/stages/pregnancy.php.
 */
final class Pregnancy extends StageDefinition
{
    public function stage(): LifeStage
    {
        return LifeStage::Pregnancy;
    }

    public function features(): array
    {
        return [
            new FeatureSpec('weekly', screen: 'pregnancy-week'),
            new FeatureSpec('appointments', screen: 'checklist', tint: 'luteal'),
            new FeatureSpec('birth-prep', screen: 'checklist', tint: 'luteal'),
        ];
    }

    public function tools(): array
    {
        return [
            new CardSpec('hospital-bag', 'bag', 'pregnancy', 'tools', 'hospital-bag', CardSpec::IN_APP),
            new CardSpec('birth-plan', 'file-text', 'primary', 'tools', where: CardSpec::IN_APP),
            new CardSpec('sisemoni', 'check-square', 'primary', 'tools', 'sisemoni', CardSpec::IN_APP),
            new CardSpec('due-date', 'calculator', 'pregnancy', 'tools', 'due-date', CardSpec::ON_SITE),
            new CardSpec('kick-count', 'heart', 'cycle', 'tools', where: CardSpec::IN_APP),
            new CardSpec('contractions', 'clock', 'postpartum', 'tools', where: CardSpec::IN_APP),
        ];
    }

    public function help(): array
    {
        return [
            new CardSpec('clinicians', 'stethoscope', 'postpartum', 'services'),
            new CardSpec('classes', 'users', 'primary', 'directory.index'),
            new CardSpec('insurance', 'shield-check', 'postpartum', 'services'),
        ];
    }

    public function heroScreen(): string
    {
        return 'pregnancy-week';
    }

    public function floatCards(): array
    {
        return [
            new FloatCardSpec('week', 'egg', 'luteal', FloatCardSpec::TOP_START),
            new FloatCardSpec('appointment', 'stethoscope', 'fertile', FloatCardSpec::BOTTOM_END),
        ];
    }

    public function eyebrowIcon(): string
    {
        return 'sprout';
    }
}

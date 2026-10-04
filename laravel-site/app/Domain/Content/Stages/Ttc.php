<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;

/**
 * /ttc — «اقدام به بارداری» (design/html/ttc.html): fertility window, companion, IVF/IUI. Copy: lang/fa/stages/ttc.php.
 */
final class Ttc extends StageDefinition
{
    public function stage(): LifeStage
    {
        return LifeStage::Ttc;
    }

    public function features(): array
    {
        return [
            new FeatureSpec('fertility', screen: 'cycle-today'),
            new FeatureSpec('companion', screen: 'ttc-companion'),
            // Design accent #FFB86B is a rare one-off (AUDIT §3.1) → nearest token, phase-luteal.
            new FeatureSpec('treatment', screen: 'checklist', tint: 'luteal'),
        ];
    }

    public function tools(): array
    {
        return [
            new CardSpec('fertility', 'calculator', 'ttc', 'tools', 'fertility', CardSpec::ON_SITE),
            new CardSpec('preconception', 'check-square', 'primary', 'tools', where: CardSpec::IN_APP),
            new CardSpec('lh-test', 'target', 'ttc', 'tools', where: CardSpec::IN_APP),
        ];
    }

    public function help(): array
    {
        return [
            new CardSpec('clinicians', 'stethoscope', 'postpartum', 'services'),
            new CardSpec('insurance', 'shield-check', 'postpartum', 'services'),
            new CardSpec('lab-results', 'flask', 'pregnancy', 'services'),
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

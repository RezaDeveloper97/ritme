<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;

/**
 * /teen — «نوجوان و والدین» (design/html/teen.html). Copy: lang/fa/stages/teen.php (mock-screen copy under its
 * `mock` key, read by mock/screens/teen-*). The teen version has no shop or services, so there is no help block
 * (help() stays empty — AUDIT §1); the second split is the parent section «مادر در جریان است، نه ناظر».
 * Mock screens are teen-specific on purpose: no fertile window, no partner copy.
 */
final class Teen extends StageDefinition
{
    public function stage(): LifeStage
    {
        return LifeStage::Teen;
    }

    public function features(): array
    {
        return [
            new FeatureSpec('simple', screen: 'teen-today'),
            new FeatureSpec('mother', screen: 'teen-mother'),
        ];
    }

    public function tools(): array
    {
        return [
            new CardSpec('school-kit', 'bag', 'teen', 'tools', where: CardSpec::IN_APP),
            new CardSpec('first-period', 'file-text', 'primary', 'tools', where: CardSpec::IN_APP),
            new CardSpec('calendar', 'calendar', 'cycle', 'tools', where: CardSpec::IN_APP),
        ];
    }

    public function heroScreen(): string
    {
        return 'teen-today';
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

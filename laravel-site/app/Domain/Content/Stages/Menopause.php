<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;

/**
 * /menopause — «یائسگی» (design/html/menopause.html). Copy: lang/fa/stages/menopause.php (mock-screen copy under
 * its `mock` key, `mock.menopause_status` for mock/screens/menopause-status).
 */
final class Menopause extends StageDefinition
{
    public function stage(): LifeStage
    {
        return LifeStage::Menopause;
    }

    public function features(): array
    {
        return [
            new FeatureSpec('symptoms', screen: 'menopause-status'),
            new FeatureSpec('checkups', screen: 'checklist', tint: 'lilac'),
            new FeatureSpec('treatment', screen: 'checklist', tint: 'lilac'),
        ];
    }

    public function tools(): array
    {
        return [
            new CardSpec('hot-flash', 'flame', 'cycle', 'tools', where: CardSpec::IN_APP),
            new CardSpec('report', 'file-text', 'primary', 'tools', where: CardSpec::IN_APP),
            new CardSpec('checkups', 'check-square', 'primary', 'tools', where: CardSpec::IN_APP),
        ];
    }

    public function help(): array
    {
        return [
            new CardSpec('clinicians', 'stethoscope', 'postpartum', 'services'),
            new CardSpec('pelvic-floor', 'users', 'primary', 'services'),
            new CardSpec('insurance', 'shield-check', 'postpartum', 'services'),
        ];
    }

    public function heroScreen(): string
    {
        return 'menopause-status';
    }

    public function floatCards(): array
    {
        return [
            new FloatCardSpec('hot-flash', 'flame', 'period', FloatCardSpec::TOP_START),
            new FloatCardSpec('mammogram', 'shield-check', 'fertile', FloatCardSpec::BOTTOM_END),
        ];
    }

    public function eyebrowIcon(): string
    {
        return 'sprout';
    }
}

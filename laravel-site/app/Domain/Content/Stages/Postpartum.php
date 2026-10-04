<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

use App\Domain\Blog\Enums\LifeStage;

/**
 * /postpartum — «پس از زایمان و کودک» (design/html/postpartum.html). Copy: lang/fa/stages/postpartum.php
 * (mock-screen copy under its `mock` key, read by mock/screens/postpartum-*).
 */
final class Postpartum extends StageDefinition
{
    public function stage(): LifeStage
    {
        return LifeStage::Postpartum;
    }

    public function features(): array
    {
        return [
            new FeatureSpec('recovery', screen: 'postpartum-baby'),
            new FeatureSpec('baby', screen: 'checklist', tint: 'fertile'),
            new FeatureSpec('family', screen: 'postpartum-partner'),
        ];
    }

    public function tools(): array
    {
        return [
            new CardSpec('safe-home', 'check-square', 'primary', 'tools', where: CardSpec::IN_APP),
            new CardSpec('buying-guide', 'file-text', 'postpartum', 'tools', where: CardSpec::IN_APP),
            new CardSpec('vaccines', 'bell', 'primary', 'tools', where: CardSpec::IN_APP),
        ];
    }

    public function help(): array
    {
        return [
            new CardSpec('directory', 'map', 'primary', 'directory.index'),
            new CardSpec('clinicians', 'stethoscope', 'postpartum', 'services'),
            new CardSpec('shop', 'store', 'muted', 'shop.index'),
        ];
    }

    public function heroScreen(): string
    {
        return 'postpartum-baby';
    }

    public function floatCards(): array
    {
        return [
            new FloatCardSpec('vaccine', 'syringe', 'luteal', FloatCardSpec::TOP_START),
            new FloatCardSpec('weight', 'chart-bar', 'fertile', FloatCardSpec::BOTTOM_END),
        ];
    }

    public function eyebrowIcon(): string
    {
        return 'sprout';
    }
}

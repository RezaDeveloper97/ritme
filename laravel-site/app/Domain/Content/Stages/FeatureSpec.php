<?php

declare(strict_types=1);

namespace App\Domain\Content\Stages;

/**
 * Structure of one feature split (eyebrow, h2, text, check list + phone mock-up). Copy is `features.<key>` of the
 * stage's lang file; `screen` is a partial in resources/views/pages/stages/mock/screens; `tint` is the mock
 * screen's accent (a cycle-phase or stage colour key, see mock/screens/checklist.blade.php).
 */
final readonly class FeatureSpec
{
    public function __construct(
        public string $key,
        public string $screen = 'checklist',
        public string $tint = 'period',
    ) {}
}

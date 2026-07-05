<?php

namespace App\Services\HomePage\Sections;

use App\Models\Affirmation;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 11 — daily affirmation (تأکید روزانه), chosen deterministically per day.
 */
class AffirmationSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'affirmation';
    }

    public function order(): int
    {
        return 110;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $affirmations = Affirmation::query()
            ->active()
            ->forPhase($context->phase())
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get();

        if ($affirmations->isEmpty()) {
            return null;
        }

        $affirmation = $affirmations[$context->date->dayOfYear % $affirmations->count()];

        return new HomeSection(
            key: $this->key(),
            type: 'affirmation',
            title: null,
            data: [
                'id' => $affirmation->id,
                'text' => $affirmation->localized('text', $context->locale),
                'icon' => 'sparkle',
            ],
            order: $this->order(),
        );
    }
}

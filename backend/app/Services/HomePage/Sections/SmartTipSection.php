<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 10 — "نکته هوشمند": a single personalized paragraph. Prefers the
 * unified MessageSystem (long message / correlations); falls back to the cycle
 * engine's localized text flags so the card is never empty when cycle data exists.
 */
class SmartTipSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'smart_tip';
    }

    public function order(): int
    {
        return 100;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $text = null;
        $source = null;

        $messages = $context->messages();
        if ($messages) {
            $primary = $messages->primaryMessage;
            $text = $primary['long_message'] ?? $primary['short_message'] ?? null;
            $source = $text ? 'message_engine' : null;

            if (!$text && !empty($messages->correlations)) {
                $text = $messages->correlations[0]['insight_message'] ?? null;
                $source = $text ? 'correlation' : null;
            }
        }

        if (!$text) {
            $flags = $context->cycleData()['text_flags'] ?? [];
            foreach (['phase_info', 'probability_message', 'pms_warning', 'fertility_status'] as $flagKey) {
                if (isset($flags[$flagKey])) {
                    $text = $this->pick($flags[$flagKey], $context->locale);
                    $source = 'cycle_engine';
                    break;
                }
            }
        }

        if (!$text) {
            return null;
        }

        return new HomeSection(
            key: $this->key(),
            type: 'smart_tip',
            title: $context->t('نکته هوشمند', 'Smart tip'),
            data: [
                'text' => $text,
                'source' => $source,
            ],
            order: $this->order(),
        );
    }
}

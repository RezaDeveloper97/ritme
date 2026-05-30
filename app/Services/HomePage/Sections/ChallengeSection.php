<?php

namespace App\Services\HomePage\Sections;

use App\Models\Challenge;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 7 — "چالش امروز": one daily challenge chosen deterministically per
 * day, with completion state and an encouraging status message.
 */
class ChallengeSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'challenge';
    }

    public function order(): int
    {
        return 70;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $challenges = Challenge::query()
            ->active()
            ->forPhase($context->phase())
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get();

        if ($challenges->isEmpty()) {
            return null;
        }

        // Deterministic pick: stable for the whole day, rotates across days.
        $challenge = $challenges[$context->date->dayOfYear % $challenges->count()];

        $isCompleted = $context->user->challengeCompletions()
            ->where('challenge_id', $challenge->id)
            ->whereDate('completion_date', $context->date)
            ->exists();

        $statusMessage = $isCompleted
            ? $context->t('عالی! امروز این چالش رو انجام دادی، با همین روند ادامه بده', 'Great! You completed today\'s challenge, keep it up')
            : $challenge->localized('description', $context->locale);

        return new HomeSection(
            key: $this->key(),
            type: 'challenge',
            title: $context->t('چالش امروز', "Today's challenge"),
            data: [
                'id' => $challenge->id,
                'title' => $challenge->localized('title', $context->locale),
                'description' => $challenge->localized('description', $context->locale),
                'difficulty' => $challenge->difficulty,
                'is_completed' => $isCompleted,
                'status_message' => $statusMessage,
            ],
            order: $this->order(),
        );
    }
}

<?php

namespace App\Services\HomePage\Sections;

use App\Services\Challenges\DailyChallengeService;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 7 — "چالش امروز": one admin-authored task chosen per user/day by
 * {@see DailyChallengeService} (cycle day + recent-log signal + no repeats),
 * with nothing attached to it but a "done" flag.
 */
class ChallengeSection extends AbstractHomeSection
{
    private DailyChallengeService $challenges;

    public function __construct(?DailyChallengeService $challenges = null)
    {
        $this->challenges = $challenges ?? new DailyChallengeService;
    }

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
        $payload = $this->challenges->payload(
            user: $context->user,
            date: $context->date,
            locale: $context->locale,
            cycleDay: $context->cycleDay(),
            recentLogs: $context->recentLogs(3),
        );

        if ($payload === null) {
            return null;
        }

        return new HomeSection(
            key: $this->key(),
            type: 'challenge',
            title: $context->t('چالش امروز', "Today's challenge"),
            data: $payload,
            order: $this->order(),
        );
    }
}

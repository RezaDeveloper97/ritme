<?php

namespace App\Services\HomePage\Sections;

use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;
use App\Services\HomePage\Support\MoonPhase;

/**
 * Section 1 — app header: identity, mode, premium badge, date, moon phase and
 * the notification bell unread count.
 */
class HeaderSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'header';
    }

    public function order(): int
    {
        return 10;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $name = $context->user->name;
        $unread = $context->user->appNotifications()->unread()->count();

        return new HomeSection(
            key: $this->key(),
            type: 'header',
            title: null,
            data: [
                'app_name' => 'ریتمی',
                'tagline' => $context->t('همراه سلامتی شما', 'Your health companion'),
                'greeting' => $name
                    ? $context->t("سلام {$name}", "Hi {$name}")
                    : $context->t('سلام', 'Hi'),
                'user' => [
                    'id' => $context->user->id,
                    'name' => $name,
                ],
                'mode' => $context->mode->value,
                'mode_label' => $context->mode->label($context->locale),
                'is_premium' => $context->profile?->isPremium() ?? false,
                'date' => $context->date->toDateString(),
                'weekday_iso' => $context->date->dayOfWeekIso,
                'moon_phase' => MoonPhase::forDate($context->date, $context->locale),
                'notifications' => [
                    'unread_count' => $unread,
                    'has_unread' => $unread > 0,
                ],
            ],
            order: $this->order(),
        );
    }
}

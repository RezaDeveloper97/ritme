<?php

namespace App\Services\HomePage\Sections;

use App\Enums\ReminderType;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 8 — "یادآور دکتر": the next upcoming doctor appointment reminder.
 * Omitted entirely when the user has none.
 */
class DoctorReminderSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'doctor_reminder';
    }

    public function order(): int
    {
        return 80;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $reminder = $context->user->reminders()
            ->active()
            ->ofType(ReminderType::DOCTOR->value)
            ->relevantOn($context->date)
            ->orderBy('scheduled_at')
            ->first();

        if (!$reminder) {
            return null;
        }

        return new HomeSection(
            key: $this->key(),
            type: 'doctor_reminder',
            title: $context->t('یادآور دکتر', 'Doctor reminder'),
            data: [
                'id' => $reminder->id,
                'title' => $reminder->title,
                'subtitle' => $reminder->subtitle,
                'notes' => $reminder->notes,
                'scheduled_at' => $reminder->scheduled_at?->toIso8601String(),
                'date' => $reminder->scheduled_at?->toDateString(),
                'time' => $reminder->scheduled_at?->format('H:i'),
                'meta' => $reminder->meta,
                'icon' => ReminderType::DOCTOR->icon(),
            ],
            order: $this->order(),
        );
    }
}

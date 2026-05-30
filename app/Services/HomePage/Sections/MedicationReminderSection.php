<?php

namespace App\Services\HomePage\Sections;

use App\Enums\ReminderType;
use App\Services\HomePage\HomeContext;
use App\Services\HomePage\HomeSection;

/**
 * Section 9 — "یادآور دارو": today's medication reminder (recurring or one-off).
 * Omitted entirely when the user has none.
 */
class MedicationReminderSection extends AbstractHomeSection
{
    public function key(): string
    {
        return 'medication_reminder';
    }

    public function order(): int
    {
        return 90;
    }

    public function build(HomeContext $context): ?HomeSection
    {
        $reminder = $context->user->reminders()
            ->active()
            ->ofType(ReminderType::MEDICATION->value)
            ->relevantOn($context->date)
            // Order by time-of-day: normalize the one-off datetime to a time so
            // it is comparable with recurring reminders' recurrence_time.
            ->orderByRaw('COALESCE(recurrence_time, TIME(scheduled_at))')
            ->first();

        if (!$reminder) {
            return null;
        }

        $time = $reminder->recurrence_time
            ? substr($reminder->recurrence_time, 0, 5)
            : $reminder->scheduled_at?->format('H:i');

        return new HomeSection(
            key: $this->key(),
            type: 'medication_reminder',
            title: $context->t('یادآور دارو', 'Medication reminder'),
            data: [
                'id' => $reminder->id,
                'title' => $reminder->title,
                'subtitle' => $reminder->subtitle,
                'notes' => $reminder->notes,
                'time' => $time,
                'recurrence' => $reminder->recurrence,
                'meta' => $reminder->meta,
                'icon' => ReminderType::MEDICATION->icon(),
            ],
            order: $this->order(),
        );
    }
}

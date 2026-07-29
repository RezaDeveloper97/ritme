<?php

namespace App\Services\HealthEngine;

use App\Enums\CycleSubphase;
use App\Enums\DataStatus;
use App\Enums\FertilityLevel;
use Carbon\Carbon;

/**
 * Turns a day's cycle state into a render-ready {@see DailyCard} using the spec's
 * rule-based title chain (§"final title logic" + §"card content by day type"). The
 * order of the checks encodes the data-status priority actual > incomplete >
 * needs_confirmation > predicted (§18), so a real logged period is never overridden
 * by a prediction. Copy is deliberately non-diagnostic (CLAUDE §11).
 *
 * $predictedNextStart is the period the user is waiting for — the last *confirmed*
 * start plus the effective cycle length, NOT rolled forward — so "period due /
 * overdue" states fire correctly. $ovulationDate is that cycle's estimate.
 */
class DailyCardBuilder
{
    private const ACTION_LOG_SYMPTOMS = 'log_symptoms';

    private const ACTION_LOG_PERIOD_END = 'log_period_end';

    private const ACTION_LOG_PERIOD_START = 'log_period_start';

    private const ACTION_CONFIRM_PERIOD_START = 'confirm_period_start';

    private const ACTION_CONFIRM_PERIOD_END = 'confirm_period_end';

    private const ACTION_PERIOD_NOT_STARTED = 'period_not_started';

    private const ACTION_STILL_BLEEDING = 'still_bleeding';

    private const ACTION_VIEW_DETAILS = 'view_details';

    private const ACTION_SET_REMINDER = 'set_reminder';

    private const ACTION_COMPLETE_DAY = 'complete_day';

    public function build(
        Carbon $selectedDate,
        Carbon $today,
        int $cycleDay,
        CycleSubphase $subphase,
        Carbon $predictedNextStart,
        OpenPeriodState $openPeriod,
        ?int $loggedPeriodDay,
        bool $loggedPeriodClosed,
        ?Carbon $estimatedOvulation = null,
        string $locale = 'en',
    ): DailyCard {
        // Set once for the whole card build so every action()/t()/num() below resolves
        // against the same locale regardless of call order.
        $this->currentLocale = $locale;

        $sel = $selectedDate->copy()->startOfDay();
        $ref = $today->copy()->startOfDay();
        $isFuture = $sel->gt($ref);
        $isToday = $sel->eq($ref);

        // 1) Inside a logged period → actual (closed) or incomplete (still open).
        if ($loggedPeriodDay !== null) {
            return $loggedPeriodClosed
                ? $this->actualPeriodDay($cycleDay, $loggedPeriodDay, $locale)
                : $this->openPeriodDay($cycleDay, $loggedPeriodDay, $openPeriod, $isToday, $locale);
        }

        $daysUntilPeriod = (int) $sel->diffInDays($predictedNextStart, false);

        // 2) A predicted period is due or overdue with nothing logged (today/past only,
        //    never the future — §"future days must not show 'not started'").
        if (! $isFuture) {
            if ($daysUntilPeriod === 0) {
                return $this->periodDue($cycleDay, $locale);
            }
            if ($daysUntilPeriod < 0) {
                return $this->periodOverdue(-$daysUntilPeriod, $locale);
            }

            // 2b) A *predicted* bleeding day (any day of a rolled-forward predicted
            //     period window) with nothing logged. Without this the card fell
            //     through to the generic "day X of your cycle" and offered no way to
            //     say "it started here" — exactly the day the user needs it most.
            if ($this->isMenstrualSubphase($subphase)) {
                return $this->predictedPeriodDay($cycleDay, $isToday, $locale);
            }
        }

        // 3) Otherwise it's a prediction: a period countdown, a fertile/ovulation window
        //    (driven by the sub-phase, so the title, fertility level and phase always
        //    agree — the ovulation *date* used elsewhere can be off by a day), or a
        //    plain cycle day.
        return $this->predictedDay($cycleDay, $subphase, $daysUntilPeriod, $isFuture, $isToday, $sel, $estimatedOvulation, $locale);
    }

    private function actualPeriodDay(int $cycleDay, int $periodDay, string $locale): DailyCard
    {
        // A closed logged period already has an end date, so no "has your period
        // ended?" here (that belongs to the open-period card). Editing/removing the
        // range is offered by the calendar's own logged-period controls.
        return new DailyCard(
            title: $this->t($locale, 'روز '.$this->num($periodDay, $locale).' پریود', 'Day '.$periodDay.' of your period'),
            subtitle: $this->t($locale,
                'این روز به‌عنوان روز پریود ثبت شده است. مراقبت ملایم می‌تواند کمک‌کننده باشد.',
                'This day is logged as a period day. Gentle self-care can help.'),
            dataStatus: DataStatus::ACTUAL,
            fertilityLevel: FertilityLevel::LOW,
            fertilityLabel: FertilityLevel::LOW->label($locale),
            badges: $this->badges($cycleDay, DataStatus::ACTUAL, $locale),
            primaryAction: $this->action(self::ACTION_LOG_SYMPTOMS, 'ثبت علائم امروز', "Log today's symptoms"),
            secondaryActions: [$this->detailsAction(true, true, $locale)],
        );
    }

    private function openPeriodDay(int $cycleDay, int $periodDay, OpenPeriodState $openPeriod, bool $isToday, string $locale): DailyCard
    {
        // Past the expected length (evaluated for today) → stop counting days and ask
        // for the end; otherwise keep showing "day X of period" (§4).
        if ($isToday && $openPeriod->endOverdue) {
            return new DailyCard(
                title: $this->t($locale, 'پایان پریود هنوز ثبت نشده', 'Period end not logged yet'),
                subtitle: $this->t($locale,
                    'برای دقیق‌تر شدن تقویم، مشخص کن خون‌ریزی چه زمانی تمام شده است.',
                    'To keep your calendar accurate, tell us when the bleeding stopped.'),
                dataStatus: DataStatus::INCOMPLETE,
                fertilityLevel: FertilityLevel::LOW,
                fertilityLabel: FertilityLevel::LOW->label($locale),
                badges: $this->badges($cycleDay, DataStatus::INCOMPLETE, $locale),
                primaryAction: $this->action(self::ACTION_LOG_PERIOD_END, 'ثبت پایان پریود', 'Log period end'),
                secondaryActions: [
                    $this->action(self::ACTION_STILL_BLEEDING, 'هنوز ادامه دارد', 'Still ongoing'),
                    $this->detailsAction(false, true, $locale),
                ],
            );
        }

        return new DailyCard(
            title: $this->t($locale, 'روز '.$this->num($periodDay, $locale).' پریود', 'Day '.$periodDay.' of your period'),
            subtitle: $this->t($locale,
                'پریود هنوز باز است. هر وقت خون‌ریزی تمام شد، پایان آن را ثبت کن.',
                'This period is still open. Log its end once the bleeding stops.'),
            dataStatus: DataStatus::INCOMPLETE,
            fertilityLevel: FertilityLevel::LOW,
            fertilityLabel: FertilityLevel::LOW->label($locale),
            badges: $this->badges($cycleDay, DataStatus::INCOMPLETE, $locale),
            primaryAction: $this->action(self::ACTION_LOG_SYMPTOMS, 'ثبت علائم امروز', "Log today's symptoms"),
            secondaryActions: [
                $this->action(self::ACTION_LOG_PERIOD_END, 'ثبت پایان پریود', 'Log period end'),
                $this->detailsAction(! $isToday, $isToday, $locale),
            ],
        );
    }

    private function periodDue(int $cycleDay, string $locale): DailyCard
    {
        return new DailyCard(
            title: $this->t($locale, 'موعد پریود پیش‌بینی‌شده رسیده', 'Your predicted period is due'),
            subtitle: $this->t($locale,
                'تا وقتی شروع پریود را ثبت نکنی، این وضعیت فقط یک پیش‌بینی است.',
                'Until you log the start, this is only a prediction.'),
            dataStatus: DataStatus::NEEDS_CONFIRMATION,
            fertilityLevel: FertilityLevel::LOW,
            fertilityLabel: FertilityLevel::LOW->label($locale),
            badges: $this->badges($cycleDay, DataStatus::NEEDS_CONFIRMATION, $locale),
            primaryAction: $this->action(self::ACTION_CONFIRM_PERIOD_START, 'پریودت شروع شده؟', 'Has your period started?'),
            secondaryActions: [
                $this->action(self::ACTION_PERIOD_NOT_STARTED, 'هنوز شروع نشده', 'Not started yet'),
                $this->detailsAction(false, true, $locale),
            ],
        );
    }

    /** Predicted bleeding days are the ones a user confirms — never a silent dead card. */
    private function isMenstrualSubphase(CycleSubphase $subphase): bool
    {
        return in_array($subphase, [
            CycleSubphase::MENSTRUATION,
            CycleSubphase::MENSTRUAL,
            CycleSubphase::MENSTRUAL_POSSIBLE,
        ], true);
    }

    private function predictedPeriodDay(int $cycleDay, bool $isToday, string $locale): DailyCard
    {
        return new DailyCard(
            title: $this->t($locale,
                'روز '.$this->num($cycleDay, $locale).' پریود پیش‌بینی‌شده',
                'Day '.$cycleDay.' of your predicted period'),
            subtitle: $this->t($locale,
                'این فقط یک پیش‌بینی است. اگر پریودت از این روز شروع شده، ثبتش کن تا تقویم دقیق‌تر شود.',
                'This is only a prediction. If your period started on this day, log it to keep your calendar accurate.'),
            dataStatus: DataStatus::NEEDS_CONFIRMATION,
            fertilityLevel: FertilityLevel::LOW,
            fertilityLabel: FertilityLevel::LOW->label($locale),
            badges: $this->badges($cycleDay, DataStatus::NEEDS_CONFIRMATION, $locale),
            primaryAction: $this->action(self::ACTION_CONFIRM_PERIOD_START, 'پریودم از این روز شروع شد', 'My period started this day'),
            secondaryActions: [
                $this->action(self::ACTION_PERIOD_NOT_STARTED, 'شروع نشده بود', "It didn't start"),
                $this->detailsAction(! $isToday, $isToday, $locale),
            ],
        );
    }

    private function periodOverdue(int $daysSince, string $locale): DailyCard
    {
        $title = $daysSince <= 2
            ? $this->t($locale, $this->num($daysSince, $locale).' روز از موعد پیش‌بینی‌شده گذشته', $daysSince.' day(s) past the predicted date')
            : $this->t($locale, 'هنوز شروع پریود ثبت نشده', 'Period start not logged yet');

        return new DailyCard(
            title: $title,
            subtitle: $this->t($locale,
                'چند روز اختلاف با پیش‌بینی طبیعی است، مخصوصاً اگر زمان تخمک‌گذاری جابه‌جا شده باشد.',
                'A few days off from the prediction is normal, especially if ovulation shifted.'),
            dataStatus: DataStatus::NEEDS_CONFIRMATION,
            fertilityLevel: FertilityLevel::LOW,
            fertilityLabel: FertilityLevel::LOW->label($locale),
            badges: [$this->statusBadge(DataStatus::NEEDS_CONFIRMATION, $locale)],
            primaryAction: $this->action(self::ACTION_LOG_PERIOD_START, 'ثبت شروع پریود', 'Log period start'),
            secondaryActions: [
                $this->action(self::ACTION_PERIOD_NOT_STARTED, 'هنوز شروع نشده', 'Not started yet'),
                $this->detailsAction(true, false, $locale),
            ],
        );
    }

    private function predictedDay(
        int $cycleDay,
        CycleSubphase $subphase,
        int $daysUntilPeriod,
        bool $isFuture,
        bool $isToday,
        Carbon $selectedDate,
        ?Carbon $estimatedOvulation,
        string $locale,
    ): DailyCard {
        // A plain day should headline its *next* event, not "day X of your cycle":
        // days until the fertile window opens (ovulation − 5, matching the window
        // shown everywhere else), used when the window is still ahead of this day.
        $daysToFertile = $estimatedOvulation !== null
            ? (int) $selectedDate->diffInDays($estimatedOvulation->copy()->subDays(5)->startOfDay(), false)
            : null;

        [$title, $subtitle] = match (true) {
            $daysUntilPeriod >= 3 && $daysUntilPeriod <= 7 => [
                $this->t($locale, 'حدود '.$this->num($daysUntilPeriod, $locale).' روز تا پریود بعدی', 'About '.$daysUntilPeriod.' days to your next period'),
                $this->t($locale, 'در روزهای پایانی چرخه، انرژی، خواب یا خلق ممکن است تغییر کند.', 'Late in the cycle, energy, sleep or mood may shift for some.'),
            ],
            $daysUntilPeriod === 2 => [
                $this->t($locale, 'حدود ۲ روز تا پریود بعدی', 'About 2 days to your next period'),
                $this->t($locale, 'پریود احتمالاً نزدیک است.', 'Your period is likely close.'),
            ],
            $daysUntilPeriod === 1 => [
                $this->t($locale, 'احتمالاً پریود نزدیک است', 'Your period is likely near'),
                $this->t($locale, 'ممکن است به‌زودی خون‌ریزی شروع شود.', 'Bleeding may start soon.'),
            ],
            $subphase === CycleSubphase::FERTILE_RISING => [
                $this->t($locale, 'پنجره باروری ممکن است نزدیک باشد', 'Your fertile window may be approaching'),
                $this->fertileNote($locale),
            ],
            $subphase === CycleSubphase::HIGH_FERTILITY => [
                $this->t($locale, 'احتمالاً در پنجره باروری هستی', "You're likely in your fertile window"),
                $this->fertileNote($locale),
            ],
            $subphase === CycleSubphase::OVULATION_LIKELY => [
                $this->t($locale, 'تخمک‌گذاری احتمالی نزدیک است', 'Ovulation is likely near'),
                $this->fertileNote($locale),
            ],
            $subphase === CycleSubphase::POST_OVULATION => [
                $this->t($locale, 'احتمالاً از پنجره باروری عبور کرده‌ای', "You've likely passed your fertile window"),
                $this->fertileNote($locale),
            ],
            // Plain in-between days headline the countdown to their next event —
            // the fertile window if it's still ahead, otherwise the next period —
            // instead of the uninformative "day X of your cycle".
            $daysToFertile !== null && $daysToFertile > 0 => [
                $this->t($locale, $this->num($daysToFertile, $locale).' روز تا پنجره باروری', $daysToFertile.' day(s) to your fertile window'),
                $this->t($locale, 'بر اساس پیش‌بینی چرخه، پنجره باروری از حدود '.$this->num($daysToFertile, $locale).' روز دیگر شروع می‌شود.', 'Based on your cycle prediction, your fertile window starts in about '.$daysToFertile.' day(s).'),
            ],
            $daysUntilPeriod > 7 => [
                $this->t($locale, $this->num($daysUntilPeriod, $locale).' روز تا پریود بعدی', $daysUntilPeriod.' day(s) to your next period'),
                $this->t($locale, 'بر اساس پیش‌بینی، پریود بعدی حدود '.$this->num($daysUntilPeriod, $locale).' روز دیگر شروع می‌شود.', 'Based on the prediction, your next period starts in about '.$daysUntilPeriod.' day(s).'),
            ],
            $isFuture && $daysUntilPeriod <= 0 => [
                $this->t($locale, 'روز '.$this->num($cycleDay, $locale).' چرخه پیش‌بینی‌شده', 'Day '.$cycleDay.' of the predicted cycle'),
                $this->t($locale, 'این یک پیش‌بینی تقویمی است و ممکن است تغییر کند.', 'This is a calendar prediction and may change.'),
            ],
            default => [
                $this->t($locale, 'روز '.$this->num($cycleDay, $locale).' چرخه', 'Day '.$cycleDay.' of your cycle'),
                $this->t($locale, 'یک روز معمول چرخه بر اساس پیش‌بینی.', 'A typical predicted cycle day.'),
            ],
        };

        $fertility = $subphase->fertilityLevel();

        return new DailyCard(
            title: $title,
            subtitle: $subtitle,
            dataStatus: DataStatus::PREDICTED,
            fertilityLevel: $fertility,
            fertilityLabel: $fertility->label($locale),
            badges: $this->badges($cycleDay, DataStatus::PREDICTED, $locale),
            primaryAction: $this->predictedPrimaryAction($isFuture, $isToday, $locale),
            secondaryActions: [$this->detailsAction(! $isToday && ! $isFuture, $isToday, $locale)],
        );
    }

    /** Future days can't log real symptoms — offer a reminder; past days offer completion. */
    private function predictedPrimaryAction(bool $isFuture, bool $isToday, string $locale): array
    {
        if ($isFuture) {
            return $this->action(self::ACTION_SET_REMINDER, 'یادآوری برای این روز', 'Set a reminder');
        }
        if ($isToday) {
            return $this->action(self::ACTION_LOG_SYMPTOMS, 'ثبت علائم امروز', "Log today's symptoms");
        }

        return $this->action(self::ACTION_COMPLETE_DAY, 'تکمیل اطلاعات این روز', 'Complete this day');
    }

    private function fertileNote(string $locale): string
    {
        return $this->t($locale,
            'در این بازه احتمال باروری بر اساس پیش‌بینی چرخه بالاتر است، اما زمان واقعی تخمک‌گذاری می‌تواند متفاوت باشد.',
            'Fertility is estimated higher here based on your cycle, but real ovulation timing can differ.');
    }

    /**
     * @return array{type: string, label: string}
     */
    private function detailsAction(bool $isPast, bool $isToday, string $locale): array
    {
        [$fa, $en] = match (true) {
            $isPast => ['جزئیات این روز', "This day's details"],
            $isToday => ['جزئیات امروز', "Today's details"],
            default => ['جزئیات پیش‌بینی', 'Prediction details'],
        };

        return $this->action(self::ACTION_VIEW_DETAILS, $fa, $en);
    }

    /**
     * @return array<string>
     */
    private function badges(int $cycleDay, DataStatus $status, string $locale): array
    {
        $cycleBadge = $this->t($locale, 'روز '.$this->num($cycleDay, $locale).' چرخه', 'Day '.$cycleDay);

        return [$cycleBadge, $this->statusBadge($status, $locale)];
    }

    private function statusBadge(DataStatus $status, string $locale): string
    {
        return $status->label($locale);
    }

    /**
     * @return array{type: string, label: string}
     */
    private function action(string $type, string $fa, string $en): array
    {
        // Labels resolve against the caller's locale via t(); store the resolved string.
        return ['type' => $type, 'label' => $this->currentLocale === 'fa' ? $fa : $en];
    }

    private string $currentLocale = 'en';

    private function t(string $locale, string $fa, string $en): string
    {
        $this->currentLocale = $locale;

        return $locale === 'fa' ? $fa : $en;
    }

    private function num(int $value, string $locale): string
    {
        if ($locale !== 'fa') {
            return (string) $value;
        }

        return strtr((string) $value, ['0' => '۰', '1' => '۱', '2' => '۲', '3' => '۳', '4' => '۴', '5' => '۵', '6' => '۶', '7' => '۷', '8' => '۸', '9' => '۹']);
    }
}

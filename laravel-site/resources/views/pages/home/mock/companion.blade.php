{{-- Phone screen «همدم»: the companion's view of a shared cycle (family split). Decorative. --}}
<div class="flex flex-col gap-3 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ __('home.mock.companion_of') }}</span>
    <span class="font-display text-d-sm text-on-night">{{ __('home.mock.greeting') }}</span>
    <div class="flex flex-col gap-2 rounded-2xl bg-night-card p-3.5">
        <div class="text-[11px] font-bold text-on-night-muted">{{ __('home.mock.cycle_of') }}</div>
        <b class="font-display text-4xl text-on-night">{{ __('home.mock.cycle_day') }}</b>
        <div class="h-2 rounded-full bg-[linear-gradient(to_left,var(--color-phase-period)_0_17%,var(--color-lilac)_17%_45%,var(--color-phase-fertile)_45%_60%,var(--color-phase-luteal)_60%)]"></div>
    </div>
    <div class="rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">{{ __('home.mock.suggestion') }}</div>
    <div class="flex justify-between rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">
        <span>{{ __('home.mock.reminder') }}</span>
        <span class="text-phase-fertile">{{ __('home.mock.reminder_time') }}</span>
    </div>
</div>

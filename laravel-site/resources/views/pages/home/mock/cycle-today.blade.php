{{-- Phone screen «امروز»: cycle ring + fertile window + voice prompt (hero, uncertainty split). Decorative. --}}
<div class="flex flex-col items-center gap-3.5 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ __('home.mock.today') }}</span>
    <div class="flex size-45 items-center justify-center rounded-full bg-[conic-gradient(var(--color-stage-cycle)_0_17%,var(--color-lilac)_17%_45%,var(--color-phase-fertile)_45%_60%,var(--color-phase-luteal)_60%_100%)]">
        <div class="flex size-36.5 flex-col items-center justify-center gap-1 rounded-full bg-night">
            <span class="font-display text-[30px] text-on-night">{{ __('home.mock.days_left') }}</span>
            <span class="text-[11px] font-bold text-on-night-muted">{{ __('home.mock.until_period') }}</span>
        </div>
    </div>
    <div class="flex w-full justify-between rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">
        <span>{{ __('home.mock.fertile_window') }}</span>
        <span class="text-phase-fertile">{{ __('home.mock.fertile_when') }}</span>
    </div>
    <div class="flex w-full items-center gap-2 rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">
        <x-icon name="mic" class="size-4 text-lilac"/>
        {{ __('home.mock.how_was_today') }}
    </div>
</div>

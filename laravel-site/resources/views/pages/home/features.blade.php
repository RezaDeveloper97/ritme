{{--
    #features — anchor of the header item «امکانات» (/#features). Five value blocks, not a feature grid: three
    cycle-care cards (desktop 3 across, phones stacked), then «وقتی مسیرت تغییر می‌کند» with two wide cards for
    TTC and pregnancy (desktop 2 across, phones stacked). Each mockup is decorative (aria-hidden) and token-built.
--}}
@php
    $t = static fn (string $key): mixed => __("home.features.{$key}");
    // Two calendar weeks (8–21 Aban); 12–16 is the probable period, 9 is today.
    $calendarDays = range(8, 21);
    $fertilityBars = ['h-3.5 bg-rose-line', 'h-4.5 bg-sand-line', 'h-7.5 bg-amber', 'h-11 bg-stage-ttc', 'h-7.5 bg-amber', 'h-4.5 bg-sand-line', 'h-3.5 bg-rose-line'];
@endphp
<section id="features" aria-labelledby="home-features-title" class="flex flex-col gap-10 bg-canvas px-30 py-24 max-xl:px-6 max-lg:gap-6 max-lg:px-5 max-lg:py-14">
    <x-site.heading id="home-features-title" :eyebrow="$t('eyebrow')" :title="$t('title')" :lead="$t('lead')"/>

    <ul class="m-0 grid list-none grid-cols-3 gap-5 p-0 max-lg:grid-cols-1 max-lg:gap-3">
        <x-home.feature-tile tint="bg-danger-soft" :title="$t('prediction.title')" :text="$t('prediction.text')" :backing="$t('prediction.backing')">
            <div class="flex w-full flex-col gap-2.5 rounded-xl bg-surface p-3.5 shadow-tile">
                <div class="flex items-center justify-between gap-2">
                    <span class="flex flex-col">
                        <span class="text-xs font-bold text-muted">{{ $t('prediction.label') }}</span>
                        <b class="text-md text-ink">{{ $t('prediction.range') }}</b>
                    </span>
                    <span class="rounded-lg bg-lavender px-2 py-0.5 text-xs font-extrabold text-primary">{{ $t('prediction.confidence') }}</span>
                </div>
                <div class="grid grid-cols-7 gap-1">
                    @foreach ($t('prediction.weekdays') as $weekday)
                        <span class="text-center text-[10px] font-bold text-muted">{{ $weekday }}</span>
                    @endforeach
                    @foreach ($calendarDays as $day)
                        <span @class([
                            'flex h-5.5 items-center justify-center rounded-[7px] text-[10.5px] font-bold',
                            'bg-ink text-white' => $day === 9,
                            'border-[1.5px] border-dashed border-rose bg-rose-mist text-rose' => $day >= 12 && $day <= 16,
                            'text-ink' => $day !== 9 && ($day < 12 || $day > 16),
                        ])>{{ fa_digits($day) }}</span>
                    @endforeach
                </div>
            </div>
        </x-home.feature-tile>

        <x-home.feature-tile tint="bg-lavender" :title="$t('body.title')" :text="$t('body.text')" :backing="$t('body.backing')">
            <div class="flex w-full flex-col gap-2.5">
                <div class="flex max-w-[92%] items-start gap-2 self-start rounded-xl rounded-es-sm bg-surface px-3.5 py-3 shadow-tile">
                    <span class="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary"><x-icon name="mic" class="size-3.5 text-white"/></span>
                    <span class="text-sm leading-[1.8] font-semibold text-ink">{{ $t('body.said') }}</span>
                </div>
                <div class="flex flex-wrap items-center gap-1.5 self-end">
                    <span class="text-xs font-bold text-muted">{{ $t('body.found') }}</span>
                    @foreach ($t('body.chips') as $chip)
                        <span class="inline-flex h-7 items-center rounded-2xl bg-rose px-3 text-sm font-bold text-white">{{ $chip }}</span>
                    @endforeach
                </div>
            </div>
        </x-home.feature-tile>

        <x-home.feature-tile tint="bg-success-soft" :title="$t('care.title')" :text="$t('care.text')" :backing="$t('care.backing')">
            <div class="flex w-full flex-col gap-2 rounded-xl bg-surface px-4 py-3.5 shadow-tile">
                <span class="self-start rounded-lg bg-lavender px-2 py-0.5 text-xs font-extrabold text-primary">{{ $t('care.phase') }}</span>
                <span class="text-sm leading-[1.9] font-semibold text-ink">{{ $t('care.tip') }}</span>
                <span class="flex items-center gap-1.5 border-t border-line pt-2 text-xs font-bold text-muted">
                    <x-icon name="bell" class="size-3.5 text-success"/>{{ $t('care.reminder') }}
                </span>
            </div>
        </x-home.feature-tile>
    </ul>

    <div class="flex flex-col gap-5 max-lg:gap-3">
        <p id="home-features-change" class="m-0 font-display text-d-sm leading-[1.25] text-ink">{{ $t('change_title') }}</p>
        <ul aria-labelledby="home-features-change" class="m-0 grid list-none grid-cols-2 gap-5 p-0 max-lg:grid-cols-1 max-lg:gap-3">
            <x-home.feature-tile tint="bg-sand" wide :title="$t('ttc.title')" :text="$t('ttc.text')" :backing="$t('ttc.backing')" :note="$t('ttc.note')">
                <div class="flex w-full max-w-80 flex-col gap-2.5 rounded-xl bg-surface px-4 py-3.5 shadow-tile">
                    <div class="flex items-baseline justify-between gap-2">
                        <span class="text-sm font-bold text-muted">{{ $t('ttc.label') }}</span>
                        <span class="rounded-lg bg-sand px-2 py-0.5 text-xs font-extrabold text-stage-ttc">{{ $t('ttc.estimated') }}</span>
                    </div>
                    <b class="text-2xl text-ink">{{ $t('ttc.range') }}</b>
                    <div class="flex h-11 items-end gap-1">
                        @foreach ($fertilityBars as $bar)<span class="flex-1 rounded-md {{ $bar }}"></span>@endforeach
                    </div>
                </div>
            </x-home.feature-tile>

            <x-home.feature-tile tint="bg-peach" wide :title="$t('pregnancy.title')" :text="$t('pregnancy.text')" :backing="$t('pregnancy.backing')">
                <div class="flex w-full max-w-80 flex-col gap-2.5 rounded-xl bg-surface px-4 py-3.5 shadow-tile">
                    <div class="flex items-center gap-3">
                        <span class="flex size-12 shrink-0 items-center justify-center rounded-full bg-peach"><x-icon name="sprout" class="size-6 text-stage-pregnancy-warm"/></span>
                        <span class="flex flex-col">
                            <span class="text-xs font-bold text-muted">{{ $t('pregnancy.week') }}</span>
                            <b class="text-lg text-ink">{{ $t('pregnancy.size') }}</b>
                        </span>
                    </div>
                    <div class="flex items-start gap-2 rounded-lg bg-canvas px-3 py-2">
                        <x-icon name="check-square" class="mt-0.5 size-4 shrink-0 text-stage-pregnancy-warm"/>
                        <span class="flex flex-col">
                            <span class="text-[11px] font-bold text-muted">{{ $t('pregnancy.task_label') }}</span>
                            <span class="text-sm leading-[1.7] font-semibold text-ink">{{ $t('pregnancy.task') }}</span>
                        </span>
                    </div>
                </div>
            </x-home.feature-tile>
        </ul>
    </div>
</section>

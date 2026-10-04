{{--
    Tools page (L3-07), design/html/tools.html. Data from App\Http\Controllers\ToolsController:
      $forms      array<string, CalculatorForm> — `due`, `fert`: field values, error, ready result lines
      $templates  array — lang `tools.text` (result sentence templates), $errorsText array — lang `tools.errors`;
                  both handed to the calculators.js module so JS shows exactly what the no-JS GET form shows
      $appLinks   AppLinksSettings, $qrUrl string — the #download card
    Copy: lang/fa/tools.php. Stable anchors (footer + home link to them): #due-date, #fertility, #hospital-bag,
    #sisemoni. Calculators: GET forms (work without JS); data-module="calculators" answers in place when JS runs.
    Checklists are non-persistent checkboxes (nothing is stored).
--}}
@extends('layouts.app')

@php
    $cards = [
        'due' => [
            'icon' => 'calculator',
            'tile' => 'bg-stage-pregnancy/10 text-stage-pregnancy',
            'result' => 'border-stage-pregnancy/27 bg-stage-pregnancy/7',
        ],
        'fert' => [
            'icon' => 'target',
            'tile' => 'bg-stage-ttc/10 text-stage-ttc',
            'result' => 'border-stage-ttc/27 bg-stage-ttc/7',
        ],
    ];
    $checklists = [
        'hospital_mother' => 'hospital-bag',
        'hospital_baby' => null,
        'sisemoni' => 'sisemoni',
    ];
    $guides = [
        'car_seat' => ['icon' => 'person', 'color' => 'postpartum'],
        'stroller' => ['icon' => 'person', 'color' => 'primary'],
        'crib' => ['icon' => 'home', 'color' => 'primary'],
        'bottle' => ['icon' => 'pill', 'color' => 'postpartum'],
    ];
@endphp

@section('content')
    <x-ui.page-intro
        :eyebrow="__('tools.intro.eyebrow')"
        :title="__('tools.intro.title')"
        :lead="__('tools.intro.lead')"
    />

    <section aria-label="{{ __('tools.calculators.label') }}" class="flex flex-col gap-4 px-30 pt-6 pb-[43px] max-lg:px-5 max-lg:pb-6">
        <div class="flex gap-6 max-lg:flex-wrap">
            @foreach ($forms as $key => $form)
                @php
                    $card = $cards[$key];
                    $id = $form->kind->anchor();
                    $lmpInvalid = $form->error?->field() === 'lmp';
                    $cycleInvalid = $form->error?->field() === 'cycle';
                @endphp
                <form
                    id="{{ $id }}"
                    method="get"
                    action="{{ route('tools') }}#{{ $id }}"
                    novalidate
                    aria-labelledby="{{ $id }}-title"
                    data-module="calculators"
                    data-kind="{{ $form->kind->value }}"
                    data-text="{{ json_encode($templates, JSON_UNESCAPED_UNICODE) }}"
                    data-errors="{{ json_encode($errorsText, JSON_UNESCAPED_UNICODE) }}"
                    class="flex flex-1 scroll-mt-6 flex-col gap-4.5 rounded-6xl border border-line bg-surface p-8 max-lg:basis-91.5 max-sm:basis-full"
                >
                    <input type="hidden" name="calc" value="{{ $form->kind->value }}">
                    <div class="flex items-center gap-3.5">
                        <span aria-hidden="true" class="flex size-13 shrink-0 items-center justify-center rounded-2xl {{ $card['tile'] }}"><x-icon :name="$card['icon']" class="size-6.5"/></span>
                        <div>
                            <h2 id="{{ $id }}-title" class="m-0 font-sans text-[21px] leading-normal font-bold">{{ __("tools.calculators.{$key}.title") }}</h2>
                            <p class="m-0 text-base font-semibold text-muted">{{ __("tools.calculators.{$key}.subtitle") }}</p>
                        </div>
                    </div>

                    <div class="flex flex-col gap-2">
                        <label for="{{ $id }}-lmp" class="text-base font-extrabold">{{ __('tools.calculators.lmp') }}</label>
                        <span @class(['flex h-14 items-center justify-between gap-2 rounded-2xl border-[1.5px] bg-surface px-4.5 text-lg font-bold text-ink focus-within:border-primary', $lmpInvalid ? 'border-danger' : 'border-line'])>
                            <input
                                id="{{ $id }}-lmp"
                                name="lmp"
                                type="text"
                                inputmode="text"
                                autocomplete="off"
                                maxlength="40"
                                value="{{ $form->lmp }}"
                                placeholder="{{ __('tools.calculators.lmp_placeholder') }}"
                                aria-describedby="{{ $id }}-error"
                                @if ($lmpInvalid) aria-invalid="true" @endif
                                class="min-w-0 grow border-0 bg-transparent [font:inherit] text-inherit outline-none placeholder:font-semibold placeholder:text-placeholder"
                            >
                            <x-icon name="calendar" class="size-4.5 shrink-0 text-muted"/>
                        </span>
                    </div>

                    <div class="flex flex-col gap-2">
                        <label for="{{ $id }}-cycle" class="text-base font-extrabold">{{ __('tools.calculators.cycle') }}</label>
                        <span @class(['flex h-14 items-center justify-between gap-2 rounded-2xl border-[1.5px] bg-surface px-4.5 text-lg font-bold text-ink focus-within:border-primary', $cycleInvalid ? 'border-danger' : 'border-line'])>
                            <input
                                id="{{ $id }}-cycle"
                                name="cycle"
                                type="text"
                                inputmode="numeric"
                                autocomplete="off"
                                maxlength="12"
                                value="{{ $form->cycle }}"
                                placeholder="{{ __('tools.calculators.cycle_placeholder') }}"
                                aria-describedby="{{ $id }}-error"
                                @if ($cycleInvalid) aria-invalid="true" @endif
                                class="min-w-0 grow border-0 bg-transparent [font:inherit] text-inherit outline-none placeholder:font-semibold placeholder:text-placeholder"
                            >
                            <x-icon name="calendar" class="size-4.5 shrink-0 text-muted"/>
                        </span>
                    </div>

                    <p id="{{ $id }}-error" role="alert" data-calc-error class="m-0 text-[13px] font-bold text-danger empty:hidden">{{ $form->error ? __('tools.errors.'.$form->error->value) : '' }}</p>

                    <button type="submit" class="inline-flex h-13.5 items-center justify-center gap-2 rounded-full border-[1.5px] border-primary bg-primary px-6.5 text-[15.5px] font-extrabold text-white hover:bg-primary-hover">{{ __('tools.calculators.submit') }}</button>

                    <p role="note" class="m-0 flex gap-2.5 rounded-[18px] border border-warn-line bg-warn-bg px-4 py-3 text-[13.5px] leading-[1.9] font-bold text-warn-text">
                        <span aria-hidden="true" class="flex size-5.5 shrink-0 items-center justify-center rounded-full bg-warn-icon font-extrabold text-white">!</span>
                        <span>{{ __("tools.calculators.{$key}.warn") }}</span>
                    </p>

                    <div data-calc-result aria-live="polite" @if (! $form->hasResult()) hidden @endif class="flex flex-col gap-1.5 rounded-4xl border p-5.5 {{ $card['result'] }}">
                        <span class="text-base font-extrabold text-muted">{{ __("tools.calculators.{$key}.result") }}</span>
                        <output for="{{ $id }}-lmp {{ $id }}-cycle" data-calc-value class="font-display text-[34px] leading-snug">{{ $form->value }}</output>
                        <span data-calc-detail class="text-base font-semibold text-muted">{{ $form->detail }}</span>
                    </div>
                </form>
            @endforeach
        </div>
        <p class="m-0 text-sm leading-relaxed font-semibold text-muted">{{ __('tools.calculators.note') }}</p>
    </section>

    <x-ui.section bg="surface" aria-labelledby="tools-checklists-title" class="flex flex-col gap-8">
        <x-ui.section-header
            id="tools-checklists-title"
            :eyebrow="__('tools.checklists.eyebrow')"
            :title="__('tools.checklists.title')"
            :lead="__('tools.checklists.lead')"
        />
        <div class="grid grid-cols-3 gap-4.5 max-sm:grid-cols-1">
            @foreach ($checklists as $key => $anchor)
                <div @if ($anchor) id="{{ $anchor }}" @endif role="group" aria-labelledby="checklist-{{ $key }}" class="flex scroll-mt-6 flex-col gap-2 rounded-4xl border border-line bg-surface p-5.5">
                    <h3 id="checklist-{{ $key }}" class="m-0 text-xl font-bold">{{ __("tools.checklists.items.{$key}.title") }}</h3>
                    @foreach ((array) __("tools.checklists.items.{$key}.items") as $item)
                        <label class="flex min-h-8 cursor-pointer items-center gap-2.5 text-md font-semibold">
                            <input type="checkbox" @checked($loop->index < 2) class="size-5 shrink-0 accent-primary">
                            {{ $item }}
                        </label>
                    @endforeach
                </div>
            @endforeach
        </div>
    </x-ui.section>

    <x-ui.section aria-labelledby="tools-guides-title" class="flex flex-col gap-7">
        <x-ui.section-header id="tools-guides-title" :eyebrow="__('tools.guides.eyebrow')" :title="__('tools.guides.title')"/>
        <div class="grid grid-cols-4 gap-4 max-lg:grid-cols-2 max-sm:grid-cols-1">
            @foreach ($guides as $key => $guide)
                <x-cards.feature
                    :icon="$guide['icon']"
                    :color="$guide['color']"
                    :title="__('tools.guides.items.'.$key.'.title')"
                    :text="__('tools.guides.items.'.$key.'.text')"
                    :where="__('tools.guides.where')"
                />
            @endforeach
        </div>
        <p class="m-0 text-base font-semibold text-muted">{{ __('tools.guides.note') }}</p>
    </x-ui.section>

    <x-ui.app-cta
        :title="__('tools.app_cta.title')"
        :lead="__('tools.app_cta.lead')"
        :links="$appLinks"
        :qr-url="$qrUrl"
    />
@endsection

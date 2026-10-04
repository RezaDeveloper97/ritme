{{--
    @include('pages.home.split', ['key' => 'uncertainty|family', 'screen' => 'cycle-today|companion', 'mediaFirst' => bool, 'bg' => 'none|surface'])
    Feature split (AUDIT §2.4 x-stage.feature-split, index ×2): eyebrow + h2 + text + check list beside a phone
    mockup on a lavender glow. Copy: lang/fa/home.php `<key>.*`.
--}}
@php
    $splitId = 'home-'.$key;
    $media = view('pages.home.mock.phone', ['size' => 'split', 'screen' => $screen]);
@endphp
<x-ui.section pad="lg" :bg="$bg ?? 'none'" aria-labelledby="{{ $splitId }}-title" class="flex items-center gap-20 max-lg:flex-wrap">
    @if ($mediaFirst ?? false)
        <div class="flex flex-1 justify-center bg-[radial-gradient(circle,var(--color-lavender),transparent_70%)] py-5 max-lg:basis-75 max-sm:basis-full">{!! $media !!}</div>
    @endif
    <div class="flex flex-1 flex-col gap-4.5 max-lg:basis-75 max-sm:basis-full">
        <x-ui.eyebrow>{{ __("home.{$key}.eyebrow") }}</x-ui.eyebrow>
        <h2 id="{{ $splitId }}-title" class="m-0 font-display text-d-lg leading-heading font-normal text-ink">{{ __("home.{$key}.title") }}</h2>
        <p class="m-0 text-xl leading-loose font-medium text-muted">{{ __("home.{$key}.text") }}</p>
        <ul class="m-0 flex list-none flex-col gap-1.5 p-0">
            @foreach (__("home.{$key}.points") as $point)
                <li class="flex items-start gap-2.5 text-lg leading-relaxed font-semibold text-ink">
                    <span class="mt-1.5 shrink-0"><x-icon name="check" class="size-4.5 text-stage-postpartum"/></span>
                    {{ $point }}
                </li>
            @endforeach
        </ul>
    </div>
    @unless ($mediaFirst ?? false)
        <div class="flex flex-1 justify-center bg-[radial-gradient(circle,var(--color-lavender),transparent_70%)] py-5 max-lg:basis-75 max-sm:basis-full">{!! $media !!}</div>
    @endunless
</x-ui.section>

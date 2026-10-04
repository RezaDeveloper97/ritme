{{--
    Feature split (AUDIT §2.4 `x-stage.feature-split`): text column (eyebrow, h2, text, check list) + phone mock-up
    on a lavender glow. $feature: StageFeatureData — column order and band colour alternate; the first split carries
    id="how". Stage-specific variants live in partials/<slug>/ (StageDefinition::featurePartial()).
--}}
<section @if ($feature->anchor) id="{{ $feature->anchor }}" @endif aria-labelledby="{{ $feature->headingId() }}" @class([
    'flex items-center gap-20 px-30 py-20 max-lg:flex-wrap max-lg:px-5 max-lg:py-11',
    'bg-surface' => $feature->surface,
])>
    @if ($feature->mediaFirst)
        @include('pages.stages.partials.feature-media', ['feature' => $feature])
    @endif
    <div class="flex flex-1 flex-col gap-4.5 max-lg:basis-75 max-sm:basis-full">
        <x-ui.eyebrow>{{ $feature->eyebrow }}</x-ui.eyebrow>
        <h2 id="{{ $feature->headingId() }}" class="m-0 font-display text-d-lg leading-heading font-normal text-ink">{{ $feature->title }}</h2>
        <p class="m-0 text-xl leading-loose font-medium text-muted">{{ $feature->text }}</p>
        @if ($feature->points !== [])
            <ul class="m-0 flex list-none flex-col gap-1.5 p-0">
                @foreach ($feature->points as $point)
                    <li class="flex items-start gap-2.5 text-lg leading-relaxed font-semibold text-ink">
                        <span aria-hidden="true" class="mt-1.5 shrink-0"><x-icon name="check" class="inline size-4.5 text-stage-postpartum"/></span>{{ $point }}
                    </li>
                @endforeach
            </ul>
        @endif
    </div>
    @unless ($feature->mediaFirst)
        @include('pages.stages.partials.feature-media', ['feature' => $feature])
    @endunless
</section>

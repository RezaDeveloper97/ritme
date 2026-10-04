{{--
    Stage hero (AUDIT §2.4 `x-stage.hero`), inside the dark header block. $hero: StageHeroData. The single h1 of the
    page; the phone, orbit and float cards are decorative (aria-hidden). «چطور کار می‌کند؟» targets the first
    feature split (#how).
--}}
<section aria-labelledby="stage-hero-title" class="flex items-center gap-10 px-30 pt-8 pb-25 max-lg:flex-wrap max-lg:px-5 max-lg:py-8">
    <div class="flex flex-1 flex-col gap-6 max-lg:basis-75 max-sm:basis-full">
        <x-ui.eyebrow tone="dark" pill :icon="$hero->eyebrowIcon">{{ $hero->eyebrow }}</x-ui.eyebrow>
        <h1 id="stage-hero-title" class="m-0 font-display text-d-3xl leading-display font-normal text-on-night"><span class="text-lilac">{{ $hero->highlight }}</span> {{ $hero->title }}</h1>
        <p class="m-0 max-w-140 text-2xl leading-loose font-medium text-on-night-muted max-sm:w-full max-sm:max-w-full">{{ $hero->lead }}</p>
        <div class="flex flex-wrap gap-3">
            <x-ui.button href="#download" tone="dark" size="xl" icon="download" class="box-content border-[1.5px] border-lilac">{{ $hero->downloadLabel }}</x-ui.button>
            <x-ui.button href="#how" variant="outline" tone="dark" size="xl">{{ $hero->howLabel }}</x-ui.button>
        </div>
    </div>
    <div aria-hidden="true" class="relative h-165 w-150 shrink-0 max-sm:w-full max-sm:max-w-full">
        <x-illustration name="hero-orbit" width="620" height="620" class="absolute end-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2"/>
        <div class="absolute end-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2">
            @include('pages.stages.mock.phone', ['size' => 'hero', 'screen' => $hero->screen, 'data' => $hero->screenData])
        </div>
        @foreach ($hero->floatCards as $card)
            @include('pages.stages.mock.float-card', ['card' => $card])
        @endforeach
    </div>
</section>

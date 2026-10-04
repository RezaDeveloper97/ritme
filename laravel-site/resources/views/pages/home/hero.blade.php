{{--
    Home hero inside the dark header block (layouts.app @section('hero')). The LCP element is the h1 (text: rendered
    with the preloaded display font, nothing lazy); the phone, orbit and float cards are decorative (aria-hidden).
--}}
<section aria-labelledby="home-title" class="flex items-center gap-10 px-30 pt-8 pb-25 max-lg:flex-wrap max-lg:px-5 max-lg:py-8">
    <div class="flex flex-1 flex-col gap-6 max-lg:basis-75 max-sm:basis-full">
        <x-ui.eyebrow tone="dark" pill icon="sprout">{{ __('home.hero.eyebrow') }}</x-ui.eyebrow>
        <h1 id="home-title" class="m-0 font-display text-d-3xl leading-display font-normal text-on-night">
            {{ __('home.hero.title') }} <span class="text-lilac">{{ __('home.hero.title_accent') }}</span>
        </h1>
        <p class="m-0 max-w-140 text-2xl leading-loose font-medium text-on-night-muted max-sm:w-full max-sm:max-w-full">{{ __('home.hero.lead') }}</p>
        <div class="flex flex-wrap gap-3">
            <x-ui.button href="#download" tone="dark" size="xl" icon="download" class="border-[1.5px] border-lilac text-[15.5px]">{{ __('home.hero.download') }}</x-ui.button>
            <x-ui.button href="#stages" variant="outline" tone="dark" size="xl" class="box-border text-[15.5px]">{{ __('home.hero.stages') }}</x-ui.button>
        </div>
    </div>
    <div aria-hidden="true" class="relative h-165 w-150 shrink-0 max-sm:w-full max-sm:max-w-full">
        <x-illustration name="hero-orbit" :width="620" :height="620" class="absolute end-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2"/>
        <div class="absolute end-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2">
            @include('pages.home.mock.phone', ['size' => 'hero', 'screen' => 'cycle-today'])
        </div>
        @include('pages.home.mock.float-card', ['icon' => 'drop', 'tone' => 'period', 'title' => __('home.hero.float_period_title'), 'text' => __('home.hero.float_period_text'), 'class' => 'top-22.5 -start-2.5'])
        @include('pages.home.mock.float-card', ['icon' => 'mic', 'tone' => 'lilac', 'title' => __('home.hero.float_voice_title'), 'text' => __('home.hero.float_voice_text'), 'class' => 'bottom-27.5 -end-5'])
    </div>
</section>

@php
    use App\Domain\Content\Enums\SitePhoto;

    // Sections switched off in the admin «صفحه‌ها» settings lose their cards (settings writes bump `pages`).
    $on = static fn (string $section): bool => \Illuminate\Support\Facades\Blade::check('siteSection', $section);
    $cards = array_values(array_filter([
        ['key' => 'cycle', 'route' => 'stage.cycle', 'photo' => SitePhoto::WindowTea, 'soon' => false],
        ['key' => 'ttc', 'route' => 'stage.ttc', 'photo' => SitePhoto::CoupleBreakfast, 'soon' => false],
        ['key' => 'pregnancy', 'route' => 'stage.pregnancy', 'photo' => SitePhoto::PregnantWindow, 'soon' => false],
        ['key' => 'postpartum', 'route' => 'stage.postpartum', 'photo' => SitePhoto::MotherBaby, 'soon' => true],
        ['key' => 'menopause', 'route' => 'stage.menopause', 'photo' => SitePhoto::MidlifePark, 'soon' => true],
        ['key' => 'teen', 'route' => 'stage.teen', 'photo' => SitePhoto::MotherTeen, 'soon' => true],
    ], static fn (array $card): bool => $on($card['key'])));
@endphp
@if ($cards !== [])
<section id="stages" aria-labelledby="home-stages-title" class="flex flex-col gap-10 bg-surface px-30 py-24 max-xl:px-6 max-lg:gap-6 max-lg:px-5 max-lg:py-14">
    <x-site.heading id="home-stages-title" :eyebrow="__('home.stages.eyebrow')" :title="__('home.stages.title')" :lead="__('home.stages.lead')"/>
    <ul class="m-0 grid list-none grid-cols-3 gap-x-6 gap-y-12 p-0 max-lg:grid-cols-2 max-lg:gap-x-3 max-lg:gap-y-6">
        @foreach ($cards as $card)
            <li><a href="{{ route($card['route']) }}" class="flex flex-col gap-3 text-ink hover:text-ink max-lg:gap-2">
                <span class="relative block h-54 overflow-hidden rounded-3xl bg-canvas max-lg:h-36 max-sm:h-30">
                    @if ($photos[$card['photo']->value] ?? null)
                        <x-picture :media="$photos[$card['photo']->value]" :alt="$card['photo']->alt()" sizes="(max-width: 700px) 50vw, 33vw" class="size-full object-cover"/>
                    @endif
                    @if ($card['soon'])
                        <span class="absolute end-3.5 top-3.5 rounded-xl bg-surface px-3 py-1 text-sm font-extrabold text-ink max-sm:end-2 max-sm:top-2 max-sm:px-2 max-sm:py-0.5 max-sm:text-xs">{{ __('home.stages.soon') }}</span>
                    @endif
                </span>
                <h3 class="m-0 text-4xl leading-normal font-bold max-sm:text-xl">{{ __("home.stages.items.{$card['key']}.title") }}</h3>
                <span class="text-md leading-[1.9] font-medium text-muted max-sm:hidden">{{ __("home.stages.items.{$card['key']}.text") }}</span>
            </a></li>
        @endforeach
    </ul>
</section>
@endif

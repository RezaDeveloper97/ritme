{{--
    Services page (L3-06), design/html/services.html. Data from App\Http\Controllers\ServicesController:
      $careItems         list<{icon, color, title, text, points}> — «از سؤال ساده تا ویزیت» (in-app services, not links)
      $emergencyNumber   string                                   — settings general.emergency_number
      $appLinks          AppLinksSettings, $qrUrl                 — the #download card
    Copy: lang/fa/services.php.
--}}
@extends('layouts.app')

@section('content')
    <x-ui.page-intro :eyebrow="__('services.intro.eyebrow')" :title="__('services.intro.title')" :lead="__('services.intro.lead')"/>

    <x-ui.section pad="none" aria-labelledby="care-title" class="flex flex-col gap-8 pt-10 pb-24 max-lg:pt-6 max-lg:pb-[52.8px]">
        <x-ui.section-header :eyebrow="__('services.care.eyebrow')" :title="__('services.care.title')" id="care-title"/>
        <ul class="m-0 grid list-none grid-cols-3 gap-4.5 p-0 max-sm:grid-cols-1">
            @foreach ($careItems as $item)
                <li class="flex flex-col gap-3.5 rounded-5xl border border-line bg-surface p-7 text-ink">
                    <x-ui.icon-tile :icon="$item['icon']" :color="$item['color']"/>
                    <h3 class="m-0 text-[21px] font-bold">{{ $item['title'] }}</h3>
                    <p class="m-0 text-md leading-relaxed font-medium text-muted">{{ $item['text'] }}</p>
                    <ul class="m-0 flex list-none flex-col gap-1 p-0">
                        @foreach ($item['points'] as $point)
                            <li class="flex items-start gap-2.5 text-lg leading-relaxed font-semibold text-ink"><span aria-hidden="true" class="mt-1.5 shrink-0"><x-icon name="check" class="inline size-4.5 text-stage-postpartum"/></span>{{ $point }}</li>
                        @endforeach
                    </ul>
                </li>
            @endforeach
        </ul>
    </x-ui.section>

    <x-ui.section bg="surface" as="div" class="flex flex-col gap-10">
        {{-- The design keeps the cards' 36px padding on phones; promo-split drops it to 24px (kit), so restore it here. --}}
        <x-ui.promo-split class="max-sm:[&>a]:p-9" :items="[
            ['href' => route('directory.index'), 'tone' => 'lavender', 'icon' => 'map', 'title' => __('services.directory.title'), 'text' => __('services.directory.text'), 'cta' => __('services.directory.cta')],
            ['href' => route('shop.index'), 'tone' => 'dashed', 'icon' => 'store', 'title' => __('services.shop.title'), 'text' => __('services.shop.text'), 'cta' => __('services.shop.cta')],
        ]"/>
    </x-ui.section>

    <x-ui.section pad="none" as="div" class="flex flex-col gap-10 pt-16 pb-24 max-lg:pt-[35.2px] max-lg:pb-[52.8px]">
        <x-ui.alert-emergency :number="$emergencyNumber">{{ __('services.emergency', ['number' => fa_digits($emergencyNumber)]) }}</x-ui.alert-emergency>
    </x-ui.section>

    <x-ui.app-cta :title="__('services.app_cta.title')" :lead="__('services.app_cta.lead')" :links="$appLinks" :qr-url="$qrUrl"/>
@endsection

{{--
    Free vs Plus page (L3-06), design/html/plus.html. Data from App\Http\Controllers\PlusController:
      $plans     list<{key, title, price, note, items, featured}> — the three plan cards; `price` is already display
                 text («۰ تومان», «قیمت در اپ», or a published price — see lang/fa/plus.php `pricing`)
      $faq       ?FaqGroupData (FaqServiceProvider composer, group `plus`; its FAQPage JSON-LD is already registered)
      $appLinks  AppLinksSettings, $qrUrl — the #download card
    Plan cards are page markup (the `x-cards.pricing` kit slot was outside this task's paths).
--}}
@extends('layouts.app')

@section('content')
    <x-ui.page-intro :eyebrow="__('plus.intro.eyebrow')" :title="__('plus.intro.title')" :lead="__('plus.intro.lead')"/>

    <x-ui.section pad="none" aria-label="{{ __('plus.plans.label') }}" class="flex flex-col gap-10 pt-6 pb-20 max-lg:pb-11">
        <ul class="m-0 flex list-none items-stretch gap-5 p-0 max-lg:flex-wrap">
            @foreach ($plans as $plan)
                <li @class([
                    'flex flex-1 flex-col gap-4 rounded-6xl border p-8 max-lg:basis-91.5 max-sm:basis-full',
                    'border-night-line bg-night text-on-night' => $plan['featured'],
                    'border-line bg-surface text-ink' => ! $plan['featured'],
                ])>
                    <h2 class="m-0 font-sans text-4xl leading-normal font-bold">{{ $plan['title'] }}</h2>
                    <p class="m-0 font-display text-d-lg leading-heading">{{ $plan['price'] }}</p>
                    <p @class(['m-0 text-base font-semibold', $plan['featured'] ? 'text-on-night-muted' : 'text-muted'])>{{ $plan['note'] }}</p>
                    <ul class="m-0 flex list-none flex-col gap-1.5 p-0">
                        @foreach ($plan['items'] as $item)
                            <li @class(['flex items-start gap-2.5 text-lg leading-relaxed font-semibold', $plan['featured'] ? 'text-on-night' : 'text-ink'])><span aria-hidden="true" class="mt-1.5 shrink-0"><x-icon name="check" @class(['inline size-4.5', $plan['featured'] ? 'text-lilac' : 'text-stage-postpartum'])/></span>{{ $item }}</li>
                        @endforeach
                    </ul>
                </li>
            @endforeach
        </ul>
    </x-ui.section>

    <x-faq :faq="$faq" variant="grid" :eyebrow="__('plus.faq.eyebrow')" :title="$faq?->title" bg="surface"/>

    <x-ui.app-cta :title="__('plus.app_cta.title')" :lead="__('plus.app_cta.lead')" :links="$appLinks" :qr-url="$qrUrl"/>
@endsection

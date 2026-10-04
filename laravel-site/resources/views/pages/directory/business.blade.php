{{--
    Business landing (L5-05), design/html/directory-business.html. Controller: App\Http\Controllers\Directory\JoinController@business.
      $faq  ?FaqGroupData — the `directory-business` FAQ group (FaqServiceProvider composer; also its FAQPage JSON-LD)
    Copy: lang/fa/directory.php `business_page` (the «شرایط همکاری» text is still the design placeholder until the
    commission model is decided). The hero card is a decorative sample of a listing card (inert, no fake rating).
--}}
@extends('layouts.app')

@php($t = 'directory.business_page.')

@section('content')
    <section aria-labelledby="business-title" class="flex items-center gap-20 bg-linear-180/srgb from-surface to-canvas px-30 pt-18 pb-24 max-lg:flex-wrap max-lg:px-5 max-lg:pt-[39.6px] max-lg:pb-[52.8px]">
        <div class="flex flex-1 flex-col gap-5.5 max-lg:basis-75 max-sm:basis-full">
            <span class="text-base font-extrabold text-primary">{{ __($t.'eyebrow') }}</span>
            <h1 id="business-title" class="m-0 font-display text-[56px] leading-display font-normal text-ink max-lg:text-[42px] max-sm:text-[34px]">{{ __($t.'title') }}</h1>
            <p class="m-0 max-w-155 text-2xl leading-loose font-medium text-muted max-sm:w-full max-sm:max-w-full">{{ __($t.'lead') }}</p>
            <div class="flex gap-3 max-lg:flex-wrap">
                <x-ui.button :href="route('directory.join')" size="xl" icon="arrow-left" icon-class="size-4.5 text-white" class="box-content border-[1.5px] border-primary">{{ __($t.'start') }}</x-ui.button>
                <x-ui.button href="#faq" variant="outline" size="xl">{{ __($t.'faq_link') }}</x-ui.button>
            </div>
        </div>
        <figure aria-label="{{ __($t.'sample.label') }}" class="relative m-0 w-115 shrink-0 max-sm:w-full max-sm:max-w-full">
            <div inert aria-hidden="true">
                <x-cards.place href="{{ route('directory.index') }}" as="b" :name="__($t.'sample.name')" :category="__($t.'sample.category')"
                               :district="__($t.'sample.district')" :ages="__($t.'sample.ages')" :slots="[__($t.'sample.slot')]"
                               illustration="place-cover-pool" verified selected/>
            </div>
            <div aria-hidden="true" class="absolute -end-10 -bottom-21 flex items-center gap-3 rounded-4xl border border-line bg-surface px-5 py-4 shadow-[0_24px_48px_-28px_rgba(40,20,90,.45)] max-lg:flex-wrap">
                <span class="flex size-11 shrink-0 items-center justify-center rounded-full bg-stage-teen/13"><x-icon name="calendar" class="size-5.5 text-stage-teen"/></span>
                <div><b class="text-md">{{ __($t.'sample.booking_title') }}</b><div class="text-sm font-semibold text-muted">{{ __($t.'sample.booking_text') }}</div></div>
            </div>
        </figure>
    </section>

    <x-ui.section pad="lg" aria-labelledby="business-why" class="flex flex-col gap-10">
        <h2 id="business-why" class="m-0 font-display text-d-lg leading-heading font-normal text-ink">{{ __($t.'why.title') }}</h2>
        <div class="grid grid-cols-3 gap-5 max-sm:grid-cols-1">
            @foreach (__($t.'why.items') as $item)
                <x-cards.value :icon="$item['icon']" :title="$item['title']" :text="$item['text']"/>
            @endforeach
        </div>
    </x-ui.section>

    <x-ui.section pad="lg" bg="surface" aria-labelledby="business-how" class="flex flex-col gap-10">
        <h2 id="business-how" class="m-0 font-display text-d-lg leading-heading font-normal text-ink">{{ __($t.'how.title') }}</h2>
        <x-ui.steps :items="__($t.'how.items')"/>
    </x-ui.section>

    <x-ui.section pad="lg" class="flex flex-col gap-10" aria-label="{{ __($t.'requirements.title') }} · {{ __($t.'terms.title') }}">
        <div class="flex gap-12 max-lg:flex-wrap">
            <div class="flex flex-1 flex-col gap-4 max-lg:basis-75 max-sm:basis-full">
                <h2 class="m-0 font-display text-[34px] leading-heading font-normal text-ink">{{ __($t.'requirements.title') }}</h2>
                <ul class="m-0 flex list-none flex-col gap-1.5 p-0">
                    @foreach (__($t.'requirements.items') as $requirement)
                        <li class="flex items-start gap-2.5 text-lg leading-relaxed font-semibold text-ink"><span aria-hidden="true" class="mt-1.5 shrink-0"><x-icon name="check" class="size-4.5 text-stage-postpartum"/></span>{{ $requirement }}</li>
                    @endforeach
                </ul>
            </div>
            <div class="flex flex-1 flex-col gap-3.5 rounded-6xl border border-line bg-surface p-8 max-lg:basis-91.5 max-sm:basis-full">
                <h2 class="m-0 font-display text-[30px] leading-heading font-normal text-ink">{{ __($t.'terms.title') }}</h2>
                <p class="m-0 text-lg leading-loose font-medium text-muted">{{ __($t.'terms.text') }}</p>
                <p class="m-0 flex items-start gap-2.5 text-base leading-relaxed font-semibold text-muted"><x-icon name="info" class="size-4.5 shrink-0 text-primary"/>{{ __($t.'terms.note') }}</p>
            </div>
        </div>
    </x-ui.section>

    <x-faq :faq="$faq" :title="__($t.'faq_title')" align="center" bg="surface" pad="lg" heading-id="faq-title" id="faq" class="scroll-mt-6"/>

    <section aria-labelledby="business-cta" class="mx-30 my-20 flex items-center justify-between gap-10 rounded-7xl bg-night px-16 py-14 max-lg:mx-5 max-lg:my-11 max-lg:flex-wrap max-sm:rounded-4xl max-sm:px-5.5 max-sm:py-7">
        <div class="flex flex-col gap-3">
            <h2 id="business-cta" class="m-0 font-display text-d-xl leading-heading font-normal text-on-night">{{ __($t.'cta.title') }}</h2>
            <p class="m-0 text-xl leading-loose font-medium text-on-night-muted">{{ __($t.'cta.text') }}</p>
        </div>
        <a href="{{ route('directory.join') }}" class="inline-flex h-13.5 items-center gap-2 rounded-full border-[1.5px] border-lilac bg-lilac px-6.5 text-[15.5px] font-extrabold text-night"><x-icon name="arrow-left" class="size-4.5 text-night"/>{{ __($t.'start') }}</a>
    </section>
@endsection

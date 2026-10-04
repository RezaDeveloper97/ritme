{{--
    Shop home (L6-02, design/html/shop.html). Data (ShopHomeController): $intro [eyebrow, title, lead], $promos
    (x-ui.promo-split items), $departments [slug, name, href, tiles, products (ProductCardData), productsTitle,
    productsLead], $media (cover MediaData by id), $demo (any demo product shown → note), $appHref (app link for the
    app-only features), $trust (value items). JSON-LD (CollectionPage + ItemList) is registered by the controller;
    BreadcrumbList Home → فروشگاه is automatic. The cart badge is filled client-side (x-shop.cart-link).
--}}
@extends('layouts.app')

@section('content')
    <section class="flex flex-col gap-3 px-30 pt-10 pb-6 max-lg:px-5 max-lg:py-6">
        <div class="flex items-center justify-between gap-4">
            <x-ui.eyebrow>{{ $intro['eyebrow'] }}</x-ui.eyebrow>
            <x-shop.cart-link class="-my-2.5"/>
        </div>
        <h1 class="m-0 font-display text-[48px] leading-display font-normal text-ink max-sm:text-[30px]">{{ $intro['title'] }}</h1>
        <p class="m-0 max-w-205 text-[16.5px] leading-loose font-medium text-muted">{{ $intro['lead'] }}</p>
        @if ($demo)
            <p class="m-0 flex items-center gap-2 text-sm font-semibold text-muted"><x-icon name="info" class="size-4 shrink-0 text-primary"/>{{ __('shop.demo_note') }}</p>
        @endif
    </section>

    @if ($promos !== [])
        <section aria-label="{{ __('shop.name') }}" class="px-30 pt-2 pb-12 max-lg:px-5 max-lg:pb-[26.4px]">
            <x-ui.promo-split :items="$promos"/>
        </section>
    @else
        <section class="px-30 pb-12 max-lg:px-5">
            <p class="m-0 rounded-4xl border border-line bg-surface p-8 text-lg leading-loose font-medium text-muted">{{ __('shop.empty') }}</p>
        </section>
    @endif

    @foreach ($departments as $department)
        @if ($department['tiles'] !== [])
            <section aria-labelledby="cats-{{ $department['slug'] }}" class="flex flex-col gap-6 px-30 py-8 max-lg:px-5">
                <x-ui.section-header size="md" :id="'cats-'.$department['slug']" :title="__('shop.home.categories_title', ['name' => $department['name']])"/>
                <ul class="m-0 grid list-none grid-cols-8 gap-4 p-0 max-lg:grid-cols-4 max-sm:grid-cols-2">
                    @foreach ($department['tiles'] as $tile)
                        <li><x-shop.category-tile :href="$tile['href']" :name="$tile['name']" :illustration="$tile['illustration']" :media="$tile['media']" :index="$loop->index"/></li>
                    @endforeach
                </ul>
            </section>
        @endif

        @if ($department['products'] !== [])
            <section aria-labelledby="products-{{ $department['slug'] }}" class="flex flex-col gap-6 px-30 py-8 max-lg:px-5">
                <x-ui.section-header size="md" :id="'products-'.$department['slug']" :title="$department['productsTitle']" :more-href="$department['href']"
                                     :more-label="__('shop.department.all')" more-icon="arrow-left">
                    @if ($department['productsLead'])<p class="m-0 -mt-2 text-base font-semibold text-muted">{{ $department['productsLead'] }}</p>@endif
                </x-ui.section-header>
                <div class="grid grid-cols-5 gap-x-5 gap-y-7 max-lg:grid-cols-3 max-sm:grid-cols-2">
                    @foreach ($department['products'] as $product)
                        <x-shop.product-card :product="$product" :media="$media" :index="$loop->index"/>
                    @endforeach
                </div>
            </section>
        @endif

        @if ($loop->first)
            <section aria-label="{{ __('shop.app.title') }}" class="mx-30 my-8 flex gap-6 max-lg:mx-5 max-lg:flex-wrap">
                <a href="{{ $appHref }}" class="flex flex-1 flex-col gap-4 rounded-6xl border border-line bg-surface p-8 text-ink transition-shadow hover:text-ink hover:shadow-card max-lg:basis-91 max-sm:basis-full">
                    <div class="flex items-center gap-4">
                        <x-ui.icon-tile icon="check-square" color="primary-soft" shape="circle"/>
                        <div class="flex flex-col">
                            <h2 class="m-0 font-display text-d-sm leading-heading font-normal text-ink">{{ __('shop.app.checklist.title') }}</h2>
                            <span class="text-base font-semibold text-muted">{{ __('shop.app.checklist.meta') }}</span>
                        </div>
                    </div>
                    <span class="text-base font-bold text-muted">{{ __('shop.app.checklist.text') }}</span>
                    <span class="flex items-center gap-1.5 text-md font-extrabold text-primary">{{ __('shop.app.checklist.cta') }}<x-icon name="arrow-left" class="size-4"/></span>
                </a>
                <a href="{{ $appHref }}" class="flex flex-1 flex-col gap-3.5 rounded-6xl border border-line bg-surface p-8 text-ink transition-shadow hover:text-ink hover:shadow-card max-lg:basis-91 max-sm:basis-full">
                    <div class="flex items-center gap-4 max-sm:flex-wrap">
                        <x-ui.icon-tile icon="bell" color="primary-soft" shape="circle"/>
                        <div class="flex grow flex-col">
                            <h2 class="m-0 font-display text-d-sm leading-heading font-normal text-ink">{{ __('shop.app.reminder.title') }}</h2>
                            <span class="text-base font-semibold text-muted">{{ __('shop.app.reminder.meta') }}</span>
                        </div>
                        <x-ui.pill tone="muted" size="sm" class="shrink-0">{{ __('shop.app.reminder.status') }}</x-ui.pill>
                    </div>
                    <span class="text-base leading-relaxed font-semibold text-muted">{{ __('shop.app.reminder.text') }}</span>
                </a>
            </section>
        @endif
    @endforeach

    <section aria-labelledby="trust-title" class="flex flex-col px-30 pt-10 pb-20 max-lg:px-5 max-lg:pt-6 max-lg:pb-11">
        <h2 id="trust-title" class="sr-only">{{ __('shop.trust.title') }}</h2>
        <div class="grid grid-cols-4 gap-5 rounded-5xl border border-line bg-surface px-8 py-7 max-lg:grid-cols-2 max-sm:grid-cols-1">
            @foreach ($trust as $item)
                <x-cards.value variant="inline" :icon="$item['icon']" :title="$item['title']" :text="$item['text']"/>
            @endforeach
        </div>
    </section>
@endsection

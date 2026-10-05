{{--
    Cart page (L6-04, design/html/shop-cart.html): /shop/cart — noindex, no-store, never page-cached.
    Data (Shop\CartController): $cart (CartSummary: resolved lines, count, subtotal, shipping quote, total), $notices
    (translated notices about dropped lines), $itemNotices (line key → notice), $status / $error (flash after a no-JS
    change), $media (cover MediaData by id), $suggestions (ProductCardData, «شاید لازم داشته باشی»).
    Single seller: one group «فروشگاه ریتمی» instead of the design's per-seller boxes; no coupon field (no coupons).
    No-JS: every change is a POST form (PRG). JS (lazy `cart`): submits those forms with fetch and swaps
    [data-cart-body] with the fresh page's, announcing the status in the live region.
--}}
@extends('layouts.app')

@section('content')
    <section aria-labelledby="cart-title" data-module="cart" class="flex flex-col gap-6 px-30 pt-9 pb-24 max-lg:px-5 max-lg:pb-[52.8px]">
        <h1 id="cart-title" class="m-0 font-display text-d-xl leading-display font-normal text-ink">{{ __('shop.cart.title') }}</h1>

        <p role="status" data-cart-status @class(['m-0 rounded-2xl bg-success-soft px-5 py-3 text-[14.5px] font-bold text-success', 'hidden' => $status === null])>{{ $status }}</p>
        <p role="alert" data-cart-error @class(['m-0 rounded-2xl bg-danger-soft px-5 py-3 text-[14.5px] font-bold text-danger', 'hidden' => $error === null && $notices === []])>{{ implode(' ', array_filter([$error, ...$notices])) }}</p>

        <div data-cart-body>
            @if ($cart->isEmpty())
                <div class="flex flex-col items-start gap-4 rounded-5xl border border-line bg-surface p-8">
                    <span aria-hidden="true" class="flex size-14 items-center justify-center rounded-full bg-primary/13"><x-icon name="cart" class="size-6 text-primary"/></span>
                    <h2 class="m-0 text-2xl font-bold">{{ __('shop.cart.empty_title') }}</h2>
                    <p class="m-0 text-base leading-loose font-medium text-muted">{{ __('shop.cart.empty_text') }}</p>
                    <x-ui.button :href="route('shop.index')" size="xl" icon="store">{{ __('shop.cart.empty_cta') }}</x-ui.button>
                </div>
            @else
                <div class="flex items-start gap-8 max-lg:flex-wrap">
                    <div class="flex flex-1 flex-col gap-4.5 max-lg:basis-75 max-sm:basis-full">
                        @if ($cart->shipping->threshold !== null && $cart->count > 0)
                            <div class="rounded-5xl border border-line bg-surface px-7 py-5">
                                <div class="mb-2.5 flex justify-between text-[14.5px] font-bold max-lg:flex-wrap">
                                    @if ($cart->shipping->free)
                                        <span>{{ __('shop.cart.free_reached') }}</span>
                                    @else
                                        <span>{{ __('shop.cart.free_remaining', ['amount' => $cart->shipping->remaining->formatShort()]) }}</span>
                                    @endif
                                    <span class="text-muted">{{ __('shop.cart.free_rule', ['amount' => $cart->shipping->threshold->formatShort()]) }}</span>
                                </div>
                                {{-- Bar without inline style: an SVG rect (RTL: filled from the right). --}}
                                <svg viewBox="0 0 100 10" preserveAspectRatio="none" role="img" aria-label="{{ __('shop.cart.free_progress', ['percent' => fa_digits($cart->shipping->progress)]) }}" class="block h-2.5 w-full overflow-hidden rounded-full">
                                    <rect width="100" height="10" class="fill-canvas"/>
                                    <rect x="{{ 100 - $cart->shipping->progress }}" width="{{ $cart->shipping->progress }}" height="10" class="fill-stage-teen"/>
                                </svg>
                            </div>
                        @endif

                        <div class="rounded-5xl border border-line bg-surface px-7 pt-5.5 pb-2">
                            <div class="flex items-center justify-between pb-1.5 max-lg:flex-wrap">
                                <span class="flex items-center gap-2 text-lg font-extrabold"><x-icon name="store" class="size-5 text-primary"/>{{ __('shop.cart.seller') }}</span>
                                <span class="flex items-center gap-1.5 text-sm-plus font-bold text-muted"><x-icon name="truck" class="size-4 text-muted"/>{{ __('shop.cart.delivery') }}</span>
                            </div>
                            <ul aria-label="{{ __('shop.cart.lines') }}" class="m-0 list-none p-0">
                                @foreach ($cart->items as $item)
                                    <x-shop.cart-line :item="$item" :notice="$itemNotices[$item->key] ?? null" :media="$media" :index="$loop->index" :first="$loop->first"/>
                                @endforeach
                            </ul>
                        </div>

                        @if ($cart->hasDemo())
                            <p class="m-0 flex items-center gap-2 text-sm font-semibold text-muted"><x-icon name="info" class="size-4 shrink-0 text-primary"/>{{ __('shop.cart.demo_note') }}</p>
                        @endif
                    </div>

                    <aside aria-labelledby="cart-summary-title" class="flex w-100 shrink-0 flex-col gap-4 max-sm:w-full max-sm:max-w-full">
                        <x-shop.cart-summary :cart="$cart"/>
                        <p class="m-0 flex gap-2.5 px-2 text-sm-plus leading-relaxed font-semibold text-muted">
                            <x-icon name="lock" class="size-4.5 shrink-0 text-stage-teen"/>{{ __('shop.cart.privacy') }}
                        </p>
                    </aside>
                </div>
            @endif
        </div>

        @if ($suggestions !== [])
            <section aria-labelledby="suggestions-title" class="flex flex-col gap-6 pt-6">
                <h2 id="suggestions-title" class="m-0 font-display text-d-md leading-heading font-normal text-ink">{{ __('shop.cart.suggestions') }}</h2>
                <div class="grid grid-cols-5 gap-x-5 gap-y-7 max-lg:grid-cols-3 max-sm:grid-cols-2">
                    @foreach ($suggestions as $card)
                        <x-shop.product-card :product="$card" :media="$media" :index="$loop->index"/>
                    @endforeach
                </div>
            </section>
        @endif
    </section>
@endsection

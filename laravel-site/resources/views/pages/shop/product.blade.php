{{--
    Product page (L6-03, design/html/shop-product.html): /shop/product/{slug}.
    Data (Shop\ProductController, DTOs + arrays): $product (ProductData), $breadcrumbs (list<BreadcrumbItem>, also the
    BreadcrumbList JSON-LD), $subnav, $gallery, $header (brand, real rating, sales), $picker (variant picker state, see
    ProductController::picker()), $cart (action|null, purchasable), $og (product:price:*), $specs, $description
    (sanitised HTML), $sizeChart, $reviews, $reviewsDemo, $reviewPagination, $reviewForm, $crossSells (cards),
    $crossSellMedia. ItemPage + Product JSON-LD are registered by the controller before the head renders.
    No-JS: colour/size are radio inputs inside the add-to-cart form, the quantity a number input, the gallery links.
    JS (lazy): `product` (per-colour sold-out sizes, price, stock, hint, size-chart row, quantity stepper), `gallery`.
--}}
@extends('layouts.app', ['navRoute' => 'shop.index'])

@push('head')
    <meta property="product:price:amount" content="{{ $og['amount'] }}">
    <meta property="product:price:currency" content="{{ $og['currency'] }}">
    <meta property="product:availability" content="{{ $og['availability'] }}">
    @if ($product->brand)
        <meta property="product:brand" content="{{ $product->brand->name }}">
    @endif
@endpush

@section('content')
    <x-shop.subnav :departments="$subnav['departments']" :links="$subnav['links']" :label="$subnav['label']"/>

    <section aria-labelledby="product-title" class="flex flex-col gap-6 px-30 pt-7 pb-10 max-lg:px-5 max-lg:pb-6">
        <x-ui.breadcrumbs :items="$breadcrumbs"/>

        <div class="flex items-start gap-14 max-lg:flex-wrap">
            <x-shop.gallery :images="$gallery['images']" :illustration="$gallery['illustration']" :name="$gallery['alt']"/>

            <div class="flex w-130 shrink-0 flex-col gap-5.5 max-sm:w-full max-sm:max-w-full" data-module="product" data-variants="{{ json_encode($picker['variants'], JSON_UNESCAPED_UNICODE) }}"
                 data-hints="{{ json_encode($picker['hints'], JSON_UNESCAPED_UNICODE) }}" data-label-in="{{ $picker['labels']['in'] }}"
                 data-label-out="{{ $picker['labels']['out'] }}" data-label-add="{{ __('shop.product.add') }}" data-label-unavailable="{{ __('shop.product.unavailable') }}">
                @if ($header['brand'])
                    <p class="m-0 flex items-center gap-1.5 text-base font-extrabold text-primary"><x-icon name="store" class="size-[17px] text-primary"/>{{ $header['brand'] }}</p>
                @endif
                <h1 id="product-title" class="m-0 font-display text-[38px] leading-display font-normal text-ink">{{ $product->title }}</h1>

                @if ($header['rating'] || $header['sales'])
                    <div class="flex items-center gap-2.5 text-[14.5px] font-semibold text-muted max-lg:flex-wrap">
                        @if ($header['rating'])
                            <a href="#reviews" class="text-ink hover:text-ink"><x-ui.rating :value="$header['rating']['value']" :count="$header['rating']['count']" class="text-[14.5px]"/></a>
                        @endif
                        @if ($header['rating'] && $header['sales'])<span aria-hidden="true" class="inline-block size-[3px] rounded-[2px] bg-muted"></span>@endif
                        @if ($header['sales'])<span>{{ __('shop.product.sales', ['count' => fa_digits(number_format($header['sales']))]) }}</span>@endif
                    </div>
                @endif

                <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
                    <span class="inline-flex flex-wrap items-baseline gap-1.5">
                        <b class="text-d-sm"><span data-price>{{ \App\Support\Text\Toman::format($picker['price']['amount']) }}</span> <span class="text-[24.5px] font-bold text-muted">تومان</span></b>
                        <s data-compare @class(['text-lg font-semibold text-muted', 'hidden' => $picker['price']['compare'] === null])><span class="sr-only">قیمت قبلی: </span><span data-compare-amount>{{ $picker['price']['compare'] === null ? '' : \App\Support\Text\Toman::format($picker['price']['compare']) }}</span></s>
                    </span>
                    <x-ui.badge :tone="$picker['stock']['in'] ? 'success' : 'danger'" data-stock role="status">{{ $picker['stock']['label'] }}</x-ui.badge>
                </div>

                {{-- Add-to-cart form: works without JS (radios + number input, POST → cart page). L6-04: `cart` module posts it as JSON and shows the toast; a refused no-JS add comes back here with `cart_error`. --}}
                <form id="buy" @if ($cart['action']) method="post" action="{{ $cart['action'] }}" data-module="cart" @endif data-cart-slot class="flex scroll-mt-24 flex-col gap-5.5">
                    @if ($cart['action'])
                        @csrf
                    @endif
                    <input type="hidden" name="product" value="{{ $product->id }}">

                    @if ($picker['colors'] !== [])
                        <div class="flex flex-col gap-2.5">
                            <b id="color-label" class="text-md">{{ __('shop.product.color') }}: <span data-color-name>{{ $picker['color'] }}</span></b>
                            <div role="radiogroup" aria-labelledby="color-label" class="flex gap-2 max-lg:flex-wrap">
                                @foreach ($picker['colors'] as $color)
                                    <label class="box-border flex size-11 cursor-pointer rounded-full border-2 border-transparent p-[3px] has-checked:border-primary has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-primary">
                                        <input type="radio" name="color" value="{{ $color['name'] }}" class="sr-only" data-color @checked($color['checked'])>
                                        <span class="sr-only">{{ __('shop.filters.color', ['name' => $color['name']]) }}</span>
                                        <svg viewBox="0 0 34 34" aria-hidden="true" class="size-full"><circle cx="17" cy="17" r="16.5" @if ($color['hex']) fill="{{ $color['hex'] }}" @else class="fill-lavender" @endif/><circle cx="17" cy="17" r="16.5" fill="none" class="stroke-line"/></svg>
                                    </label>
                                @endforeach
                            </div>
                        </div>
                    @endif

                    @if ($picker['sizes'] !== [])
                        <div class="flex flex-col gap-2.5">
                            <div class="flex justify-between max-lg:flex-wrap">
                                <b id="size-label" class="text-md">{{ __('shop.product.size') }}</b>
                                @if ($sizeChart)
                                    <a href="#size" class="flex items-center gap-1.5 text-base font-extrabold max-lg:flex-wrap"><x-icon name="ruler" class="size-[17px] text-primary"/>{{ __('shop.product.size_chart_link') }}</a>
                                @endif
                            </div>
                            <div role="radiogroup" aria-labelledby="size-label" class="flex flex-wrap gap-2">
                                @foreach ($picker['sizes'] as $size)
                                    <label @if ($size['soldout']) data-soldout @endif class="box-border flex h-12 cursor-pointer items-center rounded-xl border-[1.5px] border-line bg-surface px-4 text-base font-extrabold text-ink has-checked:border-primary has-checked:bg-lavender has-disabled:cursor-not-allowed has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-primary data-soldout:text-muted data-soldout:line-through data-soldout:opacity-50">
                                        <input type="radio" name="size" value="{{ $size['name'] }}" class="sr-only" data-size @checked($size['checked']) @disabled($size['disabled'])>{{ $size['name'] }}
                                    </label>
                                @endforeach
                            </div>
                            <span data-size-hint @class(['text-sm-plus font-semibold text-muted', 'hidden' => $picker['hint'] === null])>{{ $picker['hint'] }}</span>
                        </div>
                    @endif

                    <div class="flex gap-3 max-lg:flex-wrap">
                        <div class="flex h-14 items-center gap-3.5 rounded-full border-[1.5px] border-line px-2.5">
                            <button type="button" data-qty-step="1" hidden aria-label="{{ __('shop.product.more') }}" class="flex size-9 items-center justify-center border-0 bg-transparent text-ink"><x-icon name="plus" class="size-4.5"/></button>
                            <label for="product-qty" class="sr-only">{{ __('shop.product.quantity') }}</label>
                            <input id="product-qty" type="number" name="quantity" value="1" min="1" max="{{ $picker['quantityMax'] }}" inputmode="numeric" data-qty
                                   class="w-8 border-0 bg-transparent p-0 text-center text-lg font-bold text-ink [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none">
                            <button type="button" data-qty-step="-1" hidden aria-label="{{ __('shop.product.less') }}" class="flex size-9 items-center justify-center border-0 bg-transparent text-ink"><x-icon name="minus" class="size-4.5"/></button>
                        </div>
                        <button type="submit" data-cart-submit @if (! $cart['action'] || ! $cart['purchasable']) disabled @endif @if (! $cart['action']) aria-describedby="cart-soon" @endif
                                class="flex h-14 flex-1 items-center justify-center gap-2 rounded-full bg-primary text-lg font-extrabold text-white shadow-primary disabled:cursor-not-allowed max-lg:basis-75 max-sm:basis-full">
                            <x-icon name="cart" class="size-[19px]"/><span data-cart-label>{{ $cart['purchasable'] ? __('shop.product.add') : __('shop.product.unavailable') }}</span>
                        </button>
                        <button type="button" aria-pressed="false" aria-label="{{ __('shop.product.wishlist', ['name' => $product->title]) }}" class="flex size-14 items-center justify-center rounded-full border-[1.5px] border-line bg-transparent text-ink hover:text-primary">
                            <x-icon name="heart" class="size-[21px]"/>
                        </button>
                    </div>
                    @unless ($cart['action'])
                        <p id="cart-soon" class="sr-only">{{ __('shop.product.cart_soon') }}</p>
                    @endunless
                    @php($cartError = session('cart_error'))
                    <p role="status" data-cart-toast @if (is_string($cartError)) data-state="error" @endif @class(['m-0 flex flex-wrap items-center gap-x-3 gap-y-1 rounded-2xl bg-success-soft px-5 py-3 text-[14.5px] font-bold text-success data-[state=error]:bg-danger-soft data-[state=error]:text-danger', 'hidden' => ! is_string($cartError)])>
                        <span data-cart-toast-text>{{ is_string($cartError) ? $cartError : '' }}</span>
                        <a href="{{ route('shop.cart') }}" data-cart-toast-link @if (is_string($cartError)) hidden @endif class="text-primary underline">{{ __('shop.cart.view') }}</a>
                    </p>
                </form>

                <ul class="m-0 list-none rounded-5xl border border-line bg-surface px-5 py-1">
                    @foreach (__('shop.product.delivery') as $item)
                        <li @class(['flex min-h-11 items-center gap-3 py-3 text-ink max-lg:flex-wrap', 'border-t border-line' => ! $loop->first])>
                            <span aria-hidden="true" class="flex size-9.5 shrink-0 items-center justify-center rounded-full bg-primary/13"><x-icon :name="$item['icon']" class="size-[19px] text-primary"/></span>
                            <div class="min-w-0 grow">
                                <b class="text-sm-plus">{{ $item['title'] }}</b>
                                <div class="text-[11.5px] leading-[1.7] font-semibold text-muted">{{ $item['text'] }}</div>
                            </div>
                        </li>
                    @endforeach
                </ul>

                @if ($product->isDemo)
                    <p class="m-0 flex items-center gap-2 text-sm font-semibold text-muted"><x-icon name="info" class="size-4 shrink-0 text-primary"/>{{ __('shop.product.demo_note') }}</p>
                @endif
            </div>
        </div>
    </section>

    @if ($specs !== [] || $description || $sizeChart)
        <section aria-label="{{ __('shop.product.specs') }}" class="grid grid-cols-2 gap-10 px-30 pt-6 pb-10 max-lg:px-5 max-lg:py-6 max-sm:grid-cols-1">
            @if ($specs !== [] || $description)
                <div class="flex flex-col gap-4.5">
                    <h2 class="m-0 font-display text-d-sm leading-heading font-normal text-ink">{{ __('shop.product.specs') }}</h2>
                    @if ($specs !== [])
                        <dl class="m-0 overflow-hidden rounded-[22px] border border-line">
                            @foreach ($specs as $spec)
                                <div @class(['flex px-5 py-3.5 text-[14.5px] max-lg:flex-wrap', 'border-t border-line' => ! $loop->first, 'bg-surface' => $loop->odd, 'bg-canvas' => $loop->even])>
                                    <dt class="w-45 font-bold text-muted">{{ $spec['label'] }}</dt>
                                    <dd class="m-0 font-bold">{{ $spec['value'] }}</dd>
                                </div>
                            @endforeach
                        </dl>
                    @endif
                    @if ($description)
                        <h3 class="m-0 text-lg font-bold">{{ __('shop.product.description') }}</h3>
                        <div class="text-[14.5px] leading-loose font-medium text-muted [&_p]:m-0 [&_p+p]:mt-2 [&_ul]:m-0 [&_ul]:ps-5">{!! $description !!}</div>
                    @endif
                </div>
            @endif

            @if ($sizeChart)
                <div id="size" class="flex flex-col gap-4.5">
                    <h2 class="m-0 font-display text-d-sm leading-heading font-normal text-ink">{{ __('shop.product.size_chart') }}</h2>
                    <div class="overflow-x-auto overflow-y-hidden rounded-[22px] border border-line bg-surface">
                        <table class="w-full border-collapse text-center text-[14.5px]">
                            <caption class="sr-only">{{ __('shop.product.size_chart_caption', ['name' => $product->title]) }}</caption>
                            <thead>
                                <tr>
                                    @foreach ($sizeChart['columns'] as $column)
                                        <th scope="col" class="bg-lavender p-3.5 font-extrabold">{{ $column }}</th>
                                    @endforeach
                                </tr>
                            </thead>
                            <tbody>
                                @foreach ($sizeChart['rows'] as $row)
                                    <tr data-size-row="{{ $row[0] ?? '' }}" @if (($row[0] ?? null) === $picker['size']) data-current @endif class="data-current:bg-lavender/40">
                                        @foreach ($row as $cell)
                                            @if ($loop->first)
                                                <th scope="row" class="border-t border-line p-3.5 font-extrabold">{{ $cell }}</th>
                                            @else
                                                <td class="border-t border-line p-3.5 font-semibold">{{ $cell }}</td>
                                            @endif
                                        @endforeach
                                    </tr>
                                @endforeach
                            </tbody>
                        </table>
                    </div>
                </div>
            @endif
        </section>
    @endif

    <section id="reviews" aria-labelledby="reviews-title" class="flex flex-col gap-5 px-30 pt-6 pb-10 max-lg:px-5 max-lg:py-6">
        <div class="flex items-end justify-between gap-4 max-lg:flex-wrap">
            <div>
                <h2 id="reviews-title" class="m-0 font-display text-d-md leading-heading font-normal text-ink">{{ __('shop.product.reviews.title') }}</h2>
                <p class="m-0 mt-1 text-base font-semibold text-muted">{{ __('shop.product.reviews.lead') }}</p>
            </div>
            @if ($header['rating'])
                <p class="m-0 flex items-center gap-2 text-base font-extrabold text-ink">
                    <x-icon name="star" class="size-4.5 text-stage-ttc"/>{{ __('shop.product.reviews.summary', ['value' => \App\Support\Text\PersianDigits::number($header['rating']['value'], 1), 'count' => fa_digits($header['rating']['count'])]) }}
                </p>
            @endif
        </div>

        @if ($reviews !== [])
            <div class="grid grid-cols-3 gap-4 max-sm:grid-cols-1">
                @foreach ($reviews as $review)
                    <x-cards.review :name="$review['name']" :rating="$review['rating']" :meta="$review['meta']" :text="$review['text']"/>
                @endforeach
            </div>
            @if ($reviewsDemo)
                <p class="m-0 text-sm font-semibold text-muted">{{ __('shop.product.reviews.demo_note') }}</p>
            @endif
        @else
            <p class="m-0 text-base font-semibold text-muted">{{ __('shop.product.reviews.empty') }}</p>
        @endif

        @if ($reviewPagination)
            <nav aria-label="{{ __('shop.product.reviews.pagination') }}" class="flex items-center justify-between gap-3 text-sm-plus font-extrabold">
                @if ($reviewPagination['previous'])<a href="{{ $reviewPagination['previous'] }}" rel="prev" class="text-primary">{{ __('shop.product.reviews.previous') }}</a>@else<span></span>@endif
                <span class="text-muted">{{ $reviewPagination['label'] }}</span>
                @if ($reviewPagination['next'])<a href="{{ $reviewPagination['next'] }}" rel="next" class="text-primary">{{ __('shop.product.reviews.next') }}</a>@else<span></span>@endif
            </nav>
        @endif

        <x-shop.review-form :action="$reviewForm['action']" :sizes="$reviewForm['sizes']" :honeypot="$reviewForm['honeypot']"/>
    </section>

    @if ($crossSells !== [])
        <section aria-labelledby="related-title" class="flex flex-col gap-6 px-30 pt-6 pb-20 max-lg:px-5 max-lg:pb-11">
            <h2 id="related-title" class="m-0 font-display text-d-md leading-heading font-normal text-ink">{{ __('shop.product.related') }}</h2>
            <div class="grid grid-cols-5 gap-x-5 gap-y-7 max-lg:grid-cols-3 max-sm:grid-cols-2">
                @foreach ($crossSells as $card)
                    <x-shop.product-card :product="$card" :media="$crossSellMedia" :index="$loop->index + 3"/>
                @endforeach
            </div>
        </section>
    @endif
@endsection

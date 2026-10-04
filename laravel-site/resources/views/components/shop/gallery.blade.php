{{--
    <x-shop.gallery :images="[['id' => 12, 'url' => '…/a-desktop_1920.webp'], …]" name="بادی آستین‌بلند" illustration="product-bodysuit"/>
    Product gallery (L6-03, shop-product.html): a column of 96px thumbnails beside the 560px main image. The main image
    is the LCP (`priority`, eager); thumbnails are lazy and link to the large variant, so everything works without JS.
    The shared `gallery` data-module (L5-03) turns the links + the main image into a <dialog> lightbox (prev/next, arrow
    keys, Esc). Without photos: the product's local illustration on a tint (decorative, no lightbox).
--}}
@props(['images' => [], 'illustration' => null, 'name'])
@php($images = array_values($images))
<div {{ $attributes->class('flex flex-1 gap-3.5 max-lg:basis-75 max-lg:flex-wrap max-sm:basis-full') }} @if ($images !== []) data-module="gallery" @endif>
    @if (count($images) > 1)
        <ul class="m-0 flex list-none flex-col gap-3 p-0 max-lg:order-last max-lg:flex-row max-lg:flex-wrap" aria-label="{{ __('shop.product.gallery.thumbs') }}">
            @foreach ($images as $image)
                <li>
                    <a href="{{ $image['url'] }}" data-gallery-item @class([
                        'block size-24 shrink-0 overflow-hidden rounded-3xl bg-lavender',
                        'border-2 border-primary' => $loop->first,
                    ])>
                        <x-picture :media="$image['id']" :alt="__('shop.product.gallery.image', ['name' => $name, 'n' => fa_digits($loop->iteration)])" sizes="96px" class="size-full object-cover"/>
                    </a>
                </li>
            @endforeach
        </ul>
    @endif

    <div class="flex-1 max-lg:basis-75 max-sm:basis-full" @if (count($images) > 1) data-gallery-mosaic @endif>
        @if ($images !== [])
            @if (count($images) === 1)
                <a href="{{ $images[0]['url'] }}" data-gallery-item class="relative block h-140 overflow-hidden rounded-6xl bg-lavender">
                    <x-picture :media="$images[0]['id']" :alt="$name" sizes="(max-width: 1024px) 100vw, 45vw" class="size-full object-cover" priority/>
                </a>
            @else
                <div class="relative h-140 overflow-hidden rounded-6xl bg-lavender">
                    <x-picture :media="$images[0]['id']" :alt="$name" sizes="(max-width: 1024px) 100vw, 45vw" class="size-full object-cover" priority/>
                </div>
            @endif
        @else
            <div class="relative h-140 overflow-hidden rounded-6xl bg-stage-pregnancy/15" aria-hidden="true">
                @if ($illustration)<x-illustration :name="$illustration" class="size-full"/>@endif
            </div>
        @endif
    </div>

    @if ($images !== [])
        <dialog data-gallery-dialog aria-label="{{ __('shop.product.gallery.label', ['name' => $name]) }}"
                class="m-auto w-[min(92vw,1100px)] max-w-none overflow-visible rounded-5xl border-0 bg-night p-0 text-on-night backdrop:bg-night/80">
            <div class="p-4">
                <figure class="m-0 flex flex-col items-center gap-3">
                    <span data-gallery-stage data-gallery-img-class="max-h-[78vh] w-full rounded-3xl object-contain" class="flex min-h-60 w-full items-center justify-center"></span>
                    <figcaption data-gallery-counter aria-live="polite" class="text-sm font-bold text-on-night-muted"
                                data-template="{{ __('shop.product.gallery.counter', ['n' => ':n', 'total' => ':total']) }}"></figcaption>
                </figure>
                <div class="mt-3 flex items-center justify-center gap-2.5">
                    <button type="button" data-gallery-prev class="flex h-11 items-center gap-2 rounded-full border border-on-night/30 px-4.5 text-base font-extrabold text-on-night">
                        <x-icon name="chevron-left" class="size-4.5 rotate-180"/>{{ __('shop.product.gallery.previous') }}
                    </button>
                    <button type="button" data-gallery-next class="flex h-11 items-center gap-2 rounded-full border border-on-night/30 px-4.5 text-base font-extrabold text-on-night">
                        {{ __('shop.product.gallery.next') }}<x-icon name="chevron-left" class="size-4.5"/>
                    </button>
                    <button type="button" data-gallery-close class="flex h-11 items-center gap-2 rounded-full bg-surface px-4.5 text-base font-extrabold text-ink">
                        <x-icon name="x" class="size-4.5"/>{{ __('shop.product.gallery.close') }}
                    </button>
                </div>
            </div>
        </dialog>
    @endif
</div>

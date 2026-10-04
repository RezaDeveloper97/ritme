{{--
    <x-directory.gallery :images="[12, 13]" :full="[['id' => 12, 'url' => '…/a-desktop_1920.webp'], …]" name="آب‌پری"/>
    <x-directory.gallery :illustrations="['place-cover-pool', …]" name="آب‌پری"/>
    Place gallery (L5-03). Photos: the kit mosaic (x-ui.gallery; first image is the LCP `priority`, the rest lazy) plus
    «همه N عکس» — a native <details> listing every photo as a link to its large variant, so it works without JS; the
    `gallery` data-module turns it into a <dialog> lightbox (prev/next, arrow keys, Esc). Without photos: the
    category's local cover illustrations as a decorative mosaic (no button, no lightbox).
--}}
@props(['images' => [], 'full' => [], 'illustrations' => [], 'name'])
@if ($images !== [])
    <div {{ $attributes->class('relative') }} data-module="gallery">
        <div data-gallery-mosaic>
            <x-ui.gallery :images="$images" :alt="$name"/>
        </div>
        @if (count($full) > 1)
            <details id="photos" data-gallery-details>
                <summary data-gallery-open class="absolute end-4.5 top-96.5 box-content flex h-11 cursor-pointer list-none items-center gap-2 rounded-full border border-line bg-surface px-4.5 text-base font-extrabold text-ink max-sm:top-44 [&::-webkit-details-marker]:hidden">
                    <x-icon name="camera" class="size-4.5 text-primary"/>{{ __('directory.place.gallery.all', ['count' => fa_digits(count($full))]) }}
                </summary>
                <ul class="m-0 mt-2.5 grid list-none grid-cols-4 gap-2.5 p-0 max-sm:grid-cols-2" aria-label="{{ __('directory.place.gallery.label', ['name' => $name]) }}">
                    @foreach ($full as $image)
                        <li>
                            <a href="{{ $image['url'] }}" data-gallery-item class="block aspect-[4/3] overflow-hidden rounded-2xl bg-lavender">
                                <x-picture :media="$image['id']" :alt="__('directory.place.gallery.image', ['name' => $name, 'n' => fa_digits($loop->iteration)])" sizes="(max-width: 700px) 50vw, 25vw" class="size-full object-cover"/>
                            </a>
                        </li>
                    @endforeach
                </ul>
            </details>

            <dialog data-gallery-dialog aria-label="{{ __('directory.place.gallery.label', ['name' => $name]) }}"
                    class="m-auto w-[min(92vw,1100px)] max-w-none overflow-visible rounded-5xl border-0 bg-night p-0 text-on-night backdrop:bg-night/80">
                <div class="p-4">
                    <figure class="m-0 flex flex-col items-center gap-3">
                        <span data-gallery-stage data-gallery-img-class="max-h-[78vh] w-full rounded-3xl object-contain" class="flex min-h-60 w-full items-center justify-center"></span>
                        <figcaption data-gallery-counter aria-live="polite" class="text-sm font-bold text-on-night-muted"
                                    data-template="{{ __('directory.place.gallery.counter', ['n' => ':n', 'total' => ':total']) }}"></figcaption>
                    </figure>
                    <div class="mt-3 flex items-center justify-center gap-2.5">
                        <button type="button" data-gallery-prev class="flex h-11 items-center gap-2 rounded-full border border-on-night/30 px-4.5 text-base font-extrabold text-on-night">
                            <x-icon name="chevron-left" class="size-4.5 rotate-180"/>{{ __('directory.place.gallery.previous') }}
                        </button>
                        <button type="button" data-gallery-next class="flex h-11 items-center gap-2 rounded-full border border-on-night/30 px-4.5 text-base font-extrabold text-on-night">
                            {{ __('directory.place.gallery.next') }}<x-icon name="chevron-left" class="size-4.5"/>
                        </button>
                        <button type="button" data-gallery-close class="flex h-11 items-center gap-2 rounded-full bg-surface px-4.5 text-base font-extrabold text-ink">
                            <x-icon name="x" class="size-4.5"/>{{ __('directory.place.gallery.close') }}
                        </button>
                    </div>
                </div>
            </dialog>
        @endif
    </div>
@else
    <div {{ $attributes->class('grid grid-cols-[2fr_1fr_1fr] grid-rows-[220px_220px] gap-2.5 overflow-hidden rounded-6xl max-sm:grid-cols-1 max-sm:grid-rows-[240px]') }} aria-hidden="true">
        @foreach ($illustrations as $illustration)
            <div @class(['overflow-hidden bg-lavender', 'row-span-2 max-sm:row-span-1' => $loop->first, 'max-sm:hidden' => ! $loop->first])>
                <x-illustration :name="$illustration" width="100%" height="100%" class="block size-full"/>
            </div>
        @endforeach
    </div>
@endif

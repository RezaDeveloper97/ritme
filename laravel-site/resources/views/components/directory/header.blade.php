{{--
    <x-directory.header :place="$place"/>
    Title block of the place page (L5-03): category + verified (+ «مجموعه نمونه» for demo places) pills, the page's
    only h1, rating (real reviews only) · area · "open until" (cache-safe, decided by the controller), and the actions:
    share (copy link via the `share` module), call (tel:, when the place has a phone), directions (#address).
    `place`: array from ShowPlaceController::header().
--}}
@props(['place'])
@php($button = 'box-content flex h-11.5 items-center gap-2 rounded-full border-[1.5px] border-line bg-surface px-4.5 text-base font-extrabold text-ink hover:border-primary hover:text-ink')
<div class="flex items-end justify-between gap-6 max-lg:flex-wrap">
    <div class="flex flex-col gap-2.5">
        <div class="flex gap-2 max-lg:flex-wrap">
            <span class="box-content inline-flex h-7 items-center rounded-full border border-primary/40 px-2.5 text-[11.5px] font-bold whitespace-nowrap text-ink">{{ $place['category'] }}</span>
            @if ($place['verified'])
                <x-ui.pill size="md" icon="shield-check" icon-class="text-stage-teen" class="h-7">{{ __('directory.place.verified') }}</x-ui.pill>
            @endif
            @if ($place['demo'])
                <x-ui.pill size="md" tone="muted" class="h-7">{{ __('directory.place.demo') }}</x-ui.pill>
            @endif
        </div>
        <h1 class="m-0 font-display text-[46px] leading-display font-normal text-ink max-sm:text-[30px]">{{ $place['name'] }}</h1>
        <div class="flex items-center gap-2.5 text-md font-semibold text-muted max-lg:flex-wrap">
            @if ($place['rating'])
                <a href="#reviews" class="text-ink hover:text-ink"><x-ui.rating :value="$place['rating']['value']" :count="$place['rating']['count']" size="md"/></a>
                <span aria-hidden="true" class="inline-block size-[3px] rounded-[2px] bg-muted"></span>
            @endif
            <span>{{ $place['area'] }}</span>
            @if ($place['open'])
                <span aria-hidden="true" class="inline-block size-[3px] rounded-[2px] bg-muted"></span>
                <x-ui.pill dot>{{ $place['open'] }}</x-ui.pill>
            @endif
        </div>
    </div>
    <div class="flex gap-2.5 max-lg:flex-wrap" role="group" aria-label="{{ __('directory.place.actions.label') }}">
        <div data-module="share" data-share-copied="{{ __('directory.place.actions.copied') }}" data-share-failed="{{ __('directory.place.actions.copy_failed') }}" class="relative">
            <label class="sr-only" for="place-share-url">{{ __('directory.place.actions.share_url') }}</label>
            <input id="place-share-url" type="text" readonly value="{{ $place['url'] }}" data-share-url class="sr-only">
            <button type="button" data-share-copy hidden class="{{ $button }}">
                <x-icon name="share" class="size-4.5 text-primary"/>{{ __('directory.place.actions.share') }}
            </button>
            <span data-share-status role="status" class="sr-only"></span>
        </div>
        @if ($place['phone'])
            <a href="tel:{{ $place['phone'] }}" class="{{ $button }}"><x-icon name="phone" class="size-4.5 text-primary"/>{{ __('directory.place.actions.call') }}</a>
        @endif
        <a href="#address" class="{{ $button }}"><x-icon name="navigate" class="size-4.5 text-primary"/>{{ __('directory.place.actions.directions') }}</a>
    </div>
</div>

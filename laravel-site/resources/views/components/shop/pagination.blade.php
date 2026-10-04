{{--
    Result pages (design: 44px circles, current one filled primary): real links, page 1 without the parameter,
    window ±2. $pagination = [label, previous, next, pages: [number, href, current]] from Shop\CategoryController.
    <x-shop.pagination :pagination="$pagination"/> (shop category listing).
--}}
@props(['pagination'])
<nav aria-label="{{ $pagination['label'] }}">
    <ul class="m-0 flex list-none flex-wrap items-center justify-center gap-2 p-0">
        @if ($pagination['previous'])
            <li><a href="{{ $pagination['previous'] }}" rel="prev" class="flex h-11 items-center gap-1.5 rounded-full border-[1.5px] border-line bg-surface px-4.5 text-base font-extrabold text-ink hover:border-primary hover:text-ink"><x-icon name="chevron-left" class="size-4 rotate-180 text-primary"/>{{ __('shop.pagination.previous') }}</a></li>
        @endif
        @foreach ($pagination['pages'] as $item)
            <li>
                @if ($item['href'] === null)
                    <span aria-hidden="true" class="flex h-11 items-center px-2 text-base font-extrabold text-muted">{{ $item['number'] }}</span>
                @else
                    <a href="{{ $item['href'] }}" aria-label="{{ __('shop.pagination.page', ['page' => $item['number']]) }}" @if ($item['current']) aria-current="page" @endif @class([
                        'flex size-11 items-center justify-center rounded-full border-[1.5px] text-base font-extrabold hover:border-primary',
                        $item['current'] ? 'border-primary bg-primary text-white hover:text-white' : 'border-line bg-surface text-ink hover:text-ink',
                    ])>{{ $item['number'] }}</a>
                @endif
            </li>
        @endforeach
        @if ($pagination['next'])
            <li><a href="{{ $pagination['next'] }}" rel="next" class="flex h-11 items-center gap-1.5 rounded-full border-[1.5px] border-line bg-surface px-4.5 text-base font-extrabold text-ink hover:border-primary hover:text-ink">{{ __('shop.pagination.next') }}<x-icon name="chevron-left" class="size-4 text-primary"/></a></li>
        @endif
    </ul>
</nav>

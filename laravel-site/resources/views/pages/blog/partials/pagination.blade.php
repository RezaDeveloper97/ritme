{{--
    Magazine pagination: real links (?page=n, page 1 without the parameter), window ±2 around the current page.
    $pagination = [label, previous, next, pages: [number (Persian digits or «…»), href, current]] from BlogListingController.
--}}
<nav aria-label="{{ $pagination['label'] }}" class="pt-4">
    <ul class="m-0 flex list-none flex-wrap items-center justify-center gap-2 p-0">
        @if ($pagination['previous'])
            <li><a href="{{ $pagination['previous'] }}" rel="prev" class="flex h-10.5 items-center gap-1.5 rounded-full border-[1.5px] border-line bg-surface px-4.5 text-base font-bold text-ink transition-colors hover:border-primary hover:text-ink"><x-icon name="chevron-left" class="size-4 rotate-180 text-primary"/>{{ __('blog.pagination.previous') }}</a></li>
        @endif
        @foreach ($pagination['pages'] as $item)
            <li>
                @if ($item['href'] === null)
                    <span aria-hidden="true" class="flex h-10.5 items-center px-2 text-base font-bold text-muted">{{ $item['number'] }}</span>
                @else
                    <a href="{{ $item['href'] }}" aria-label="{{ __('blog.pagination.page', ['page' => $item['number']]) }}" @if ($item['current']) aria-current="page" @endif @class([
                        'flex size-10.5 items-center justify-center rounded-full border-[1.5px] text-base font-bold text-ink transition-colors hover:border-primary hover:text-ink',
                        $item['current'] ? 'border-primary bg-lavender' : 'border-line bg-surface',
                    ])>{{ $item['number'] }}</a>
                @endif
            </li>
        @endforeach
        @if ($pagination['next'])
            <li><a href="{{ $pagination['next'] }}" rel="next" class="flex h-10.5 items-center gap-1.5 rounded-full border-[1.5px] border-line bg-surface px-4.5 text-base font-bold text-ink transition-colors hover:border-primary hover:text-ink">{{ __('blog.pagination.next') }}<x-icon name="chevron-left" class="size-4 text-primary"/></a></li>
        @endif
    </ul>
</nav>

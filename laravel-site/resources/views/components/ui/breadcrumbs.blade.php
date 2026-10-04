{{-- Rendered by App\View\Components\Ui\Breadcrumbs. 13.5px/600 muted, «/» separators (AUDIT §2.1), as an ordered list. --}}
<nav aria-label="مسیر" {{ $attributes->class('text-sm-plus font-semibold text-muted') }}>
    <ol class="flex flex-wrap items-center gap-2">
        @foreach ($visible() as $item)
            <li class="flex items-center gap-2">
                @if (! $loop->first)<span aria-hidden="true">/</span>@endif
                @if ($loop->last || $item->url === null)
                    <span @if ($loop->last) aria-current="page" @endif class="text-ink">{{ $item->name }}</span>
                @else
                    <a href="{{ $item->url }}" class="text-muted hover:text-primary">{{ $item->name }}</a>
                @endif
            </li>
        @endforeach
    </ol>
</nav>

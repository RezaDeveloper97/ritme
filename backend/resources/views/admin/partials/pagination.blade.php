@if ($paginator->hasPages())
    <nav class="pagination" role="navigation">
        @if ($paginator->onFirstPage())
            <span class="muted" style="padding:6px 11px">قبلی</span>
        @else
            <a href="{{ $paginator->previousPageUrl() }}" rel="prev">قبلی</a>
        @endif

        @foreach ($elements as $element)
            @if (is_string($element))
                <span class="muted" style="padding:6px 11px">{{ $element }}</span>
            @endif
            @if (is_array($element))
                @foreach ($element as $page => $url)
                    @if ($page == $paginator->currentPage())
                        <span class="active"><span>{{ $page }}</span></span>
                    @else
                        <a href="{{ $url }}">{{ $page }}</a>
                    @endif
                @endforeach
            @endif
        @endforeach

        @if ($paginator->hasMorePages())
            <a href="{{ $paginator->nextPageUrl() }}" rel="next">بعدی</a>
        @else
            <span class="muted" style="padding:6px 11px">بعدی</span>
        @endif
    </nav>
@endif

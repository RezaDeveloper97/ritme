{{--
    <x-blog.toc :outline="$page->body->outline"/> — «در این مقاله»: links to the body's h2/h3 ids (h3 indented).
    The first entry carries the design's highlighted state (the reader starts at the top; no scroll-spy JS).
    Renders nothing without headings.
--}}
@props(['outline' => [], 'title' => 'در این مقاله'])
@if ($outline !== [])
<nav aria-label="{{ $title }}" {{ $attributes->class('flex flex-col gap-1') }}>
    <b class="mb-2 text-md">{{ $title }}</b>
    @foreach ($outline as $heading)
        <a href="#{{ $heading['id'] }}" @class([
            'border-s-2 px-3.5 py-2 text-[14.5px] hover:text-primary',
            'ps-7' => $heading['level'] > 2,
            'border-s-primary font-extrabold text-ink' => $loop->first,
            'border-s-line font-semibold text-muted' => ! $loop->first,
        ])>{{ $heading['text'] }}</a>
    @endforeach
</nav>
@endif

{{--
    <x-ui.chip-nav label="دسته‌های مجله" size="sm|md|lg" :items="[
        ['label' => 'همه', 'href' => route('blog.index'), 'active' => true],
        ['label' => 'چرخه و پریود', 'href' => '…', 'icon' => 'drop'],
    ]"/>
    Link chips — the design's only "tabs" (blog/directory categories, shop sort + sizes, contact topics, faq side nav;
    AUDIT §2.2 correction: no JS tabs). Plain links in a labelled <nav>; the active chip gets `aria-current`
    (`current="page"` for category pages, `"true"` for sort/topic selections) plus the lavender fill + primary border.
    Sizes: sm 40px/13.5 (contact) · md 42px/14 (blog, faq) · lg 46px/14 with icons (directory). `vertical` stacks them.
--}}
@props(['items' => [], 'label', 'size' => 'md', 'current' => 'page', 'vertical' => false])
@php
    $sizeClasses = match ($size) {
        'sm' => 'h-10 px-3.5 text-sm-plus',
        'lg' => 'h-11.5 px-4.5 text-base',
        default => 'h-10.5 px-4.5 text-base',
    };
@endphp
<nav aria-label="{{ $label }}" {{ $attributes }}>
    <ul @class(['m-0 flex list-none gap-2 p-0', 'flex-col items-start' => $vertical, 'flex-wrap' => ! $vertical, 'gap-2.5' => $size === 'lg'])>
        @foreach ($items as $item)
            @php($active = (bool) ($item['active'] ?? false))
            <li><a href="{{ $item['href'] }}" @if ($active) aria-current="{{ $current }}" @endif @class([
                'flex items-center gap-2 rounded-full border-[1.5px] font-bold text-ink transition-colors hover:border-primary hover:text-ink',
                $sizeClasses,
                $active ? 'border-primary bg-lavender' : 'border-line bg-surface',
            ])>@if (! empty($item['icon']))<x-icon :name="$item['icon']" class="size-4.5 text-primary"/>@endif{{ $item['label'] }}</a></li>
        @endforeach
    </ul>
</nav>

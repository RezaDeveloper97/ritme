{{--
    <x-cards.feature href="{{ route('tools') }}" icon="calculator" color="ttc" title="محاسبه روزهای باروری" text="…" where="روی سایت"/>
    Tool card (AUDIT §2.3: stage «کارهای کوچک», index tools, tools guides): radius 24, p 22, horizontal 48px rounded-16
    icon tile + title 17 + text 14 + where-tag («روی سایت» / «در اپ»). Without `href` it renders a plain <div>.
--}}
@props(['href' => null, 'icon', 'color' => 'primary', 'title', 'text' => null, 'where' => null, 'as' => 'h3'])
@php($tag = $href ? 'a' : 'div')
<{{ $tag }} @if ($href) href="{{ $href }}" @endif {{ $attributes->class(['flex items-start gap-3.5 rounded-4xl border border-line bg-surface p-5.5 text-ink', 'transition-shadow hover:text-ink hover:shadow-card' => $href]) }}>
    <x-ui.icon-tile :icon="$icon" :color="$color" size="sm"/>
    <span class="flex flex-col gap-1">
        <{{ $as }} class="m-0 text-xl font-bold">{{ $title }}</{{ $as }}>
        @if ($text)<span class="text-base leading-[1.8] font-medium text-muted">{{ $text }}</span>@endif
        @if ($where)<span class="text-[12.5px] font-extrabold text-primary">{{ $where }}</span>@endif
    </span>
</{{ $tag }}>

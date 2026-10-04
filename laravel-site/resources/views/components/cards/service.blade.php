{{--
    <x-cards.service href="{{ route('services') }}" icon="stethoscope" color="postpartum" title="پزشک و ماما" text="…"/>
    Service card (AUDIT §2.3: stage «وقتی کمک بیشتری», index, services): radius 28, p 28, vertical 56px rounded-18 icon
    tile + title 20 + text 15 + «بیشتر ←».
--}}
@props(['href', 'icon', 'color' => 'primary', 'title', 'text' => null, 'cta' => 'بیشتر', 'as' => 'h3'])
<a href="{{ $href }}" {{ $attributes->class('flex flex-col gap-3.5 rounded-5xl border border-line bg-surface p-7 text-ink transition-shadow hover:text-ink hover:shadow-card') }}>
    <x-ui.icon-tile :icon="$icon" :color="$color"/>
    <{{ $as }} class="m-0 text-3xl font-bold">{{ $title }}</{{ $as }}>
    @if ($text)<span class="text-md leading-relaxed font-medium text-muted">{{ $text }}</span>@endif
    <span class="flex items-center gap-1.5 text-base font-extrabold text-primary">{{ $cta }}<x-icon name="arrow-left" class="size-4"/></span>
</a>

{{--
    <x-cards.stage href="{{ route('stage.cycle') }}" icon="drop" color="cycle" title="چرخه و پریود" text="…"/>
    Life-stage card (AUDIT §2.3, index ×6): white, radius 28, p 24; 56px round icon tile in `stage/10`; title 20;
    text 14.5; «ببین ریتمی چه می‌کند ←». The whole card is the link; the title is an h3 (`as`).
--}}
@props(['href', 'icon', 'color' => 'primary', 'title', 'text' => null, 'cta' => 'ببین ریتمی چه می‌کند', 'as' => 'h3'])
<a href="{{ $href }}" {{ $attributes->class('flex flex-col gap-3 rounded-5xl border border-line bg-surface p-6 text-ink transition-shadow hover:text-ink hover:shadow-card') }}>
    <x-ui.icon-tile :icon="$icon" :color="$color" shape="circle"/>
    <{{ $as }} class="m-0 text-3xl font-bold">{{ $title }}</{{ $as }}>
    @if ($text)<span class="text-[14.5px] leading-relaxed font-medium text-muted">{{ $text }}</span>@endif
    <span class="flex items-center gap-1.5 text-base font-extrabold text-primary">{{ $cta }}<x-icon name="arrow-left" class="size-4"/></span>
</a>

{{--
    <x-cards.value icon="heart" color="cycle" title="بی‌قضاوت" text="…"/>                         (tile: about values, plus, social-responsibility)
    <x-cards.value variant="inline" icon="truck" title="ارسال سریع" text="…"/>                    (shop trust strip, privacy tools)
    <x-cards.value variant="compact" icon="lock" title="قفل اپ با رمز یا اثر انگشت"/>              (index privacy grid)
    Icon + title + text, not a link (AUDIT §2.3). tile: white radius 28 p 28 with a 56px tile; inline: 48px round
    primary/13 tile beside the text, no card chrome (put several in a bordered grid); compact: radius 22 p 20 with a
    bare 26px icon and a bold title.
--}}
@props(['icon', 'color' => 'primary', 'title', 'text' => null, 'variant' => 'tile', 'as' => 'h3'])
@if ($variant === 'inline')
<div {{ $attributes->class('flex items-center gap-3.5 max-lg:flex-wrap') }}>
    <x-ui.icon-tile :icon="$icon" color="primary-soft" size="sm" shape="circle"/>
    <div>
        <{{ $as }} class="m-0 text-[15.5px] font-bold">{{ $title }}</{{ $as }}>
        @if ($text)<div class="text-sm font-semibold text-muted">{{ $text }}</div>@endif
    </div>
</div>
@elseif ($variant === 'compact')
<div {{ $attributes->class('flex flex-col gap-2.5 rounded-[22px] border border-line bg-surface p-5') }}>
    <x-icon :name="$icon" class="size-6.5 text-primary"/>
    <{{ $as }} class="m-0 text-lg font-bold">{{ $title }}</{{ $as }}>
    @if ($text)<span class="text-base leading-relaxed font-medium text-muted">{{ $text }}</span>@endif
</div>
@else
<div {{ $attributes->class('flex flex-col gap-3.5 rounded-5xl border border-line bg-surface p-7 text-ink') }}>
    <x-ui.icon-tile :icon="$icon" :color="$color"/>
    <{{ $as }} class="m-0 text-3xl font-bold">{{ $title }}</{{ $as }}>
    @if ($text)<span class="text-md leading-relaxed font-medium text-muted">{{ $text }}</span>@endif
</div>
@endif

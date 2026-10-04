{{--
    <x-ui.section-header eyebrow="مجله" title="برای همین مرحله" lead="…" align="start|center" tone="light|dark"
                         size="lg|md|sm" as="h2" id="readings-title" more-href="{{ route('blog.index') }}" more-label="همه"/>
    Eyebrow + h2 (Lalezar 40 / lh 1.2) + optional lead (17px, lh 2, muted, max-w 760) — AUDIT §2.2. With `more-href`
    the header becomes a space-between row with a trailing «همه» link (`more-icon="arrow-left"` adds the arrow, as on shop rows). Give `id` and point the section's
    `aria-labelledby` at it. The slot (optional) is appended under the lead.
--}}
@props([
    'eyebrow' => null,
    'title',
    'lead' => null,
    'align' => 'start',
    'tone' => 'light',
    'size' => 'lg',
    'as' => 'h2',
    'id' => null,
    'moreHref' => null,
    'moreLabel' => 'همه',
    'moreIcon' => null,
])
@php
    $center = $align === 'center';
    $dark = $tone === 'dark';
    $titleSize = match ($size) {
        'md' => 'text-d-md',
        'sm' => 'text-d-sm',
        'xl' => 'text-d-xl',
        default => 'text-d-lg',
    };
@endphp
@if ($moreHref)
<div {{ $attributes->class('flex items-end justify-between gap-4 max-lg:flex-wrap') }}>
@endif
<div @if (! $moreHref) {{ $attributes->class(['flex flex-col gap-3', 'items-center text-center' => $center]) }} @else class="flex flex-col gap-3" @endif>
    @if ($eyebrow)<x-ui.eyebrow :tone="$tone">{{ $eyebrow }}</x-ui.eyebrow>@endif
    <{{ $as }} @if ($id) id="{{ $id }}" @endif @class(['m-0 font-display font-normal leading-heading', $titleSize, $dark ? 'text-on-night' : 'text-ink'])>{{ $title }}</{{ $as }}>
    @if ($lead)<p @class(['m-0 max-w-190 text-xl leading-loose font-medium max-sm:max-w-full', $dark ? 'text-on-night-muted' : 'text-muted'])>{{ $lead }}</p>@endif
    {{ $slot }}
</div>
@if ($moreHref)
    <x-ui.button :href="$moreHref" variant="ghost" :tone="$tone" icon-class="size-4" :icon-end="$moreIcon">{{ $moreLabel }}</x-ui.button>
</div>
@endif

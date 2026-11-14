{{--
    <x-site.button href="#download" icon="download">دانلود رایگان ریتمی</x-site.button>
    <x-site.button href="…" variant="outline">انتخاب</x-site.button>   <x-site.button variant="rose" icon="phone">…</x-site.button>
    Redesign pill button, 54px (size="md": 52, "sm": 44): `ink` (default, white text), `outline` (1.5px ink border),
    `white` (white fill, rose text — on rose cards), `rose` (emergency). Renders a <button> when no href is given.
--}}
@props(['href' => null, 'variant' => 'ink', 'icon' => null, 'size' => 'lg', 'type' => 'button'])
@php
    $classes = match ($variant) {
        'outline' => 'border-[1.5px] border-ink bg-surface text-ink hover:bg-canvas hover:text-ink',
        'white' => 'bg-surface text-rose hover:bg-rose-mist hover:text-rose',
        'rose' => 'bg-rose text-white hover:bg-rose/90 hover:text-white',
        default => 'bg-ink text-white hover:bg-ink/90 hover:text-white',
    };
    $sizes = match ($size) {
        'md' => 'h-13 px-6 text-lg',
        'sm' => 'h-11 px-5 text-md',
        default => 'h-13.5 px-7 text-lg',
    };
    $class = "inline-flex shrink-0 items-center justify-center gap-2.5 rounded-full font-extrabold transition-colors {$sizes} {$classes}";
@endphp
@if ($href !== null)
<a href="{{ $href }}" {{ $attributes->class($class) }}>@if ($icon)<x-icon :name="$icon" class="size-4.5"/>@endif{{ $slot }}</a>
@else
<button type="{{ $type }}" {{ $attributes->class($class) }}>@if ($icon)<x-icon :name="$icon" class="size-4.5"/>@endif{{ $slot }}</button>
@endif

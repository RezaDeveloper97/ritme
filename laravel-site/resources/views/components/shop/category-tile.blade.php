{{--
    <x-shop.category-tile :href="$url" name="لباس نوزاد" illustration="product-bodysuit" :media="$coverId" :index="0"/>
    Category tile of the shop home (shop.html «دسته‌های …»): 120px rounded tile on a stage tint (cycling in the
    design's order) with the cover photo or illustration, name under it. The whole tile is the link.
--}}
@props(['href', 'name', 'illustration' => null, 'media' => null, 'index' => 0])
@php
    $tints = ['bg-stage-pregnancy/15', 'bg-primary/13', 'bg-stage-postpartum/12', 'bg-primary/12', 'bg-primary/10', 'bg-stage-pregnancy/12', 'bg-primary/10', 'bg-stage-pregnancy/13'];
@endphp
<a href="{{ $href }}" {{ $attributes->class('flex flex-col items-center gap-2.5 text-center text-ink hover:text-primary') }}>
    <span class="relative block h-30 w-full shrink-0 overflow-hidden rounded-5xl {{ $tints[$index % count($tints)] }}">
        @if ($media)
            <x-picture :media="$media" decorative sizes="(max-width: 700px) 50vw, (max-width: 1024px) 25vw, 12vw" class="size-full object-cover"/>
        @elseif ($illustration)
            <x-illustration :name="$illustration" class="size-full"/>
        @endif
    </span>
    <span class="text-base font-bold">{{ $name }}</span>
</a>

{{--
    <x-cards.article :post="$postCard"/>                                       (PostCardData from the Blog context)
    <x-cards.article href="…" title="…" stage="cycle" label="چرخه" :minutes="5" :media="$mediaId"/>
    <x-cards.article :post="$featured" featured reviewer="…"/>                (blog's large horizontal card)
    Article card (AUDIT §2.3: index, blog, article related, stage «برای همین مرحله»): radius 24, 180px cover —
    <x-picture> of the cover when there is one, else the stage gradient (`stage/33 → stage/7`) with a book icon — stage
    label in the stage colour, title 18 lh 1.7 (h3), «مطالعه ۵ دقیقه». Explicit props override the DTO's values.
    The design's cover is 180px + 18px padding as content-box, so the (border-box) cover is h-54 = 216px. The featured
    text block keeps the design's `32px 0 32px 32px` padding (ps-0 pe-8) at every width, also when the card wraps.
--}}
@props([
    'post' => null,
    'href' => null,
    'title' => null,
    'stage' => null,
    'label' => null,
    'minutes' => null,
    'media' => null,
    'mobileMedia' => null,
    'excerpt' => null,
    'reviewer' => null,
    'featured' => false,
    'as' => 'h3',
])
@php
    if ($post instanceof \App\Domain\Blog\Data\PostCardData) {
        $href ??= route('blog.show', $post->slug);
        $title ??= $post->title;
        $stage ??= $post->lifeStage;
        $label ??= $post->category?->label ?? $post->lifeStage?->label();
        $minutes ??= $post->readingTime;
        $media ??= $post->coverMediaId;
        $mobileMedia ??= $post->mobileCoverMediaId;
        $excerpt ??= $post->excerpt;
    }
    $stageKey = $stage instanceof \App\Domain\Blog\Enums\LifeStage ? $stage->value : (string) ($stage ?? 'menopause');
    [$gradient, $labelColor] = match ($stageKey) {
        'cycle' => ['from-stage-cycle/33 to-stage-cycle/7', 'text-stage-cycle'],
        'ttc' => ['from-stage-ttc/33 to-stage-ttc/7', 'text-stage-ttc'],
        'pregnancy' => ['from-stage-pregnancy/33 to-stage-pregnancy/7', 'text-stage-pregnancy'],
        'postpartum' => ['from-stage-postpartum/33 to-stage-postpartum/7', 'text-stage-postpartum'],
        'teen' => ['from-stage-teen/33 to-stage-teen/7', 'text-stage-teen'],
        default => ['from-stage-menopause/33 to-stage-menopause/7', 'text-stage-menopause'],
    };
    $minutesText = $minutes !== null ? fa_digits($minutes) : null;
@endphp
@if ($featured)
<a href="{{ $href }}" {{ $attributes->class('flex items-center gap-8 overflow-hidden rounded-6xl border border-line bg-surface text-ink transition-shadow hover:text-ink hover:shadow-card max-lg:flex-wrap') }}>
    <span class="flex h-85 w-140 shrink-0 items-center justify-center overflow-hidden bg-linear-135/srgb {{ $gradient }} max-lg:w-full">
        @if ($media)
            <x-picture :media="$media" :mobile="$mobileMedia" decorative sizes="(max-width: 1024px) 100vw, 560px" class="size-full object-cover"/>
        @else
            <x-icon name="book" class="size-22.5 {{ $labelColor }}" stroke="1.8"/>
        @endif
    </span>
    <span class="flex flex-col gap-3.5 py-8 ps-0 pe-8">
        <span class="text-base font-extrabold text-primary">{{ $label }}@if ($minutesText) · {{ $minutesText }} دقیقه @endif</span>
        <{{ $as }} class="m-0 font-display text-[34px] leading-heading font-normal text-ink">{{ $title }}</{{ $as }}>
        @if ($excerpt)<span class="text-lg leading-loose font-medium text-muted">{{ $excerpt }}</span>@endif
        @if ($reviewer)<span class="text-sm-plus font-bold text-muted">بازبینی علمی: {{ $reviewer }}</span>@endif
    </span>
</a>
@else
<a href="{{ $href }}" {{ $attributes->class('flex flex-col overflow-hidden rounded-4xl border border-line bg-surface text-ink transition-shadow hover:text-ink hover:shadow-card') }}>
    <span class="relative flex h-54 items-end overflow-hidden bg-linear-135/srgb p-4.5 {{ $gradient }}">
        @if ($media)
            <x-picture :media="$media" :mobile="$mobileMedia" decorative sizes="(max-width: 700px) 100vw, 33vw" picture-class="absolute inset-0" class="size-full object-cover"/>
        @else
            <x-icon name="book" class="size-7.5 {{ $labelColor }}" stroke="1.8"/>
        @endif
    </span>
    <span class="flex flex-col gap-2.5 p-5">
        @if ($label)<span class="text-sm font-extrabold {{ $labelColor }}">{{ $label }}</span>@endif
        <{{ $as }} class="m-0 text-2xl leading-[1.7] font-bold">{{ $title }}</{{ $as }}>
        @if ($minutesText)<span class="text-sm font-semibold text-muted">مطالعه {{ $minutesText }} دقیقه</span>@endif
    </span>
</a>
@endif

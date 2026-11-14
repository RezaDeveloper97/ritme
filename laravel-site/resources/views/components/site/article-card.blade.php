{{--
    <x-site.article-card :post="$postCard"/>   (PostCardData from the Blog context)
    Redesign article card: borderless, 200px cover (radius 16; the post cover through <x-picture>, else the stage's
    soft tint), stage/category label 13/800 in the stage colour, title 18/800 lh 1.7 (h3), «مطالعه ۵ دقیقه».
--}}
@props(['post', 'as' => 'h3'])
@php
    $stageKey = $post->lifeStage?->value ?? 'menopause';
    [$tint, $labelColor] = match ($stageKey) {
        'cycle' => ['bg-danger-soft', 'text-rose'],
        'ttc' => ['bg-sand', 'text-stage-ttc'],
        'pregnancy' => ['bg-peach', 'text-stage-pregnancy-warm'],
        'postpartum' => ['bg-aqua-soft', 'text-stage-postpartum'],
        'teen' => ['bg-success-soft', 'text-stage-teen'],
        default => ['bg-lavender', 'text-primary'],
    };
    $label = $post->category?->label ?? $post->lifeStage?->label();
@endphp
<a href="{{ route('blog.show', $post->slug) }}" {{ $attributes->class('group flex flex-col gap-3.5 text-ink hover:text-ink') }}>
    <span class="relative block h-50 overflow-hidden rounded-xl {{ $tint }}">
        @if ($post->coverMediaId)
            <x-picture :media="$post->coverMediaId" :mobile="$post->mobileCoverMediaId" decorative sizes="(max-width: 700px) 100vw, 33vw" picture-class="absolute inset-0" class="size-full object-cover transition-transform duration-300 group-hover:scale-[1.03] motion-reduce:transition-none"/>
        @endif
    </span>
    <span class="flex flex-col gap-1.5">
        @if ($label)<span class="text-sm font-extrabold {{ $labelColor }}">{{ $label }}</span>@endif
        <{{ $as }} class="m-0 text-2xl leading-[1.7] font-extrabold group-hover:text-primary">{{ $post->title }}</{{ $as }}>
        <span class="text-sm font-semibold text-muted">مطالعه {{ fa_digits($post->readingTime) }} دقیقه</span>
    </span>
</a>

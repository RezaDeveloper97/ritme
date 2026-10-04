{{--
    <x-blog.cover :cover="$page->cover" :mobile="$page->mobileCover" :alt="$page->coverAlt" :stage="$post->lifeStage"/>
    Article cover (design: 360px, radius 32). With media: <x-picture> as the LCP (`priority` → eager, high
    fetchpriority, preload). Without: the design's stage gradient with the drop icon (decorative).
--}}
@props(['cover' => null, 'mobile' => null, 'alt' => '', 'stage' => null])
@php
    $stageKey = $stage instanceof \App\Domain\Blog\Enums\LifeStage ? $stage->value : (string) ($stage ?? 'cycle');
    [$gradient, $iconColor] = match ($stageKey) {
        'ttc' => ['from-stage-ttc/20', 'text-stage-ttc'],
        'pregnancy' => ['from-stage-pregnancy/20', 'text-stage-pregnancy'],
        'postpartum' => ['from-stage-postpartum/20', 'text-stage-postpartum'],
        'menopause' => ['from-stage-menopause/20', 'text-stage-menopause'],
        'teen' => ['from-stage-teen/20', 'text-stage-teen'],
        default => ['from-stage-cycle/20', 'text-stage-cycle'],
    };
@endphp
@if ($cover)
    <x-picture :media="$cover" :mobile="$mobile" :alt="$alt" priority
               sizes="{{ \App\Domain\Blog\Rendering\ArticleBodyRenderer::SIZES }}"
               picture-class="block" class="h-90 w-full rounded-6xl object-cover max-sm:h-60"/>
@else
    <div aria-hidden="true" class="flex h-90 items-center justify-center rounded-6xl bg-linear-135/srgb {{ $gradient }} to-primary/13 max-lg:flex-wrap">
        <x-icon name="drop" class="size-25 {{ $iconColor }}"/>
    </div>
@endif

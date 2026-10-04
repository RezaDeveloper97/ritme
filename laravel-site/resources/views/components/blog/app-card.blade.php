{{--
    <x-blog.app-card :stage="$post->lifeStage" :href="$page->downloadUrl"/> — the sidebar «… در ریتمی» card of the
    article (design: night card, radius 28). It is the page's `#download` target (the header's «دانلود اپ» lands
    here); the button leads to the full download block (store badges + QR) on the home page.
    Copy is generic for every article (the design's «دفترچه درد» copy only fits one post) and per-stage icon.
--}}
@props([
    'stage' => null,
    'href',
    'title' => __('blog.article.app_card.title'),
    'lead' => __('blog.article.app_card.lead'),
    'cta' => __('blog.article.app_card.cta'),
    'id' => 'download',
])
@php
    $stageKey = $stage instanceof \App\Domain\Blog\Enums\LifeStage ? $stage->value : (string) ($stage ?? 'cycle');
    $icon = match ($stageKey) {
        'ttc' => 'egg',
        'pregnancy' => 'heart',
        'postpartum' => 'bottle',
        'menopause' => 'thermometer',
        'teen' => 'sprout',
        default => 'flame',
    };
@endphp
<section id="{{ $id }}" aria-labelledby="{{ $id }}-title" {{ $attributes->class('flex flex-col gap-3 rounded-5xl bg-night p-6 text-on-night') }}>
    <x-icon :name="$icon" class="size-7.5 text-phase-period"/>
    <h2 id="{{ $id }}-title" class="m-0 font-sans text-2xl leading-normal font-bold text-on-night">{{ $title }}</h2>
    <p class="m-0 text-base leading-relaxed text-on-night-muted">{{ $lead }}</p>
    <a href="{{ $href }}" class="inline-flex h-13.5 items-center gap-2 rounded-full border-[1.5px] border-lilac bg-lilac px-6.5 text-[15.5px] font-extrabold text-night hover:text-night">
        <x-icon name="download" class="size-4.5 text-night"/>{{ $cta }}
    </a>
</section>

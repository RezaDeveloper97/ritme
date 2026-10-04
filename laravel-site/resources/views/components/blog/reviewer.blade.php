{{--
    <x-blog.reviewer :post="$post" :author-url="…" :reviewer-url="…"/> — author + «بازبینی پزشکی» box (E-E-A-T).
    Says only what the data supports: who wrote it, who reviewed it and when; the note makes clear a review is
    not personal medical advice (content red lines: no diagnosis or guarantee claims).
--}}
@props(['post', 'authorUrl' => null, 'reviewerUrl' => null])
@php
    /** @var \App\Domain\Blog\Data\PostData $post */
    $author = $post->author;
    $reviewer = $post->reviewer;
@endphp
@if ($author || $reviewer)
<section aria-labelledby="article-people-title" {{ $attributes->class('flex flex-col gap-4 rounded-5xl border border-line bg-surface p-6') }}>
    <h2 id="article-people-title" class="sr-only">{{ __('blog.article.people.title') }}</h2>
    @if ($author)
        <div class="flex items-center gap-3">
            @if ($author->avatarMediaId)
                <x-picture :media="$author->avatarMediaId" :alt="$author->name" sizes="48px" class="size-12 rounded-full object-cover" picture-class="shrink-0"/>
            @else
                <x-ui.icon-tile icon="user" color="primary" size="sm" shape="circle"/>
            @endif
            <div class="flex flex-col">
                <span class="text-sm font-bold text-muted">{{ __('blog.article.people.author') }}</span>
                @if ($authorUrl)
                    <a href="{{ $authorUrl }}" class="text-md font-extrabold text-ink hover:text-primary">{{ $author->name }}</a>
                @else
                    <span class="text-md font-extrabold text-ink">{{ $author->name }}</span>
                @endif
                @if ($author->jobTitle)<span class="text-sm font-semibold text-muted">{{ $author->jobTitle }}</span>@endif
            </div>
        </div>
    @endif
    @if ($reviewer)
        <div @class(['flex flex-col gap-2', 'border-t border-t-line pt-4' => $author])>
            <span class="flex items-center gap-1.5 text-sm font-extrabold text-stage-postpartum">
                <x-icon name="shield-check" class="size-4 shrink-0"/>{{ __('blog.article.people.review') }}
            </span>
            <p class="m-0 text-base leading-relaxed font-semibold text-ink">
                @if ($reviewerUrl)
                    <a href="{{ $reviewerUrl }}" class="font-extrabold text-ink hover:text-primary">{{ $reviewer->name }}</a>@else<b>{{ $reviewer->name }}</b>@endif
                @if ($reviewer->credentials)<span class="text-muted">، {{ $reviewer->credentials }}</span>@endif
            </p>
            @if ($post->reviewedAt)
                <span class="text-sm font-semibold text-muted">{{ __('blog.article.people.reviewed_at') }} <time datetime="{{ $post->reviewedAt->format('Y-m-d') }}">{{ jdate($post->reviewedAt, 'j F Y') }}</time></span>
            @endif
            <p class="m-0 text-sm leading-relaxed text-muted">{{ __('blog.article.people.note') }}</p>
        </div>
    @endif
</section>
@endif

{{--
    <x-blog.adjacent :previous="$page->previous" :next="$page->next" :category="$post->category"/> — older / newer
    article of the same category. Renders nothing when the post is alone in its category.
--}}
@props(['previous' => null, 'next' => null, 'category' => null])
@if ($previous || $next)
<nav aria-label="{{ $category ? 'مقاله‌های قبلی و بعدی در '.$category->name : 'مقاله‌های قبلی و بعدی' }}" {{ $attributes->class('flex flex-col gap-2') }}>
    @foreach ([['post' => $next, 'label' => 'مقاله بعدی'], ['post' => $previous, 'label' => 'مقاله قبلی']] as $item)
        @if ($item['post'])
            <a href="{{ route('blog.show', $item['post']->slug) }}" class="flex flex-col gap-1 rounded-4xl border border-line bg-surface px-5 py-4 text-ink hover:border-primary hover:text-ink">
                <span class="text-sm font-bold text-muted">{{ $item['label'] }}</span>
                <span class="text-base leading-relaxed font-extrabold">{{ $item['post']->title }}</span>
            </a>
        @endif
    @endforeach
</nav>
@endif

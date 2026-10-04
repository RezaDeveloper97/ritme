{{--
    <x-blog.disclaimer/> — the medical disclaimer under every article (design wording, lang blog.article.disclaimer).
    Slot replaces the text.
--}}
<p {{ $attributes->class('m-0') }}>{{ $slot->isEmpty() ? __('blog.article.disclaimer') : $slot }}</p>

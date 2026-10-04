{{--
    <x-blog.disclaimer/> — the medical disclaimer under every article (design wording). Slot replaces the text.
--}}
<p {{ $attributes->class('m-0') }}>{{ $slot->isEmpty() ? 'این مطلب جایگزین مشاوره پزشکی نیست.' : $slot }}</p>

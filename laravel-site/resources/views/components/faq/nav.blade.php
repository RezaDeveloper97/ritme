{{--
    <x-faq.nav :groups="$groups" label="دسته‌های سؤال‌ها"/>
    /faq side nav (AUDIT §2.2 correction): plain links to the category anchors (`#<group slug>`), 15px, radius 14.
    The first link starts as current (lavender fill, 800, ink); the faq-filter module moves `aria-current` to the
    category the visitor jumps to. Works without JS (anchors).
--}}
@props(['groups' => [], 'label'])
<nav aria-label="{{ $label }}" {{ $attributes->class('w-60 shrink-0') }}>
    <ul class="m-0 flex list-none flex-col gap-1 p-0">
        @foreach ($groups as $group)
            <li><a href="#{{ $group->slug }}" @if ($loop->first) aria-current="true" @endif
                   class="block rounded-lg px-4 py-2.5 text-md font-semibold text-muted transition-colors hover:text-ink aria-[current=true]:bg-lavender aria-[current=true]:font-extrabold aria-[current=true]:text-ink">{{ $group->title }}</a></li>
        @endforeach
    </ul>
</nav>

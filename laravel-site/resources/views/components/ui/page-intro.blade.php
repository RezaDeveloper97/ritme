{{--
    <x-ui.page-intro eyebrow="مجله ریتمی" title="خواندنی‌های کوتاه، دقیق و بی‌قضاوت" lead="…" size="lg">
        <x-ui.chip-nav …/>   (optional: search, chips)
    </x-ui.page-intro>
    Light-page header band (AUDIT §2.4: blog, contact, faq, plus, services, tools, shop, directory): white → canvas
    gradient, eyebrow + the page's single h1 (Lalezar: lg 54 · md 48/52 · sm 44) + lead (17.5, muted, max-w 820),
    paddings 64/48 with the page gutters. `plain` drops the gradient (shop). The slot follows the lead.
--}}
@props(['eyebrow' => null, 'title', 'lead' => null, 'size' => 'lg', 'plain' => false, 'as' => 'h1'])
@php
    $titleSize = match ($size) {
        'md' => 'text-[52px] max-lg:text-[38px] max-sm:text-[30px]',
        'sm' => 'text-d-xl',
        default => 'text-d-2xl',
    };
@endphp
<section {{ $attributes->class(['flex flex-col gap-4.5 px-30 pt-16 pb-12 max-lg:px-5 max-lg:pt-[35.2px] max-lg:pb-[26.4px]', 'bg-linear-180/srgb from-white to-canvas' => ! $plain]) }}>
    @if ($eyebrow)<x-ui.eyebrow>{{ $eyebrow }}</x-ui.eyebrow>@endif
    <{{ $as }} class="m-0 font-display leading-display font-normal text-ink {{ $titleSize }}">{{ $title }}</{{ $as }}>
    @if ($lead)<p class="m-0 max-w-205 text-[17.5px] leading-loose font-medium text-muted">{{ $lead }}</p>@endif
    {{ $slot }}
</section>

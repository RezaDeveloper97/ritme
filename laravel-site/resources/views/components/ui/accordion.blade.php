{{--
    <x-ui.accordion :items="[['question' => '…', 'answer' => '…'], …]" heading-level="3"/>
    Native <details>/<summary> — no JS (AUDIT §2.2). First item open; summary 18/800, answer 16 lh 2, bottom border,
    inside a max-w 880 column. Answers are escaped; pass an HtmlString for trusted (sanitised) HTML. FAQPage JSON-LD is
    the page's job (x-faq, L3-09). `open="none"` closes all, `open="all"` opens all.
--}}
@props(['items' => [], 'open' => 'first'])
<div {{ $attributes->class('flex w-full max-w-220 flex-col') }}>
    @foreach ($items as $item)
        <details @if ($open === 'all' || ($open === 'first' && $loop->first)) open @endif class="border-b border-line py-5">
            <summary class="min-h-8 cursor-pointer text-2xl font-extrabold text-ink">{{ $item['question'] }}</summary>
            <div class="mt-2.5 text-lg leading-loose text-muted">{{ $item['answer'] }}</div>
        </details>
    @endforeach
</div>

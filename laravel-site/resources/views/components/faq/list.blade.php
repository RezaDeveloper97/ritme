{{--
    <x-faq.list :entries="[['question' => '…', 'answer' => '<p>…</p>'], …]" variant="list|grid" open="first|all|none"/>
    The markup of x-faq (use x-faq; this is its inner part). Answers are printed unescaped — sanitised HTML or trusted
    lang copy only. list: native <details> (first open), grid: 2-column cards (1 column ≤ 700).
--}}
@props(['entries' => [], 'variant' => 'list', 'open' => 'first'])
@php($answerHtml = '[&_a]:font-extrabold [&_a]:text-primary [&_ol]:list-decimal [&_ol]:ps-5 [&_p+p]:mt-3 [&_ul]:list-disc [&_ul]:ps-5')
@if ($variant === 'grid')
    <div {{ $attributes->class('grid grid-cols-2 gap-4 max-sm:grid-cols-1') }}>
        @foreach ($entries as $entry)
            <div class="flex flex-col gap-2 rounded-[22px] border border-line bg-surface p-5.5" data-faq-item>
                <h3 class="m-0 text-xl font-bold text-ink">{{ $entry['question'] }}</h3>
                <div class="text-md leading-relaxed font-medium text-muted {{ $answerHtml }}">{!! $entry['answer'] !!}</div>
            </div>
        @endforeach
    </div>
@else
    <div {{ $attributes->class('flex w-full max-w-220 flex-col') }}>
        @foreach ($entries as $entry)
            <details @if ($open === 'all' || ($open === 'first' && $loop->first)) open @endif class="border-b border-line py-5" data-faq-item>
                <summary class="min-h-8 cursor-pointer text-2xl font-extrabold text-ink">{{ $entry['question'] }}</summary>
                <div class="mt-2.5 text-lg leading-loose text-muted {{ $answerHtml }}">{!! $entry['answer'] !!}</div>
            </details>
        @endforeach
    </div>
@endif

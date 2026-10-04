{{--
    Terms of use (L3-08, AUDIT §8) — no design page; the privacy-page layout. Data from App\Http\Controllers\TermsController:
      $updatedAt  ?CarbonImmutable — «آخرین به‌روزرسانی» (also the page's dateModified)
      $sections   list<{title, paragraphs, items, link}> — lang/fa/terms.php (skeleton; «[…]» = legal team to fill)
--}}
@extends('layouts.app')

@section('hero')
    <section aria-labelledby="terms-title" class="flex max-w-305 flex-col gap-5.5 px-30 max-lg:max-w-255 pt-14 pb-27.5 max-lg:px-5 max-lg:pt-[30.8px] max-lg:pb-[60.5px]">
        <x-ui.eyebrow tone="dark">{{ __('terms.hero.eyebrow') }}</x-ui.eyebrow>
        <h1 id="terms-title" class="m-0 font-display text-[64px] leading-display font-normal text-on-night max-lg:text-[46px] max-sm:text-[34px]">{{ __('terms.hero.title_before') }}<span class="text-lilac">{{ __('terms.hero.title_highlight') }}</span>{{ __('terms.hero.title_after') }}</h1>
        <p class="m-0 max-w-180 text-[19px] leading-loose font-medium text-on-night-muted">{{ __('terms.hero.lead') }}</p>
    </section>
@endsection

@section('content')
    <x-ui.section aria-labelledby="terms-doc-title" class="flex flex-col gap-6">
        <div class="flex flex-col gap-1 rounded-5xl bg-lavender px-8 py-7">
            <h2 id="terms-doc-title" class="m-0 font-sans text-[19px] leading-normal font-bold">{{ __('terms.doc.title') }}</h2>
            @if ($updatedAt !== null)
                <p class="m-0 text-md font-semibold text-muted">{{ __('terms.doc.updated', ['date' => '']) }}<time datetime="{{ $updatedAt->format('Y-m-d') }}">{{ jdate($updatedAt, 'j F Y') }}</time></p>
            @endif
        </div>
        <div class="flex flex-col gap-8 rounded-5xl border border-line bg-surface p-10 max-sm:p-6">
            @foreach ($sections as $n => $section)
                <section aria-labelledby="terms-{{ $n + 1 }}" class="flex flex-col gap-3">
                    <h3 id="terms-{{ $n + 1 }}" class="m-0 text-xl font-bold text-ink">{{ fa_digits($n + 1) }}. {{ $section['title'] }}</h3>
                    @foreach ($section['paragraphs'] as $paragraph)
                        <p class="m-0 text-md leading-loose font-medium text-muted">{{ $paragraph }}</p>
                    @endforeach
                    @if ($section['items'] !== [])
                        <ul class="m-0 flex list-disc flex-col gap-2 ps-6 text-md leading-loose font-medium text-muted marker:text-primary">
                            @foreach ($section['items'] as $item)
                                <li>{{ $item }}</li>
                            @endforeach
                        </ul>
                    @endif
                    @if ($section['link'] !== null)
                        <a href="{{ $section['link']['url'] }}" class="self-start text-md font-extrabold">{{ $section['link']['text'] }}</a>
                    @endif
                </section>
            @endforeach
        </div>
    </x-ui.section>
@endsection

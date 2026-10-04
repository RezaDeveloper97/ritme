{{--
    Draft preview of a magazine post (PostPreviewController, signed URL, noindex). Plain data only; the body is the
    stored HTML, already sanitised on save (RichHtmlSanitizer). The real article template arrives with L4-03.
--}}
@extends('layouts.app', ['appCta' => false])

@section('content')
    <x-ui.section pad="lg">
        <p class="rounded-2xl border border-line bg-surface px-5 py-4 text-md font-bold text-ink" role="status">
            پیش‌نمایش — وضعیت: {{ $status }}. این صفحه برای موتورهای جست‌وجو نمایه نمی‌شود.
        </p>
        <article class="mt-8 max-w-[760px]">
            @if ($category)
                <p class="text-md font-bold text-primary">{{ $category }}</p>
            @endif
            <h1 class="mt-4 font-display text-d-xl leading-display text-ink">{{ $title }}</h1>
            @if ($excerpt)
                <p class="mt-4 text-xl leading-relaxed text-muted">{{ $excerpt }}</p>
            @endif
            <p class="mt-4 text-md text-muted">
                @if ($author) {{ $author }} · @endif
                @if ($reviewer) بازبینی: {{ $reviewer }} · @endif
                {{ $readingTime }} دقیقه مطالعه
            </p>
            @if ($coverId)
                <x-picture :media="$coverId" :mobile="$coverMobileId" sizes="(max-width: 760px) 100vw, 760px" priority class="mt-8 w-full rounded-2xl"/>
            @endif
            <div class="rt-prose mt-8">{!! $body !!}</div>
        </article>
    </x-ui.section>
@endsection

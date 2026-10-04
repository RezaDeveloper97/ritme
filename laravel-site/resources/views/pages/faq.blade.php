{{--
    FAQ page (L3-09), design/html/faq.html. Data from App\Http\Controllers\FaqController:
      $groups   list<FaqGroupData> — the /faq categories (`is_listed`, ≥ 1 published item), admin-ordered; their
                questions are the page's FAQPage JSON-LD (PageFaq)
      $appLinks AppLinksSettings, $qrUrl string — the #download card
    Category anchors = group slugs (side nav links). The search box filters questions in place with the faq-filter
    module; without JS every question stays visible.
--}}
@extends('layouts.app')

@section('content')
    <x-ui.page-intro eyebrow="سؤالات متداول" title="جواب سؤال‌های رایج" lead="جوابت را پیدا نکردی؟ از صفحه تماس یا پشتیبانی داخل اپ بپرس.">
        @if ($groups !== [])
            <div role="search" class="flex h-14.5 w-155 items-center gap-2.5 rounded-full border-[1.5px] border-line bg-surface px-5 focus-within:border-primary max-sm:w-full max-sm:max-w-full">
                <x-icon name="search" class="size-5 shrink-0 text-muted"/>
                <label for="faq-search" class="sr-only">جست‌وجو در سؤال‌ها</label>
                <input id="faq-search" type="search" autocomplete="off" enterkeyhint="search" placeholder="جست‌وجو در سؤال‌ها"
                       data-module="faq-filter" aria-controls="faq-groups"
                       class="h-full min-w-0 flex-1 border-0 bg-transparent text-lg text-ink outline-none placeholder:text-muted [&::-webkit-search-cancel-button]:appearance-none">
            </div>
        @endif
    </x-ui.page-intro>

    @if ($groups !== [])
        <section aria-label="سؤال‌ها بر اساس دسته" class="flex flex-col gap-10 px-30 pt-6 pb-24 max-lg:px-5 max-lg:pb-[52.8px]">
            <div class="flex items-start gap-12 max-lg:flex-wrap">
                <x-faq.nav :groups="$groups" label="دسته‌های سؤال‌ها"/>
                <div id="faq-groups" class="flex flex-1 flex-col gap-9 max-lg:basis-75 max-sm:basis-full">
                    @foreach ($groups as $group)
                        <section id="{{ $group->slug }}" aria-labelledby="{{ $group->slug }}-title" class="flex scroll-mt-6 flex-col gap-1.5" data-faq-group>
                            <h2 id="{{ $group->slug }}-title" class="m-0 font-display text-d-sm leading-heading font-normal text-ink">{{ $group->title }}</h2>
                            <x-faq :faq="$group" bare/>
                        </section>
                    @endforeach
                    <p class="m-0 text-lg text-muted" role="status" data-faq-empty hidden>سؤالی با این عبارت پیدا نشد. از <a href="{{ route('contact') }}" class="font-extrabold text-primary">صفحه تماس</a> بپرس.</p>
                </div>
            </div>
        </section>
    @endif

    <x-ui.app-cta
        title="ریتمی را رایگان نصب کن"
        lead="سؤالت را از داخل اپ هم می‌توانی بپرسی."
        :links="$appLinks"
        :qr-url="$qrUrl"
    />
@endsection

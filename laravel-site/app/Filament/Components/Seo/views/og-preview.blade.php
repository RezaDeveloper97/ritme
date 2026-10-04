{{-- Share-card preview (Open Graph / Twitter summary_large_image) for SeoFields. Data: $og (see SeoFields::ogPreview). --}}
<div class="flex flex-col gap-2" data-seo-og-preview>
    <p class="text-sm font-medium text-gray-950 dark:text-white">پیش‌نمایش کارت اشتراک‌گذاری</p>
    <figure class="max-w-lg overflow-hidden rounded-xl bg-white ring-1 ring-gray-950/10 dark:bg-gray-900 dark:ring-white/10">
        @if ($og['image'])
            <img src="{{ $og['image'] }}" alt="" width="1200" height="630" loading="lazy" class="aspect-[1200/630] w-full object-cover">
        @else
            <div class="flex aspect-[1200/630] w-full items-center justify-center bg-gray-100 text-sm text-gray-500 dark:bg-gray-800 dark:text-gray-400">بدون تصویر — کارت کوچک نمایش داده می‌شود</div>
        @endif
        <figcaption class="flex flex-col gap-1 border-t border-gray-950/5 p-3 dark:border-white/10">
            <span class="text-xs uppercase text-gray-500 dark:text-gray-400" dir="ltr">{{ $og['host'] }}</span>
            <span class="font-semibold text-gray-950 dark:text-white" data-og-title>{{ $og['title'] }}</span>
            <span class="text-sm text-gray-600 dark:text-gray-300">{{ $og['description'] }}</span>
        </figcaption>
    </figure>
</div>

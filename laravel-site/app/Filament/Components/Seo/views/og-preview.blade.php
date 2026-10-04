{{-- Share-card previews (Open Graph / Twitter summary_large_image, Telegram, WhatsApp) for SeoFields. Data: $og (see SeoFields::ogPreview). Local only: no external assets. --}}
<div class="flex flex-col gap-3" data-seo-og-preview>
    <p class="text-sm font-medium text-gray-950 dark:text-white">پیش‌نمایش کارت اشتراک‌گذاری</p>

    <div class="grid gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
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

        <div class="flex flex-col gap-4">
            <figure class="max-w-sm rounded-2xl bg-sky-50 p-2 dark:bg-gray-800" data-og-telegram>
                <figcaption class="mb-1 px-1 text-xs text-gray-500 dark:text-gray-400">تلگرام</figcaption>
                <div class="rounded-xl bg-white p-2 shadow-sm dark:bg-gray-900">
                    <span class="block truncate text-xs text-sky-700 dark:text-sky-300" dir="ltr">{{ $og['host'] }}</span>
                    <div class="mt-1 border-s-2 border-sky-500 ps-2">
                        <span class="block text-sm font-semibold text-sky-700 dark:text-sky-300">{{ $og['site'] ?? $og['host'] }}</span>
                        <span class="block text-sm font-semibold text-gray-950 dark:text-white">{{ $og['title'] }}</span>
                        <span class="line-clamp-3 text-xs text-gray-700 dark:text-gray-300">{{ $og['description'] }}</span>
                        @if ($og['image'])
                            <img src="{{ $og['image'] }}" alt="" width="1200" height="630" loading="lazy" class="mt-2 aspect-[1200/630] w-full rounded-lg object-cover">
                        @endif
                    </div>
                </div>
            </figure>

            <figure class="max-w-sm rounded-2xl bg-emerald-50 p-2 dark:bg-gray-800" data-og-whatsapp>
                <figcaption class="mb-1 px-1 text-xs text-gray-500 dark:text-gray-400">واتس‌اپ</figcaption>
                <div class="overflow-hidden rounded-xl bg-emerald-100 dark:bg-emerald-950">
                    <div class="flex gap-2 bg-black/5 p-2 dark:bg-white/5">
                        @if ($og['image'])
                            <img src="{{ $og['image'] }}" alt="" width="1200" height="630" loading="lazy" class="size-16 shrink-0 rounded-md object-cover">
                        @endif
                        <span class="min-w-0">
                            <span class="block truncate text-sm font-semibold text-gray-950 dark:text-white">{{ $og['title'] }}</span>
                            <span class="line-clamp-2 text-xs text-gray-700 dark:text-gray-300">{{ $og['description'] }}</span>
                            <span class="block truncate text-[11px] text-gray-500 dark:text-gray-400" dir="ltr">{{ $og['host'] }}</span>
                        </span>
                    </div>
                </div>
            </figure>
        </div>
    </div>
</div>

{{-- Google result preview (desktop + mobile) for SeoFields. Data: $serp (see SeoFields::serpPreview). Local only: no external assets. --}}
<div class="flex flex-col gap-3" data-seo-serp-preview>
    <div class="flex items-center justify-between gap-2">
        <p class="text-sm font-medium text-gray-950 dark:text-white">پیش‌نمایش در گوگل</p>
        @if ($serp['noindex'])
            <span class="rounded-md bg-danger-50 px-2 py-0.5 text-xs font-medium text-danger-700 dark:bg-danger-400/10 dark:text-danger-400">noindex — در نتایج نمایش داده نمی‌شود</span>
        @endif
    </div>

    <div class="grid gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <figure class="rounded-xl bg-white p-4 ring-1 ring-gray-950/10 dark:bg-gray-900 dark:ring-white/10">
            <figcaption class="mb-2 text-xs text-gray-500 dark:text-gray-400">دسکتاپ</figcaption>
            <div class="flex items-center gap-2">
                <span class="flex size-7 shrink-0 items-center justify-center rounded-full bg-gray-100 text-xs font-bold text-primary-600 dark:bg-gray-800">{{ mb_substr($serp['site'], 0, 1) }}</span>
                <span class="min-w-0 leading-tight">
                    <span class="block truncate text-sm text-gray-900 dark:text-gray-100">{{ $serp['site'] }}</span>
                    <span class="block truncate text-xs text-gray-600 dark:text-gray-400" dir="ltr">{{ $serp['breadcrumb'] }}</span>
                </span>
            </div>
            <p class="mt-1 text-xl leading-snug text-blue-800 dark:text-blue-300" data-serp-title>{{ $serp['desktopTitle'] }}</p>
            <p class="mt-1 text-sm leading-relaxed text-gray-700 dark:text-gray-300" data-serp-description>{{ $serp['desktopDescription'] }}</p>
        </figure>

        <figure class="max-w-sm rounded-2xl bg-white p-4 shadow-sm ring-1 ring-gray-950/10 dark:bg-gray-900 dark:ring-white/10">
            <figcaption class="mb-2 text-xs text-gray-500 dark:text-gray-400">موبایل</figcaption>
            <div class="flex items-center gap-2">
                <span class="flex size-6 shrink-0 items-center justify-center rounded-full bg-gray-100 text-[10px] font-bold text-primary-600 dark:bg-gray-800">{{ mb_substr($serp['site'], 0, 1) }}</span>
                <span class="min-w-0 leading-tight">
                    <span class="block truncate text-xs text-gray-900 dark:text-gray-100">{{ $serp['site'] }}</span>
                    <span class="block truncate text-[11px] text-gray-600 dark:text-gray-400" dir="ltr">{{ $serp['host'] }}</span>
                </span>
            </div>
            <p class="mt-2 text-lg leading-snug text-blue-800 dark:text-blue-300">{{ $serp['mobileTitle'] }}</p>
            <p class="mt-1 text-sm leading-relaxed text-gray-700 dark:text-gray-300">{{ $serp['mobileDescription'] }}</p>
        </figure>
    </div>
    <p class="text-xs text-gray-500 dark:text-gray-400">تقریبی است؛ گوگل ممکن است عنوان یا توضیح را خودش بازنویسی کند.</p>
</div>

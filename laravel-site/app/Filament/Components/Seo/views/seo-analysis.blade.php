{{-- Live SEO content analysis for SeoFields. Data: $analysis (?SeoAnalysis). Local only: no external assets.
     Re-renders with every debounced form update; x-intersect refreshes when the tab is shown (body edited elsewhere). --}}
@use('App\Domain\Seo\Analysis\Severity')
@php
    $tones = [
        'pass' => ['dot' => 'bg-success-500', 'text' => 'text-success-700 dark:text-success-400'],
        'info' => ['dot' => 'bg-gray-400', 'text' => 'text-gray-600 dark:text-gray-400'],
        'warning' => ['dot' => 'bg-warning-500', 'text' => 'text-warning-700 dark:text-warning-400'],
        'error' => ['dot' => 'bg-danger-500', 'text' => 'text-danger-700 dark:text-danger-400'],
    ];
    $scoreTone = [
        'success' => 'bg-success-50 text-success-700 ring-success-600/20 dark:bg-success-400/10 dark:text-success-400',
        'warning' => 'bg-warning-50 text-warning-700 ring-warning-600/20 dark:bg-warning-400/10 dark:text-warning-400',
        'danger' => 'bg-danger-50 text-danger-700 ring-danger-600/20 dark:bg-danger-400/10 dark:text-danger-400',
    ];
@endphp
<div class="flex flex-col gap-3" data-seo-analysis x-data x-intersect="$wire.$refresh()">
    @if ($analysis === null)
        <p class="text-sm text-gray-500 dark:text-gray-400">تحلیل در دسترس نیست.</p>
    @else
        <div class="flex flex-wrap items-center gap-3">
            <span class="inline-flex items-baseline gap-1 rounded-xl px-3 py-1.5 ring-1 {{ $scoreTone[$analysis->color()] }}" data-seo-score="{{ $analysis->score }}">
                <span class="text-2xl font-bold leading-none">{{ fa_digits($analysis->score) }}</span>
                <span class="text-xs">از ۱۰۰</span>
            </span>
            <span class="text-sm text-gray-700 dark:text-gray-300">
                {{ $analysis->needsWork() ? 'نیاز به کار دارد' : ($analysis->color() === 'success' ? 'آماده انتشار' : 'قابل قبول، اما بهتر می‌شود') }}
            </span>
            <span class="flex flex-wrap gap-2 text-xs text-gray-600 dark:text-gray-400">
                <span>{{ fa_digits($analysis->count(Severity::Error)) }} مشکل</span>
                <span aria-hidden="true">·</span>
                <span>{{ fa_digits($analysis->count(Severity::Warning)) }} قابل بهبود</span>
                <span aria-hidden="true">·</span>
                <span>{{ fa_digits($analysis->count(Severity::Pass)) }} خوب</span>
            </span>
            <button type="button" wire:click="$refresh" wire:loading.attr="disabled"
                class="ms-auto rounded-lg px-2.5 py-1 text-xs font-medium text-primary-600 ring-1 ring-gray-950/10 hover:bg-gray-50 dark:text-primary-400 dark:ring-white/10 dark:hover:bg-white/5">
                بررسی دوباره
            </button>
        </div>

        <ul class="divide-y divide-gray-950/5 rounded-xl bg-white ring-1 ring-gray-950/10 dark:divide-white/5 dark:bg-gray-900 dark:ring-white/10">
            @foreach ($analysis->checks as $check)
                <li class="flex items-start gap-3 px-4 py-2.5" data-seo-check="{{ $check->key }}" data-severity="{{ $check->severity->value }}">
                    <span class="mt-1.5 size-2.5 shrink-0 rounded-full {{ $tones[$check->severity->value]['dot'] }}" aria-hidden="true"></span>
                    <span class="min-w-0 flex-1">
                        <span class="flex flex-wrap items-baseline gap-x-2">
                            <span class="text-sm font-medium text-gray-950 dark:text-white">{{ $check->label }}</span>
                            <span class="text-xs {{ $tones[$check->severity->value]['text'] }}">{{ $check->severity->label() }}</span>
                        </span>
                        <span class="block text-sm text-gray-600 dark:text-gray-400">{{ $check->message }}</span>
                    </span>
                </li>
            @endforeach
        </ul>
        <p class="text-xs text-gray-500 dark:text-gray-400">راهنماست، نه حکم: متن را برای خواننده بنویسید. واژه‌های خط قرمز و ادعای تشخیص پیش از انتشار باید اصلاح شوند.</p>
    @endif
</div>

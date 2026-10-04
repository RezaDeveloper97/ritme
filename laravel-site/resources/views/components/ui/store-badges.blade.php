{{--
    <x-ui.store-badges :links="$appLinks" tone="dark|light"/>
    The four download badges (کافه‌بازار، مایکت، گوگل‌پلی، نسخه iOS): 52px content + 1px border (54, as the design), radius 14, «دریافت از» 10.5px at 75%
    (AUDIT §2.2). `links` is the `AppLinksSettings` DTO (or a store-key ⇒ URL array) handed down by the controller —
    this component never reads settings itself. A badge whose link is empty (or not http(s)/relative) is hidden;
    nothing renders when all are empty.
--}}
@props(['links' => null, 'tone' => 'dark'])
@php
    $urls = $links instanceof \App\Domain\Settings\Data\AppLinksSettings ? $links->toArray() : (array) ($links ?? []);
    $badges = [];
    foreach (['bazaar' => 'کافه‌بازار', 'myket' => 'مایکت', 'google_play' => 'گوگل‌پلی', 'app_store' => 'نسخه iOS'] as $store => $label) {
        $url = $urls[$store] ?? null;
        if (is_string($url) && preg_match('~^(https?://|/|#)~i', $url) === 1) {
            $badges[] = ['label' => $label, 'url' => $url, 'external' => preg_match('~^https?://~i', $url) === 1];
        }
    }
    $badgeClasses = $tone === 'dark'
        ? 'border-night-line bg-night-card text-on-night hover:border-lilac hover:text-on-night'
        : 'border-line bg-surface text-ink hover:border-primary hover:text-ink';
@endphp
@if ($badges !== [])
<ul {{ $attributes->class('m-0 flex list-none flex-wrap gap-2.5 p-0') }} aria-label="دریافت اپ ریتمی">
    @foreach ($badges as $badge)
        <li><a href="{{ $badge['url'] }}" @if ($badge['external']) rel="noopener" @endif class="box-content flex h-13 items-center gap-2.5 rounded-lg border px-4.5 text-base font-extrabold transition-colors {{ $badgeClasses }}">
            <x-icon name="download" class="size-4.5"/>
            <span class="flex flex-col leading-snug"><span class="text-2xs font-semibold opacity-75">دریافت از</span>{{ $badge['label'] }}</span>
        </a></li>
    @endforeach
</ul>
@endif

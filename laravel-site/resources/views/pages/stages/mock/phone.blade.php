{{--
    CSS phone mock-up (AUDIT §2.4 `x-mock.phone`): night screen, deep bezel, notch, shadow-phone. Always decorative —
    the caller wraps it in aria-hidden (or it adds aria-hidden itself). $size: hero (300×620, content 1:1) |
    feature (255×527, content scaled .85). $screen: partial in mock/screens, $data: its copy (array).
    The screen HTML is a cache-aside fragment (`pages` ns) keyed by screen, copy, template mtime and the Vite
    manifest hash (sprite URLs inside).
--}}
@php
    $hero = ($size ?? 'feature') === 'hero';
    $screenView = 'pages.stages.mock.screens.'.$screen;
    $screenPath = resource_path('views/pages/stages/mock/screens/'.$screen.'.blade.php');
    $screenHtml = app(\App\Support\Cache\CacheAside::class)->remember(
        \App\Support\Cache\CacheKey::make('pages', 'mock', $screen, md5(serialize($data ?? [])), (string) (@filemtime($screenPath) ?: 0), (string) app(\Illuminate\Foundation\Vite::class)->manifestHash()),
        null,
        static fn (): string => view($screenView, ['data' => $data ?? []])->render(),
    );
@endphp
<div aria-hidden="true" @class([
    'box-border shrink-0 overflow-hidden bg-night shadow-phone',
    'h-155 w-75 rounded-[48px] border-10 border-night-deep' => $hero,
    'h-[527px] w-[255px] rounded-7xl border-8 border-night-deep max-sm:rounded-4xl' => ! $hero,
])>
    <div @class(['flex items-center justify-center', 'h-8.5' => $hero, 'h-7' => ! $hero])>
        <span @class(['rounded-md bg-night-deep', 'h-5.5 w-22.5' => $hero, 'h-4.5 w-19' => ! $hero])></span>
    </div>
    <div @class(['mx-auto w-70 origin-top', 'scale-85 max-sm:scale-none' => ! $hero])>{!! $screenHtml !!}</div>
</div>

{{--
    @include('pages.home.mock.phone', ['size' => 'hero|split', 'screen' => 'cycle-today|companion'])
    CSS phone mockup (AUDIT §2.4 x-mock.phone): hero 300×620 (border 10, content at 1.0), feature splits 255×527
    (border 8, content scaled .85). Decorative: aria-hidden, the words are lang/fa/home.php `mock.*`.
--}}
@php($hero = ($size ?? 'split') === 'hero')
<div aria-hidden="true" @class([
    'box-border shrink-0 overflow-hidden bg-night border-night-deep shadow-phone',
    'h-155 w-75 rounded-[48px] border-10' => $hero,
    'h-[527px] w-[255px] rounded-7xl border-8 max-sm:rounded-4xl' => ! $hero,
])>
    <div @class(['flex items-center justify-center', 'h-8.5' => $hero, 'h-7' => ! $hero])>
        <span @class(['rounded-md bg-night-deep', 'h-5.5 w-22.5' => $hero, 'h-4.5 w-19' => ! $hero])></span>
    </div>
    <div @class(['mx-auto w-70 origin-top max-sm:scale-none', 'scale-85' => ! $hero])>
        @include('pages.home.mock.'.$screen)
    </div>
</div>

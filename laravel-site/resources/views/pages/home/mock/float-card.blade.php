{{--
    @include('pages.home.mock.float-card', ['icon' => 'drop', 'tone' => 'period|lilac', 'title' => '…', 'text' => '…', 'class' => 'top-22.5 -start-2.5'])
    Glass card floating beside the hero phone (AUDIT §2.4 x-mock.float-card). Position classes come from the caller.
--}}
<div class="{{ $class ?? '' }} absolute flex min-w-52.5 items-center gap-3 rounded-[22px] border border-night-line bg-night-card/82 px-4 py-3.5 shadow-float backdrop-blur-sm">
    <span @class([
        'flex size-10.5 shrink-0 items-center justify-center rounded-lg',
        'bg-phase-period/15 text-phase-period' => ($tone ?? 'lilac') === 'period',
        'bg-lilac/15 text-lilac' => ($tone ?? 'lilac') !== 'period',
    ])><x-icon :name="$icon" class="size-5.5"/></span>
    <span class="flex flex-col gap-0.5">
        <b class="text-base text-on-night">{{ $title }}</b>
        <span class="text-[12.5px] font-semibold text-on-night-muted">{{ $text }}</span>
    </span>
</div>

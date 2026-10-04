{{--
    Stage pills (AUDIT §2.1 `x-layout.stage-nav`) — first row of the dark hero block on the six stage pages.
    $items: list<StageNavItemData>, $label: nav aria-label. Active pill: border in the stage colour + stage/20 fill.
    (Lives here, not in components/layout, because this task owns only pages/stages; index can @include it.)
--}}
@php
    $activeClasses = static fn (string $color): string => match ($color) {
        'ttc' => 'border-stage-ttc bg-stage-ttc/20',
        'pregnancy' => 'border-stage-pregnancy bg-stage-pregnancy/20',
        'postpartum' => 'border-stage-postpartum bg-stage-postpartum/20',
        'menopause' => 'border-stage-menopause bg-stage-menopause/20',
        'teen' => 'border-stage-teen bg-stage-teen/20',
        default => 'border-stage-cycle bg-stage-cycle/20',
    };
@endphp
<nav aria-label="{{ $label }}" class="pt-2">
    <ul class="m-0 flex list-none flex-wrap gap-2 px-30 max-lg:px-5">
        @foreach ($items as $item)
            <li>
                <a href="{{ $item->url }}" @if ($item->active) aria-current="page" @endif @class([
                    'box-content flex h-10 items-center gap-2 rounded-full border px-4 text-sm-plus text-on-night transition-colors hover:text-on-night',
                    $activeClasses($item->color).' font-extrabold' => $item->active,
                    'border-night-line font-semibold hover:border-lilac' => ! $item->active,
                ])>
                    <x-icon :name="$item->icon" @class(['size-4', 'text-white' => $item->active, 'text-on-night-muted' => ! $item->active])/>{{ $item->label }}
                </a>
            </li>
        @endforeach
    </ul>
</nav>

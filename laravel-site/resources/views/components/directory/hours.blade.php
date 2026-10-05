{{--
    <x-directory.hours :rows="[['label' => 'شنبه', 'hours' => '۹ تا ۲۰', 'closed' => false, 'today' => true], …]"/>
    Weekly opening hours (L5-03): seven tiles, Saturday first; closed days dashed + muted (L9-03: no opacity fade, which
    dropped the text to 3.3:1), today's tile outlined in primary
    with an «امروز» label (the controller flags today only when it cannot change within the page-cache TTL).
--}}
@props(['rows' => []])
<ul {{ $attributes->class('m-0 grid list-none grid-cols-7 gap-2 p-0 max-lg:grid-cols-4 max-sm:grid-cols-2') }}>
    @foreach ($rows as $row)
        <li @class([
            'flex flex-col gap-1.5 rounded-xl border px-2 py-3 text-center',
            'border-primary bg-lavender' => $row['today'],
            'border-line bg-surface' => ! $row['today'] && ! $row['closed'],
            'border-dashed border-line bg-canvas text-muted' => $row['closed'] && ! $row['today'],
        ]) @if ($row['today']) aria-current="date" @endif>
            <b class="text-sm-plus">{{ $row['label'] }}@if ($row['today'])<span class="sr-only"> ({{ __('directory.place.hours.today') }})</span>@endif</b>
            <span class="text-[12.5px] leading-[1.7] font-semibold text-muted">{{ $row['hours'] }}</span>
        </li>
    @endforeach
</ul>

{{--
    <x-ui.timeline label="وضعیت درخواست" :items="[
        ['title' => 'درخواست ثبت شد', 'time' => 'امروز ۱۲:۴۰', 'state' => 'done'],
        ['title' => 'بررسی مدارک', 'time' => '…', 'state' => 'current'],
        ['title' => 'انتشار صفحه مجموعه', 'time' => '…', 'state' => 'todo'],
    ]"/>
    Vertical status list (AUDIT §2.2: directory-join-done, shop-done per seller): 14px dot (done = postpartum teal,
    current = primary, todo = hollow) joined by a 2px line, title 13.5 + time 11.5. Times are display strings —
    format dates with the Jalali helper before passing them. States are spoken via sr-only text.
--}}
@props(['items' => [], 'label' => null])
@php($stateText = ['done' => 'انجام شد', 'current' => 'در جریان', 'todo' => 'در انتظار'])
<ol @if ($label) aria-label="{{ $label }}" @endif {{ $attributes->class('m-0 flex list-none flex-col p-0') }}>
    @foreach ($items as $item)
        @php($state = $item['state'] ?? 'todo')
        <li class="flex gap-3" @if ($state === 'current') aria-current="step" @endif>
            <span aria-hidden="true" class="flex flex-col items-center">
                <span @class([
                    'mt-1 size-3.5 shrink-0 rounded-full border-2',
                    'border-stage-postpartum bg-stage-postpartum' => $state === 'done',
                    'border-primary bg-primary' => $state === 'current',
                    'border-line bg-transparent' => $state === 'todo',
                ])></span>
                @unless ($loop->last)<span class="min-h-7 w-0.5 grow bg-line"></span>@endunless
            </span>
            <span class="flex flex-col pb-3.5">
                <b @class(['text-sm-plus', $state === 'todo' ? 'text-muted' : 'text-ink'])>{{ $item['title'] }}<span class="sr-only"> ({{ $stateText[$state] ?? '' }})</span></b>
                @if (! empty($item['time']))<span class="text-[11.5px] font-semibold text-muted">{{ $item['time'] }}</span>@endif
            </span>
        </li>
    @endforeach
</ol>

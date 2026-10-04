{{--
    <x-ui.stepper variant="list|bar" :current="2" :steps="[['label' => 'معرفی', 'hint' => 'کامل شد'], ['label' => 'مکان و تصاویر', 'hint' => 'در حال پر کردن'], …]" label="مراحل"/>
    Progress header with done / current / todo states (AUDIT §2.2). `list`: the directory-join side nav (36px circles —
    check for done, Persian number otherwise; the current row is a white card). `bar`: the shop-checkout progress
    (segments + labels «سبد › ارسال و پرداخت › تأیید»). An ordered list; the current step has aria-current="step" and
    every state has screen-reader text. Optional slot (list): a note under the steps.
--}}
@props(['steps' => [], 'current' => 1, 'variant' => 'list', 'label' => 'مراحل'])
@php
    $stateOf = static fn (int $i): string => $i < $current ? 'done' : ($i === $current ? 'current' : 'todo');
    $stateText = ['done' => 'انجام شد', 'current' => 'مرحله فعلی', 'todo' => 'مانده'];
@endphp
@if ($variant === 'bar')
<nav aria-label="{{ $label }}" {{ $attributes->class('flex flex-col gap-2') }}>
    <div aria-hidden="true" class="flex gap-1.5">
        @foreach ($steps as $step)
            <span @class(['h-1.5 grow rounded-full', $loop->iteration <= $current ? 'bg-primary' : 'bg-line'])></span>
        @endforeach
    </div>
    <ol class="m-0 flex list-none justify-between p-0 text-[11.5px] font-bold">
        @foreach ($steps as $step)
            @php($state = $stateOf($loop->iteration))
            <li @if ($state === 'current') aria-current="step" @endif @class(['text-ink' => $state === 'current', 'text-muted' => $state !== 'current'])>{{ $step['label'] }}<span class="sr-only"> ({{ $stateText[$state] }})</span></li>
        @endforeach
    </ol>
</nav>
@else
<nav aria-label="{{ $label }}" {{ $attributes->class('flex flex-col gap-1.5') }}>
    <ol class="m-0 flex list-none flex-col gap-1.5 p-0">
        @foreach ($steps as $step)
            @php($state = $stateOf($loop->iteration))
            <li @if ($state === 'current') aria-current="step" @endif @class([
                'flex items-center gap-3.5 rounded-3xl border px-4 py-3.5',
                'border-line bg-surface' => $state === 'current',
                'border-transparent' => $state !== 'current',
            ])>
                <span aria-hidden="true" @class([
                    'flex size-9 shrink-0 items-center justify-center rounded-full text-base font-extrabold',
                    'bg-stage-teen text-white' => $state === 'done',
                    'bg-primary text-white' => $state === 'current',
                    'bg-line text-muted' => $state === 'todo',
                ])>@if ($state === 'done')<x-icon name="check" class="size-4.5"/>@else{{ fa_digits($loop->iteration) }}@endif</span>
                <span class="flex flex-col">
                    <b @class(['text-md', $state === 'todo' ? 'text-muted' : 'text-ink'])>{{ $step['label'] }}</b>
                    <span class="text-[12.5px] font-semibold text-muted">{{ $step['hint'] ?? '' }}<span class="sr-only">{{ ! empty($step['hint']) ? ' · ' : '' }}{{ $stateText[$state] }}</span></span>
                </span>
            </li>
        @endforeach
    </ol>
    {{ $slot }}
</nav>
@endif

{{--
    <x-home.feature-tile tint="bg-danger-soft" :title="…" :text="…" :backing="[…]" note="…" wide>…decorative mockup…</x-home.feature-tile>
    One value block of the home #features section: white card with a tinted stage holding the aria-hidden UI mockup
    (slot), then the title, a muted line, the features behind it as small chips and an optional caveat note.
    `wide` lays the stage beside the text on desktop (the two «وقتی مسیرت تغییر می‌کند» cards); phones always stack.
--}}
@props(['tint', 'title', 'text', 'backing' => [], 'note' => null, 'wide' => false])
<li @class(['flex flex-col rounded-4xl border border-line bg-surface p-2', 'xl:flex-row xl:items-stretch' => $wide])>
    <div aria-hidden="true" @class([
        'flex items-center justify-center overflow-hidden rounded-2xl p-4.5 max-lg:h-40',
        'h-47.5' => ! $wide,
        'h-47.5 xl:h-auto xl:w-[46%] xl:shrink-0' => $wide,
        $tint,
    ])>{{ $slot }}</div>
    <div class="flex flex-1 flex-col gap-2 px-3.5 pt-4 pb-3.5">
        <h3 class="m-0 text-2xl leading-normal font-extrabold text-ink">{{ $title }}</h3>
        <p class="m-0 text-md leading-[1.9] font-medium text-muted">{{ $text }}</p>
        @if ($backing !== [])
            <ul class="m-0 mt-auto flex list-none flex-wrap gap-1.5 p-0 pt-2" aria-label="{{ __('home.features.backing_label') }}">
                @foreach ($backing as $item)
                    <li class="rounded-lg bg-canvas px-2.5 py-1 text-xs font-bold text-muted">{{ $item }}</li>
                @endforeach
            </ul>
        @endif
        @if ($note)
            <p class="m-0 flex items-start gap-1.5 text-xs leading-[1.8] font-semibold text-muted">
                <x-icon name="info" class="mt-0.5 size-3.5 shrink-0"/>{{ $note }}
            </p>
        @endif
    </div>
</li>

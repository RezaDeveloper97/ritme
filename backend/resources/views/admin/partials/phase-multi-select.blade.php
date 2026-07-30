{{--
    Multi-phase picker: an article may be tagged with SEVERAL cycle phases and
    is then shown in each of them.

    $phases   array<int, array{value: string, label: string}> — from
              CycleSubphase::options(), the same list /admin/phase-contents shows.
    $selected array<int, string> — currently stored phase keys (empty = general)
--}}
@php $selected = old('cycle_phases', $selected ?? []) ?: []; @endphp

<div class="field full">
    <label>فازهای سیکل</label>
    <span class="hint">
        هر تعداد فاز که بخواهید انتخاب کنید؛ مقاله در همه‌ی آن‌ها نمایش داده می‌شود.
        اگر هیچ فازی انتخاب نشود، مقاله عمومی است و در همه‌ی فازها دیده می‌شود.
    </span>

    <div class="check-grid">
        @foreach ($phases as $phase)
            <label class="check-item" for="phase_{{ $loop->index }}">
                <input type="checkbox" id="phase_{{ $loop->index }}" name="cycle_phases[]"
                       value="{{ $phase['value'] }}" @checked(in_array($phase['value'], $selected, true))>
                <span>{{ $phase['label'] }}</span>
            </label>
        @endforeach
    </div>

    <div class="btn-row" style="margin-top:6px">
        <button type="button" class="btn sm ghost" data-check-all="1">انتخاب همه</button>
        <button type="button" class="btn sm ghost" data-check-all="0">حذف انتخاب‌ها</button>
    </div>

    @error('cycle_phases') <span class="err">{{ $message }}</span> @enderror
    @error('cycle_phases.*') <span class="err">{{ $message }}</span> @enderror
</div>

@once
    <script>
        // Bulk toggles for the phase grid — plain delegation, no framework.
        document.addEventListener('click', function (event) {
            var button = event.target.closest('[data-check-all]');
            if (!button) return;

            var grid = button.closest('.field').querySelector('.check-grid');
            var checked = button.dataset.checkAll === '1';
            grid.querySelectorAll('input[type=checkbox]').forEach(function (box) {
                box.checked = checked;
            });
        });
    </script>
@endonce

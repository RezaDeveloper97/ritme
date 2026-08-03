{{--
    Sub-phase picker for a recommendation.

    Every canonical sub-phase is rendered once, tagged with the phase it belongs
    to, and the list is filtered to the phase currently selected above — picking
    "لوتئال میانی" on a menstruation recommendation would be a row that can never
    fire, so it is never offered. The server enforces the same rule
    (RecommendationController::validated), the script is only the affordance.

    $subphases  array<int, array{value: string, label: string}> — CycleSubphase::options()
    $phaseOf    array<string, string> — sub-phase key => the phase key it sits in
    $selected   array<int, string> — currently stored sub-phase keys (empty = all)
--}}
@php $selected = old('cycle_subphases', $selected ?? []) ?: []; @endphp

<div class="field full" data-subphase-picker>
    <label>زیرفازها (اختیاری)</label>
    <span class="hint">
        برای محدودکردن توصیه به بخشی از فاز انتخاب‌شده. اگر هیچ‌کدام انتخاب نشود،
        توصیه در تمام زیرفازهای آن فاز نمایش داده می‌شود.
    </span>

    <div class="check-grid">
        @foreach ($subphases as $subphase)
            <label class="check-item" for="subphase_{{ $loop->index }}"
                   data-subphase-of="{{ $phaseOf[$subphase['value']] ?? '' }}">
                <input type="checkbox" id="subphase_{{ $loop->index }}" name="cycle_subphases[]"
                       value="{{ $subphase['value'] }}" @checked(in_array($subphase['value'], $selected, true))>
                <span>{{ $subphase['label'] }}</span>
            </label>
        @endforeach
    </div>

    @error('cycle_subphases') <span class="err">{{ $message }}</span> @enderror
    @error('cycle_subphases.*') <span class="err">{{ $message }}</span> @enderror
</div>

<script>
    (function () {
        var phaseSelect = document.getElementById('cycle_phase');
        var picker = document.querySelector('[data-subphase-picker]');
        if (!phaseSelect || !picker) return;

        function sync() {
            var phase = phaseSelect.value;
            var shown = 0;

            picker.querySelectorAll('[data-subphase-of]').forEach(function (item) {
                // No phase chosen means the row applies everywhere, so every
                // sub-phase is a legal target.
                var visible = !phase || item.dataset.subphaseOf === phase;
                item.hidden = !visible;
                // Clear a selection that just became impossible, so what the
                // admin sees is what gets submitted.
                if (visible) shown++;
                else item.querySelector('input').checked = false;
            });

            // A phase with a single sub-phase (menstruation) can't be narrowed,
            // so the picker would be a checkbox that changes nothing.
            picker.hidden = shown < 2;
        }

        phaseSelect.addEventListener('change', sync);
        sync();
    })();
</script>

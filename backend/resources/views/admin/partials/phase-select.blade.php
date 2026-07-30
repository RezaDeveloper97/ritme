{{--
    Phase picker shared by every admin content form.

    $phases   array<int, array{value: string, label: string}> — built by the
              controller from CyclePhase::options() or CycleSubphase::options(),
              so the option list has exactly one source of truth per model.
    $selected currently stored phase key (nullable)
--}}
<div class="field">
    <label for="cycle_phase">فاز سیکل</label>
    <select id="cycle_phase" name="cycle_phase">
        <option value="">— همه فازها —</option>
        @foreach ($phases as $phase)
            <option value="{{ $phase['value'] }}" @selected(old('cycle_phase', $selected) === $phase['value'])>{{ $phase['label'] }}</option>
        @endforeach
    </select>
    @error('cycle_phase') <span class="err">{{ $message }}</span> @enderror
</div>

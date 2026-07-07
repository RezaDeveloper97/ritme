<div class="field">
    <label for="cycle_phase">فاز سیکل</label>
    <select id="cycle_phase" name="cycle_phase">
        <option value="">— همه فازها —</option>
        @foreach ($phases as $phase)
            <option value="{{ $phase->value }}" @selected(old('cycle_phase', $selected) === $phase->value)>{{ $phase->label('fa') }}</option>
        @endforeach
    </select>
</div>

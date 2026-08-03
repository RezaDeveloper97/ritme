{{--
    Cycle-day targeting shared by admin content forms.

    $challenge   model holding cycle_day_from / cycle_day_to (nullable)
    $maxCycleDay upper bound of the range (Challenge::MAX_CYCLE_DAY)

    Both bounds are optional and independent: empty/empty = every day,
    from-only = "from day N onwards", to-only = "up to day N".
--}}
<div class="field">
    <label for="cycle_day_from">روز چرخه — از</label>
    <input type="number" id="cycle_day_from" name="cycle_day_from" min="1" max="{{ $maxCycleDay }}"
           value="{{ old('cycle_day_from', $challenge->cycle_day_from) }}" placeholder="از ابتدای چرخه">
    @error('cycle_day_from') <span class="err">{{ $message }}</span> @enderror
</div>

<div class="field">
    <label for="cycle_day_to">روز چرخه — تا</label>
    <input type="number" id="cycle_day_to" name="cycle_day_to" min="1" max="{{ $maxCycleDay }}"
           value="{{ old('cycle_day_to', $challenge->cycle_day_to) }}" placeholder="تا انتهای چرخه">
    @error('cycle_day_to') <span class="err">{{ $message }}</span> @enderror
</div>

<div class="field" style="grid-column: 1 / -1">
    <span class="hint">
        بازه‌ی روزهایی از چرخه که این چالش در آن‌ها نمایش داده می‌شود (۱ تا {{ $maxCycleDay }}).
        اگر هر دو خالی بمانند، چالش در همه‌ی روزها قابل نمایش است.
        در هر روز، چالش‌هایی که همان روز را هدف گرفته‌اند بر چالش‌های بدون بازه اولویت دارند.
    </span>
</div>

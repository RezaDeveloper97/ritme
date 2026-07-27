package ir.ritmeapp.ritme.adapter.inbound.ui.onboarding

import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JalaliDatePicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JalaliMonthCalendar
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RulerPicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.Segmented
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.WheelPicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.ChronicCondition
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.PregnancyAgeSource
import ir.ritmeapp.ritme.domain.model.PregnancyIntention
import kotlin.math.roundToInt

private const val WEIGHT_MIN_KG = 30
private const val WEIGHT_MAX_KG = 150
private const val HEIGHT_MIN_CM = 120
private const val HEIGHT_MAX_CM = 220
private const val PERIOD_MIN = 1
private const val PERIOD_MAX = 10
private const val CYCLE_MIN = 15
private const val CYCLE_MAX = 60
private const val LB_PER_KG = 2.205
private const val CM_PER_FT = 30.48
private const val BASIS_WEEK_MIN = 1
private const val BASIS_WEEK_MAX = 42
private const val BASIS_DAY_MAX = 6
private const val UNIT_TOGGLE_WIDTH = 170

/** A step's start-aligned title, optional subtitle and optional helper line — the shared header. */
@Composable
internal fun StepHeader(
    @StringRes title: Int,
    @StringRes subtitle: Int?,
    colors: RitmeColors,
    @StringRes helper: Int? = null,
) {
    Column(Modifier.fillMaxWidth()) {
        Text(
            text = stringResource(title),
            style = MaterialTheme.typography.headlineSmall,
            color = colors.ink,
        )
        if (subtitle != null) {
            Spacer(Modifier.height(10.dp))
            Text(
                text = stringResource(subtitle),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
            )
        }
        if (helper != null) {
            Spacer(Modifier.height(8.dp))
            Text(
                text = stringResource(helper),
                style = MaterialTheme.typography.bodySmall,
                color = colors.inkMuted,
            )
        }
    }
}

/** Picker steps: title pinned to the top, the picker block vertically centered in the space below. */
@Composable
private fun PickerStep(header: @Composable () -> Unit, picker: @Composable ColumnScope.() -> Unit) {
    Column(Modifier.fillMaxSize()) {
        header()
        Column(
            modifier = Modifier.fillMaxWidth().weight(1f),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
            content = picker,
        )
    }
}

/** List steps (intention / conditions / dating basis): scrollable, content hugging the top. */
@Composable
private fun ListStep(content: @Composable ColumnScope.() -> Unit) {
    Column(
        modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()),
        content = content,
    )
}

@Composable
internal fun NameStep(name: String, onName: (String) -> Unit, colors: RitmeColors) {
    PickerStep(
        header = { StepHeader(R.string.ob_name_title, R.string.ob_name_subtitle, colors) },
        picker = {
            Column(Modifier.fillMaxWidth()) {
                Text(
                    text = stringResource(R.string.ob_name_label),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                )
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = name,
                    onValueChange = onName,
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = { Text(stringResource(R.string.ob_name_placeholder), color = colors.placeholder) },
                    trailingIcon = {
                        Icon(
                            painter = painterResource(R.drawable.ic_pencil),
                            contentDescription = null,
                            tint = colors.placeholder,
                            modifier = Modifier.size(18.dp),
                        )
                    },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                    shape = MaterialTheme.shapes.medium,
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedBorderColor = colors.pink,
                        unfocusedBorderColor = colors.outline,
                        cursorColor = colors.pink,
                    ),
                )
            }
        },
    )
}

@Composable
internal fun BirthdayStep(date: JalaliDate, onDate: (JalaliDate) -> Unit, minYear: Int, maxYear: Int, colors: RitmeColors) {
    PickerStep(
        header = { StepHeader(R.string.ob_birthday_title, R.string.ob_common_subtitle, colors) },
        picker = { JalaliDatePicker(value = date, onValueChange = onDate, minYear = minYear, maxYear = maxYear) },
    )
}

@Composable
internal fun WeightStep(kg: Int, unit: WeightUnit, onKg: (Int) -> Unit, onUnit: (WeightUnit) -> Unit, colors: RitmeColors) {
    PickerStep(
        header = { StepHeader(R.string.ob_weight_title, R.string.ob_common_subtitle, colors, helper = R.string.ob_measure_helper) },
        picker = {
            UnitToggle(
                options = listOf(stringResource(R.string.ob_unit_lb), stringResource(R.string.ob_unit_kg)),
                selectedIndex = if (unit == WeightUnit.KG) 1 else 0,
                onSelected = { onUnit(if (it == 1) WeightUnit.KG else WeightUnit.LB) },
            )
            Spacer(Modifier.height(30.dp))
            if (unit == WeightUnit.KG) {
                RulerPicker(WEIGHT_MIN_KG, WEIGHT_MAX_KG, kg, stringResource(R.string.ob_unit_kg), onKg)
            } else {
                RulerPicker(
                    min = (WEIGHT_MIN_KG * LB_PER_KG).roundToInt(),
                    max = (WEIGHT_MAX_KG * LB_PER_KG).roundToInt(),
                    value = (kg * LB_PER_KG).roundToInt(),
                    unit = stringResource(R.string.ob_unit_lb),
                    onValue = { onKg((it / LB_PER_KG).roundToInt()) },
                )
            }
        },
    )
}

@Composable
internal fun HeightStep(cm: Int, unit: HeightUnit, onCm: (Int) -> Unit, onUnit: (HeightUnit) -> Unit, colors: RitmeColors) {
    PickerStep(
        header = { StepHeader(R.string.ob_height_title, R.string.ob_common_subtitle, colors, helper = R.string.ob_measure_helper) },
        picker = {
            UnitToggle(
                options = listOf(stringResource(R.string.ob_unit_ft), stringResource(R.string.ob_unit_cm)),
                selectedIndex = if (unit == HeightUnit.CM) 1 else 0,
                onSelected = { onUnit(if (it == 1) HeightUnit.CM else HeightUnit.FT) },
            )
            Spacer(Modifier.height(30.dp))
            if (unit == HeightUnit.CM) {
                RulerPicker(HEIGHT_MIN_CM, HEIGHT_MAX_CM, cm, stringResource(R.string.ob_unit_cm), onCm)
            } else {
                RulerPicker(
                    min = (HEIGHT_MIN_CM / CM_PER_FT).roundToInt(),
                    max = (HEIGHT_MAX_CM / CM_PER_FT).roundToInt(),
                    value = (cm / CM_PER_FT).roundToInt(),
                    unit = stringResource(R.string.ob_unit_ft),
                    onValue = { onCm((it * CM_PER_FT).roundToInt()) },
                )
            }
        },
    )
}

/** The 170dp-wide kg/lb (or cm/ft) toggle, centered above the ruler like the web `.seg`. */
@Composable
private fun UnitToggle(options: List<String>, selectedIndex: Int, onSelected: (Int) -> Unit) {
    Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
        Segmented(
            options = options,
            selectedIndex = selectedIndex,
            onSelected = onSelected,
            modifier = Modifier.width(UNIT_TOGGLE_WIDTH.dp),
        )
    }
}

@Composable
internal fun PeriodStep(days: Int, onDays: (Int) -> Unit, colors: RitmeColors) {
    UnitWheelStep(R.string.ob_period_title, R.string.ob_common_subtitle, PERIOD_MIN, PERIOD_MAX, days, onDays, colors)
}

@Composable
internal fun CycleStep(days: Int, onDays: (Int) -> Unit, colors: RitmeColors) {
    UnitWheelStep(R.string.ob_cycle_title, R.string.ob_cycle_subtitle, CYCLE_MIN, CYCLE_MAX, days, onDays, colors)
}

/** A day-count wheel whose rows already carry the «روز» unit (web `PeriodLenPage`/`CycleDurationPage`). */
@Composable
private fun UnitWheelStep(
    @StringRes title: Int,
    @StringRes subtitle: Int,
    min: Int,
    max: Int,
    value: Int,
    onValue: (Int) -> Unit,
    colors: RitmeColors,
) {
    val dayUnit = stringResource(R.string.onboarding_days_unit)
    PickerStep(
        header = { StepHeader(title, subtitle, colors) },
        picker = {
            WheelPicker(
                count = max - min + 1,
                selectedIndex = (value - min).coerceIn(0, max - min),
                onSelected = { onValue(min + it) },
                label = { "${(min + it).toPersianDigits()} $dayUnit" },
                modifier = Modifier.width(150.dp),
            )
        },
    )
}

@Composable
internal fun LastPeriodStep(selected: JalaliDate?, onSelect: (JalaliDate) -> Unit, colors: RitmeColors) {
    PickerStep(
        header = { StepHeader(R.string.ob_last_period_title, R.string.ob_common_subtitle, colors) },
        picker = {
            JalaliMonthCalendar(selected = selected, onSelect = onSelect)
            Spacer(Modifier.height(16.dp))
            Text(
                text = stringResource(R.string.ob_last_period_hint),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth(),
            )
        },
    )
}

@Composable
internal fun IntentionStep(selected: PregnancyIntention?, onSelect: (PregnancyIntention) -> Unit, colors: RitmeColors) {
    ListStep {
        StepHeader(R.string.ob_intention_title, R.string.ob_intention_subtitle, colors)
        Spacer(Modifier.height(18.dp))
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            PregnancyIntention.entries.forEach { intention ->
                RadioCard(
                    label = stringResource(intentionLabel(intention)),
                    selected = selected == intention,
                    colors = colors,
                    onClick = { onSelect(intention) },
                )
            }
        }
    }
}

@Composable
internal fun ConditionsStep(selected: List<ChronicCondition>, onToggle: (ChronicCondition) -> Unit, colors: RitmeColors) {
    ListStep {
        StepHeader(R.string.ob_conditions_title, R.string.ob_conditions_subtitle, colors)
        Spacer(Modifier.height(16.dp))
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            ChronicCondition.entries.forEach { condition ->
                SquareCheckRow(
                    label = stringResource(conditionLabel(condition)),
                    checked = condition in selected,
                    colors = colors,
                    onClick = { onToggle(condition) },
                )
            }
        }
    }
}

@Composable
internal fun PregnancyBasisStep(
    basis: PregnancyBasisDraft,
    today: JalaliDate,
    onIntent: (OnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    ListStep {
        StepHeader(R.string.ob_basis_title, R.string.ob_basis_subtitle, colors)
        Spacer(Modifier.height(16.dp))
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            PregnancyAgeSource.entries.forEach { source ->
                SourceCard(
                    title = stringResource(basisSourceLabel(source)),
                    hint = stringResource(basisHint(source)),
                    selected = basis.source == source,
                    colors = colors,
                    onClick = { onIntent(OnboardingIntent.BasisSourceSelected(source)) },
                )
            }
        }

        when (basis.source) {
            PregnancyAgeSource.LMP -> {
                Spacer(Modifier.height(16.dp))
                JalaliDatePicker(
                    value = basis.lmp ?: today,
                    onValueChange = { onIntent(OnboardingIntent.BasisLmpChanged(it)) },
                    minYear = today.year - 1,
                    maxYear = today.year,
                )
            }

            PregnancyAgeSource.ULTRASOUND -> {
                Spacer(Modifier.height(16.dp))
                JalaliDatePicker(
                    value = basis.ultrasoundDate ?: today,
                    onValueChange = { onIntent(OnboardingIntent.BasisUltrasoundDateChanged(it)) },
                    minYear = today.year - 1,
                    maxYear = today.year,
                )
                Spacer(Modifier.height(14.dp))
                NumberStepper(
                    label = stringResource(R.string.preg_ob_weeks),
                    value = basis.ultrasoundWeeks,
                    min = BASIS_WEEK_MIN,
                    max = BASIS_WEEK_MAX,
                    onChange = { onIntent(OnboardingIntent.BasisUltrasoundWeeksChanged(it)) },
                    colors = colors,
                )
                Spacer(Modifier.height(12.dp))
                DaysField(
                    days = basis.ultrasoundDays,
                    onDays = { onIntent(OnboardingIntent.BasisUltrasoundDaysChanged(it)) },
                    colors = colors,
                )
            }

            PregnancyAgeSource.MANUAL -> {
                Spacer(Modifier.height(16.dp))
                NumberStepper(
                    label = stringResource(R.string.preg_ob_weeks),
                    value = basis.manualWeeks,
                    min = BASIS_WEEK_MIN,
                    max = BASIS_WEEK_MAX,
                    onChange = { onIntent(OnboardingIntent.BasisManualWeeksChanged(it)) },
                    colors = colors,
                )
                Spacer(Modifier.height(12.dp))
                DaysField(
                    days = basis.manualDays,
                    onDays = { onIntent(OnboardingIntent.BasisManualDaysChanged(it)) },
                    colors = colors,
                )
            }

            null -> Unit
        }

        Spacer(Modifier.height(18.dp))
        Text(
            text = stringResource(R.string.ob_basis_note),
            style = MaterialTheme.typography.bodySmall,
            color = colors.inkMuted,
        )
        Spacer(Modifier.height(12.dp))
    }
}

/** A radio card (circle + check) with a bold label — the intention options (web `IntentionPage`). */
@Composable
private fun RadioCard(label: String, selected: Boolean, colors: RitmeColors, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(if (selected) colors.pinkContainer else colors.surface)
            .border(2.dp, if (selected) colors.pink else colors.outline, RoundedCornerShape(16.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 15.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioDot(selected, colors)
        Spacer(Modifier.width(12.dp))
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.Bold),
            color = if (selected) colors.pink else colors.ink,
        )
    }
}

/** A radio card that also carries a hint line — the dating-source options (web `PregnancyBasisPage`). */
@Composable
private fun SourceCard(title: String, hint: String, selected: Boolean, colors: RitmeColors, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(if (selected) colors.pinkContainer else colors.surface)
            .border(2.dp, if (selected) colors.pink else colors.outline, RoundedCornerShape(16.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 13.dp),
        verticalAlignment = Alignment.Top,
    ) {
        RadioDot(selected, colors)
        Spacer(Modifier.width(11.dp))
        Column {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Bold),
                color = if (selected) colors.pink else colors.ink,
            )
            Spacer(Modifier.height(3.dp))
            Text(text = hint, style = MaterialTheme.typography.bodySmall, color = colors.inkMuted)
        }
    }
}

@Composable
private fun RadioDot(selected: Boolean, colors: RitmeColors) {
    Box(
        modifier = Modifier
            .size(22.dp)
            .clip(CircleShape)
            .background(if (selected) colors.pink else colors.outline),
        contentAlignment = Alignment.Center,
    ) {
        if (selected) {
            Icon(
                painter = painterResource(R.drawable.ic_check),
                contentDescription = null,
                tint = colors.onPink,
                modifier = Modifier.size(14.dp),
            )
        }
    }
}

/** A rounded-square checkbox row with a pink-tinted selected card (web `ConditionsPage`). */
@Composable
private fun SquareCheckRow(label: String, checked: Boolean, colors: RitmeColors, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(if (checked) colors.pinkContainer else colors.surface)
            .border(2.dp, if (checked) colors.pink else colors.outline, RoundedCornerShape(14.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 15.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(22.dp)
                .clip(RoundedCornerShape(7.dp))
                .background(if (checked) colors.pink else colors.surface)
                .border(2.dp, if (checked) colors.pink else colors.outline, RoundedCornerShape(7.dp)),
            contentAlignment = Alignment.Center,
        ) {
            if (checked) {
                Icon(
                    painter = painterResource(R.drawable.ic_check),
                    contentDescription = null,
                    tint = colors.onPink,
                    modifier = Modifier.size(13.dp),
                )
            }
        }
        Spacer(Modifier.width(12.dp))
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge.copy(fontWeight = FontWeight.Bold),
            color = if (checked) colors.pink else colors.ink,
        )
    }
}

/** A small −/＋ number entry (web `NumberField`); an unset value reads «—» so validation can require it. */
@Composable
private fun NumberStepper(label: String, value: Int?, min: Int, max: Int, onChange: (Int?) -> Unit, colors: RitmeColors) {
    Column(Modifier.fillMaxWidth()) {
        Text(text = label, style = MaterialTheme.typography.labelLarge, color = colors.ink, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(8.dp))
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(48.dp)
                .clip(RoundedCornerShape(12.dp))
                .border(1.dp, colors.outline, RoundedCornerShape(12.dp)),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            StepperButton("−", colors) { onChange(value?.let { (it - 1).coerceAtLeast(min) }) }
            Text(
                text = value?.toPersianDigits() ?: "—",
                modifier = Modifier.weight(1f),
                textAlign = TextAlign.Center,
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
            )
            StepperButton("＋", colors) { onChange(((value ?: (min - 1)) + 1).coerceAtMost(max)) }
        }
    }
}

@Composable
private fun StepperButton(glyph: String, colors: RitmeColors, onClick: () -> Unit) {
    Box(
        modifier = Modifier.size(48.dp).clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(text = glyph, style = MaterialTheme.typography.titleLarge, color = colors.pink)
    }
}

/** The «روز» segmented 0…6 selector used for the ultrasound/manual day offset. */
@Composable
private fun DaysField(days: Int, onDays: (Int) -> Unit, colors: RitmeColors) {
    Column(Modifier.fillMaxWidth()) {
        Text(
            text = stringResource(R.string.onboarding_days_unit),
            style = MaterialTheme.typography.labelLarge,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        Segmented(
            options = (0..BASIS_DAY_MAX).map { it.toPersianDigits() },
            selectedIndex = days.coerceIn(0, BASIS_DAY_MAX),
            onSelected = onDays,
        )
    }
}

@StringRes
private fun intentionLabel(intention: PregnancyIntention): Int = when (intention) {
    PregnancyIntention.AVOIDING -> R.string.ob_intention_avoiding
    PregnancyIntention.TRYING -> R.string.ob_intention_trying
    PregnancyIntention.PREGNANT -> R.string.ob_intention_pregnant
    PregnancyIntention.UNSURE -> R.string.ob_intention_unsure
}

@StringRes
private fun conditionLabel(condition: ChronicCondition): Int = when (condition) {
    ChronicCondition.PCOS -> R.string.ob_condition_pcos
    ChronicCondition.HYPOTHYROIDISM -> R.string.ob_condition_hypothyroidism
    ChronicCondition.HYPERTHYROIDISM -> R.string.ob_condition_hyperthyroidism
    ChronicCondition.HYPERTENSION -> R.string.ob_condition_hypertension
    ChronicCondition.HEART_DISEASE -> R.string.ob_condition_heart_disease
    ChronicCondition.DIABETES -> R.string.ob_condition_diabetes
}

@StringRes
private fun basisSourceLabel(source: PregnancyAgeSource): Int = when (source) {
    PregnancyAgeSource.LMP -> R.string.ob_basis_source_lmp
    PregnancyAgeSource.ULTRASOUND -> R.string.ob_basis_source_ultrasound
    PregnancyAgeSource.MANUAL -> R.string.ob_basis_source_manual
}

@StringRes
private fun basisHint(source: PregnancyAgeSource): Int = when (source) {
    PregnancyAgeSource.LMP -> R.string.ob_basis_hint_lmp
    PregnancyAgeSource.ULTRASOUND -> R.string.ob_basis_hint_ultrasound
    PregnancyAgeSource.MANUAL -> R.string.ob_basis_hint_manual
}

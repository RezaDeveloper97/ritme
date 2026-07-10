package ir.ritmeapp.ritme.adapter.inbound.ui.onboarding

import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JalaliDatePicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.WheelPicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.ChronicCondition
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.PregnancyIntention

/** A step's title + optional subtitle, shared across every wizard step for a consistent rhythm. */
@Composable
internal fun StepHeader(@StringRes title: Int, @StringRes subtitle: Int?, colors: RitmeColors) {
    Text(
        text = stringResource(title),
        style = MaterialTheme.typography.headlineSmall,
        color = colors.ink,
    )
    if (subtitle != null) {
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(subtitle),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
        )
    }
    Spacer(Modifier.height(28.dp))
}

@Composable
internal fun NameStep(name: String, onName: (String) -> Unit, colors: RitmeColors) {
    Column {
        StepHeader(R.string.onboarding_name_title, R.string.onboarding_name_subtitle, colors)
        OutlinedTextField(
            value = name,
            onValueChange = onName,
            modifier = Modifier.fillMaxWidth(),
            placeholder = { Text(stringResource(R.string.onboarding_name_placeholder)) },
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
}

@Composable
internal fun BirthdayStep(date: JalaliDate, onDate: (JalaliDate) -> Unit, minYear: Int, maxYear: Int, colors: RitmeColors) {
    Column {
        StepHeader(R.string.onboarding_birthday_title, R.string.onboarding_birthday_subtitle, colors)
        JalaliDatePicker(value = date, onValueChange = onDate, minYear = minYear, maxYear = maxYear)
    }
}

@Composable
internal fun LastPeriodStep(date: JalaliDate, onDate: (JalaliDate) -> Unit, minYear: Int, maxYear: Int, colors: RitmeColors) {
    Column {
        StepHeader(R.string.onboarding_last_period_title, R.string.onboarding_last_period_subtitle, colors)
        JalaliDatePicker(value = date, onValueChange = onDate, minYear = minYear, maxYear = maxYear)
    }
}

@Composable
internal fun NumberStep(
    @StringRes title: Int,
    @StringRes subtitle: Int?,
    min: Int,
    max: Int,
    value: Int,
    onValue: (Int) -> Unit,
    @StringRes unit: Int,
    colors: RitmeColors,
) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Column(Modifier.fillMaxWidth()) {
            StepHeader(title, subtitle, colors)
        }
        WheelPicker(
            count = max - min + 1,
            selectedIndex = (value - min).coerceIn(0, max - min),
            onSelected = { onValue(min + it) },
            label = { (min + it).toPersianDigits() },
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(unit),
            style = MaterialTheme.typography.labelLarge,
            color = colors.inkMuted,
        )
    }
}

@Composable
internal fun IntentionStep(selected: PregnancyIntention?, onSelect: (PregnancyIntention) -> Unit, colors: RitmeColors) {
    Column {
        StepHeader(R.string.onboarding_intention_title, R.string.onboarding_intention_subtitle, colors)
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            PregnancyIntention.entries.forEach { intention ->
                SelectableCard(
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
    Column {
        StepHeader(R.string.onboarding_conditions_title, R.string.onboarding_conditions_subtitle, colors)
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            ChronicCondition.entries.forEach { condition ->
                CheckableRow(
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
private fun SelectableCard(label: String, selected: Boolean, colors: RitmeColors, onClick: () -> Unit) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(if (selected) colors.pinkContainer else colors.surface)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) colors.pink else colors.outline,
                shape = RoundedCornerShape(16.dp),
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 18.dp, vertical = 18.dp),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge.copy(
                fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal,
            ),
            color = if (selected) colors.pink else colors.ink,
        )
    }
}

@Composable
private fun CheckableRow(label: String, checked: Boolean, colors: RitmeColors, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(colors.surface)
            .border(1.dp, if (checked) colors.pink else colors.outline, RoundedCornerShape(14.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(24.dp)
                .clip(CircleShape)
                .background(if (checked) colors.pink else colors.background),
            contentAlignment = Alignment.Center,
        ) {
            if (checked) {
                Text("✓", style = MaterialTheme.typography.labelMedium, color = colors.onPink)
            }
        }
        Spacer(Modifier.size(14.dp))
        Text(text = label, style = MaterialTheme.typography.bodyLarge, color = colors.ink)
    }
}

@StringRes
private fun intentionLabel(intention: PregnancyIntention): Int = when (intention) {
    PregnancyIntention.AVOIDING -> R.string.intention_avoiding
    PregnancyIntention.TRYING -> R.string.intention_trying
    PregnancyIntention.PREGNANT -> R.string.intention_pregnant
    PregnancyIntention.UNSURE -> R.string.intention_unsure
}

@StringRes
private fun conditionLabel(condition: ChronicCondition): Int = when (condition) {
    ChronicCondition.PCOS -> R.string.condition_pcos
    ChronicCondition.HYPOTHYROIDISM -> R.string.condition_hypothyroidism
    ChronicCondition.HYPERTHYROIDISM -> R.string.condition_hyperthyroidism
    ChronicCondition.HYPERTENSION -> R.string.condition_hypertension
    ChronicCondition.HEART_DISEASE -> R.string.condition_heart_disease
    ChronicCondition.DIABETES -> R.string.condition_diabetes
}

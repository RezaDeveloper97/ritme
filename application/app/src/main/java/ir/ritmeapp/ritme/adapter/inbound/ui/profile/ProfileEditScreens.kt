package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JalaliDatePicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.ScreenHeader
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.WheelPicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The two profile edit forms (web `/profile/personal` + `/profile/health`): prefilled pickers
 * over `GET /profile`, saved as partial `POST /profile` updates.
 */
@Composable
fun ProfilePersonalScreen(
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: ProfilePersonalViewModel = viewModel(factory = container.profilePersonalViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:profile_personal:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.ProfilePersonal.route, null, System.currentTimeMillis()),
        )
    }

    EditScaffold(
        title = stringResource(R.string.edit_personal_title),
        loading = state.loading,
        saveState = state.saveState,
        validationError = if (state.birthdayInFuture) stringResource(R.string.edit_error_birthday_future) else null,
        onBack = onBack,
        onSave = { viewModel.onIntent(ProfilePersonalIntent.Save) },
        modifier = modifier,
        colors = colors,
    ) {
        item(key = "name") {
            SurfaceCard {
                FieldTitle(stringResource(R.string.edit_name_label), colors)
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = state.name,
                    onValueChange = { viewModel.onIntent(ProfilePersonalIntent.NameChanged(it)) },
                    placeholder = { Text(stringResource(R.string.edit_name_placeholder), color = colors.inkMuted) },
                    singleLine = true,
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedBorderColor = colors.pink,
                        unfocusedBorderColor = colors.outline,
                    ),
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }
        item(key = "birthday") {
            SurfaceCard {
                FieldTitle(stringResource(R.string.edit_birthday_label), colors)
                Spacer(Modifier.height(8.dp))
                JalaliDatePicker(
                    value = state.birthday,
                    onValueChange = { viewModel.onIntent(ProfilePersonalIntent.BirthdayChanged(it)) },
                    minYear = ProfilePersonalUiState.MIN_BIRTH_YEAR,
                    maxYear = todayJalali().year,
                )
            }
        }
        item(key = "weight") {
            SurfaceCard {
                FieldTitle(stringResource(R.string.edit_weight_label), colors)
                WheelPicker(
                    count = ProfilePersonalUiState.MAX_WEIGHT_KG - ProfilePersonalUiState.MIN_WEIGHT_KG + 1,
                    selectedIndex = state.weightKg - ProfilePersonalUiState.MIN_WEIGHT_KG,
                    onSelected = {
                        viewModel.onIntent(
                            ProfilePersonalIntent.WeightChanged(ProfilePersonalUiState.MIN_WEIGHT_KG + it),
                        )
                    },
                    label = { (ProfilePersonalUiState.MIN_WEIGHT_KG + it).toPersianDigits() },
                    visibleCount = 3,
                )
                UnitLabel(stringResource(R.string.onboarding_weight_unit), colors)
            }
        }
        item(key = "height") {
            SurfaceCard {
                FieldTitle(stringResource(R.string.edit_height_label), colors)
                WheelPicker(
                    count = ProfilePersonalUiState.MAX_HEIGHT_CM - ProfilePersonalUiState.MIN_HEIGHT_CM + 1,
                    selectedIndex = state.heightCm - ProfilePersonalUiState.MIN_HEIGHT_CM,
                    onSelected = {
                        viewModel.onIntent(
                            ProfilePersonalIntent.HeightChanged(ProfilePersonalUiState.MIN_HEIGHT_CM + it),
                        )
                    },
                    label = { (ProfilePersonalUiState.MIN_HEIGHT_CM + it).toPersianDigits() },
                    visibleCount = 3,
                )
                UnitLabel(stringResource(R.string.onboarding_height_unit), colors)
            }
        }
    }
}

@Composable
fun ProfileHealthScreen(
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: ProfileHealthViewModel = viewModel(factory = container.profileHealthViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:profile_health:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.ProfileHealth.route, null, System.currentTimeMillis()),
        )
    }

    EditScaffold(
        title = stringResource(R.string.edit_health_title),
        loading = state.loading,
        saveState = state.saveState,
        validationError = if (state.lastPeriodInFuture) stringResource(R.string.edit_error_last_period_future) else null,
        onBack = onBack,
        onSave = { viewModel.onIntent(ProfileHealthIntent.Save) },
        modifier = modifier,
        colors = colors,
    ) {
        item(key = "cycle") {
            val dayUnit = stringResource(R.string.onboarding_days_unit)
            SurfaceCard {
                FieldTitle(stringResource(R.string.edit_cycle_label), colors)
                WheelPicker(
                    count = ProfileHealthUiState.MAX_CYCLE_DAYS - ProfileHealthUiState.MIN_CYCLE_DAYS + 1,
                    selectedIndex = state.cycleDuration - ProfileHealthUiState.MIN_CYCLE_DAYS,
                    onSelected = {
                        viewModel.onIntent(ProfileHealthIntent.CycleChanged(ProfileHealthUiState.MIN_CYCLE_DAYS + it))
                    },
                    label = { "${(ProfileHealthUiState.MIN_CYCLE_DAYS + it).toPersianDigits()} $dayUnit" },
                    visibleCount = 3,
                )
            }
        }
        item(key = "period") {
            val dayUnit = stringResource(R.string.onboarding_days_unit)
            SurfaceCard {
                FieldTitle(stringResource(R.string.edit_period_label), colors)
                WheelPicker(
                    count = ProfileHealthUiState.MAX_PERIOD_DAYS - ProfileHealthUiState.MIN_PERIOD_DAYS + 1,
                    selectedIndex = state.periodDuration - ProfileHealthUiState.MIN_PERIOD_DAYS,
                    onSelected = {
                        viewModel.onIntent(ProfileHealthIntent.PeriodChanged(ProfileHealthUiState.MIN_PERIOD_DAYS + it))
                    },
                    label = { "${(ProfileHealthUiState.MIN_PERIOD_DAYS + it).toPersianDigits()} $dayUnit" },
                    visibleCount = 3,
                )
            }
        }
        item(key = "lastperiod") {
            SurfaceCard {
                FieldTitle(stringResource(R.string.edit_last_period_label), colors)
                Spacer(Modifier.height(4.dp))
                Text(
                    stringResource(R.string.edit_last_period_hint),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.inkMuted,
                )
                Spacer(Modifier.height(8.dp))
                JalaliDatePicker(
                    value = state.lastPeriod,
                    onValueChange = { viewModel.onIntent(ProfileHealthIntent.LastPeriodChanged(it)) },
                    minYear = todayJalali().year - 2,
                    maxYear = todayJalali().year,
                )
            }
        }
    }
}

/** Shared shell: header, scrolling form body, validation line, and the sticky save button. */
@Composable
private fun EditScaffold(
    title: String,
    loading: Boolean,
    saveState: EditSaveState,
    validationError: String?,
    onBack: () -> Unit,
    onSave: () -> Unit,
    modifier: Modifier,
    colors: RitmeColors,
    content: androidx.compose.foundation.lazy.LazyListScope.() -> Unit,
) {
    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        topBar = { ScreenHeader(title = title, onBack = onBack) },
        bottomBar = {
            Column(Modifier.fillMaxWidth().padding(16.dp)) {
                val error = validationError
                    ?: if (saveState == EditSaveState.ERROR) stringResource(R.string.error_generic) else null
                if (error != null) {
                    Text(
                        text = error,
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.error,
                        modifier = Modifier.fillMaxWidth().padding(bottom = 6.dp),
                    )
                }
                Button(
                    onClick = onSave,
                    enabled = !loading && validationError == null && saveState != EditSaveState.SAVING,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = colors.pink,
                        contentColor = colors.onPink,
                        disabledContainerColor = colors.outline,
                        disabledContentColor = colors.inkMuted,
                    ),
                    shape = RoundedCornerShape(14.dp),
                    modifier = Modifier.fillMaxWidth().height(48.dp),
                ) {
                    Text(
                        text = when (saveState) {
                            EditSaveState.SAVING -> stringResource(R.string.edit_saving)
                            EditSaveState.SAVED -> stringResource(R.string.edit_saved)
                            else -> stringResource(R.string.edit_save)
                        },
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        },
    ) { padding ->
        if (loading) {
            Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = colors.pink)
            }
        } else {
            LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
                content = content,
            )
        }
    }
}

@Composable
private fun FieldTitle(text: String, colors: RitmeColors) {
    Text(
        text = text,
        style = MaterialTheme.typography.bodyMedium,
        color = colors.ink,
        fontWeight = FontWeight.Bold,
    )
}

@Composable
private fun UnitLabel(text: String, colors: RitmeColors) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelSmall,
        color = colors.inkMuted,
        modifier = Modifier
            .fillMaxWidth()
            .background(colors.surface)
            .padding(top = 4.dp),
        textAlign = androidx.compose.ui.text.style.TextAlign.Center,
    )
}

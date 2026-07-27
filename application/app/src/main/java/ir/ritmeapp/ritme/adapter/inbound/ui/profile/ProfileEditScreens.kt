package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JalaliDatePicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RulerPicker
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
 * over `GET /profile`, saved as partial `POST /profile` updates. A successful save pops back to
 * the profile (web `router.back()`), so neither form has a lingering "saved" state.
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

    LaunchedScreenTracking(Destination.ProfilePersonal.route, "profile_personal")

    LaunchedEffect(Unit) {
        viewModel.effects.collect { effect ->
            when (effect) {
                ProfileEditEffect.NavigateBack -> onBack()
            }
        }
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
            Column {
                SectionLabel(stringResource(R.string.edit_name_label), colors)
                NameField(
                    value = state.name,
                    onValueChange = { viewModel.onIntent(ProfilePersonalIntent.NameChanged(it)) },
                    colors = colors,
                )
            }
        }
        item(key = "birthday") {
            Column {
                SectionLabel(stringResource(R.string.edit_birthday_label), colors)
                SurfaceCard {
                    JalaliDatePicker(
                        value = state.birthday,
                        onValueChange = { viewModel.onIntent(ProfilePersonalIntent.BirthdayChanged(it)) },
                        minYear = ProfilePersonalUiState.MIN_BIRTH_YEAR,
                        maxYear = todayJalali().year,
                    )
                }
            }
        }
        item(key = "weight") {
            Column {
                SectionLabel(stringResource(R.string.edit_weight_label), colors)
                SurfaceCard {
                    RulerPicker(
                        min = ProfilePersonalUiState.MIN_WEIGHT_KG,
                        max = ProfilePersonalUiState.MAX_WEIGHT_KG,
                        value = state.weightKg,
                        unit = stringResource(R.string.onboarding_weight_unit),
                        onValue = { viewModel.onIntent(ProfilePersonalIntent.WeightChanged(it)) },
                    )
                }
            }
        }
        item(key = "height") {
            Column {
                SectionLabel(stringResource(R.string.edit_height_label), colors)
                SurfaceCard {
                    RulerPicker(
                        min = ProfilePersonalUiState.MIN_HEIGHT_CM,
                        max = ProfilePersonalUiState.MAX_HEIGHT_CM,
                        value = state.heightCm,
                        unit = stringResource(R.string.onboarding_height_unit),
                        onValue = { viewModel.onIntent(ProfilePersonalIntent.HeightChanged(it)) },
                    )
                }
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

    LaunchedScreenTracking(Destination.ProfileHealth.route, "profile_health")

    LaunchedEffect(Unit) {
        viewModel.effects.collect { effect ->
            when (effect) {
                ProfileEditEffect.NavigateBack -> onBack()
            }
        }
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
            Column {
                SectionLabel(stringResource(R.string.edit_cycle_label), colors)
                SurfaceCard {
                    WheelPicker(
                        count = ProfileHealthUiState.MAX_CYCLE_DAYS - ProfileHealthUiState.MIN_CYCLE_DAYS + 1,
                        selectedIndex = state.cycleDuration - ProfileHealthUiState.MIN_CYCLE_DAYS,
                        onSelected = {
                            viewModel.onIntent(ProfileHealthIntent.CycleChanged(ProfileHealthUiState.MIN_CYCLE_DAYS + it))
                        },
                        label = { "${(ProfileHealthUiState.MIN_CYCLE_DAYS + it).toPersianDigits()} $dayUnit" },
                        modifier = Modifier.width(WHEEL_WIDTH).align(Alignment.CenterHorizontally),
                    )
                }
            }
        }
        item(key = "period") {
            val dayUnit = stringResource(R.string.onboarding_days_unit)
            Column {
                SectionLabel(stringResource(R.string.edit_period_label), colors)
                SurfaceCard {
                    WheelPicker(
                        count = ProfileHealthUiState.MAX_PERIOD_DAYS - ProfileHealthUiState.MIN_PERIOD_DAYS + 1,
                        selectedIndex = state.periodDuration - ProfileHealthUiState.MIN_PERIOD_DAYS,
                        onSelected = {
                            viewModel.onIntent(ProfileHealthIntent.PeriodChanged(ProfileHealthUiState.MIN_PERIOD_DAYS + it))
                        },
                        label = { "${(ProfileHealthUiState.MIN_PERIOD_DAYS + it).toPersianDigits()} $dayUnit" },
                        modifier = Modifier.width(WHEEL_WIDTH).align(Alignment.CenterHorizontally),
                    )
                }
            }
        }
        item(key = "lastperiod") {
            Column {
                SectionLabel(stringResource(R.string.edit_last_period_label), colors)
                SurfaceCard {
                    JalaliDatePicker(
                        value = state.lastPeriod,
                        onValueChange = { viewModel.onIntent(ProfileHealthIntent.LastPeriodChanged(it)) },
                        minYear = todayJalali().year - 1,
                        maxYear = todayJalali().year,
                    )
                }
                Spacer(Modifier.height(8.dp))
                Text(
                    text = stringResource(R.string.edit_last_period_hint),
                    fontSize = 12.sp,
                    color = colors.inkMuted,
                    modifier = Modifier.padding(horizontal = 4.dp),
                )
            }
        }
    }
}

/** Records a last-safe-screen entry + breadcrumb on first render (CLAUDE.md §7.2). */
@Composable
private fun LaunchedScreenTracking(route: String, breadcrumb: String) {
    val container = LocalAppContainer.current
    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:$breadcrumb:enter")
        container.safeScreenTracker.record(SafeScreen(route, null, System.currentTimeMillis()))
    }
}

/**
 * Shared shell: header, scrolling form body, validation line, and the sticky save button.
 * While loading it shows only centered muted text (no spinner, no save button), matching the
 * web loading placeholder.
 */
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
    content: LazyListScope.() -> Unit,
) {
    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        topBar = { ScreenHeader(title = title, onBack = onBack) },
        bottomBar = {
            if (!loading) {
                Column(
                    Modifier
                        .fillMaxWidth()
                        .padding(start = 16.dp, end = 16.dp, top = 6.dp, bottom = 8.dp),
                ) {
                    val error = validationError
                        ?: if (saveState == EditSaveState.ERROR) stringResource(R.string.error_generic) else null
                    if (error != null) {
                        Text(
                            text = error,
                            fontSize = 13.sp,
                            color = colors.error,
                            textAlign = TextAlign.Center,
                            modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
                        )
                    }
                    RitmePrimaryButton(
                        text = if (saveState == EditSaveState.SAVING) {
                            stringResource(R.string.edit_saving)
                        } else {
                            stringResource(R.string.edit_save)
                        },
                        onClick = onSave,
                        enabled = saveState != EditSaveState.SAVING,
                    )
                }
            }
        },
    ) { padding ->
        if (loading) {
            Column(
                Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .padding(horizontal = 18.dp, vertical = 24.dp),
            ) {
                Text(
                    text = stringResource(R.string.loading),
                    fontSize = 14.sp,
                    color = colors.inkMuted,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        } else {
            LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(start = 18.dp, end = 18.dp, top = 12.dp, bottom = 12.dp),
                verticalArrangement = Arrangement.spacedBy(20.dp),
                content = content,
            )
        }
    }
}

/** The web `SectionLabel`: a quiet muted 13sp/bold caption above each section's card. */
@Composable
private fun SectionLabel(text: String, colors: RitmeColors) {
    Text(
        text = text,
        fontSize = 13.sp,
        fontWeight = FontWeight.Bold,
        color = colors.inkMuted,
        modifier = Modifier.padding(start = 4.dp, end = 4.dp, bottom = 8.dp),
    )
}

/**
 * The web `.field`: a 52dp bordered row (no card) with a start-aligned text input and a trailing
 * pencil affordance, tinted like the web placeholder color (#A9B2BC).
 */
@Composable
private fun NameField(value: String, onValueChange: (String) -> Unit, colors: RitmeColors) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(52.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(colors.surface)
            .border(1.5.dp, colors.outline, RoundedCornerShape(14.dp))
            .padding(horizontal = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            singleLine = true,
            textStyle = MaterialTheme.typography.bodyLarge.copy(fontSize = 15.sp, color = colors.ink),
            cursorBrush = SolidColor(colors.pink),
            modifier = Modifier.weight(1f),
            decorationBox = { inner ->
                if (value.isEmpty()) {
                    Text(
                        text = stringResource(R.string.edit_name_placeholder),
                        style = MaterialTheme.typography.bodyLarge.copy(fontSize = 15.sp),
                        color = colors.placeholder,
                    )
                }
                inner()
            },
        )
        Icon(
            painter = painterResource(R.drawable.ic_pencil),
            contentDescription = null,
            tint = colors.placeholder,
            modifier = Modifier.size(18.dp),
        )
    }
}

/** Web `DaysWheel` centers the single-value wheel in a fixed 150px column. */
private val WHEEL_WIDTH = 150.dp

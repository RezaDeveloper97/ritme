package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.adapter.inbound.ui.log.LogSaveState
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.PregnancyLogTab
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.FetalMovementLog
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.PregnancyStatus
import ir.ritmeapp.ritme.domain.model.PregnancySymptom
import ir.ritmeapp.ritme.domain.model.PregnancySymptomLog
import ir.ritmeapp.ritme.domain.model.PregnancyWeeklyLog
import ir.ritmeapp.ritme.domain.model.SymptomSeverity
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyLogsUseCase
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyTrackerUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** The weekly check-in draft, kept flat for simple copy-updates. */
@Immutable
data class WeeklyDraft(
    val weightKg: Double? = null,
    val hasSwelling: Boolean = false,
    val swellingLocations: List<String> = emptyList(),
    val hasShortnessOfBreath: Boolean = false,
    val hasBpDevice: Boolean = false,
    val systolic: Int? = null,
    val diastolic: Int? = null,
    val fastingSugar: Double? = null,
    val postMealSugar: Double? = null,
    val mood: String? = null,
    val anxiety: SymptomSeverity? = null,
    val moodSwings: SymptomSeverity? = null,
    val depression: SymptomSeverity? = null,
    val notes: String = "",
) {
    val hasAnything: Boolean
        get() = weightKg != null || hasSwelling || hasShortnessOfBreath || hasBpDevice ||
            systolic != null || fastingSugar != null || postMealSugar != null || mood != null ||
            anxiety != null || moodSwings != null || depression != null || notes.isNotBlank()
}

/** The fetal-movement draft. */
@Immutable
data class MovementDraft(
    val status: String? = null,
    val count: Int? = null,
    val firstTime: String? = null,
    val lastTime: String? = null,
    val notes: String = "",
)

/**
 * The pregnancy log screen's single immutable state (§4 MVI): the active tab plus the three
 * form drafts, each with its own save progress.
 */
@Immutable
data class PregnancyLogUiState(
    val tab: PregnancyLogTab,
    val week: Int = 1,
    val fetalTrackingActive: Boolean = false,
    val date: JalaliDate,
    val isToday: Boolean = true,
    val symptoms: Map<PregnancySymptom, SymptomSeverity> = emptyMap(),
    val symptomNotes: String = "",
    val symptomSave: LogSaveState = LogSaveState.IDLE,
    val alertsRaised: Int = 0,
    val weekly: WeeklyDraft = WeeklyDraft(),
    val weeklySave: LogSaveState = LogSaveState.IDLE,
    val movement: MovementDraft = MovementDraft(),
    val movementSave: LogSaveState = LogSaveState.IDLE,
)

sealed interface PregnancyLogIntent {
    data class SelectTab(val tab: PregnancyLogTab) : PregnancyLogIntent
    data object PreviousDay : PregnancyLogIntent
    data object NextDay : PregnancyLogIntent
    data class SymptomToggled(val symptom: PregnancySymptom, val on: Boolean) : PregnancyLogIntent
    data class SymptomSeverityChanged(val symptom: PregnancySymptom, val severity: SymptomSeverity) : PregnancyLogIntent
    data class SymptomNotesChanged(val notes: String) : PregnancyLogIntent
    data object SaveSymptoms : PregnancyLogIntent
    data class WeeklyChanged(val draft: WeeklyDraft) : PregnancyLogIntent
    data object SaveWeekly : PregnancyLogIntent
    data class MovementChanged(val draft: MovementDraft) : PregnancyLogIntent
    data object SaveMovement : PregnancyLogIntent
}

/**
 * Pregnancy log state machine over `/pregnancy/symptoms|weekly|fetal-movement`. Symptom and
 * movement forms are day-scoped (shared day switcher); the weekly check-in is week-scoped,
 * anchored on the current gestational week from `/pregnancy/status`.
 */
class PregnancyLogViewModel(
    private val logs: PregnancyLogsUseCase,
    private val tracker: PregnancyTrackerUseCase,
    private val today: JalaliDate,
    initialTab: PregnancyLogTab,
) : ViewModel() {

    private val _state = MutableStateFlow(PregnancyLogUiState(tab = initialTab, date = today))
    val state: StateFlow<PregnancyLogUiState> = _state.asStateFlow()

    private var prefillJob: Job? = null

    init {
        viewModelScope.launch {
            val status = tracker.status().getOrNull()
            _state.update {
                it.copy(
                    week = (status?.currentWeek ?: 1).coerceIn(1, PregnancyStatus.TOTAL_WEEKS),
                    fetalTrackingActive = status?.fetalMovementTrackingActive ?: false,
                )
            }
            prefillWeekly(_state.value.week)
        }
        prefillSymptoms(today)
    }

    fun onIntent(intent: PregnancyLogIntent) {
        when (intent) {
            is PregnancyLogIntent.SelectTab -> _state.update { it.copy(tab = intent.tab) }
            PregnancyLogIntent.PreviousDay -> moveDay(-1)
            PregnancyLogIntent.NextDay -> moveDay(+1)
            is PregnancyLogIntent.SymptomToggled -> toggleSymptom(intent.symptom, intent.on)
            is PregnancyLogIntent.SymptomSeverityChanged -> _state.update {
                it.copy(
                    symptoms = it.symptoms + (intent.symptom to intent.severity),
                    symptomSave = LogSaveState.IDLE,
                )
            }

            is PregnancyLogIntent.SymptomNotesChanged ->
                _state.update { it.copy(symptomNotes = intent.notes, symptomSave = LogSaveState.IDLE) }

            PregnancyLogIntent.SaveSymptoms -> saveSymptoms()
            is PregnancyLogIntent.WeeklyChanged ->
                _state.update { it.copy(weekly = intent.draft, weeklySave = LogSaveState.IDLE) }

            PregnancyLogIntent.SaveWeekly -> saveWeekly()
            is PregnancyLogIntent.MovementChanged ->
                _state.update { it.copy(movement = intent.draft, movementSave = LogSaveState.IDLE) }

            PregnancyLogIntent.SaveMovement -> saveMovement()
        }
    }

    private fun moveDay(delta: Int) {
        val target = _state.value.date.addDays(delta)
        if (target.toJdn() > today.toJdn()) return
        _state.update {
            it.copy(
                date = target,
                isToday = target.toJdn() == today.toJdn(),
                symptoms = emptyMap(),
                symptomNotes = "",
                symptomSave = LogSaveState.IDLE,
                movementSave = LogSaveState.IDLE,
            )
        }
        prefillSymptoms(target)
    }

    private fun prefillSymptoms(day: JalaliDate) {
        prefillJob?.cancel()
        prefillJob = viewModelScope.launch {
            val log = logs.symptomLog(day.toGregorian()).getOrNull() ?: return@launch
            _state.update { current ->
                if (current.date != day) return@update current
                current.copy(symptoms = log.symptoms, symptomNotes = log.notes.orEmpty())
            }
        }
    }

    private fun prefillWeekly(week: Int) {
        viewModelScope.launch {
            val log = logs.weeklyLog(week).getOrNull() ?: return@launch
            _state.update {
                it.copy(
                    weekly = WeeklyDraft(
                        weightKg = log.weightKg,
                        hasSwelling = log.hasSwelling,
                        swellingLocations = log.swellingLocations,
                        hasShortnessOfBreath = log.hasShortnessOfBreath,
                        hasBpDevice = log.hasBloodPressureDevice,
                        systolic = log.systolicPressure,
                        diastolic = log.diastolicPressure,
                        fastingSugar = log.fastingBloodSugar,
                        postMealSugar = log.postMealBloodSugar,
                        mood = log.overallMood,
                        anxiety = log.anxietySeverity,
                        moodSwings = log.moodSwingsSeverity,
                        depression = log.depressionSeverity,
                        notes = log.notes.orEmpty(),
                    ),
                )
            }
        }
    }

    private fun toggleSymptom(symptom: PregnancySymptom, on: Boolean) {
        _state.update { current ->
            val updated = if (on) {
                current.symptoms + (symptom to SymptomSeverity.MILD)
            } else {
                current.symptoms - symptom
            }
            current.copy(symptoms = updated, symptomSave = LogSaveState.IDLE)
        }
    }

    private fun saveSymptoms() {
        val snapshot = _state.value
        if (snapshot.symptoms.isEmpty() && snapshot.symptomNotes.isBlank()) return
        if (snapshot.symptomSave == LogSaveState.SAVING) return
        viewModelScope.launch {
            Breadcrumbs.add("pregnancy_log:save_symptoms")
            _state.update { it.copy(symptomSave = LogSaveState.SAVING) }
            val result = logs.saveSymptomLog(
                PregnancySymptomLog(
                    date = snapshot.date.toGregorian(),
                    symptoms = snapshot.symptoms,
                    notes = snapshot.symptomNotes.takeIf { it.isNotBlank() },
                ),
            )
            _state.update {
                when (result) {
                    is AppResult.Failure -> it.copy(symptomSave = LogSaveState.ERROR)
                    is AppResult.Success -> it.copy(symptomSave = LogSaveState.SAVED, alertsRaised = result.value)
                }
            }
        }
    }

    private fun saveWeekly() {
        val snapshot = _state.value
        if (!snapshot.weekly.hasAnything || snapshot.weeklySave == LogSaveState.SAVING) return
        viewModelScope.launch {
            Breadcrumbs.add("pregnancy_log:save_weekly")
            _state.update { it.copy(weeklySave = LogSaveState.SAVING) }
            val draft = snapshot.weekly
            val result = logs.saveWeeklyLog(
                PregnancyWeeklyLog(
                    week = snapshot.week,
                    date = today.toGregorian(),
                    weightKg = draft.weightKg,
                    hasSwelling = draft.hasSwelling,
                    swellingLocations = draft.swellingLocations,
                    hasShortnessOfBreath = draft.hasShortnessOfBreath,
                    hasBloodPressureDevice = draft.hasBpDevice,
                    systolicPressure = draft.systolic.takeIf { draft.hasBpDevice },
                    diastolicPressure = draft.diastolic.takeIf { draft.hasBpDevice },
                    fastingBloodSugar = draft.fastingSugar.takeIf { draft.hasBpDevice },
                    postMealBloodSugar = draft.postMealSugar.takeIf { draft.hasBpDevice },
                    overallMood = draft.mood,
                    anxietySeverity = draft.anxiety,
                    moodSwingsSeverity = draft.moodSwings,
                    depressionSeverity = draft.depression,
                    notes = draft.notes.takeIf { it.isNotBlank() },
                ),
            )
            _state.update {
                it.copy(weeklySave = if (result is AppResult.Success) LogSaveState.SAVED else LogSaveState.ERROR)
            }
        }
    }

    private fun saveMovement() {
        val snapshot = _state.value
        val status = snapshot.movement.status ?: return
        if (snapshot.movementSave == LogSaveState.SAVING) return
        viewModelScope.launch {
            Breadcrumbs.add("pregnancy_log:save_movement")
            _state.update { it.copy(movementSave = LogSaveState.SAVING) }
            val draft = snapshot.movement
            val result = logs.saveFetalMovement(
                FetalMovementLog(
                    date = snapshot.date.toGregorian(),
                    week = snapshot.week,
                    movementStatus = status,
                    movementCount = draft.count,
                    firstMovementTime = draft.firstTime,
                    lastMovementTime = draft.lastTime,
                    notes = draft.notes.takeIf { it.isNotBlank() },
                ),
            )
            _state.update {
                it.copy(movementSave = if (result is AppResult.Success) LogSaveState.SAVED else LogSaveState.ERROR)
            }
        }
    }
}

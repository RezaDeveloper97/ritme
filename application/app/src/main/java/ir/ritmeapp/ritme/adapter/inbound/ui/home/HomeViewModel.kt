package ir.ritmeapp.ritme.adapter.inbound.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JwtClaims
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderDraft
import ir.ritmeapp.ritme.domain.model.ReminderType
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.GetBannersUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetHomeDashboardUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetTrackingModeUseCase
import ir.ritmeapp.ritme.domain.port.inbound.ManageRemindersUseCase
import ir.ritmeapp.ritme.domain.port.inbound.StartPeriodUseCase
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Drives the Home dashboard: loads the cycle + daily-message bundle, the promo banners, and the
 * tracking mode, and runs the two-tap «شروع پریود» action. Personalizes the greeting from the
 * stored token (display only, never for authorization). Depends only on ports (§4).
 */
class HomeViewModel(
    private val getDashboard: GetHomeDashboardUseCase,
    private val getBanners: GetBannersUseCase,
    private val startPeriod: StartPeriodUseCase,
    private val getTrackingMode: GetTrackingModeUseCase,
    private val manageReminders: ManageRemindersUseCase,
    private val tokenStore: TokenStore,
    private val today: JalaliDate,
) : ViewModel() {

    private val _state = MutableStateFlow(HomeUiState())
    val state: StateFlow<HomeUiState> = _state.asStateFlow()

    private var confirmTimeout: Job? = null

    init {
        loadGreeting()
        load()
        loadBanners()
        loadMode()
        loadReminders()
    }

    fun onRetry() = load()

    /**
     * Adds a to-do/reminder to today's day-planner (web `DayTasks` add). Mirrors the web widget:
     * the new item is scheduled for today so it surfaces both here and on the daily-log planner.
     */
    fun onAddTask(type: ReminderType, title: String) {
        val clean = title.trim()
        if (clean.isEmpty()) return
        viewModelScope.launch {
            Breadcrumbs.add("home:task:add")
            val draft = ReminderDraft(type = type, title = clean, scheduledAt = today.toIso())
            manageReminders.create(draft).getOrNull()?.let { created ->
                _state.update { it.copy(reminders = it.reminders + created) }
            }
        }
    }

    /** Removes a day-planner item (web `DayTasks` delete). */
    fun onDeleteTask(reminder: Reminder) {
        viewModelScope.launch {
            Breadcrumbs.add("home:task:delete")
            if (manageReminders.delete(reminder.id) is AppResult.Success) {
                _state.update { s -> s.copy(reminders = s.reminders.filterNot { it.id == reminder.id }) }
            }
        }
    }

    /** Checks a today's task/reminder off (or back on) — done maps to an inactive reminder. */
    fun onToggleTask(reminder: Reminder) {
        viewModelScope.launch {
            Breadcrumbs.add("home:task:toggle")
            val draft = ReminderDraft(
                type = reminder.type,
                title = reminder.title,
                subtitle = reminder.subtitle,
                notes = reminder.notes,
                scheduledAt = reminder.scheduledAt,
                recurrence = reminder.recurrence,
                recurrenceTime = reminder.recurrenceTime,
                isActive = !reminder.isActive,
            )
            manageReminders.update(reminder.id, draft).getOrNull()?.let { updated ->
                _state.update { current ->
                    current.copy(reminders = current.reminders.map { if (it.id == updated.id) updated else it })
                }
            }
        }
    }

    /** First tap arms the confirmation; the second within the window actually records the period. */
    fun onStartPeriodTapped() {
        when (_state.value.startPeriod) {
            StartPeriodStatus.IDLE, StartPeriodStatus.ERROR -> armConfirmation()
            StartPeriodStatus.CONFIRMING -> submitStartPeriod()
            StartPeriodStatus.PENDING -> Unit
        }
    }

    private fun armConfirmation() {
        _state.update { it.copy(startPeriod = StartPeriodStatus.CONFIRMING) }
        confirmTimeout?.cancel()
        confirmTimeout = viewModelScope.launch {
            delay(CONFIRM_WINDOW_MS)
            _state.update {
                if (it.startPeriod == StartPeriodStatus.CONFIRMING) it.copy(startPeriod = StartPeriodStatus.IDLE) else it
            }
        }
    }

    private fun submitStartPeriod() {
        confirmTimeout?.cancel()
        viewModelScope.launch {
            Breadcrumbs.add("home:start_period")
            _state.update { it.copy(startPeriod = StartPeriodStatus.PENDING) }
            when (startPeriod(today)) {
                is AppResult.Failure -> _state.update { it.copy(startPeriod = StartPeriodStatus.ERROR) }
                is AppResult.Success -> {
                    _state.update { it.copy(startPeriod = StartPeriodStatus.IDLE) }
                    load()
                }
            }
        }
    }

    private fun load() {
        Breadcrumbs.add("ui:home:load")
        _state.update { it.copy(loading = true, isError = false, errorMessage = null) }
        viewModelScope.launch {
            when (val result = getDashboard()) {
                is AppResult.Success ->
                    _state.update { it.copy(loading = false, dashboard = result.value, isError = false, errorMessage = null) }

                is AppResult.Failure ->
                    _state.update { it.copy(loading = false, isError = true, errorMessage = serverMessageOf(result.error)) }
            }
        }
    }

    private fun loadBanners() {
        viewModelScope.launch {
            getBanners().getOrNull()?.let { banners -> _state.update { it.copy(banners = banners) } }
        }
    }

    private fun loadMode() {
        viewModelScope.launch {
            getTrackingMode().getOrNull()?.let { mode -> _state.update { it.copy(mode = mode) } }
        }
    }

    private fun loadReminders() {
        viewModelScope.launch {
            manageReminders.list().getOrNull()?.let { list -> _state.update { it.copy(reminders = list) } }
        }
    }

    private fun loadGreeting() {
        viewModelScope.launch {
            val name = tokenStore.load()?.let { JwtClaims.firstName(it.accessToken) }
            if (name != null) _state.update { it.copy(greetingName = name) }
        }
    }

    /** Prefer the backend's Persian message; connectivity errors fall back to a generic string. */
    private fun serverMessageOf(error: AppError): String? = (error as? AppError.Http)?.message

    private companion object {
        const val CONFIRM_WINDOW_MS = 4_000L
    }
}

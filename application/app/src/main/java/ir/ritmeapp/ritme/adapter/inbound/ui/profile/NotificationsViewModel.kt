package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.NotificationsPage
import ir.ritmeapp.ritme.domain.port.inbound.NotificationsUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** The notifications inbox state (§4 MVI). */
@Immutable
data class NotificationsUiState(
    val loading: Boolean = true,
    val isError: Boolean = false,
    val page: NotificationsPage? = null,
)

sealed interface NotificationsIntent {
    data object Retry : NotificationsIntent
    data class MarkRead(val id: Long) : NotificationsIntent
    data object MarkAllRead : NotificationsIntent
}

/** Notifications inbox over `/home/notifications`; read-marking optimistically updates the list. */
class NotificationsViewModel(
    private val notifications: NotificationsUseCase,
) : ViewModel() {

    private val _state = MutableStateFlow(NotificationsUiState())
    val state: StateFlow<NotificationsUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun onIntent(intent: NotificationsIntent) {
        when (intent) {
            NotificationsIntent.Retry -> load()
            is NotificationsIntent.MarkRead -> markRead(intent.id)
            NotificationsIntent.MarkAllRead -> markAllRead()
        }
    }

    private fun load() {
        viewModelScope.launch {
            _state.update { it.copy(loading = true, isError = false) }
            when (val result = notifications.page(PER_PAGE)) {
                is AppResult.Failure -> _state.update { it.copy(loading = false, isError = true) }
                is AppResult.Success -> _state.update { it.copy(loading = false, page = result.value) }
            }
        }
    }

    private fun markRead(id: Long) {
        val page = _state.value.page ?: return
        if (page.items.firstOrNull { it.id == id }?.isRead != false) return
        // Optimistic: flip locally, then tell the server; a failure just refreshes.
        _state.update { current ->
            val updated = current.page ?: return@update current
            current.copy(
                page = updated.copy(
                    unreadCount = (updated.unreadCount - 1).coerceAtLeast(0),
                    items = updated.items.map { if (it.id == id) it.copy(isRead = true) else it },
                ),
            )
        }
        viewModelScope.launch {
            Breadcrumbs.add("notifications:read")
            if (notifications.markRead(id) is AppResult.Failure) load()
        }
    }

    private fun markAllRead() {
        _state.update { current ->
            val page = current.page ?: return@update current
            current.copy(page = page.copy(unreadCount = 0, items = page.items.map { it.copy(isRead = true) }))
        }
        viewModelScope.launch {
            Breadcrumbs.add("notifications:read_all")
            if (notifications.markAllRead() is AppResult.Failure) load()
        }
    }

    private companion object {
        const val PER_PAGE = 50
    }
}

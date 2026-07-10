package ir.ritmeapp.ritme.adapter.inbound.ui.home

import androidx.compose.runtime.Immutable
import ir.ritmeapp.ritme.domain.model.Banner
import ir.ritmeapp.ritme.domain.model.BannerSlot
import ir.ritmeapp.ritme.domain.model.HomeDashboard
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.TrackingMode

/** The two-tap «شروع پریود» flow: idle → confirming → pending, with an error side-track. */
enum class StartPeriodStatus { IDLE, CONFIRMING, PENDING, ERROR }

/**
 * Immutable snapshot the Home screen renders from. [dashboard] holds the loaded cycle + message data
 * (null while loading or on error); [greetingName] is read from the JWT for display only. A single
 * flat state (with [loading]/[isError] flags) keeps the long dashboard simple to render (§4).
 */
@Immutable
data class HomeUiState(
    val loading: Boolean = true,
    val greetingName: String? = null,
    val dashboard: HomeDashboard? = null,
    val isError: Boolean = false,
    val errorMessage: String? = null,
    val banners: Map<BannerSlot, List<Banner>> = emptyMap(),
    val startPeriod: StartPeriodStatus = StartPeriodStatus.IDLE,
    val mode: TrackingMode = TrackingMode.CYCLE,
    val reminders: List<Reminder> = emptyList(),
)

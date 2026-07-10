package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.compose.runtime.Immutable
import ir.ritmeapp.ritme.domain.model.TrackingMode
import ir.ritmeapp.ritme.domain.model.UserProfile

/**
 * The profile hub's single immutable state (§4 MVI): the loaded account/health/BMI bundle,
 * the tracking mode, and the progress flags of the destructive actions.
 */
@Immutable
data class ProfileUiState(
    val loading: Boolean = true,
    val isError: Boolean = false,
    val profile: UserProfile? = null,
    val mode: TrackingMode = TrackingMode.CYCLE,
    val loggingOut: Boolean = false,
    val switchingMode: Boolean = false,
    val exporting: Boolean = false,
    val exportError: Boolean = false,
    val deleteSheetOpen: Boolean = false,
    val deleting: Boolean = false,
    val deleteError: Boolean = false,
)

/** Everything the user can do on the profile hub (navigation rows are plain lambdas). */
sealed interface ProfileIntent {
    data object Retry : ProfileIntent
    data object Logout : ProfileIntent
    data object SwitchToCycleMode : ProfileIntent
    data object Export : ProfileIntent
    data object OpenDeleteSheet : ProfileIntent
    data object CloseDeleteSheet : ProfileIntent
    data object ConfirmDelete : ProfileIntent
}

/** One-shot side effects: navigation after sign-out/delete and handing the export to the OS. */
sealed interface ProfileEffect {
    data object SignedOut : ProfileEffect
    data class ShareExport(val json: String) : ProfileEffect
}

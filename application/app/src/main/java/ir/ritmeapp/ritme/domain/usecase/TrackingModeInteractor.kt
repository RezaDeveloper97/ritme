package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.TrackingMode
import ir.ritmeapp.ritme.domain.port.inbound.GetTrackingModeUseCase
import ir.ritmeapp.ritme.domain.port.outbound.AppPreferences
import ir.ritmeapp.ritme.domain.port.outbound.ModeGateway

/**
 * Default [GetTrackingModeUseCase]: asks the backend, caches the answer, and falls back to the
 * cached mode when offline — the bottom navigation must never flip to the wrong tracker just
 * because one request failed.
 */
class TrackingModeInteractor(
    private val modeGateway: ModeGateway,
    private val appPreferences: AppPreferences,
) : GetTrackingModeUseCase {

    override suspend fun invoke(): AppResult<TrackingMode> =
        when (val result = modeGateway.currentMode()) {
            is AppResult.Success -> {
                appPreferences.saveTrackingMode(result.value.apiValue)
                result
            }

            is AppResult.Failure -> {
                val cached = appPreferences.lastTrackingMode()
                if (cached != null) AppResult.Success(TrackingMode.fromApi(cached)) else result
            }
        }
}

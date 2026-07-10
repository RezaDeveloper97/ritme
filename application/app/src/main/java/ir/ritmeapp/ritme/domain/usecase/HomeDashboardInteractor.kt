package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.CyclePredictions
import ir.ritmeapp.ritme.domain.model.HomeDashboard
import ir.ritmeapp.ritme.domain.port.inbound.GetHomeDashboardUseCase
import ir.ritmeapp.ritme.domain.port.outbound.CycleGateway
import ir.ritmeapp.ritme.domain.port.outbound.MessageGateway
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

/**
 * Default [GetHomeDashboardUseCase]. Fetches the cycle calculation and the daily message
 * concurrently (§5 — the two calls are independent), derives predictions, and bundles them. The
 * cycle call is required; a failed message call degrades gracefully to no message rather than
 * failing the whole screen.
 */
class HomeDashboardInteractor(
    private val cycleGateway: CycleGateway,
    private val messageGateway: MessageGateway,
) : GetHomeDashboardUseCase {

    override suspend fun invoke(): AppResult<HomeDashboard> = coroutineScope {
        val cycleDeferred = async { cycleGateway.today() }
        val messageDeferred = async { messageGateway.daily() }

        when (val cycle = cycleDeferred.await()) {
            is AppResult.Failure -> cycle
            is AppResult.Success -> {
                val message = (messageDeferred.await() as? AppResult.Success)?.value
                AppResult.Success(
                    HomeDashboard(
                        calculation = cycle.value,
                        predictions = CyclePredictions.from(cycle.value),
                        message = message,
                    ),
                )
            }
        }
    }
}

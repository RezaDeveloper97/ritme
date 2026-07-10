package ir.ritmeapp.ritme.domain.model

/**
 * Everything the Home screen needs in one bundle: the raw [calculation], its derived
 * [predictions] (null until there's cycle data), and the optional [message] (null when the profile
 * is incomplete or the message service has nothing for the day). Assembled by the Home use case so
 * the ViewModel gets a single, ready-to-render value.
 */
data class HomeDashboard(
    val calculation: CycleCalculation,
    val predictions: CyclePredictions?,
    val message: DailyMessage?,
)

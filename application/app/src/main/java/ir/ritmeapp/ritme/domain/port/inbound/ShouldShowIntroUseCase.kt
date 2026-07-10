package ir.ritmeapp.ritme.domain.port.inbound

/**
 * Inbound port: should the first-run welcome intro be shown before login? The app root asks this
 * once at startup for a logged-out user to decide between the Welcome carousel and Login.
 */
interface ShouldShowIntroUseCase {
    suspend operator fun invoke(): Boolean
}

package ir.ritmeapp.ritme.domain.port.inbound

/**
 * Inbound port: record that the welcome intro has been seen. The Welcome screen invokes this when
 * the carousel is completed or skipped, so it is never shown again.
 */
interface MarkIntroSeenUseCase {
    suspend operator fun invoke()
}

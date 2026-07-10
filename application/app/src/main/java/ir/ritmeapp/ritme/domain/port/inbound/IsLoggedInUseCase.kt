package ir.ritmeapp.ritme.domain.port.inbound

/**
 * Inbound port: is there a persisted session, i.e. can this launch skip the login flow?
 * The app root uses it at startup to pick the first screen. Kept boolean and I/O-free at the
 * call site (the implementation reads storage) so the UI stays ignorant of how a session is
 * persisted.
 */
interface IsLoggedInUseCase {
    suspend operator fun invoke(): Boolean
}

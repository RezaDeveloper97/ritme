package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult

/**
 * Inbound port: irreversibly delete the account and all its data, then clear the local
 * session. Unlike logout, this fails hard when the server can't be reached — the account
 * must actually be gone before local state is wiped.
 */
interface DeleteAccountUseCase {
    suspend operator fun invoke(): AppResult<Unit>
}

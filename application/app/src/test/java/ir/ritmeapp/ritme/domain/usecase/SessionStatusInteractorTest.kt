package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AuthTokens
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class SessionStatusInteractorTest {

    /** Hand-written fake port (CLAUDE.md §9 — no mocking framework). */
    private class FakeTokenStore(private var stored: AuthTokens? = null) : TokenStore {
        override suspend fun save(tokens: AuthTokens) { stored = tokens }
        override suspend fun load(): AuthTokens? = stored
        override suspend fun clear() { stored = null }
    }

    @Test
    fun `reports logged in when a token is stored`() = runBlocking {
        val interactor = SessionStatusInteractor(FakeTokenStore(AuthTokens("abc")))
        assertTrue(interactor())
    }

    @Test
    fun `reports logged out when no token is stored`() = runBlocking {
        val interactor = SessionStatusInteractor(FakeTokenStore(null))
        assertFalse(interactor())
    }

    @Test
    fun `reflects a cleared session as logged out`() = runBlocking {
        val store = FakeTokenStore(AuthTokens("abc"))
        store.clear()
        assertFalse(SessionStatusInteractor(store)())
    }
}

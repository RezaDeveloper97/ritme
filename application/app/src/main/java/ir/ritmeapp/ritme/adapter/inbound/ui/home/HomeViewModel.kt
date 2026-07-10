package ir.ritmeapp.ritme.adapter.inbound.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JwtClaims
import ir.ritmeapp.ritme.domain.port.inbound.GetHomeServicesUseCase
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Drives the home screen: exposes the service catalog immediately and, once the stored
 * access token is read off the IO dispatcher, personalizes the greeting. Depends only on
 * ports (§4); the JWT is read for display only, never for authorization.
 */
class HomeViewModel(
    getCatalog: GetHomeServicesUseCase,
    private val tokenStore: TokenStore,
) : ViewModel() {

    private val _state = MutableStateFlow(HomeUiState(services = getCatalog()))
    val state: StateFlow<HomeUiState> = _state.asStateFlow()

    init {
        Breadcrumbs.add("ui:home:load")
        viewModelScope.launch {
            val name = tokenStore.load()?.let { JwtClaims.firstName(it.accessToken) }
            if (name != null) _state.update { it.copy(greetingName = name) }
        }
    }
}

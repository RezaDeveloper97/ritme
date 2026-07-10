package ir.ritmeapp.ritme.adapter.inbound.ui.welcome

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.port.inbound.MarkIntroSeenUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Drives the first-run welcome carousel. Its only job is to remember that the intro was seen (so it
 * never gates a returning visitor again) and then signal the screen to move on to login. Depends
 * only on the inbound port (§4).
 */
class WelcomeViewModel(
    private val markIntroSeen: MarkIntroSeenUseCase,
) : ViewModel() {

    private val _finished = MutableStateFlow(false)
    val finished: StateFlow<Boolean> = _finished.asStateFlow()

    /** Called when the user finishes or skips the intro. */
    fun onCompleted() {
        Breadcrumbs.add("ui:welcome:completed")
        viewModelScope.launch {
            markIntroSeen()
            _finished.value = true
        }
    }
}

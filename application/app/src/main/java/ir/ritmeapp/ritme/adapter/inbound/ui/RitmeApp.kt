package ir.ritmeapp.ritme.adapter.inbound.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeNavHost
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.rememberNavigator
import ir.ritmeapp.ritme.adapter.outbound.di.AppContainer
import ir.ritmeapp.ritme.platform.crash.RecoveryState

/**
 * App root: provides the composition-root [AppContainer] to the tree, hosts the hand-written
 * navigation, and shows the calm one-time "you're back where you left off" notice when this
 * launch is a crash recovery (CLAUDE.md §7.3).
 */
@Composable
fun RitmeApp(
    container: AppContainer,
    recovery: RecoveryState,
) {
    CompositionLocalProvider(LocalAppContainer provides container) {
        val navigator = rememberNavigator(start = Destination.Login)
        val snackbarHostState = remember { SnackbarHostState() }
        val recoveryMessage = stringResource(R.string.recovery_message)

        LaunchedEffect(recovery.recovered) {
            if (recovery.recovered) {
                snackbarHostState.showSnackbar(recoveryMessage)
            }
        }

        Box(Modifier.fillMaxSize()) {
            RitmeNavHost(navigator)
            SnackbarHost(snackbarHostState, Modifier.align(Alignment.BottomCenter))
        }
    }
}

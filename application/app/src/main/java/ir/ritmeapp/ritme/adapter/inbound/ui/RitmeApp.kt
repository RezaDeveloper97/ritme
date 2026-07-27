package ir.ritmeapp.ritme.adapter.inbound.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import ir.ritmeapp.ritme.R
import kotlinx.coroutines.delay
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeNavHost
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.rememberNavigator
import ir.ritmeapp.ritme.adapter.outbound.di.AppContainer
import ir.ritmeapp.ritme.platform.crash.RecoveryState

/**
 * App root: provides the composition-root [AppContainer] to the tree, resolves the first
 * screen from the persisted session + crash-recovery state, hosts the hand-written navigation,
 * and shows the calm one-time "you're back where you left off" notice when this launch is a
 * crash recovery (CLAUDE.md §7.3).
 *
 * The start destination is decided once, off the main thread: a returning user with a stored
 * session opens on [Destination.Home] (and stays there after a crash-recovery relaunch),
 * everyone else on [Destination.Login]. A brief [StartupSplash] covers that single token read
 * so the app never flashes the login screen at an already-signed-in user.
 */
@Composable
fun RitmeApp(
    container: AppContainer,
    recovery: RecoveryState,
) {
    CompositionLocalProvider(LocalAppContainer provides container) {
        val start by produceState<Destination?>(initialValue = null, container) {
            value = resolveStartDestination(container)
        }

        // Hold the splash for a minimum time so the brand always shows briefly (web parity:
        // ~2.2s), while the start destination resolves off the main thread. A tap skips the
        // remaining hold, so an impatient returning user isn't blocked once data is ready.
        var minHoldElapsed by remember { mutableStateOf(false) }
        var skipped by remember { mutableStateOf(false) }
        LaunchedEffect(Unit) {
            delay(SPLASH_MIN_HOLD_MILLIS)
            minHoldElapsed = true
        }

        val resolved = start
        if (resolved != null && (minHoldElapsed || skipped)) {
            AuthenticatedShell(start = resolved, recovery = recovery)
        } else {
            Box(
                Modifier
                    .fillMaxSize()
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        onClick = { skipped = true },
                    ),
            ) {
                StartupSplash(Modifier.fillMaxSize())
            }
        }
    }
}

/** Minimum time the brand splash stays up on a cold start, mirroring the web's ~2.2s hold. */
private const val SPLASH_MIN_HOLD_MILLIS = 2200L

@Composable
private fun AuthenticatedShell(
    start: Destination,
    recovery: RecoveryState,
) {
    val navigator = rememberNavigator(start = start)
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

/**
 * Picks the first screen. A stored session opens Home (honoring a crash relaunch for the only
 * cold-start landing that makes sense today, §7.3). A logged-out visitor who hasn't seen the
 * first-run intro opens the Welcome carousel; otherwise Login. Otp and Onboarding are transient
 * post-auth steps and never a cold-start destination.
 */
private suspend fun resolveStartDestination(container: AppContainer): Destination = when {
    container.isLoggedInUseCase() -> Destination.Home
    container.shouldShowIntroUseCase() -> Destination.Welcome
    else -> Destination.Login
}

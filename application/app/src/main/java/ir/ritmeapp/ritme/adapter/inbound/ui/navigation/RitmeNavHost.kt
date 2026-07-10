package ir.ritmeapp.ritme.adapter.inbound.ui.navigation

import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import ir.ritmeapp.ritme.adapter.inbound.ui.home.HomeScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.login.LoginScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.otp.OtpScreen

/**
 * Renders the current destination and wires both ways "back" must work: the hardware/gesture
 * Back button (via [BackHandler]) and the edge swipe ([SwipeBackContainer]) perform the exact
 * same `pop` (CLAUDE.md §5b parity). The previous destination is rendered beneath the current
 * one so the swipe reveals it. Screen-to-screen navigation is expressed here as `push`/
 * `replaceAll`, keeping every screen ignorant of the others.
 */
@Composable
fun RitmeNavHost(
    navigator: Navigator,
    modifier: Modifier = Modifier,
) {
    BackHandler(enabled = navigator.canPop) { navigator.pop() }

    val current = navigator.current
    val previous = navigator.backStack.getOrNull(navigator.backStack.lastIndex - 1)

    SwipeBackContainer(
        enabled = navigator.canPop,
        onSwipeBack = { navigator.pop() },
        modifier = modifier,
        behind = previous?.let { dest -> { DestinationContent(dest, navigator) } },
    ) {
        DestinationContent(current, navigator)
    }
}

@Composable
private fun DestinationContent(destination: Destination, navigator: Navigator) {
    when (destination) {
        Destination.Login -> LoginScreen(
            onValidated = { mobile -> navigator.push(Destination.Otp(mobile)) },
        )

        is Destination.Otp -> OtpScreen(
            mobile = destination.mobile,
            onAuthenticated = { navigator.replaceAll(Destination.Home) },
        )

        Destination.Home -> HomeScreen()
    }
}

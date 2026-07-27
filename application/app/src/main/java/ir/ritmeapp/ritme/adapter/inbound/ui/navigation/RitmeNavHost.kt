package ir.ritmeapp.ritme.adapter.inbound.ui.navigation

import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import ir.ritmeapp.ritme.adapter.inbound.ui.calendar.CalendarScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.cycle.CycleScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.home.HomeScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.log.LogScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.login.LoginScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.onboarding.OnboardingScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.otp.OtpScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy.PregnancyLogScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy.PregnancyOnboardingScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy.PregnancyScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.InfoScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.NotificationsScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.ProfileHealthScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.ProfilePersonalScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.ProfileScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.RemindersScreen
import ir.ritmeapp.ritme.adapter.inbound.ui.welcome.WelcomeScreen

/**
 * Renders the current destination and wires both ways "back" must work: the hardware/gesture
 * Back button (via [BackHandler]) and the edge swipe ([SwipeBackContainer]) perform the exact
 * same `pop` (CLAUDE.md §5b parity). The previous destination is rendered beneath the current
 * one so the swipe reveals it. Screen-to-screen navigation is expressed here as `push`/
 * `replaceAll`, keeping every screen ignorant of the others.
 *
 * Tab switches from the bottom bar `replaceAll` (tabs are peers, not a stack); everything
 * else `push`es so swipe-back works.
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

/** Bottom-bar taps: switching tabs resets the stack to that tab (web tabs are peers). */
private fun tabNavigate(navigator: Navigator, destination: Destination) {
    if (navigator.current::class == destination::class) return
    navigator.replaceAll(destination)
}

@Composable
private fun DestinationContent(destination: Destination, navigator: Navigator) {
    when (destination) {
        Destination.Welcome -> WelcomeScreen(
            onFinished = { navigator.replaceAll(Destination.Login) },
        )

        Destination.Login -> LoginScreen(
            onValidated = { mobile, newUser -> navigator.push(Destination.Otp(mobile, newUser)) },
        )

        is Destination.Otp -> OtpScreen(
            mobile = destination.mobile,
            onAuthenticated = {
                // A brand-new account finishes signup first; a returning one goes straight home.
                navigator.replaceAll(if (destination.newUser) Destination.Onboarding else Destination.Home)
            },
        )

        Destination.Onboarding -> OnboardingScreen(
            onDone = { navigator.replaceAll(Destination.Home) },
        )

        Destination.Home -> HomeScreen(
            onNavigate = { target -> tabNavigate(navigator, target) },
        )

        Destination.Calendar -> CalendarScreen(
            onNavigate = { target ->
                // The «ویرایش» jump into a specific day's log is a sub-flow, not a tab switch.
                if (target is Destination.Log && target.dateIso != null) {
                    navigator.push(target)
                } else {
                    tabNavigate(navigator, target)
                }
            },
        )

        Destination.Cycle -> CycleScreen(
            onNavigate = { target -> tabNavigate(navigator, target) },
        )

        is Destination.Log -> LogScreen(
            initialDateIso = destination.dateIso,
            onNavigate = { target -> tabNavigate(navigator, target) },
        )

        Destination.Profile -> ProfileScreen(
            onNavigate = { target -> tabNavigate(navigator, target) },
            onOpenSubScreen = { target -> navigator.push(target) },
            onSignedOut = { navigator.replaceAll(Destination.Login) },
        )

        Destination.ProfilePersonal -> ProfilePersonalScreen(onBack = { navigator.pop() })

        Destination.ProfileHealth -> ProfileHealthScreen(onBack = { navigator.pop() })

        Destination.ProfileReminders -> RemindersScreen(onBack = { navigator.pop() })

        Destination.ProfileNotifications -> NotificationsScreen(onBack = { navigator.pop() })

        is Destination.ProfileInfo -> InfoScreen(topic = destination.topic, onBack = { navigator.pop() })

        Destination.Pregnancy -> PregnancyScreen(
            onNavigate = { target -> tabNavigate(navigator, target) },
            onOpenLog = { tab -> navigator.push(Destination.PregnancyLog(tab)) },
            onStartOnboarding = { navigator.push(Destination.PregnancyOnboarding) },
        )

        Destination.PregnancyOnboarding -> PregnancyOnboardingScreen(
            onBack = { navigator.pop() },
            onCompleted = { navigator.replaceAll(Destination.Pregnancy) },
        )

        is Destination.PregnancyLog -> PregnancyLogScreen(
            initialTab = destination.tab,
            onBack = { navigator.pop() },
            onNavigate = { target -> tabNavigate(navigator, target) },
        )
    }
}

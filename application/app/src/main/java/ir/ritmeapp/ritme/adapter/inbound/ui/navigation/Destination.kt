package ir.ritmeapp.ritme.adapter.inbound.ui.navigation

/**
 * The closed set of screens in the app, modelled as a sealed hierarchy so navigation is
 * type-safe and `when` over it stays exhaustive (CLAUDE.md §4). [route] is the small, stable
 * string used for crash-recovery records; data-carrying destinations hold only minimal args.
 */
sealed interface Destination {
    val route: String

    data object Login : Destination {
        override val route: String = "login"
    }

    /** One-time-code step for a mobile number that has just been SMSed a code. */
    data class Otp(val mobile: String) : Destination {
        override val route: String = "otp"
    }

    data object Home : Destination {
        override val route: String = "home"
    }
}

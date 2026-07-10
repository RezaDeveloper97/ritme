package ir.ritmeapp.ritme.adapter.inbound.ui.navigation

/**
 * The closed set of screens in the app, modelled as a sealed hierarchy so navigation is
 * type-safe and `when` over it stays exhaustive (CLAUDE.md §4). [route] is the small, stable
 * string used for crash-recovery records; data-carrying destinations hold only minimal args.
 */
sealed interface Destination {
    val route: String

    /** First-run intro carousel, shown once before login to a brand-new visitor. */
    data object Welcome : Destination {
        override val route: String = "welcome"
    }

    data object Login : Destination {
        override val route: String = "login"
    }

    /**
     * One-time-code step for a mobile number that has just been SMSed a code. [newUser] carries the
     * backend's account-existence flag so a verified brand-new account is routed into onboarding
     * while a returning one goes straight home.
     */
    data class Otp(val mobile: String, val newUser: Boolean) : Destination {
        override val route: String = ROUTE

        companion object {
            /** Stable route string, usable without an instance (e.g. safe-screen records). */
            const val ROUTE = "otp"
        }
    }

    /** Post-signup profile wizard, shown to a newly-registered account before home. */
    data object Onboarding : Destination {
        override val route: String = "onboarding"
    }

    data object Home : Destination {
        override val route: String = "home"
    }

    /** The Jalali cycle calendar tab. */
    data object Calendar : Destination {
        override val route: String = "calendar"
    }

    /** The cycle-analysis tab (donut, timeline, summary). */
    data object Cycle : Destination {
        override val route: String = "cycle"
    }

    /**
     * The daily health-log form. [dateIso] preselects a day (from the calendar's «ویرایش»);
     * null means today.
     */
    data class Log(val dateIso: String? = null) : Destination {
        override val route: String = ROUTE

        companion object {
            const val ROUTE = "log"
        }
    }

    /** The pregnancy tracker tab (week browser, content, alerts). */
    data object Pregnancy : Destination {
        override val route: String = "pregnancy"
    }

    /** Pregnancy-mode setup: dating source + medical history. */
    data object PregnancyOnboarding : Destination {
        override val route: String = "pregnancy_onboarding"
    }

    /** The pregnancy log form. [tab] picks the initial section. */
    data class PregnancyLog(val tab: PregnancyLogTab = PregnancyLogTab.SYMPTOMS) : Destination {
        override val route: String = ROUTE

        companion object {
            const val ROUTE = "pregnancy_log"
        }
    }

    /** The profile hub tab. */
    data object Profile : Destination {
        override val route: String = "profile"
    }

    /** Edit name / birthday / weight / height. */
    data object ProfilePersonal : Destination {
        override val route: String = "profile_personal"
    }

    /** Edit cycle length / period length / last period. */
    data object ProfileHealth : Destination {
        override val route: String = "profile_health"
    }

    /** Reminder list + create/edit. */
    data object ProfileReminders : Destination {
        override val route: String = "profile_reminders"
    }

    /** In-app notifications inbox. */
    data object ProfileNotifications : Destination {
        override val route: String = "profile_notifications"
    }

    /** Static info pages (privacy / terms / about / help). */
    data class ProfileInfo(val topic: InfoTopic) : Destination {
        override val route: String = ROUTE

        companion object {
            const val ROUTE = "profile_info"
        }
    }
}

/** The three sections of the pregnancy log screen. */
enum class PregnancyLogTab { SYMPTOMS, WEEKLY, MOVEMENT }

/** The static informational topics under profile. */
enum class InfoTopic { PRIVACY, TERMS, ABOUT, HELP }

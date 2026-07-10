package ir.ritmeapp.ritme.domain.model

/**
 * How the user relates to pregnancy, captured once at onboarding. This is Ritme's mode switch:
 * [PREGNANT] puts the account in pregnancy mode (cycle questions are skipped), everything else
 * is cycle mode. [apiValue] is the exact string the backend's `pregnancy_intention` enum expects
 * (`avoiding` / `pregnant` / `trying` / `unsure`); the server derives `user_goal` from it.
 */
enum class PregnancyIntention(val apiValue: String) {
    AVOIDING("avoiding"),
    PREGNANT("pregnant"),
    TRYING("trying"),
    UNSURE("unsure"),
    ;

    /** Cycle mode collects period/cycle data; pregnancy mode does not. */
    val isCycleMode: Boolean get() = this != PREGNANT

    companion object {
        /** Maps a backend token back to the enum; null for absent/unknown values. */
        fun fromApi(value: String?): PregnancyIntention? = entries.firstOrNull { it.apiValue == value }
    }
}

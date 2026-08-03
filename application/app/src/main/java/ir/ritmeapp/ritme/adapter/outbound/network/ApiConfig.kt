package ir.ritmeapp.ritme.adapter.outbound.network

/**
 * Centralized backend host + endpoint paths — the one place to change hosts. [BASE_URL] is
 * the Ritme production origin; every path below is absolute from it so a single
 * [ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient] serves auth and diagnostics.
 */
object ApiConfig {
    // Ritme production origin (Laravel backend), TLS-terminated at the nginx proxy.
    // Must stay https: the server 301-redirects http → https, and a 301 on a POST is
    // rewritten to a GET by the HTTP stack, silently dropping the request body.
    // For local dev point this at http://10.0.2.2:8010 (emulator → host :8010, test OTP 1111).
    const val BASE_URL = "https://ritmeapp.ir"

    // Ritme API is versioned under /api/v1 (backend/routes/api.php).
    private const val V1 = "/api/v1"
    private const val AUTH = "$V1/auth"
    private const val PREGNANCY = "$V1/pregnancy"

    // --- Auth ---------------------------------------------------------------

    /** Step 1: SMS a one-time code to a mobile number (`{mobile, is_test}`). */
    const val SEND_OTP_PATH = "$AUTH/send-otp"

    /** Step 2: exchange the one-time code for an access token (`{mobile, code}`). */
    const val VERIFY_OTP_PATH = "$AUTH/verify-otp"

    /** Revokes the current access token. */
    const val LOGOUT_PATH = "$AUTH/logout"

    /** The authenticated user's account record (id, name, mobile). */
    const val USER_PATH = "$AUTH/user"

    // --- Profile ------------------------------------------------------------

    /** `GET` full profile + BMI; `POST` create/update (onboarding + settings). */
    const val PROFILE_PATH = "$V1/profile"

    /** Full data export of everything the account owns. */
    const val PROFILE_EXPORT_PATH = "$V1/profile/export"

    /** `DELETE` — irreversibly removes the account and all its data. */
    const val ACCOUNT_PATH = "$V1/account"

    // --- Home / dashboard ----------------------------------------------------

    /** Server-driven Home feed: ordered `sections[]` for the current mode. */
    const val HOME_PATH = "$V1/home"

    /** Toggle one Home checklist task: `POST /home/tasks/{id}/toggle`. */
    fun homeTaskTogglePath(taskId: Long): String = "$V1/home/tasks/$taskId/toggle"

    /** Toggle the Home challenge item: `POST /home/challenges/{id}/toggle`. */
    fun homeChallengeTogglePath(challengeId: Long): String = "$V1/home/challenges/$challengeId/toggle"

    /** In-app notifications list (`?per_page=`). */
    const val NOTIFICATIONS_PATH = "$V1/home/notifications"

    /** Mark a single notification read. */
    fun notificationReadPath(notificationId: Long): String = "$V1/home/notifications/$notificationId/read"

    /** Mark every notification read. */
    const val NOTIFICATIONS_READ_ALL_PATH = "$V1/home/notifications/read-all"

    /** Calendar-scheduled promo banners grouped by slot (`home_top|home_middle|home_bottom`). */
    const val BANNERS_PATH = "$V1/banners"

    // --- Cycle tracking -------------------------------------------------------

    /** Today's freshly-computed cycle calculation for the Home hero. */
    const val CYCLE_TODAY_PATH = "$V1/cycle/today"

    /** Cycle calculation for one specific day: `GET /cycle/date/{YYYY-MM-DD}`. */
    fun cycleDatePath(isoDate: String): String = "$V1/cycle/date/$isoDate"

    /** Whole-month calculations + summary for the calendar screen. */
    fun cycleMonthPath(year: Int, month: Int): String = "$V1/cycle/month/$year/$month"

    /** Records that a period started on a date (`{date}`); re-anchors the cycle. */
    const val PERIOD_START_PATH = "$V1/cycle/period/start"

    /** Closes the ongoing period with an end date (`{date}`). */
    const val PERIOD_END_PATH = "$V1/cycle/period/end"

    /** Every logged period, newest first (`data.periods[]`). */
    const val PERIOD_HISTORY_PATH = "$V1/cycle/period/history"

    /** `PUT` move a logged period's start/end · `DELETE` remove it entirely. */
    fun periodPath(periodId: Long): String = "$V1/cycle/period/$periodId"

    /** Kicks off the async cycle recalculation. */
    const val CYCLE_RECALCULATE_PATH = "$V1/cycle/recalculate"

    /** Recalculation progress (`is_processing`). */
    const val CYCLE_STATUS_PATH = "$V1/cycle/status"

    // --- Daily health logs ------------------------------------------------------

    /** `GET` list (paginated) / `POST` upsert-by-date of daily health logs. */
    const val HEALTH_LOGS_PATH = "$V1/health-logs"

    /** One day's health log: `GET`/`DELETE /health-logs/{YYYY-MM-DD}`. */
    fun healthLogPath(isoDate: String): String = "$V1/health-logs/$isoDate"

    // --- Reminders ---------------------------------------------------------------

    /** `GET` list / `POST` create reminders. */
    const val REMINDERS_PATH = "$V1/reminders"

    /** `PUT` update / `DELETE` one reminder. */
    fun reminderPath(reminderId: Long): String = "$V1/reminders/$reminderId"

    // --- Smart messages -------------------------------------------------------------

    /** The personalized daily message (recommendations + smart tip) for the Home screen. */
    const val MESSAGES_DAILY_PATH = "$V1/messages/daily"

    /** Current tracking mode (`cycle|pregnancy`) + goal/subscription flags. */
    const val MESSAGES_MODE_PATH = "$V1/messages/mode"

    // --- Pregnancy mode ---------------------------------------------------------------

    /** Switch the account into pregnancy mode. */
    const val PREGNANCY_ACTIVATE_PATH = "$PREGNANCY/activate"

    /** Switch back to cycle mode. */
    const val PREGNANCY_DEACTIVATE_PATH = "$PREGNANCY/deactivate"

    /** Submit pregnancy onboarding answers (age source + history). */
    const val PREGNANCY_ONBOARDING_PATH = "$PREGNANCY/onboarding"

    /** `GET`/`PUT` the pregnancy profile. */
    const val PREGNANCY_PROFILE_PATH = "$PREGNANCY/profile"

    /** Gestational age, due date, current week, and risk flags for the tracker. */
    const val PREGNANCY_STATUS_PATH = "$PREGNANCY/status"

    /** `GET` list / `POST` upsert-by-date of daily pregnancy symptom logs. */
    const val PREGNANCY_SYMPTOMS_PATH = "$PREGNANCY/symptoms"

    /** One day's pregnancy symptom log: `GET`/`DELETE`. */
    fun pregnancySymptomPath(isoDate: String): String = "$PREGNANCY/symptoms/$isoDate"

    /** `GET` list / `POST` upsert-by-week of weekly monitoring logs. */
    const val PREGNANCY_WEEKLY_PATH = "$PREGNANCY/weekly"

    /** `GET` list / `POST` upsert-by-date of fetal-movement logs. */
    const val PREGNANCY_FETAL_MOVEMENT_PATH = "$PREGNANCY/fetal-movement"

    /** Localized week-by-week educational content: `GET /pregnancy/content/{1..40}`. */
    fun pregnancyContentPath(week: Int): String = "$PREGNANCY/content/$week"

    /** All safety alerts raised from logged symptoms. */
    const val PREGNANCY_ALERTS_PATH = "$PREGNANCY/alerts"

    /** Marks one alert read. */
    fun pregnancyAlertReadPath(alertId: Long): String = "$PREGNANCY/alerts/$alertId/read"

    /** Dismisses one alert. */
    fun pregnancyAlertDismissPath(alertId: Long): String = "$PREGNANCY/alerts/$alertId/dismiss"

    /** Marks every alert read. */
    const val PREGNANCY_ALERTS_READ_ALL_PATH = "$PREGNANCY/alerts/read-all"

    // --- Diagnostics --------------------------------------------------------------------

    /** Diagnostics sink for hand-built crash / non-fatal reports (CLAUDE.md §7.4). */
    const val CRASH_REPORTS_PATH = "/api/diagnostics/crash-reports"
}

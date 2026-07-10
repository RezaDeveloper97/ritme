package ir.ritmeapp.ritme.adapter.outbound.network

/**
 * Centralized backend host + endpoint paths — the one place to change hosts. [BASE_URL] is
 * the Ritme production origin; every path below is absolute from it so a single
 * [ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient] serves auth and diagnostics.
 */
object ApiConfig {
    // Ritme production origin (Laravel backend). Served over plain HTTP for now — the
    // server has no TLS/443 yet, so HTTPS fails to connect. Cleartext to this host is
    // opted in via res/xml/network_security_config.xml. Move back to https once TLS is live.
    // For local dev point this at http://10.0.2.2:8010 (emulator → host :8010, test OTP 1111).
    const val BASE_URL = "http://ritmeapp.ir"

    // Ritme API is versioned under /api/v1 (backend/routes/api.php).
    private const val AUTH = "/api/v1/auth"

    /** Step 1: SMS a one-time code to a mobile number (`{mobile, is_test}`). */
    const val SEND_OTP_PATH = "$AUTH/send-otp"

    /** Step 2: exchange the one-time code for an access token (`{mobile, code}`). */
    const val VERIFY_OTP_PATH = "$AUTH/verify-otp"

    /** Revokes the current access token. */
    const val LOGOUT_PATH = "$AUTH/logout"

    /** The authenticated user's profile. */
    const val USER_PATH = "$AUTH/user"

    /** Diagnostics sink for hand-built crash / non-fatal reports (CLAUDE.md §7.4). */
    const val CRASH_REPORTS_PATH = "/api/diagnostics/crash-reports"
}

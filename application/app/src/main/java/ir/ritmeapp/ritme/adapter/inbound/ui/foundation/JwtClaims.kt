package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import android.util.Base64
import org.json.JSONObject

/**
 * Reads display-only claims out of a JWT access token without verifying its signature —
 * this is purely to greet the user by name, never for authorization decisions (the server
 * enforces those). Returns `null` for anything it cannot decode, so callers degrade to a
 * generic greeting rather than crashing on an unexpected token shape.
 */
object JwtClaims {

    /** The user's first name from the token payload, or `null` if unavailable. */
    fun firstName(accessToken: String): String? = payload(accessToken)
        ?.optString("firstName")
        ?.takeIf { it.isNotBlank() }

    private fun payload(token: String): JSONObject? = try {
        val segment = token.split('.').getOrNull(1) ?: return null
        val decoded = Base64.decode(segment, Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP)
        JSONObject(String(decoded, Charsets.UTF_8))
    } catch (e: Exception) {
        null
    }
}

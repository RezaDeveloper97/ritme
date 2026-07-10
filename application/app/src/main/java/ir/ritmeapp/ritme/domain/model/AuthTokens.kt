package ir.ritmeapp.ritme.domain.model

/**
 * The bearer token the Ritme backend issues on a successful OTP verification
 * (`POST /api/v1/auth/verify-otp`). The [accessToken] authorizes every subsequent API call
 * via the `Authorization: Bearer …` header. Ritme's Passport setup issues no refresh token —
 * re-authentication is simply a fresh OTP — so this is deliberately a single-field value
 * object. Persistence is the [ir.ritmeapp.ritme.domain.port.outbound.TokenStore]'s job.
 */
data class AuthTokens(
    val accessToken: String,
)

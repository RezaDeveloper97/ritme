package ir.ritmeapp.ritme.domain.model

/**
 * An Iranian mobile number in canonical national form (`09XXXXXXXXX`).
 *
 * Construction goes only through [parse], so an instance of this type is *always* valid —
 * downstream code never has to re-check. Parsing tolerates the messy ways users actually
 * type numbers (Persian/Arabic digits, spaces/dashes, `+98`, `0098`, `98`, bare `9...`)
 * and normalizes them to a single representation.
 */
class PhoneNumber private constructor(
    /** Canonical national form, always matching `^09\d{9}$`. */
    val national: String,
) {
    /** E.164 form (`+989XXXXXXXX`), the shape most backends expect. */
    val e164: String get() = "+98" + national.substring(1)

    override fun equals(other: Any?): Boolean = other is PhoneNumber && other.national == national
    override fun hashCode(): Int = national.hashCode()
    override fun toString(): String = national

    companion object {
        private val NATIONAL_PATTERN = Regex("^09\\d{9}$")

        /** Convenience used by the UI to drive a field's "valid" state without allocating. */
        fun isValid(raw: String): Boolean = parse(raw) is AppResult.Success

        /**
         * Parses arbitrary user input into a [PhoneNumber], or a
         * [AppError.Validation] if it is not a recognizable Iranian mobile number.
         */
        fun parse(raw: String): AppResult<PhoneNumber> {
            val latin = raw.trim().map(::foldDigit).joinToString("")
            val digitsAndPlus = latin.filter { it.isDigit() || it == '+' }
            var s = digitsAndPlus.removePrefix("+")
            s = when {
                s.startsWith("0098") -> "0" + s.removePrefix("0098")
                s.startsWith("98") && s.length == 12 -> "0" + s.substring(2)
                s.startsWith("9") && s.length == 10 -> "0$s"
                else -> s
            }
            return if (NATIONAL_PATTERN.matches(s)) {
                AppResult.Success(PhoneNumber(s))
            } else {
                AppResult.Failure(AppError.Validation("INVALID_PHONE_NUMBER"))
            }
        }

        /** Folds Persian (۰–۹) and Arabic-Indic (٠–٩) digits to ASCII; passes others through. */
        private fun foldDigit(c: Char): Char = when (c) {
            in '۰'..'۹' -> '0' + (c - '۰') // Persian
            in '٠'..'٩' -> '0' + (c - '٠') // Arabic-Indic
            else -> c
        }
    }
}

package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.input.OffsetMapping
import androidx.compose.ui.text.input.TransformedText
import androidx.compose.ui.text.input.VisualTransformation

private const val PERSIAN_ZERO = '۰'

/** Converts ASCII digits in this string to Persian digits (۰۱۲۳), leaving other chars intact. */
fun String.toPersianDigits(): String =
    map { c -> if (c in '0'..'9') PERSIAN_ZERO + (c - '0') else c }.joinToString("")

/** Convenience for numbers shown to the user. */
fun Int.toPersianDigits(): String = toString().toPersianDigits()

/**
 * Converts Persian (۰–۹) and Arabic-Indic (٠–٩) digits in this string to ASCII `0`–`9`,
 * leaving every other character intact. Use at input boundaries — an OTP code or phone
 * number a user types on a Persian keyboard must be normalized to ASCII before it is sent
 * to the backend, which only ever speaks ASCII digits.
 */
fun String.toAsciiDigits(): String = map { c ->
    when (c) {
        in '۰'..'۹' -> '0' + (c - '۰') // Persian, U+06F0..U+06F9
        in '٠'..'٩' -> '0' + (c - '٠') // Arabic-Indic, U+0660..U+0669
        else -> c
    }
}.joinToString("")

/**
 * Displays ASCII digits as Persian inside a text field without altering the stored value.
 * The substitution is strictly 1:1 in length, so cursor offsets map identically.
 */
object PersianDigitsTransformation : VisualTransformation {
    override fun filter(text: AnnotatedString): TransformedText =
        TransformedText(AnnotatedString(text.text.toPersianDigits()), OffsetMapping.Identity)
}

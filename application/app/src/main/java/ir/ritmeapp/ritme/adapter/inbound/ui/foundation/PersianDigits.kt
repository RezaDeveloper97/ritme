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
 * Displays ASCII digits as Persian inside a text field without altering the stored value.
 * The substitution is strictly 1:1 in length, so cursor offsets map identically.
 */
object PersianDigitsTransformation : VisualTransformation {
    override fun filter(text: AnnotatedString): TransformedText =
        TransformedText(AnnotatedString(text.text.toPersianDigits()), OffsetMapping.Identity)
}

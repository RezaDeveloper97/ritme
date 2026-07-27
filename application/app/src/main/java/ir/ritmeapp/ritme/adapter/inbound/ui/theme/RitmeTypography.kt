package ir.ritmeapp.ritme.adapter.inbound.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import ir.ritmeapp.ritme.R

/**
 * The app's default font family: Vazirmatn, bundled directly under `res/font/`
 * (CLAUDE.md §5c — ship the font files, no font library). Real weight files back each
 * [FontWeight] the type scale uses, so no synthetic bolding is needed.
 */
val RitmeFontFamily: FontFamily = FontFamily(
    Font(R.font.vazirmatn_regular, FontWeight.Normal),
    Font(R.font.vazirmatn_medium, FontWeight.Medium),
    Font(R.font.vazirmatn_semibold, FontWeight.SemiBold),
    Font(R.font.vazirmatn_bold, FontWeight.Bold),
)

/**
 * A compact, brand-tuned type scale built on [RitmeFontFamily]. EVERY Material3 text style
 * is defined here so that no `Text` — whatever style it names — can ever fall back to the
 * system font (the web renders 100% of copy in Vazirmatn; this keeps Android identical).
 * Sizes mirror the web tokens in `globals.css` (`.titr` 20/800, `.sub` 13, tab labels 11…).
 */
val RitmeTypography = Typography(
    displayLarge = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Bold, fontSize = 40.sp, lineHeight = 48.sp),
    displayMedium = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Bold, fontSize = 34.sp, lineHeight = 42.sp),
    displaySmall = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Bold, fontSize = 30.sp, lineHeight = 38.sp),
    headlineLarge = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Bold, fontSize = 28.sp, lineHeight = 36.sp),
    headlineMedium = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Bold, fontSize = 24.sp, lineHeight = 30.sp),
    // Web `.titr`: 20px / 800 — the standard screen-title size used by StepHeader & screens.
    headlineSmall = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Bold, fontSize = 20.sp, lineHeight = 28.sp),
    titleLarge = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.SemiBold, fontSize = 22.sp, lineHeight = 28.sp),
    titleMedium = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.SemiBold, fontSize = 18.sp, lineHeight = 24.sp),
    titleSmall = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.SemiBold, fontSize = 15.sp, lineHeight = 20.sp),
    bodyLarge = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Normal, fontSize = 16.sp, lineHeight = 24.sp),
    bodyMedium = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Normal, fontSize = 14.sp, lineHeight = 20.sp),
    // Web `.sub`: 13px muted body copy.
    bodySmall = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Normal, fontSize = 13.sp, lineHeight = 22.sp),
    labelLarge = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.SemiBold, fontSize = 16.sp, lineHeight = 20.sp),
    labelMedium = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Medium, fontSize = 13.sp, lineHeight = 16.sp),
    // Web tab labels / captions: 11px.
    labelSmall = TextStyle(fontFamily = RitmeFontFamily, fontWeight = FontWeight.Medium, fontSize = 11.sp, lineHeight = 14.sp),
)

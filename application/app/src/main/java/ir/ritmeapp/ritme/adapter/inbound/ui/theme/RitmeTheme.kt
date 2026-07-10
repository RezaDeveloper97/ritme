package ir.ritmeapp.ritme.adapter.inbound.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.res.colorResource
import androidx.compose.ui.unit.LayoutDirection
import ir.ritmeapp.ritme.R

/**
 * The one place that turns brand tokens into a usable theme (CLAUDE.md §5c): it resolves
 * every color from resources (so day/night is automatic and no hex lives in Kotlin),
 * exposes them through [LocalRitmeColors], maps the relevant ones onto a Material3
 * [androidx.compose.material3.ColorScheme] so stock components are themed, sets the Persian
 * typography + shapes, and forces an RTL layout direction app-wide.
 */
@Composable
fun RitmeTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    val colors = RitmeColors(
        pink = colorResource(R.color.ritme_pink),
        pinkDark = colorResource(R.color.ritme_pink_dark),
        pinkLight = colorResource(R.color.ritme_pink_light),
        pinkContainer = colorResource(R.color.ritme_pink_container),
        accent = colorResource(R.color.ritme_accent),
        ink = colorResource(R.color.ritme_ink),
        inkMuted = colorResource(R.color.ritme_ink_muted),
        surface = colorResource(R.color.ritme_surface),
        background = colorResource(R.color.ritme_background),
        outline = colorResource(R.color.ritme_outline),
        success = colorResource(R.color.ritme_success),
        warning = colorResource(R.color.ritme_warning),
        error = colorResource(R.color.ritme_error),
        info = colorResource(R.color.ritme_info),
        periodContainer = colorResource(R.color.ritme_period_container),
        fertileContainer = colorResource(R.color.ritme_fertile_container),
        ovulationContainer = colorResource(R.color.ritme_ovulation_container),
        violetContainer = colorResource(R.color.ritme_violet_container),
        onPink = colorResource(R.color.white),
        onAccent = colorResource(R.color.ritme_on_accent),
    )

    val base = if (darkTheme) darkColorScheme() else lightColorScheme()
    val colorScheme = base.copy(
        primary = colors.pink,
        onPrimary = colors.onPink,
        primaryContainer = colors.pinkContainer,
        onPrimaryContainer = colors.ink,
        secondary = colors.accent,
        onSecondary = colors.onAccent,
        background = colors.background,
        onBackground = colors.ink,
        surface = colors.surface,
        onSurface = colors.ink,
        surfaceVariant = colors.pinkContainer,
        onSurfaceVariant = colors.inkMuted,
        outline = colors.outline,
        error = colors.error,
        onError = colors.onPink,
    )

    CompositionLocalProvider(
        LocalRitmeColors provides colors,
        LocalLayoutDirection provides LayoutDirection.Rtl,
    ) {
        MaterialTheme(
            colorScheme = colorScheme,
            typography = RitmeTypography,
            shapes = RitmeShapes,
            content = content,
        )
    }
}

package ir.ritmeapp.ritme.adapter.inbound.ui.theme

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

/**
 * The full set of Ritme brand tokens, resolved from `colors.xml` / `values-night`
 * (the single source of truth, CLAUDE.md §5c). Screens read these via [LocalRitmeColors]
 * instead of hardcoding any hex. Immutable so Compose can skip recomposition reliably.
 */
@Immutable
data class RitmeColors(
    val pink: Color,
    val pinkDark: Color,
    val pinkLight: Color,
    val pinkContainer: Color,
    val accent: Color,
    val ink: Color,
    val inkMuted: Color,
    val surface: Color,
    val background: Color,
    val outline: Color,
    val success: Color,
    val warning: Color,
    val error: Color,
    val info: Color,
    /** White content drawn on top of brand pink (buttons, headers). */
    val onPink: Color,
    /** Dark content drawn on top of the light accent accent; constant across themes. */
    val onAccent: Color,
)

/** Provided by [RitmeTheme]; reading it outside the theme is a programming error. */
val LocalRitmeColors = staticCompositionLocalOf<RitmeColors> {
    error("RitmeColors not provided — wrap content in RitmeTheme")
}

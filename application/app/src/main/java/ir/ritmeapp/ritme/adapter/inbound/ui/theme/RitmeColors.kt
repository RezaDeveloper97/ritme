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
    /** Soft surfaces behind cycle-phase markers (calendar days, phase chips). */
    val periodContainer: Color,
    val fertileContainer: Color,
    val ovulationContainer: Color,
    val violetContainer: Color,
    /** White content drawn on top of brand pink (buttons, headers). */
    val onPink: Color,
    /** Dark content drawn on top of the light accent accent; constant across themes. */
    val onAccent: Color,
    // ── Extended tokens mirroring specific web gradients / phase colors ──
    /** Orchid gradient end (#BA68C8): splash/hero/FAB/active-tab underline. */
    val violetGrad: Color,
    /** Home full-page backdrop gradient stops (pink → mint, web `.home-grad`). */
    val homeGradStart: Color,
    val homeGradEnd: Color,
    /** Active-tab background lavender end (web `.tab.on`). */
    val tabActiveEnd: Color,
    /** Steel used for inactive tab icons / soft-button labels (web #58636E). */
    val steel: Color,
    /** Ovulation / recommendation green accent (web --green-dot #34C77B). */
    val greenDot: Color,
    /** Deeper pink for the home hero phase-explanation box (web #F96C9C). */
    val phasePink: Color,
    /** Ovulation tile background (web #F0FDFA). */
    val mintContainer: Color,
    /** Today-status "normal" chip (web #D0FBFF bg / #478F96 ink). */
    val cyanContainer: Color,
    val cyanInk: Color,
    /** Ruler-picker tick marks (web `.tk .line` #D5DBE2 minor / #9AA4AF major). */
    val tickMinor: Color,
    val tickMajor: Color,
    /** Primary-button 1px border (web .btn-primary border #FE87B0). */
    val pinkBorder: Color,
    /** Primary-button disabled fill (web .btn-primary:disabled #F4D4E1). */
    val pinkDisabled: Color,
    /** Ghost-button border (web .btn-ghost border #FAD5E6). */
    val pinkGhostBorder: Color,
    /** Text-field placeholder (web input::placeholder #A9B2BC). */
    val placeholder: Color,
)

/** Provided by [RitmeTheme]; reading it outside the theme is a programming error. */
val LocalRitmeColors = staticCompositionLocalOf<RitmeColors> {
    error("RitmeColors not provided — wrap content in RitmeTheme")
}

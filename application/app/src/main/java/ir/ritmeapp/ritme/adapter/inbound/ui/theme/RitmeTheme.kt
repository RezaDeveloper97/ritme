package ir.ritmeapp.ritme.adapter.inbound.ui.theme

import android.content.Context
import android.content.res.Configuration
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.res.colorResource
import androidx.compose.ui.unit.LayoutDirection
import ir.ritmeapp.ritme.R

/**
 * The one place that turns brand tokens into a usable theme (CLAUDE.md §5c): it resolves
 * every color from resources, exposes them through [LocalRitmeColors], maps the relevant
 * ones onto a Material3 [androidx.compose.material3.ColorScheme] so stock components are
 * themed, sets the Persian typography + shapes, and forces an RTL layout direction app-wide.
 *
 * **Light-only, to mirror the web 1:1.** The web app is deliberately light-only right now —
 * `frontend/src/shared/theme/store.ts` resolves every appearance preference to
 * `data-theme="light"` and keeps the dark machinery dormant "so it can be re-enabled later".
 * Android matches that exactly: colors are resolved from a **night-disabled configuration
 * context**, so the palette stays light even on a device set to system dark mode. The
 * `values-night/colors.xml` tokens are kept in the project as the same dormant machinery,
 * ready for the day the web turns dark mode back on — at which point this can read
 * `isSystemInDarkTheme()` again.
 */
@Composable
fun RitmeTheme(
    content: @Composable () -> Unit,
) {
    val context = LocalContext.current
    val lightContext = remember(context) { context.withNightModeDisabled() }

    CompositionLocalProvider(LocalContext provides lightContext) {
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
            violetGrad = colorResource(R.color.ritme_violet_grad),
            homeGradStart = colorResource(R.color.ritme_home_grad_start),
            homeGradEnd = colorResource(R.color.ritme_home_grad_end),
            tabActiveEnd = colorResource(R.color.ritme_tab_active_end),
            steel = colorResource(R.color.ritme_steel),
            greenDot = colorResource(R.color.ritme_green_dot),
            phasePink = colorResource(R.color.ritme_phase_pink),
            mintContainer = colorResource(R.color.ritme_mint_container),
            cyanContainer = colorResource(R.color.ritme_cyan_container),
            cyanInk = colorResource(R.color.ritme_cyan_ink),
            tickMinor = colorResource(R.color.ritme_tick_minor),
            tickMajor = colorResource(R.color.ritme_tick_major),
            pinkBorder = colorResource(R.color.ritme_pink_border),
            pinkDisabled = colorResource(R.color.ritme_pink_disabled),
            pinkGhostBorder = colorResource(R.color.ritme_pink_ghost_border),
            placeholder = colorResource(R.color.ritme_placeholder),
        )

        val colorScheme = lightColorScheme().copy(
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
}

/**
 * Returns a context whose configuration has the night UI-mode bit cleared, so
 * `colorResource`/resource lookups resolve the light (`values/`) qualifier regardless of the
 * device's dark setting. This is how the app stays light-only in lockstep with the web.
 */
private fun Context.withNightModeDisabled(): Context {
    val configuration = Configuration(resources.configuration).apply {
        uiMode = (uiMode and Configuration.UI_MODE_NIGHT_MASK.inv()) or Configuration.UI_MODE_NIGHT_NO
    }
    return createConfigurationContext(configuration)
}

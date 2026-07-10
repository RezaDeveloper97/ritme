package ir.ritmeapp.ritme.adapter.inbound.ui.navigation

import androidx.compose.runtime.staticCompositionLocalOf
import ir.ritmeapp.ritme.adapter.outbound.di.AppContainer

/**
 * Exposes the composition-root [AppContainer] to the UI tree so screens can obtain their
 * ViewModel factories and shared infrastructure (e.g. the safe-screen tracker) without
 * any global singletons. Provided once at the app root.
 */
val LocalAppContainer = staticCompositionLocalOf<AppContainer> {
    error("AppContainer not provided — wrap content in RitmeApp")
}

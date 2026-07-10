package ir.ritmeapp.ritme

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import ir.ritmeapp.ritme.adapter.inbound.ui.RitmeApp
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeTheme
import ir.ritmeapp.ritme.platform.RitmeApplication
import ir.ritmeapp.ritme.platform.crash.CrashRecovery

/**
 * The single entry Activity. Resolves whether this launch is a crash recovery (§7.3),
 * consumes the recovery flag, and hands control to Compose. All app wiring lives in the
 * [RitmeApplication]'s [ir.ritmeapp.ritme.adapter.outbound.di.AppContainer].
 */
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        val container = (application as RitmeApplication).container
        val recovery = CrashRecovery.resolve(intent, filesDir)
        CrashRecovery.clearFlag(filesDir)

        setContent {
            RitmeTheme {
                RitmeApp(container = container, recovery = recovery)
            }
        }
    }
}

package ir.ritmeapp.ritme.platform

import android.app.Application
import ir.ritmeapp.ritme.BuildConfig
import ir.ritmeapp.ritme.adapter.outbound.di.AppContainer
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import ir.ritmeapp.ritme.platform.log.RitmeLog

/**
 * Application entry point and owner of the composition root. `onCreate` stays intentionally
 * minimal for fast cold start (§5): build the container, arm the Crash Guard *first* so it
 * protects everything after it, then kick deferred startup work off the main thread.
 */
class RitmeApplication : Application() {

    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        RitmeLog.debugEnabled = BuildConfig.DEBUG
        container = AppContainer(this)
        container.crashGuard.install()
        container.appStartupCoordinator.onAppStart()
        Breadcrumbs.add("app:start")
    }
}

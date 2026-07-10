package ir.ritmeapp.ritme.adapter.outbound.di

import android.app.Application
import android.content.Context
import android.os.Build
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import ir.ritmeapp.ritme.MainActivity
import ir.ritmeapp.ritme.adapter.inbound.ui.home.HomeViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.login.LoginViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.otp.OtpViewModel
import ir.ritmeapp.ritme.adapter.outbound.image.ImageLoader
import ir.ritmeapp.ritme.adapter.outbound.image.RitmeImageLoader
import ir.ritmeapp.ritme.adapter.outbound.network.ApiConfig
import ir.ritmeapp.ritme.adapter.outbound.network.AuthGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.CrashReportUploader
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpResponseCache
import ir.ritmeapp.ritme.adapter.outbound.persistence.RitmeSqliteHelper
import ir.ritmeapp.ritme.adapter.outbound.persistence.SharedPrefsTokenStore
import ir.ritmeapp.ritme.adapter.outbound.persistence.SqliteSafeStateRepository
import ir.ritmeapp.ritme.application.service.AppStartupCoordinator
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.domain.port.inbound.GetHomeServicesUseCase
import ir.ritmeapp.ritme.domain.port.inbound.SendOtpUseCase
import ir.ritmeapp.ritme.domain.port.inbound.VerifyOtpUseCase
import ir.ritmeapp.ritme.domain.port.outbound.AuthGateway
import ir.ritmeapp.ritme.domain.port.outbound.DiagnosticsReportUploader
import ir.ritmeapp.ritme.domain.port.outbound.SafeStateRepository
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.domain.usecase.HomeServicesProvider
import ir.ritmeapp.ritme.domain.usecase.SendOtpInteractor
import ir.ritmeapp.ritme.domain.usecase.VerifyOtpInteractor
import ir.ritmeapp.ritme.platform.crash.CrashGuard
import ir.ritmeapp.ritme.platform.crash.CrashRecovery
import ir.ritmeapp.ritme.platform.crash.CrashReportStore
import ir.ritmeapp.ritme.platform.crash.SafeScreenTracker
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import java.io.File

/**
 * The single composition root (CLAUDE.md §3 — manual DI, no Hilt/Koin/Dagger). Every
 * concrete adapter is constructed exactly once here and exposed to the rest of the app
 * only through its port interface. No other class instantiates a cross-layer dependency.
 */
class AppContainer(application: Application) {

    private val appContext: Context = application.applicationContext
    private val appScope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val versionInfo: VersionInfo = readVersionInfo(appContext)
    private val reportsDir = CrashRecovery.reportsDir(appContext.filesDir)

    // --- Outbound adapters (exposed as ports) ------------------------------
    private val httpResponseCache = HttpResponseCache(
        diskDir = File(appContext.cacheDir, "http_responses"),
    )
    private val httpClient = HttpClient(
        baseUrl = ApiConfig.BASE_URL,
        responseCache = httpResponseCache,
    )
    private val authGateway: AuthGateway = AuthGatewayAdapter(httpClient)
    private val tokenStore: TokenStore = SharedPrefsTokenStore(appContext)
    private val diagnosticsUploader: DiagnosticsReportUploader = CrashReportUploader(httpClient, reportsDir)

    /** Hand-written image loader (§3 — no Coil/Glide); UI reads it via LocalAppContainer. */
    val imageLoader: ImageLoader = RitmeImageLoader(
        context = appContext,
        diskDir = File(appContext.cacheDir, "images"),
    )

    private val sqliteHelper = RitmeSqliteHelper(appContext)
    private val safeStateRepository: SafeStateRepository = SqliteSafeStateRepository(sqliteHelper)

    // --- Domain use cases --------------------------------------------------
    private val sendOtpUseCase: SendOtpUseCase = SendOtpInteractor(authGateway)
    private val verifyOtpUseCase: VerifyOtpUseCase = VerifyOtpInteractor(authGateway, tokenStore)
    private val getHomeServicesUseCase: GetHomeServicesUseCase = HomeServicesProvider()

    // --- Crash resilience --------------------------------------------------
    val safeScreenTracker = SafeScreenTracker(safeStateRepository, appScope)
    private val crashReportStore = CrashReportStore(reportsDir)
    val crashGuard = CrashGuard(
        appContext = appContext,
        store = crashReportStore,
        currentSafeScreen = { safeScreenTracker.current },
        appVersionName = versionInfo.name,
        appVersionCode = versionInfo.code,
        entryActivityClass = MainActivity::class.java,
    )

    // --- Application services ----------------------------------------------
    val appStartupCoordinator = AppStartupCoordinator(diagnosticsUploader, appScope)

    // --- ViewModel factories ----------------------------------------------
    fun loginViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { LoginViewModel(sendOtpUseCase) }
    }

    fun otpViewModelFactory(mobile: String): ViewModelProvider.Factory = viewModelFactory {
        initializer {
            val phone = (PhoneNumber.parse(mobile) as? AppResult.Success)?.value
                ?: error("OTP screen requires a valid mobile number, got: $mobile")
            OtpViewModel(phone, verifyOtpUseCase, sendOtpUseCase)
        }
    }

    fun homeViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { HomeViewModel(getHomeServicesUseCase, tokenStore) }
    }

    private data class VersionInfo(val name: String, val code: Long)

    private fun readVersionInfo(context: Context): VersionInfo = try {
        val info = context.packageManager.getPackageInfo(context.packageName, 0)
        val code = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            info.longVersionCode
        } else {
            @Suppress("DEPRECATION") info.versionCode.toLong()
        }
        VersionInfo(info.versionName ?: "unknown", code)
    } catch (e: Exception) {
        VersionInfo("unknown", -1L)
    }
}

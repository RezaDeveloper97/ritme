package ir.ritmeapp.ritme.adapter.outbound.di

import android.app.Application
import android.content.Context
import android.os.Build
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import ir.ritmeapp.ritme.MainActivity
import ir.ritmeapp.ritme.adapter.inbound.ui.calendar.CalendarViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.cycle.CycleViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.home.HomeViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.log.LogViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.login.LoginViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.onboarding.OnboardingViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.PregnancyLogTab
import ir.ritmeapp.ritme.adapter.inbound.ui.otp.OtpViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy.PregnancyLogViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy.PregnancyOnboardingViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy.PregnancyViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.NotificationsViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.ProfileHealthViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.ProfilePersonalViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.ProfileViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.profile.RemindersViewModel
import ir.ritmeapp.ritme.adapter.inbound.ui.welcome.WelcomeViewModel
import ir.ritmeapp.ritme.adapter.outbound.image.ImageLoader
import ir.ritmeapp.ritme.adapter.outbound.image.RitmeImageLoader
import ir.ritmeapp.ritme.adapter.outbound.network.ApiConfig
import ir.ritmeapp.ritme.adapter.outbound.network.AuthGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.BannerGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.CrashReportUploader
import ir.ritmeapp.ritme.adapter.outbound.network.CycleGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.PeriodLogGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.HealthLogGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.MessageGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.ModeGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.NotificationGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.PregnancyGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.ProfileGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.ReminderGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.SessionGatewayAdapter
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpResponseCache
import ir.ritmeapp.ritme.adapter.outbound.persistence.RitmeSqliteHelper
import ir.ritmeapp.ritme.adapter.outbound.persistence.SharedPrefsAppPreferences
import ir.ritmeapp.ritme.adapter.outbound.persistence.SharedPrefsTokenStore
import ir.ritmeapp.ritme.adapter.outbound.persistence.SqliteSafeStateRepository
import ir.ritmeapp.ritme.application.service.AppStartupCoordinator
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.domain.port.inbound.CycleInsightsUseCase
import ir.ritmeapp.ritme.domain.port.inbound.DeleteAccountUseCase
import ir.ritmeapp.ritme.domain.port.inbound.ExportDataUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetBannersUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetHomeDashboardUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetTrackingModeUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetUserProfileUseCase
import ir.ritmeapp.ritme.domain.port.inbound.HealthLogUseCase
import ir.ritmeapp.ritme.domain.port.inbound.IsLoggedInUseCase
import ir.ritmeapp.ritme.domain.port.inbound.LogoutUseCase
import ir.ritmeapp.ritme.domain.port.inbound.ManageRemindersUseCase
import ir.ritmeapp.ritme.domain.port.inbound.MarkIntroSeenUseCase
import ir.ritmeapp.ritme.domain.port.inbound.NotificationsUseCase
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyLogsUseCase
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyTrackerUseCase
import ir.ritmeapp.ritme.domain.port.inbound.SaveProfileUseCase
import ir.ritmeapp.ritme.domain.port.inbound.SendOtpUseCase
import ir.ritmeapp.ritme.domain.port.inbound.ShouldShowIntroUseCase
import ir.ritmeapp.ritme.domain.port.inbound.ManagePeriodsUseCase
import ir.ritmeapp.ritme.domain.port.inbound.StartPeriodUseCase
import ir.ritmeapp.ritme.domain.port.inbound.VerifyOtpUseCase
import ir.ritmeapp.ritme.domain.port.outbound.AppPreferences
import ir.ritmeapp.ritme.domain.port.outbound.AuthGateway
import ir.ritmeapp.ritme.domain.port.outbound.BannerGateway
import ir.ritmeapp.ritme.domain.port.outbound.CycleGateway
import ir.ritmeapp.ritme.domain.port.outbound.PeriodLogGateway
import ir.ritmeapp.ritme.domain.port.outbound.DiagnosticsReportUploader
import ir.ritmeapp.ritme.domain.port.outbound.HealthLogGateway
import ir.ritmeapp.ritme.domain.port.outbound.MessageGateway
import ir.ritmeapp.ritme.domain.port.outbound.ModeGateway
import ir.ritmeapp.ritme.domain.port.outbound.NotificationGateway
import ir.ritmeapp.ritme.domain.port.outbound.PregnancyGateway
import ir.ritmeapp.ritme.domain.port.outbound.ProfileGateway
import ir.ritmeapp.ritme.domain.port.outbound.ReminderGateway
import ir.ritmeapp.ritme.domain.port.outbound.SafeStateRepository
import ir.ritmeapp.ritme.domain.port.outbound.SessionGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.domain.usecase.CycleInsightsInteractor
import ir.ritmeapp.ritme.domain.usecase.DeleteAccountInteractor
import ir.ritmeapp.ritme.domain.usecase.ExportDataInteractor
import ir.ritmeapp.ritme.domain.usecase.GetBannersInteractor
import ir.ritmeapp.ritme.domain.usecase.GetUserProfileInteractor
import ir.ritmeapp.ritme.domain.usecase.HealthLogInteractor
import ir.ritmeapp.ritme.domain.usecase.HomeDashboardInteractor
import ir.ritmeapp.ritme.domain.usecase.LogoutInteractor
import ir.ritmeapp.ritme.domain.usecase.ManageRemindersInteractor
import ir.ritmeapp.ritme.domain.usecase.MarkIntroSeenInteractor
import ir.ritmeapp.ritme.domain.usecase.NotificationsInteractor
import ir.ritmeapp.ritme.domain.usecase.PregnancyLogsInteractor
import ir.ritmeapp.ritme.domain.usecase.PregnancyTrackerInteractor
import ir.ritmeapp.ritme.domain.usecase.SaveProfileInteractor
import ir.ritmeapp.ritme.domain.usecase.SendOtpInteractor
import ir.ritmeapp.ritme.domain.usecase.SessionStatusInteractor
import ir.ritmeapp.ritme.domain.usecase.ShouldShowIntroInteractor
import ir.ritmeapp.ritme.domain.usecase.ManagePeriodsInteractor
import ir.ritmeapp.ritme.domain.usecase.StartPeriodInteractor
import ir.ritmeapp.ritme.domain.usecase.TrackingModeInteractor
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
    private val profileGateway: ProfileGateway = ProfileGatewayAdapter(httpClient, tokenStore)
    private val cycleGateway: CycleGateway = CycleGatewayAdapter(httpClient, tokenStore)
    private val periodLogGateway: PeriodLogGateway = PeriodLogGatewayAdapter(httpClient, tokenStore)
    private val messageGateway: MessageGateway = MessageGatewayAdapter(httpClient, tokenStore)
    private val bannerGateway: BannerGateway = BannerGatewayAdapter(httpClient, tokenStore)
    private val reminderGateway: ReminderGateway = ReminderGatewayAdapter(httpClient, tokenStore)
    private val sessionGateway: SessionGateway = SessionGatewayAdapter(httpClient, tokenStore)
    private val healthLogGateway: HealthLogGateway = HealthLogGatewayAdapter(httpClient, tokenStore)
    private val modeGateway: ModeGateway = ModeGatewayAdapter(httpClient, tokenStore)
    private val notificationGateway: NotificationGateway = NotificationGatewayAdapter(httpClient, tokenStore)
    private val pregnancyGateway: PregnancyGateway = PregnancyGatewayAdapter(httpClient, tokenStore)
    private val appPreferences: AppPreferences = SharedPrefsAppPreferences(appContext)
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
    private val getHomeDashboardUseCase: GetHomeDashboardUseCase = HomeDashboardInteractor(cycleGateway, messageGateway)
    private val saveProfileUseCase: SaveProfileUseCase = SaveProfileInteractor(profileGateway)
    private val markIntroSeenUseCase: MarkIntroSeenUseCase = MarkIntroSeenInteractor(appPreferences)
    private val cycleInsightsUseCase: CycleInsightsUseCase = CycleInsightsInteractor(cycleGateway)
    private val healthLogUseCase: HealthLogUseCase = HealthLogInteractor(healthLogGateway)
    private val getTrackingModeUseCase: GetTrackingModeUseCase = TrackingModeInteractor(modeGateway, appPreferences)
    private val startPeriodUseCase: StartPeriodUseCase = StartPeriodInteractor(periodLogGateway)
    private val managePeriodsUseCase: ManagePeriodsUseCase = ManagePeriodsInteractor(periodLogGateway)
    private val getBannersUseCase: GetBannersUseCase = GetBannersInteractor(bannerGateway)
    private val getUserProfileUseCase: GetUserProfileUseCase = GetUserProfileInteractor(profileGateway)
    private val logoutUseCase: LogoutUseCase = LogoutInteractor(sessionGateway, tokenStore)
    private val deleteAccountUseCase: DeleteAccountUseCase = DeleteAccountInteractor(sessionGateway, tokenStore)
    private val manageRemindersUseCase: ManageRemindersUseCase = ManageRemindersInteractor(reminderGateway)
    private val notificationsUseCase: NotificationsUseCase = NotificationsInteractor(notificationGateway)
    private val pregnancyTrackerUseCase: PregnancyTrackerUseCase = PregnancyTrackerInteractor(pregnancyGateway)
    private val pregnancyLogsUseCase: PregnancyLogsUseCase = PregnancyLogsInteractor(pregnancyGateway)
    private val exportDataUseCase: ExportDataUseCase = ExportDataInteractor(sessionGateway)

    /** Read once at startup to decide whether the app opens on Home or Login. */
    val isLoggedInUseCase: IsLoggedInUseCase = SessionStatusInteractor(tokenStore)

    /** Read once at startup to decide whether a logged-out visitor sees the intro first. */
    val shouldShowIntroUseCase: ShouldShowIntroUseCase = ShouldShowIntroInteractor(appPreferences)

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
    fun welcomeViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { WelcomeViewModel(markIntroSeenUseCase) }
    }

    fun loginViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { LoginViewModel(sendOtpUseCase) }
    }

    fun onboardingViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { OnboardingViewModel(saveProfileUseCase, todayJalali()) }
    }

    fun otpViewModelFactory(mobile: String): ViewModelProvider.Factory = viewModelFactory {
        initializer {
            val phone = (PhoneNumber.parse(mobile) as? AppResult.Success)?.value
                ?: error("OTP screen requires a valid mobile number, got: $mobile")
            OtpViewModel(phone, verifyOtpUseCase, sendOtpUseCase)
        }
    }

    fun homeViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer {
            HomeViewModel(
                getDashboard = getHomeDashboardUseCase,
                getBanners = getBannersUseCase,
                startPeriod = startPeriodUseCase,
                getTrackingMode = getTrackingModeUseCase,
                manageReminders = manageRemindersUseCase,
                tokenStore = tokenStore,
                today = todayJalali(),
            )
        }
    }

    fun logViewModelFactory(initialDateIso: String?): ViewModelProvider.Factory = viewModelFactory {
        initializer { LogViewModel(healthLogUseCase, manageRemindersUseCase, todayJalali(), initialDateIso) }
    }

    fun calendarViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer {
            CalendarViewModel(
                cycleInsightsUseCase,
                healthLogUseCase,
                getTrackingModeUseCase,
                managePeriodsUseCase,
                todayJalali(),
            )
        }
    }

    fun cycleViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { CycleViewModel(getHomeDashboardUseCase, cycleInsightsUseCase, getTrackingModeUseCase) }
    }

    fun profileViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer {
            ProfileViewModel(
                getUserProfile = getUserProfileUseCase,
                getTrackingMode = getTrackingModeUseCase,
                logout = logoutUseCase,
                deleteAccount = deleteAccountUseCase,
                exportData = exportDataUseCase,
                pregnancyTracker = pregnancyTrackerUseCase,
            )
        }
    }

    fun profilePersonalViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { ProfilePersonalViewModel(getUserProfileUseCase, saveProfileUseCase, todayJalali()) }
    }

    fun profileHealthViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { ProfileHealthViewModel(getUserProfileUseCase, saveProfileUseCase, todayJalali()) }
    }

    fun remindersViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { RemindersViewModel(manageRemindersUseCase) }
    }

    fun notificationsViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { NotificationsViewModel(notificationsUseCase) }
    }

    fun pregnancyViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { PregnancyViewModel(pregnancyTrackerUseCase) }
    }

    fun pregnancyOnboardingViewModelFactory(): ViewModelProvider.Factory = viewModelFactory {
        initializer { PregnancyOnboardingViewModel(pregnancyTrackerUseCase, todayJalali()) }
    }

    fun pregnancyLogViewModelFactory(initialTab: PregnancyLogTab): ViewModelProvider.Factory = viewModelFactory {
        initializer {
            PregnancyLogViewModel(pregnancyLogsUseCase, pregnancyTrackerUseCase, todayJalali(), initialTab)
        }
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

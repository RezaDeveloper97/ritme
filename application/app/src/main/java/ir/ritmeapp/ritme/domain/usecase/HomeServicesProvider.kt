package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.HomeService
import ir.ritmeapp.ritme.domain.model.ServiceAccent
import ir.ritmeapp.ritme.domain.port.inbound.GetHomeServicesUseCase

/**
 * Default [GetHomeServicesUseCase]: the curated set of Ritme services shown on the home
 * screen, mirroring what the web app offers (cycle tracking, pregnancy mode, daily logs,
 * calendar, content). Held as pure domain data — no I/O — so it is instant and always
 * available on the home screen. When a remote services endpoint is wired, only this
 * class changes.
 */
class HomeServicesProvider : GetHomeServicesUseCase {

    override fun invoke(): List<HomeService> = CATALOG

    private companion object {
        val CATALOG = listOf(
            HomeService(
                id = "cycle_tracker",
                title = "پیگیری چرخهٔ قاعدگی",
                tagline = "ثبت و پیش‌بینی دوره‌های شما",
                accent = ServiceAccent.Pink,
            ),
            HomeService(
                id = "period_log",
                title = "ثبت پریود",
                tagline = "ثبت روزهای پریود و علائم",
                accent = ServiceAccent.Pink,
            ),
            HomeService(
                id = "pregnancy",
                title = "حالت بارداری",
                tagline = "همراه شما در ۴۰ هفتهٔ بارداری",
                accent = ServiceAccent.Accent,
            ),
            HomeService(
                id = "daily_log",
                title = "یادداشت روزانه",
                tagline = "حال و روز و علائم امروزتان",
                accent = ServiceAccent.Info,
            ),
            HomeService(
                id = "calendar",
                title = "تقویم من",
                tagline = "نمای ماهانهٔ چرخه و روزهای مهم",
                accent = ServiceAccent.Pink,
            ),
            HomeService(
                id = "content",
                title = "مطالب و راهنما",
                tagline = "نکات سلامتی متناسب با شما",
                accent = ServiceAccent.Info,
            ),
        )
    }
}

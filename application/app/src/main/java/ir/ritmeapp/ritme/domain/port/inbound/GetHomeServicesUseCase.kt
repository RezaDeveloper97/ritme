package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.HomeService

/**
 * Inbound port: the list of Ritme services offered on the home screen. Modeled as a
 * use case so the source (today a curated in-app list; tomorrow a remote endpoint)
 * can change without touching the UI.
 */
interface GetHomeServicesUseCase {
    operator fun invoke(): List<HomeService>
}

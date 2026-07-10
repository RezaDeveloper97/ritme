package ir.ritmeapp.ritme.adapter.inbound.ui.home

import ir.ritmeapp.ritme.domain.model.HomeService

/**
 * Immutable snapshot the home screen renders from. [greetingName] is decoded from the login
 * token for a personal welcome (or null → generic greeting); [services] is the home
 * catalog. A plain data class — Compose can skip recomposition when it is unchanged (§5).
 */
data class HomeUiState(
    val greetingName: String? = null,
    val services: List<HomeService> = emptyList(),
)

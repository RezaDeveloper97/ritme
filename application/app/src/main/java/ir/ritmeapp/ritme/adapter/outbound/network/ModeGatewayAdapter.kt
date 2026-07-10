package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.map
import ir.ritmeapp.ritme.domain.model.TrackingMode
import ir.ritmeapp.ritme.domain.port.outbound.ModeGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/** Network [ModeGateway] over `GET /messages/mode`. */
class ModeGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : ModeGateway {

    override suspend fun currentMode(): AppResult<TrackingMode> {
        Breadcrumbs.add("api:mode:start")
        return when (val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.MESSAGES_MODE_PATH)) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:mode:http_${result.value.statusCode}")
                result.value.readEnvelope("Mode request rejected")
                    .map { TrackingMode.fromApi(it.dataObject().stringOrNull("mode")) }
            }
        }
    }
}

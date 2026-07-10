package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpResponse
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import org.json.JSONObject

/**
 * Shared handling of the backend's uniform `{success, message, data}` envelope so every new
 * gateway adapter doesn't re-implement the same status/JSON/`success:false` triage. Returns
 * the whole parsed root (callers usually read `data` from it) or the mapped failure.
 */
internal fun HttpResponse.readEnvelope(rejectionMessage: String): AppResult<JSONObject> = try {
    val root = if (body.isBlank()) JSONObject() else JSONObject(body)
    if (root.optBoolean("success", false)) {
        AppResult.Success(root)
    } else {
        val message = root.stringOrNull("message") ?: rejectionMessage
        AppResult.Failure(AppError.Http(statusCode, message, body))
    }
} catch (e: Exception) {
    AppResult.Failure(AppError.Parsing("Malformed response ($statusCode)", e))
}

/** The envelope's `data` object, or an empty object when the backend sent none. */
internal fun JSONObject.dataObject(): JSONObject = optJSONObject("data") ?: JSONObject()

package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpRequest
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.port.outbound.DiagnosticsReportUploader
import ir.ritmeapp.ritme.platform.log.RitmeLog
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

/**
 * Drains queued crash/error reports (CLAUDE.md §7.4). Scans the reports directory and
 * POSTs each file through the hand-written [HttpClient]. A report file is deleted **only**
 * after a 2xx — so a failed upload simply stays on disk and retries next launch, and a
 * successful one is never sent twice.
 */
class CrashReportUploader(
    private val httpClient: HttpClient,
    private val reportsDir: File,
    private val ioDispatcher: CoroutineDispatcher = Dispatchers.IO,
) : DiagnosticsReportUploader {

    override suspend fun uploadPending() = withContext(ioDispatcher) {
        val files = reportsDir.listFiles { file -> file.isFile && file.name.endsWith(".json") }
            ?: return@withContext
        for (file in files) {
            val body = runCatching { file.readText() }.getOrNull() ?: continue
            val request = HttpRequest(
                method = HttpMethod.POST,
                path = ApiConfig.CRASH_REPORTS_PATH,
                body = body,
            )
            when (val result = httpClient.execute(request)) {
                is AppResult.Success ->
                    if (result.value.isSuccessful) {
                        file.delete()
                    } else {
                        RitmeLog.w(TAG, "Report upload rejected: HTTP ${result.value.statusCode}")
                    }

                is AppResult.Failure ->
                    RitmeLog.w(TAG, "Report upload failed: ${result.error.message}")
            }
        }
    }

    private companion object {
        const val TAG = "CrashReportUploader"
    }
}

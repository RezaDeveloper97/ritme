package ir.ritmeapp.ritme.platform.crash

import java.io.File

/**
 * Persists a [CrashPayload] as one small JSON file under `filesDir/crash_reports/`
 * (CLAUDE.md §7.3). JSON is hand-written — deliberately *not* using any JSON library —
 * because this runs while the process is dying: no reflection, no allocation surprises,
 * just a flat string build. The [CrashReportUploader] later drains these files.
 */
class CrashReportStore(
    private val reportsDir: File,
) {

    /** Writes [payload] to disk, returning the file, or `null` if writing failed. */
    fun writeReport(payload: CrashPayload): File? = try {
        if (!reportsDir.exists()) reportsDir.mkdirs()
        val file = File(reportsDir, "${payload.timestampMillis}.json")
        file.writeText(payload.toJson())
        file
    } catch (t: Throwable) {
        null
    }

    private fun CrashPayload.toJson(): String {
        val sb = StringBuilder(512)
        sb.append('{')
        appendNumber(sb, "timestampMillis", timestampMillis); sb.append(',')
        appendString(sb, "severity", severity); sb.append(',')
        appendString(sb, "appVersionName", appVersionName); sb.append(',')
        appendNumber(sb, "appVersionCode", appVersionCode); sb.append(',')
        appendNumber(sb, "sdkInt", sdkInt.toLong()); sb.append(',')
        appendString(sb, "osVersion", osVersion); sb.append(',')
        appendString(sb, "manufacturer", manufacturer); sb.append(',')
        appendString(sb, "deviceModel", deviceModel); sb.append(',')
        appendNumber(sb, "freeMemoryBytes", freeMemoryBytes); sb.append(',')
        appendString(sb, "threadName", threadName); sb.append(',')
        appendString(sb, "stackTrace", stackTrace); sb.append(',')
        appendString(sb, "safeRoute", safeRoute); sb.append(',')
        appendString(sb, "safeArgs", safeArgs); sb.append(',')
        sb.append("\"breadcrumbs\":[")
        breadcrumbs.forEachIndexed { index, crumb ->
            if (index > 0) sb.append(',')
            sb.append(quote(crumb))
        }
        sb.append(']')
        sb.append('}')
        return sb.toString()
    }

    private fun appendString(sb: StringBuilder, key: String, value: String?) {
        sb.append(quote(key)).append(':').append(quote(value))
    }

    private fun appendNumber(sb: StringBuilder, key: String, value: Long) {
        sb.append(quote(key)).append(':').append(value)
    }

    /** RFC 8259-compliant JSON string escaping; `null` renders as the JSON literal `null`. */
    private fun quote(value: String?): String {
        if (value == null) return "null"
        val sb = StringBuilder(value.length + 2)
        sb.append('"')
        for (c in value) {
            when (c) {
                '"' -> sb.append("\\\"")
                '\\' -> sb.append("\\\\")
                '\n' -> sb.append("\\n")
                '\r' -> sb.append("\\r")
                '\t' -> sb.append("\\t")
                '\b' -> sb.append("\\b")
                '\u000C' -> sb.append("\\f")
                else -> if (c < ' ') sb.append("\\u").append(c.code.toString(16).padStart(4, '0')) else sb.append(c)
            }
        }
        sb.append('"')
        return sb.toString()
    }
}

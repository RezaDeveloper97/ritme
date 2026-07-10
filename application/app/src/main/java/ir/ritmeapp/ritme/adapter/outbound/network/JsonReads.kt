package ir.ritmeapp.ritme.adapter.outbound.network

import org.json.JSONArray
import org.json.JSONObject

/**
 * Small null-aware readers over `org.json` (which ships in the Android SDK — not a third-party add,
 * §3). `org.json`'s `opt*` accessors coerce missing/null to a zero-value, losing the "field absent"
 * distinction these endpoints rely on (an empty calculation sends real nulls); these restore it.
 */
internal fun JSONObject.stringOrNull(key: String): String? =
    if (!has(key) || isNull(key)) null else optString(key).takeIf { it.isNotEmpty() }

internal fun JSONObject.intOrNull(key: String): Int? =
    if (!has(key) || isNull(key)) null else optInt(key)

internal fun JSONObject.doubleOrNull(key: String): Double? =
    if (!has(key) || isNull(key)) null else optDouble(key)

/** Reads a JSON array of strings into a list, or an empty list when absent. */
internal fun JSONObject.stringList(key: String): List<String> {
    val array: JSONArray = optJSONArray(key) ?: return emptyList()
    return (0 until array.length()).mapNotNull { array.optString(it).takeIf { s -> s.isNotEmpty() } }
}

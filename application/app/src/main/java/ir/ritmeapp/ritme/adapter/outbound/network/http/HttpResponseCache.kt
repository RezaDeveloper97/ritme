package ir.ritmeapp.ritme.adapter.outbound.network.http

import java.io.File

/**
 * Hand-written two-tier (memory + disk) response cache for GET-heavy screens
 * (CLAUDE.md §5 — no OkHttp cache). Keyed by the full request URL; each entry carries
 * its own absolute expiry so stale data is never served, only re-fetched.
 *
 * Memory tier is a small access-ordered LRU; disk tier survives process death so a cold
 * start can still paint pregnancy content / banners instantly while fresh data loads.
 * All methods are synchronized — entries are tiny (JSON bodies) and callers are already
 * on Dispatchers.IO, so a simple monitor beats a hand-rolled lock-free structure (§3).
 */
class HttpResponseCache(
    private val diskDir: File?,
    private val maxMemoryEntries: Int = DEFAULT_MAX_MEMORY_ENTRIES,
    private val clock: () -> Long = System::currentTimeMillis,
) {

    private data class Entry(val statusCode: Int, val body: String, val expiresAtMillis: Long)

    private val memory = object : LinkedHashMap<String, Entry>(16, 0.75f, true) {
        override fun removeEldestEntry(eldest: MutableMap.MutableEntry<String, Entry>): Boolean =
            size > maxMemoryEntries
    }

    /** Returns the cached response for [url], or null if absent or expired. */
    @Synchronized
    fun get(url: String): HttpResponse? {
        val now = clock()
        memory[url]?.let { entry ->
            if (entry.expiresAtMillis > now) return entry.toResponse()
            memory.remove(url)
        }
        val entry = readFromDisk(url) ?: return null
        return if (entry.expiresAtMillis > now) {
            memory[url] = entry // promote disk hit into the memory tier
            entry.toResponse()
        } else {
            fileFor(url)?.delete()
            null
        }
    }

    /** Stores a successful response for [url], valid for [ttlMillis] from now. */
    @Synchronized
    fun put(url: String, response: HttpResponse, ttlMillis: Long) {
        if (ttlMillis <= 0) return
        val entry = Entry(response.statusCode, response.body, clock() + ttlMillis)
        memory[url] = entry
        writeToDisk(url, entry)
    }

    /** Drops every entry from both tiers (e.g. on logout). */
    @Synchronized
    fun clear() {
        memory.clear()
        diskDir?.listFiles()?.forEach { it.delete() }
    }

    private fun Entry.toResponse() = HttpResponse(statusCode, emptyMap(), body)

    // Disk format: line 1 = "<expiresAtMillis> <statusCode>", rest = raw body.
    private fun writeToDisk(url: String, entry: Entry) {
        val file = fileFor(url) ?: return
        try {
            file.parentFile?.mkdirs()
            file.writeText("${entry.expiresAtMillis} ${entry.statusCode}\n${entry.body}")
        } catch (_: java.io.IOException) {
            // A failed cache write must never break the request path — just skip caching.
        }
    }

    private fun readFromDisk(url: String): Entry? {
        val file = fileFor(url) ?: return null
        return try {
            if (!file.exists()) return null
            val text = file.readText()
            val newline = text.indexOf('\n')
            if (newline <= 0) return null
            val (expires, status) = text.substring(0, newline).split(' ', limit = 2)
            Entry(status.toInt(), text.substring(newline + 1), expires.toLong())
        } catch (_: Exception) {
            file.delete() // corrupt entry: drop it rather than fail the request
            null
        }
    }

    private fun fileFor(url: String): File? =
        diskDir?.let { File(it, cacheFileName(url)) }

    /** Stable, filesystem-safe name derived from the URL (FNV-1a — no crypto needed). */
    private fun cacheFileName(url: String): String {
        var hash = FNV_OFFSET_BASIS
        for (ch in url) {
            hash = hash xor ch.code.toLong()
            hash *= FNV_PRIME
        }
        return "resp_${hash.toULong().toString(RADIX_HEX)}"
    }

    private companion object {
        const val DEFAULT_MAX_MEMORY_ENTRIES = 32
        const val FNV_OFFSET_BASIS = -0x340d631b7bdddcdbL // 14695981039346656037
        const val FNV_PRIME = 0x100000001b3L
        const val RADIX_HEX = 16
    }
}

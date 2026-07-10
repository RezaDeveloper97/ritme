package ir.ritmeapp.ritme.adapter.outbound.network.http

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import java.io.File

class HttpResponseCacheTest {

    private lateinit var diskDir: File
    private var nowMillis = 1_000_000L
    private val clock: () -> Long = { nowMillis }

    @Before
    fun createDiskDir() {
        diskDir = File.createTempFile("cache_test", "").apply { delete(); mkdirs() }
    }

    private fun response(body: String) = HttpResponse(200, emptyMap(), body)

    @Test
    fun `returns what was stored while the TTL is alive`() {
        val cache = HttpResponseCache(diskDir, clock = clock)
        cache.put("https://x/a", response("""{"v":1}"""), ttlMillis = 5_000)

        assertEquals("""{"v":1}""", cache.get("https://x/a")?.body)
    }

    @Test
    fun `expired entries are not served`() {
        val cache = HttpResponseCache(diskDir, clock = clock)
        cache.put("https://x/a", response("old"), ttlMillis = 5_000)

        nowMillis += 5_001
        assertNull(cache.get("https://x/a"))
    }

    @Test
    fun `disk tier survives a new cache instance (process restart)`() {
        HttpResponseCache(diskDir, clock = clock)
            .put("https://x/banners", response("""{"banners":[1]}"""), ttlMillis = 60_000)

        val rebooted = HttpResponseCache(diskDir, clock = clock)
        assertEquals("""{"banners":[1]}""", rebooted.get("https://x/banners")?.body)
    }

    @Test
    fun `memory tier evicts least-recently-used beyond capacity but disk still serves`() {
        val cache = HttpResponseCache(diskDir, maxMemoryEntries = 2, clock = clock)
        cache.put("u1", response("1"), 60_000)
        cache.put("u2", response("2"), 60_000)
        cache.put("u3", response("3"), 60_000) // evicts u1 from memory

        // All three still resolvable — u1 comes back from disk.
        assertEquals("1", cache.get("u1")?.body)
        assertEquals("2", cache.get("u2")?.body)
        assertEquals("3", cache.get("u3")?.body)
    }

    @Test
    fun `clear drops both tiers`() {
        val cache = HttpResponseCache(diskDir, clock = clock)
        cache.put("u1", response("1"), 60_000)

        cache.clear()

        assertNull(cache.get("u1"))
        assertEquals(0, diskDir.listFiles().orEmpty().size)
    }

    @Test
    fun `zero or negative TTL is never stored`() {
        val cache = HttpResponseCache(diskDir, clock = clock)
        cache.put("u1", response("1"), 0)
        cache.put("u2", response("2"), -5)

        assertNull(cache.get("u1"))
        assertNull(cache.get("u2"))
    }

    @Test
    fun `corrupt disk entry is dropped, not crashed on`() {
        val cache = HttpResponseCache(diskDir, clock = clock)
        cache.put("u1", response("good"), 60_000)
        diskDir.listFiles().orEmpty().forEach { it.writeText("garbage-no-newline") }

        val fresh = HttpResponseCache(diskDir, clock = clock)
        assertNull(fresh.get("u1"))
    }

    @Test
    fun `works memory-only when no disk dir is given`() {
        val cache = HttpResponseCache(diskDir = null, clock = clock)
        cache.put("u1", response("1"), 60_000)

        assertEquals("1", cache.get("u1")?.body)
    }
}

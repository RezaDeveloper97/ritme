package ir.ritmeapp.ritme.adapter.outbound.network.http

import com.sun.net.httpserver.HttpServer
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.File
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicInteger

/**
 * Integration tests for the hand-written HTTP client (CLAUDE.md §9) against a real local
 * server — `com.sun.net.httpserver` ships with the JDK, so no test dependency is added.
 */
class HttpClientTest {

    private lateinit var server: HttpServer
    private lateinit var baseUrl: String

    @Before
    fun startServer() {
        server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.start()
        baseUrl = "http://127.0.0.1:${server.address.port}"
    }

    @After
    fun stopServer() {
        server.stop(0)
    }

    private fun respond(path: String, status: Int, body: String, onHit: () -> Unit = {}) {
        server.createContext(path) { exchange ->
            onHit()
            val bytes = body.toByteArray(Charsets.UTF_8)
            exchange.responseHeaders.add("Content-Type", "application/json")
            exchange.sendResponseHeaders(status, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
    }

    @Test
    fun `GET returns status and body on success`() = runBlocking {
        respond("/api/ping", 200, """{"ok":true}""")
        val client = HttpClient(baseUrl)

        val result = client.execute(HttpRequest(HttpMethod.GET, "/api/ping"))

        val response = (result as AppResult.Success).value
        assertEquals(200, response.statusCode)
        assertEquals("""{"ok":true}""", response.body)
    }

    @Test
    fun `POST sends the JSON body and content type`() = runBlocking {
        var receivedBody = ""
        var receivedContentType = ""
        server.createContext("/api/login") { exchange ->
            receivedBody = exchange.requestBody.readBytes().toString(Charsets.UTF_8)
            receivedContentType = exchange.requestHeaders.getFirst("Content-Type").orEmpty()
            exchange.sendResponseHeaders(200, 2)
            exchange.responseBody.use { it.write("{}".toByteArray()) }
        }
        val client = HttpClient(baseUrl)

        client.execute(HttpRequest(HttpMethod.POST, "/api/login", body = """{"mobile":"0912"}"""))

        assertEquals("""{"mobile":"0912"}""", receivedBody)
        assertTrue(receivedContentType.startsWith("application/json"))
    }

    @Test
    fun `query parameters are URL-encoded`() = runBlocking {
        var receivedQuery = ""
        server.createContext("/api/search") { exchange ->
            receivedQuery = exchange.requestURI.rawQuery.orEmpty()
            exchange.sendResponseHeaders(200, 2)
            exchange.responseBody.use { it.write("{}".toByteArray()) }
        }
        val client = HttpClient(baseUrl)

        client.execute(
            HttpRequest(HttpMethod.GET, "/api/search", query = mapOf("q" to "دوره ماهانه")),
        )

        assertTrue(receivedQuery.startsWith("q=%D8%AF"))
    }

    @Test
    fun `default and per-request headers are sent`() = runBlocking {
        var auth = ""
        var custom = ""
        server.createContext("/api/me") { exchange ->
            auth = exchange.requestHeaders.getFirst("Authorization").orEmpty()
            custom = exchange.requestHeaders.getFirst("X-Ritme").orEmpty()
            exchange.sendResponseHeaders(200, 2)
            exchange.responseBody.use { it.write("{}".toByteArray()) }
        }
        val client = HttpClient(baseUrl, defaultHeaders = mapOf("Authorization" to "Bearer t"))

        client.execute(HttpRequest(HttpMethod.GET, "/api/me", headers = mapOf("X-Ritme" to "1")))

        assertEquals("Bearer t", auth)
        assertEquals("1", custom)
    }

    @Test
    fun `4xx completes as Success carrying the error body`() = runBlocking {
        respond("/api/otp", 422, """{"error":"invalid code"}""")
        val client = HttpClient(baseUrl)

        val result = client.execute(HttpRequest(HttpMethod.POST, "/api/otp", body = "{}"))

        val response = (result as AppResult.Success).value
        assertEquals(422, response.statusCode)
        assertEquals("""{"error":"invalid code"}""", response.body)
    }

    @Test
    fun `unreachable host maps to a Network failure, not a throw`() = runBlocking {
        val client = HttpClient("http://nonexistent.invalid")

        val result = client.execute(HttpRequest(HttpMethod.GET, "/api/ping"))

        val failure = (result as AppResult.Failure).error
        assertTrue(failure is AppError.Network)
    }

    @Test
    fun `slow server maps to a Timeout failure`() = runBlocking {
        server.createContext("/api/slow") { exchange ->
            Thread.sleep(500)
            exchange.sendResponseHeaders(200, 2)
            exchange.responseBody.use { it.write("{}".toByteArray()) }
        }
        val client = HttpClient(baseUrl, readTimeoutMs = 100)

        val result = client.execute(HttpRequest(HttpMethod.GET, "/api/slow"))

        val failure = (result as AppResult.Failure).error
        assertTrue(failure is AppError.Timeout)
    }

    @Test
    fun `cached GET is served without a second network hit`() = runBlocking {
        val hits = AtomicInteger(0)
        respond("/api/banners", 200, """{"banners":[]}""") { hits.incrementAndGet() }
        val cacheDir = File.createTempFile("http_cache", "").let {
            it.delete(); it.mkdirs(); it
        }
        val client = HttpClient(baseUrl, responseCache = HttpResponseCache(cacheDir))
        val request = HttpRequest(HttpMethod.GET, "/api/banners", cacheTtlMillis = 60_000)

        val first = client.execute(request)
        val second = client.execute(request)

        assertEquals(1, hits.get())
        assertEquals(
            (first as AppResult.Success).value.body,
            (second as AppResult.Success).value.body,
        )
    }

    @Test
    fun `GET without a TTL always hits the network`() = runBlocking {
        val hits = AtomicInteger(0)
        respond("/api/live", 200, "{}") { hits.incrementAndGet() }
        val client = HttpClient(baseUrl, responseCache = HttpResponseCache(diskDir = null))
        val request = HttpRequest(HttpMethod.GET, "/api/live")

        client.execute(request)
        client.execute(request)

        assertEquals(2, hits.get())
    }

    @Test
    fun `error responses are never cached`() = runBlocking {
        val hits = AtomicInteger(0)
        respond("/api/flaky", 500, """{"error":"boom"}""") { hits.incrementAndGet() }
        val client = HttpClient(baseUrl, responseCache = HttpResponseCache(diskDir = null))
        val request = HttpRequest(HttpMethod.GET, "/api/flaky", cacheTtlMillis = 60_000)

        client.execute(request)
        client.execute(request)

        assertEquals(2, hits.get())
    }
}

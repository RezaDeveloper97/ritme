package ir.ritmeapp.ritme.adapter.outbound.image

import android.app.ActivityManager
import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.util.LruCache
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import kotlinx.coroutines.withContext
import java.io.File
import java.io.IOException
import java.net.HttpURLConnection
import java.net.SocketTimeoutException
import java.net.URL
import kotlin.coroutines.cancellation.CancellationException

/**
 * Hand-written three-tier image loader (CLAUDE.md §3 — no Coil/Glide, §5 low-end budget):
 * memory `LruCache` sized off the device's `memoryClass` (never a fixed MB), a disk cache
 * under `cacheDir/images/`, then the network. Every decode is a two-pass, `inSampleSize`-
 * downsampled decode targeting the view size, in RGB_565 (half the RAM, images here are
 * opaque banners/photos). In-flight decodes are capped so a fast scroll cannot OOM.
 */
class RitmeImageLoader(
    context: Context,
    private val diskDir: File,
    private val connectTimeoutMs: Int = DEFAULT_CONNECT_TIMEOUT_MS,
    private val readTimeoutMs: Int = DEFAULT_READ_TIMEOUT_MS,
    private val ioDispatcher: CoroutineDispatcher = Dispatchers.IO,
) : ImageLoader {

    private val memoryCache: LruCache<String, Bitmap>

    /** Bounds concurrent fetch+decode work (§5 — cap in-flight decodes). */
    private val inFlight = Semaphore(MAX_CONCURRENT_LOADS)

    init {
        val activityManager =
            context.getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        val maxKb = (activityManager.memoryClass * KB_PER_MB) / MEMORY_CLASS_FRACTION
        memoryCache = object : LruCache<String, Bitmap>(maxKb) {
            override fun sizeOf(key: String, value: Bitmap): Int =
                value.byteCount / BYTES_PER_KB
        }
    }

    override suspend fun load(
        url: String,
        reqWidthPx: Int,
        reqHeightPx: Int,
    ): AppResult<Bitmap> = withContext(ioDispatcher) {
        val key = "$url@${reqWidthPx}x$reqHeightPx"
        memoryCache.get(key)?.let { return@withContext AppResult.Success(it) }
        inFlight.withPermit {
            // Re-check after waiting: another coroutine may have loaded the same key.
            memoryCache.get(key)?.let { return@withPermit AppResult.Success(it) }
            val bytes = diskBytes(url) ?: when (val fetched = fetch(url)) {
                is AppResult.Success -> fetched.value.also { writeDisk(url, it) }
                is AppResult.Failure -> return@withPermit fetched
            }
            val bitmap = decodeDownsampled(bytes, reqWidthPx, reqHeightPx)
                ?: return@withPermit decodeFailure(url)
            memoryCache.put(key, bitmap)
            AppResult.Success(bitmap)
        }
    }

    // --- Network tier -------------------------------------------------------

    private fun fetch(url: String): AppResult<ByteArray> = try {
        val connection = (URL(url).openConnection() as HttpURLConnection).apply {
            connectTimeout = connectTimeoutMs
            readTimeout = readTimeoutMs
            instanceFollowRedirects = true
        }
        val status = connection.responseCode
        if (status in 200..299) {
            AppResult.Success(connection.inputStream.use { it.readBytes() })
        } else {
            connection.errorStream?.close()
            AppResult.Failure(AppError.Http(status, "Image fetch failed: $url"))
        }
    } catch (e: CancellationException) {
        throw e
    } catch (e: SocketTimeoutException) {
        AppResult.Failure(AppError.Timeout("Image fetch timed out: $url", e))
    } catch (e: IOException) {
        AppResult.Failure(AppError.Network("Image fetch error: $url", e))
    } catch (e: Exception) {
        AppResult.Failure(AppError.Unexpected("Unexpected image failure: $url", e))
    }

    // --- Disk tier ----------------------------------------------------------

    private fun diskBytes(url: String): ByteArray? {
        val file = fileFor(url)
        return try {
            if (file.exists()) file.readBytes() else null
        } catch (_: IOException) {
            null
        }
    }

    private fun writeDisk(url: String, bytes: ByteArray) {
        try {
            diskDir.mkdirs()
            fileFor(url).writeBytes(bytes)
        } catch (_: IOException) {
            // A failed cache write must never fail the load — next time re-fetches.
        }
    }

    private fun fileFor(url: String): File = File(diskDir, "img_${fnv1a(url)}")

    private fun fnv1a(value: String): String {
        var hash = FNV_OFFSET_BASIS
        for (ch in value) {
            hash = hash xor ch.code.toLong()
            hash *= FNV_PRIME
        }
        return hash.toULong().toString(RADIX_HEX)
    }

    // --- Decode (two-pass, downsampled — §5) ---------------------------------

    private fun decodeDownsampled(bytes: ByteArray, reqW: Int, reqH: Int): Bitmap? {
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeByteArray(bytes, 0, bytes.size, bounds)
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return null
        val options = BitmapFactory.Options().apply {
            inSampleSize = calculateInSampleSize(bounds, reqW, reqH)
            inPreferredConfig = Bitmap.Config.RGB_565
        }
        return BitmapFactory.decodeByteArray(bytes, 0, bytes.size, options)
    }

    /** Largest power-of-two keeping both dimensions >= the requested size. */
    private fun calculateInSampleSize(
        bounds: BitmapFactory.Options,
        reqW: Int,
        reqH: Int,
    ): Int {
        if (reqW <= 0 || reqH <= 0) return 1
        var sampleSize = 1
        var halfWidth = bounds.outWidth / 2
        var halfHeight = bounds.outHeight / 2
        while (halfWidth / sampleSize >= reqW && halfHeight / sampleSize >= reqH) {
            sampleSize *= 2
        }
        return sampleSize
    }

    private companion object {
        const val DEFAULT_CONNECT_TIMEOUT_MS = 10_000
        const val DEFAULT_READ_TIMEOUT_MS = 15_000
        const val MAX_CONCURRENT_LOADS = 3
        const val MEMORY_CLASS_FRACTION = 8 // ~1/8 of the app's memory class
        const val KB_PER_MB = 1024
        const val BYTES_PER_KB = 1024
        const val FNV_OFFSET_BASIS = -0x340d631b7bdddcdbL
        const val FNV_PRIME = 0x100000001b3L
        const val RADIX_HEX = 16
    }
}

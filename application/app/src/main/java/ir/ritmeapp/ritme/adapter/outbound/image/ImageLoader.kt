package ir.ritmeapp.ritme.adapter.outbound.image

import android.graphics.Bitmap
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult

/**
 * Outbound capability for loading remote images (CLAUDE.md §3 — no Coil/Glide).
 *
 * Lives in the adapter layer (not `domain/port`) deliberately: a decoded [Bitmap] is an
 * Android type, and the domain layer must stay free of Android imports (§2). No use case
 * ever needs an image — only Compose UI does — so the interface sits at the outbound
 * adapter boundary and is handed to the UI through the composition root.
 */
interface ImageLoader {

    /**
     * Loads [url] decoded and downsampled to roughly [reqWidthPx] x [reqHeightPx]
     * (§5 low-end budget — never full resolution). Checks memory, then disk, then the
     * network; failures come back as [AppResult.Failure], never a throw.
     */
    suspend fun load(url: String, reqWidthPx: Int, reqHeightPx: Int): AppResult<Bitmap>
}

/** Marker error helper so callers can distinguish decode failures from transport ones. */
internal fun decodeFailure(url: String): AppResult.Failure =
    AppResult.Failure(AppError.Parsing("Could not decode image: $url"))

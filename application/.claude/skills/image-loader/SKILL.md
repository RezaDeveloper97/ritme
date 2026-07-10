---
name: image-loader
description: Hand-build (no Coil/Glide) the Ritme image loader — LruCache memory tier + disk cache + downsampled BitmapFactory decode on Dispatchers.IO, sized for low-end devices. Use whenever loading, caching, or displaying remote images (product/plan thumbnails, banners, avatars), or when a screen shows network images and there is no loader yet.
---

# Custom Image Loader (no third-party library)

CLAUDE.md §3 bans Coil/Glide. Images are loaded by a hand-written loader living in
`adapter/outbound/network` (fetch) + `adapter/outbound/persistence` (disk cache), behind
a `domain/port/outbound` interface. §5 low-end-device budget is non-negotiable.

## Port (domain)
```kotlin
// domain/port/outbound/ImageLoaderPort.kt
interface ImageLoaderPort {
    /** Returns a decoded bitmap sized for [reqWidthPx] x [reqHeightPx], or an AppError. */
    suspend fun load(url: String, reqWidthPx: Int, reqHeightPx: Int): Result<Bitmap, AppError>
}
```
The Compose side gets a small `rememberImageBitmap(url, size)` helper (inbound/ui) that
calls the injected port on `Dispatchers.IO` and exposes state — never decodes on main.

## Three tiers (check in order)
1. **Memory** — `LruCache<String, Bitmap>` keyed by `url + "@" + targetSize`. Size it off
   the device, not a constant:
   ```kotlin
   val maxKb = (am.memoryClass * 1024) / 8   // ~1/8 of app memory class
   object : LruCache<String, Bitmap>(maxKb) { override fun sizeOf(k,b)=b.byteCount/1024 }
   ```
2. **Disk** — hand-written cache under `cacheDir/images/`, filename = hash(key). Decode
   from disk (still downsampled) on hit; promote into memory.
3. **Network** — fetch via the existing hand-written HTTP client (keep-alive, gzip,
   timeouts). Write bytes to disk, then decode.

## Decode rules (the low-end part)
Always two-pass decode, never full-resolution:
```kotlin
val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
BitmapFactory.decodeByteArray(bytes, 0, bytes.size, bounds)      // measure only
val opts = BitmapFactory.Options().apply {
    inSampleSize = calcInSampleSize(bounds, reqWidthPx, reqHeightPx) // power-of-two
    inPreferredConfig = Bitmap.Config.RGB_565                         // half the RAM if no alpha
}
BitmapFactory.decodeByteArray(bytes, 0, bytes.size, opts)
```
- `calcInSampleSize` halves until `w/2 < req` — target the *view* size, never the source.
- Cap in-flight decodes (e.g. a coroutine `Semaphore(2–3)`) so a fast scroll can't OOM.
- Cancel decodes for recycled/scrolled-off items (coroutine cancellation).
- All I/O + decode on `Dispatchers.IO`; surface to UI as state.

## Errors & resilience
- Return `Result<Bitmap, AppError>` with a `RIT-1xxx` (network) / `RIT-2xxx` (decode)
  code — never throw across the port (§4).
- Missing/failed image → show a placeholder, never a crash or blank that looks broken.

## Checklist
- [ ] Loading goes through `ImageLoaderPort`, wired in `adapter/outbound/di` — no Coil/Glide.
- [ ] Memory cache sized off `memoryClass`, not a hardcoded MB.
- [ ] Every decode uses `inSampleSize` + bounds pass; targets view size; `RGB_565` when opaque.
- [ ] Decode/I/O on `Dispatchers.IO`, bounded concurrency, cancellable on scroll.
- [ ] Disk cache in `cacheDir`, promotes to memory on hit.
- [ ] Failures return `Result.Error(RIT-…)` and render a placeholder.

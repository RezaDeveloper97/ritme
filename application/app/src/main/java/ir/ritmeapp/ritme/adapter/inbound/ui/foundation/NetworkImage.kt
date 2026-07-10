package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.produceState
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.unit.IntSize
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.domain.model.AppResult
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue

/**
 * Displays a remote image through the hand-written [ir.ritmeapp.ritme.adapter.outbound.image.ImageLoader]
 * (CLAUDE.md §3 — no Coil/Glide). Waits for the composable's measured size so the decode
 * is downsampled to the actual view, never full resolution (§5). While loading or on
 * failure it shows a calm surface-variant placeholder — never a crash or a broken frame.
 * The load is keyed on url+size, so scrolled-off items cancel naturally via composition.
 */
@Composable
fun NetworkImage(
    url: String,
    contentDescription: String?,
    modifier: Modifier = Modifier,
    contentScale: ContentScale = ContentScale.Crop,
) {
    val imageLoader = LocalAppContainer.current.imageLoader
    var size by remember { mutableStateOf(IntSize.Zero) }
    val bitmap by produceState<ImageBitmap?>(initialValue = null, url, size) {
        if (size == IntSize.Zero) return@produceState
        val result = imageLoader.load(url, size.width, size.height)
        value = (result as? AppResult.Success)?.value?.asImageBitmap()
    }
    Box(modifier = modifier.onSizeChanged { size = it }) {
        val current = bitmap
        if (current != null) {
            Image(
                bitmap = current,
                contentDescription = contentDescription,
                contentScale = contentScale,
                modifier = Modifier.matchParentSize(),
            )
        } else {
            Box(
                Modifier
                    .matchParentSize()
                    .background(MaterialTheme.colorScheme.surfaceVariant),
            )
        }
    }
}

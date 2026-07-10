package ir.ritmeapp.ritme.adapter.inbound.ui.navigation

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.input.pointer.util.VelocityTracker
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

private val EdgeWidth = 20.dp
private const val ParallaxFraction = 0.25f
private const val ScrimMaxAlpha = 0.25f
private const val CommitFraction = 0.4f

/**
 * The single, reusable Telegram-style edge swipe-to-go-back wrapper (CLAUDE.md §5b).
 * Every non-root screen is wrapped in exactly one of these; the root passes
 * `enabled = false` and it degrades to a plain container.
 *
 * Performance: the finger is tracked 1:1 via a raw `graphicsLayer` translation (no tree
 * recomposition per frame, RTL-safe because pixel translation is not mirrored). The
 * previous screen is rendered underneath with a subtle parallax + dim. Release past 40%
 * (or a fast fling) commits the pop; otherwise it springs back.
 */
@Composable
fun SwipeBackContainer(
    enabled: Boolean,
    onSwipeBack: () -> Unit,
    modifier: Modifier = Modifier,
    behind: (@Composable () -> Unit)? = null,
    content: @Composable () -> Unit,
) {
    if (!enabled) {
        Box(modifier.fillMaxSize()) { content() }
        return
    }

    val scope = rememberCoroutineScope()
    val density = LocalDensity.current
    val edgePx = with(density) { EdgeWidth.toPx() }
    val flingThresholdPx = with(density) { 800.dp.toPx() }

    val offsetX = remember { Animatable(0f) }
    var widthPx by remember { mutableFloatStateOf(0f) }

    Box(
        modifier
            .fillMaxSize()
            .onSizeChanged { widthPx = it.width.toFloat() },
    ) {
        val progress = if (widthPx > 0f) (offsetX.value / widthPx).coerceIn(0f, 1f) else 0f

        if (behind != null) {
            Box(
                Modifier
                    .fillMaxSize()
                    .graphicsLayer { translationX = -ParallaxFraction * widthPx * (1f - progress) },
            ) { behind() }
            Box(
                Modifier
                    .fillMaxSize()
                    .background(Color.Black.copy(alpha = ScrimMaxAlpha * (1f - progress))),
            )
        }

        Box(
            Modifier
                .fillMaxSize()
                .graphicsLayer { translationX = offsetX.value }
                .pointerInput(widthPx) {
                    val velocityTracker = VelocityTracker()
                    var fromEdge = false
                    detectHorizontalDragGestures(
                        onDragStart = { start ->
                            fromEdge = start.x <= edgePx
                            velocityTracker.resetTracking()
                        },
                        onHorizontalDrag = { change, dragAmount ->
                            if (fromEdge) {
                                change.consume()
                                velocityTracker.addPosition(change.uptimeMillis, change.position)
                                val target = (offsetX.value + dragAmount).coerceIn(0f, widthPx)
                                scope.launch { offsetX.snapTo(target) }
                            }
                        },
                        onDragEnd = {
                            if (fromEdge) {
                                val velocity = velocityTracker.calculateVelocity().x
                                val commit = widthPx > 0f &&
                                    (offsetX.value > widthPx * CommitFraction || velocity > flingThresholdPx)
                                scope.launch {
                                    if (commit) {
                                        offsetX.animateTo(widthPx, tween(durationMillis = 220))
                                        onSwipeBack()
                                    } else {
                                        offsetX.animateTo(0f, spring())
                                    }
                                }
                            }
                        },
                        onDragCancel = {
                            scope.launch { offsetX.animateTo(0f, spring()) }
                        },
                    )
                },
        ) { content() }
    }
}

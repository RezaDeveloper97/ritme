package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.gestures.snapping.rememberSnapFlingBehavior
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors

/**
 * A hand-built vertical wheel/spinner picker (CLAUDE.md §3 — no library). Snapping comes from
 * Compose's own [rememberSnapFlingBehavior]; index math is kept trivial by padding the list with
 * blank spacer rows on each side so the centered real item equals the list's first visible index.
 * The centered value is highlighted (pink, bold) and edges fade toward the surface, giving a
 * premium feel without per-frame custom layout — the drawing stays cheap on scroll (§5).
 *
 * Controlled component: callers pass the current [selectedIndex] and receive changes via
 * [onSelected] once a rest position settles.
 */
@Composable
fun WheelPicker(
    count: Int,
    selectedIndex: Int,
    onSelected: (Int) -> Unit,
    label: (Int) -> String,
    modifier: Modifier = Modifier,
    itemHeight: Dp = 44.dp,
    visibleCount: Int = 5,
) {
    val colors = LocalRitmeColors.current
    val half = visibleCount / 2
    val lastIndex = (count - 1).coerceAtLeast(0)
    // Start already centered on the initial selection: without this the first derived read sees
    // index 0 and reports it as a "selection" before the corrective scroll runs, snapping every
    // picker whose initial value isn't the first item down to the minimum (e.g. a year wheel
    // defaulting to 1405 inside a 1404..1405 range silently became 1404).
    val listState = rememberLazyListState(
        initialFirstVisibleItemIndex = selectedIndex.coerceIn(0, lastIndex),
    )
    val flingBehavior = rememberSnapFlingBehavior(listState)
    val itemHeightPx = with(LocalDensity.current) { itemHeight.toPx() }

    // The real item under the center slot: firstVisibleItemIndex, bumped by one when dragged more
    // than half a row past it. The leading spacers make this the real index directly.
    val centered by remember(count) {
        derivedStateOf {
            val bump = if (listState.firstVisibleItemScrollOffset > itemHeightPx / 2) 1 else 0
            (listState.firstVisibleItemIndex + bump).coerceIn(0, lastIndex)
        }
    }

    LaunchedEffect(centered, listState.isScrollInProgress) {
        if (!listState.isScrollInProgress && centered != selectedIndex) onSelected(centered)
    }

    LaunchedEffect(selectedIndex) {
        if (!listState.isScrollInProgress && centered != selectedIndex) {
            listState.scrollToItem(selectedIndex.coerceIn(0, lastIndex))
        }
    }

    Box(modifier = modifier.height(itemHeight * visibleCount), contentAlignment = Alignment.Center) {
        // Selection band behind the values (web `.wheel-band`): a pink-tinted rounded slot.
        Box(
            Modifier
                .fillMaxWidth()
                .height(itemHeight)
                .clip(RoundedCornerShape(12.dp))
                .background(colors.pinkLight.copy(alpha = 0.10f))
                .border(1.dp, colors.pinkLight.copy(alpha = 0.30f), RoundedCornerShape(12.dp)),
        )
        LazyColumn(
            state = listState,
            flingBehavior = flingBehavior,
            modifier = Modifier
                .fillMaxWidth()
                .edgeFade(colors.surface),
        ) {
            items(count = half, key = { "top_$it" }) { SpacerRow(itemHeight) }
            items(count = count, key = { "row_$it" }) { index ->
                val isCentered = index == centered
                Box(
                    Modifier.fillMaxWidth().height(itemHeight),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = label(index),
                        textAlign = TextAlign.Center,
                        // Web `.wi`: centered value is dark ink/bold; the rest is a light gray.
                        color = if (isCentered) colors.ink else colors.ink.copy(alpha = 0.28f),
                        style = if (isCentered) {
                            MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.Bold)
                        } else {
                            MaterialTheme.typography.titleMedium
                        },
                    )
                }
            }
            items(count = half, key = { "bottom_$it" }) { SpacerRow(itemHeight) }
        }
    }
}

@Composable
private fun SpacerRow(itemHeight: Dp) {
    Spacer(Modifier.fillMaxWidth().height(itemHeight))
}

/** Fades the top and bottom rows toward the surface color so the centered value reads as focused. */
private fun Modifier.edgeFade(surface: Color): Modifier = drawWithContent {
    drawContent()
    val fade = size.height * 0.28f
    drawRect(
        brush = Brush.verticalGradient(listOf(surface, Color.Transparent), startY = 0f, endY = fade),
    )
    drawRect(
        brush = Brush.verticalGradient(
            listOf(Color.Transparent, surface),
            startY = size.height - fade,
            endY = size.height,
        ),
    )
}

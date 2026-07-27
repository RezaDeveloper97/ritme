package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.snapping.rememberSnapFlingBehavior
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
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
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import kotlin.math.roundToInt

/**
 * A hand-built horizontal scroll-snap ruler for weight / height entry (CLAUDE.md §3 — no
 * library), mirroring the web `RulerPicker` + `.ruler` CSS: a big tabular readout of the
 * current value + unit above, a horizontal ruler of tick marks (major every 5 with a Persian
 * label), a fixed brand-pink center pointer, and surface edge fades. The ruler itself scrolls
 * left→right (LTR) even inside the RTL app, exactly like the web (`dir="ltr"`), so larger
 * numbers are always to the right.
 *
 * Controlled: the caller passes [value] and receives changes via [onValue] once a rest
 * position settles. [toDisplay] formats the readout (e.g. one-decimal weight, integer height).
 */
@Composable
fun RulerPicker(
    min: Int,
    max: Int,
    value: Int,
    unit: String,
    onValue: (Int) -> Unit,
    modifier: Modifier = Modifier,
    toDisplay: (Int) -> String = { it.toPersianDigits() },
) {
    val colors = LocalRitmeColors.current
    val count = (max - min + 1).coerceAtLeast(1)
    val tickWidth = 12.dp
    val tickPx = with(LocalDensity.current) { tickWidth.toPx() }

    val listState = rememberLazyListState(
        initialFirstVisibleItemIndex = (value - min).coerceIn(0, count - 1),
    )
    val fling = rememberSnapFlingBehavior(listState)

    val centered by remember(count, tickPx) {
        derivedStateOf {
            val scrolled = listState.firstVisibleItemIndex * tickPx + listState.firstVisibleItemScrollOffset
            (scrolled / tickPx).roundToInt().coerceIn(0, count - 1)
        }
    }

    LaunchedEffect(centered, listState.isScrollInProgress) {
        if (!listState.isScrollInProgress && (min + centered) != value) onValue(min + centered)
    }
    LaunchedEffect(value) {
        val target = (value - min).coerceIn(0, count - 1)
        if (!listState.isScrollInProgress && centered != target) listState.scrollToItem(target)
    }

    Column(modifier = modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Row(verticalAlignment = Alignment.Bottom) {
            Text(
                text = toDisplay(min + centered),
                style = MaterialTheme.typography.displayMedium.copy(fontSize = 48.sp, fontWeight = FontWeight.Bold),
                color = colors.ink,
            )
            Spacer(Modifier.width(6.dp))
            Text(
                text = unit,
                style = MaterialTheme.typography.bodyLarge.copy(fontSize = 15.sp),
                color = colors.inkMuted,
                modifier = Modifier.padding(bottom = 8.dp),
            )
        }
        Spacer(Modifier.height(16.dp))
        CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
            BoxWithConstraints(Modifier.fillMaxWidth().height(70.dp)) {
                val half = ((maxWidth - tickWidth) / 2).coerceAtLeast(0.dp)
                LazyRow(
                    state = listState,
                    flingBehavior = fling,
                    horizontalArrangement = Arrangement.Start,
                    verticalAlignment = Alignment.Bottom,
                    contentPadding = PaddingValues(horizontal = half),
                    modifier = Modifier.fillMaxSize(),
                ) {
                    items(count, key = { it }) { index -> Tick(min + index, colors, tickWidth) }
                }
                // Fixed brand-pink pointer over the horizontal center.
                Box(
                    Modifier
                        .align(Alignment.TopCenter)
                        .width(3.dp)
                        .height(44.dp)
                        .clip(RoundedCornerShape(2.dp))
                        .background(colors.pink),
                )
                Box(Modifier.fillMaxSize().edgeFadeHorizontal(colors.surface))
            }
        }
    }
}

@Composable
private fun Tick(tickValue: Int, colors: RitmeColors, tickWidth: Dp) {
    val major = tickValue % 5 == 0
    Column(
        modifier = Modifier.width(tickWidth).padding(bottom = 6.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .width(2.dp)
                .height(if (major) 32.dp else 18.dp)
                .clip(RoundedCornerShape(2.dp))
                .background(if (major) colors.tickMajor else colors.tickMinor),
        )
        Spacer(Modifier.height(5.dp))
        Box(Modifier.height(12.dp), contentAlignment = Alignment.Center) {
            if (major) {
                Text(
                    text = tickValue.toPersianDigits(),
                    style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp),
                    color = colors.tickMajor,
                )
            }
        }
    }
}

/** Fades the leading/trailing edges of the ruler toward the surface so the center reads focused. */
private fun Modifier.edgeFadeHorizontal(surface: Color): Modifier = drawWithContent {
    drawContent()
    val fade = size.width * 0.14f
    drawRect(brush = Brush.horizontalGradient(listOf(surface, Color.Transparent), startX = 0f, endX = fade))
    drawRect(
        brush = Brush.horizontalGradient(
            listOf(Color.Transparent, surface),
            startX = size.width - fade,
            endX = size.width,
        ),
    )
}

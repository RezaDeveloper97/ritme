package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors

/**
 * A hand-built bottom sheet (CLAUDE.md §3 — no library) mirroring the web `.sheet`: a dimmed
 * backdrop that dismisses on tap, and a surface panel that rises from the bottom with a grip,
 * 26dp top corners, and internal padding. Place it as the LAST child of the screen's root
 * `Box` so it overlays the content; it consumes no touches while hidden. The caller owns the
 * open/close state (`visible`) and content (§10).
 */
@Composable
fun RitmeBottomSheet(
    visible: Boolean,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit,
) {
    val colors = LocalRitmeColors.current
    Box(modifier = modifier.fillMaxSize()) {
        AnimatedVisibility(
            visible = visible,
            enter = fadeIn(tween(200)),
            exit = fadeOut(tween(200)),
        ) {
            Box(
                Modifier
                    .fillMaxSize()
                    .background(colors.ink.copy(alpha = 0.38f))
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        onClick = onDismiss,
                    ),
            )
        }
        AnimatedVisibility(
            visible = visible,
            modifier = Modifier.align(Alignment.BottomCenter),
            enter = slideInVertically(tween(320)) { it } + fadeIn(tween(220)),
            exit = slideOutVertically(tween(240)) { it } + fadeOut(tween(160)),
        ) {
            Column(
                Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(topStart = 26.dp, topEnd = 26.dp))
                    .background(colors.surface)
                    // Swallow taps on the panel so they don't fall through to the backdrop.
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        onClick = {},
                    )
                    .padding(top = 10.dp, start = 20.dp, end = 20.dp, bottom = 24.dp),
            ) {
                Box(
                    Modifier
                        .align(Alignment.CenterHorizontally)
                        .padding(bottom = 16.dp)
                        .width(44.dp)
                        .height(5.dp)
                        .clip(RoundedCornerShape(99.dp))
                        .background(colors.outline),
                )
                content()
            }
        }
    }
}

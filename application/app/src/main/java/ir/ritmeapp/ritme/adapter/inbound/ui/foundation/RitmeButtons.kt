package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors

/**
 * The shared primary CTA — mirrors the web `.btn-primary` (globals.css): 44dp tall, 14dp
 * radius, brand-pink fill with a 1px light-pink border and a soft pink glow shadow; the
 * disabled state is a solid light pink with no glow. The label is always supplied by the
 * caller (i18n, CLAUDE.md §5c) and is 14sp/bold white. A [loading] flag swaps the label for a
 * small spinner. No Material `Button` — this is a custom-drawn control so it matches the web
 * pixel-for-pixel (§5).
 */
@Composable
fun RitmePrimaryButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    loading: Boolean = false,
    leadingIcon: (@Composable () -> Unit)? = null,
) {
    val colors = LocalRitmeColors.current
    val shape = RoundedCornerShape(14.dp)
    val active = enabled && !loading
    Box(
        modifier = modifier
            .fillMaxWidth()
            .height(44.dp)
            .then(
                if (active) {
                    Modifier.shadow(
                        elevation = 12.dp,
                        shape = shape,
                        ambientColor = colors.pink,
                        spotColor = colors.pink,
                    )
                } else {
                    Modifier
                },
            )
            .clip(shape)
            .background(if (active) colors.pink else colors.pinkDisabled)
            .border(1.dp, if (active) colors.pinkBorder else Color.Transparent, shape)
            .clickable(enabled = active, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        if (loading) {
            CircularProgressIndicator(
                modifier = Modifier.size(20.dp),
                color = colors.onPink,
                strokeWidth = 2.dp,
            )
        } else {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                leadingIcon?.invoke()
                Text(
                    text = text,
                    style = MaterialTheme.typography.labelLarge.copy(
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Bold,
                    ),
                    color = colors.onPink,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

/**
 * Secondary/outline CTA — mirrors the web `.btn-ghost`: surface fill, brand-pink label, a
 * 1.5dp very-light-pink border, 44dp tall with a fully rounded (24dp) shape.
 */
@Composable
fun RitmeGhostButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    leadingIcon: (@Composable () -> Unit)? = null,
) {
    val colors = LocalRitmeColors.current
    val shape = RoundedCornerShape(24.dp)
    Box(
        modifier = modifier
            .fillMaxWidth()
            .height(44.dp)
            .clip(shape)
            .background(colors.surface)
            .border(1.5.dp, colors.pinkGhostBorder, shape)
            .clickable(enabled = enabled, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            leadingIcon?.invoke()
            Text(
                text = text,
                style = MaterialTheme.typography.labelLarge.copy(fontSize = 14.sp, fontWeight = FontWeight.Bold),
                color = colors.pink,
            )
        }
    }
}

/**
 * Low-emphasis pill — mirrors the web `.btn-soft`: neutral `line` fill, steel label, 40dp
 * tall, fully rounded (24dp), 13sp/bold.
 */
@Composable
fun RitmeSoftButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    leadingIcon: (@Composable () -> Unit)? = null,
) {
    val colors = LocalRitmeColors.current
    val shape = RoundedCornerShape(24.dp)
    Box(
        modifier = modifier
            .height(40.dp)
            .clip(shape)
            .background(colors.background)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 16.dp),
        contentAlignment = Alignment.Center,
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            leadingIcon?.invoke()
            Text(
                text = text,
                style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp, fontWeight = FontWeight.Bold),
                color = colors.steel,
            )
        }
    }
}

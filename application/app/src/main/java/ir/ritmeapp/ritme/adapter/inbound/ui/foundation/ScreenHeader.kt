package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors

/**
 * The shared sub-screen header, matching the web pattern (`NavBack` + start-aligned title):
 * a transparent, mirrored back chevron on the start edge, the title immediately after it
 * (start-aligned, NOT centered — the web header groups the title next to the back arrow),
 * and an optional trailing action pushed to the end. In the app-wide RTL layout the start
 * edge is the right, so `chevron_right` reads as "back" in fa exactly like the web.
 */
@Composable
fun ScreenHeader(
    title: String,
    onBack: (() -> Unit)?,
    modifier: Modifier = Modifier,
    trailing: (@Composable () -> Unit)? = null,
) {
    val colors = LocalRitmeColors.current
    Row(
        modifier = modifier
            .fillMaxWidth()
            .height(56.dp)
            .padding(horizontal = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (onBack != null) {
            HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.action_back), onBack)
        } else {
            Spacer(Modifier.width(4.dp))
        }
        Spacer(Modifier.width(2.dp))
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.weight(1f))
        if (trailing != null) {
            Row(verticalAlignment = Alignment.CenterVertically) { trailing() }
        }
    }
}

/**
 * A small tappable header icon, matching the web `.iconbtn`: 36dp touch target, transparent
 * background (no filled gray circle), ink-tinted 24dp glyph.
 */
@Composable
fun HeaderIconButton(
    @DrawableRes icon: Int,
    contentDescription: String?,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    tint: Color = LocalRitmeColors.current.ink,
) {
    Box(
        modifier = modifier
            .size(36.dp)
            .clip(CircleShape)
            .background(Color.Transparent)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = contentDescription,
            tint = tint,
            modifier = Modifier.size(24.dp),
        )
    }
}

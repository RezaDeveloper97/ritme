package ir.ritmeapp.ritme.adapter.inbound.ui

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors

/**
 * The brief branded gate shown at cold start while the app decides where to open (Home for a
 * returning session, otherwise Login — see [RitmeApp]). Deliberately static and cheap so it
 * costs nothing on the cold-start path (CLAUDE.md §5); the resolution it covers is a single
 * token read off the IO dispatcher.
 */
@Composable
fun StartupSplash(modifier: Modifier = Modifier) {
    val colors = LocalRitmeColors.current
    Box(
        modifier = modifier.background(colors.pinkContainer),
        contentAlignment = Alignment.Center,
    ) {
        Image(
            painter = painterResource(R.drawable.ritme_logo),
            contentDescription = stringResource(R.string.cd_app_logo),
            modifier = Modifier
                .size(148.dp)
                .clip(RoundedCornerShape(34.dp)),
        )
    }
}

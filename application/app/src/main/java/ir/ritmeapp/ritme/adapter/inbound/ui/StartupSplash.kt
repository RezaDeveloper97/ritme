package ir.ritmeapp.ritme.adapter.inbound.ui

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors

/**
 * The brief branded gate shown at cold start while the app decides where to open (Home for a
 * returning session, otherwise Login — see [RitmeApp]). Deliberately static and cheap so it
 * costs nothing on the cold-start path (CLAUDE.md §5); the resolution it covers is a single
 * token read off the IO dispatcher. Mirrors the web splash (`auth.splash`): a pink→violet
 * gradient field with the logo, wordmark, tagline, a spinner, and a copyright line.
 */
@Composable
fun StartupSplash(modifier: Modifier = Modifier) {
    val colors = LocalRitmeColors.current
    val white = colors.onPink

    Box(
        modifier = modifier
            .clipToBounds()
            .background(
                // Web: linear-gradient(160deg, #E91E63 → #BA68C8). On a portrait screen the
                // default top-start → bottom-end diagonal lands at ~the same visual angle.
                Brush.linearGradient(listOf(colors.pink, colors.violetGrad)),
            ),
        contentAlignment = Alignment.Center,
    ) {
        // Figma: faint concentric-circles mark peeking from the top trailing corner.
        Canvas(
            modifier = Modifier
                .align(Alignment.TopEnd)
                .offset(x = 30.dp, y = (-55).dp)
                .size(180.dp),
        ) {
            val ringColor = white.copy(alpha = 0.12f)
            val center = Offset(size.width * 43f / 86f, size.height * 43f / 86f)
            val stroke = size.width * 1.5f / 86f
            drawCircle(ringColor, radius = size.width * 36f / 86f, center = center, style = Stroke(width = stroke))
            drawCircle(ringColor, radius = size.width * 25f / 86f, center = center, style = Stroke(width = stroke))
            drawCircle(ringColor, radius = size.width * 14.5f / 86f, center = center)
        }

        // Logo + wordmark + tagline, centered.
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Image(
                painter = painterResource(R.drawable.ritme_logo),
                contentDescription = stringResource(R.string.cd_app_logo),
                modifier = Modifier
                    .size(84.dp)
                    .shadow(elevation = 20.dp, shape = RoundedCornerShape(26.dp))
                    .clip(RoundedCornerShape(26.dp)),
            )
            Text(
                text = stringResource(R.string.app_name),
                color = white,
                fontSize = 34.sp,
                fontWeight = FontWeight.Black,
                letterSpacing = 1.sp,
                modifier = Modifier.padding(top = 6.dp),
            )
            Text(
                text = stringResource(R.string.splash_tagline),
                color = white.copy(alpha = 0.92f),
                fontSize = 13.sp,
                fontWeight = FontWeight.Medium,
            )
        }

        // Bottom: spinner + copyright.
        val spinner = rememberInfiniteTransition(label = "splashSpinner")
        val angle by spinner.animateFloat(
            initialValue = 0f,
            targetValue = 360f,
            animationSpec = infiniteRepeatable(
                animation = tween(durationMillis = 1000, easing = LinearEasing),
                repeatMode = RepeatMode.Restart,
            ),
            label = "splashSpinnerAngle",
        )
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(10.dp),
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .padding(bottom = 30.dp),
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_refresh),
                contentDescription = null,
                tint = white.copy(alpha = 0.95f),
                modifier = Modifier
                    .size(24.dp)
                    .rotate(angle),
            )
            Text(
                text = stringResource(R.string.splash_copyright),
                color = white.copy(alpha = 0.8f),
                fontSize = 10.sp,
            )
        }
    }
}

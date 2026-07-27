package ir.ritmeapp.ritme.adapter.inbound.ui.welcome

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.launch

/**
 * The first-run welcome carousel (three intro slides). It is the app's start destination for a
 * logged-out visitor who hasn't seen it; finishing or skipping marks it seen and hands off to
 * login. Being a start destination it is intentionally not swipe-back-wrapped.
 */
@Composable
fun WelcomeScreen(
    onFinished: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: WelcomeViewModel = viewModel(factory = container.welcomeViewModelFactory())
    val finished by viewModel.finished.collectAsStateWithLifecycle()

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:welcome:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Welcome.route, null, System.currentTimeMillis()),
        )
    }

    LaunchedEffect(finished) {
        if (finished) onFinished()
    }

    IntroPager(onComplete = viewModel::onCompleted, modifier = modifier)
}

@Composable
private fun IntroPager(
    onComplete: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    val scope = rememberCoroutineScope()
    val lastIndex = SLIDES.lastIndex
    var page by remember { mutableIntStateOf(0) }
    val offsetX = remember { Animatable(0f) }

    Scaffold(modifier = modifier.fillMaxSize(), containerColor = colors.surface) { padding ->
        BoxWithConstraints(
            Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            val widthPx = constraints.maxWidth.toFloat()

            fun goTo(target: Int) {
                scope.launch {
                    val clamped = target.coerceIn(0, lastIndex)
                    if (clamped == page) {
                        offsetX.animateTo(0f, tween(250))
                    } else {
                        val direction = if (clamped > page) -1f else 1f
                        offsetX.animateTo(direction * widthPx, tween(280))
                        page = clamped
                        offsetX.snapTo(0f)
                    }
                }
            }

            Box(
                Modifier
                    .fillMaxSize()
                    .pointerInput(widthPx, lastIndex) {
                        detectHorizontalDragGestures(
                            onHorizontalDrag = { change, drag ->
                                change.consume()
                                scope.launch { offsetX.snapTo(offsetX.value + drag) }
                            },
                            onDragEnd = {
                                val threshold = widthPx * 0.2f
                                when {
                                    offsetX.value < -threshold && page < lastIndex -> settle(scope, offsetX, widthPx) { page += 1 }
                                    offsetX.value > threshold && page > 0 -> settle(scope, offsetX, -widthPx) { page -= 1 }
                                    else -> scope.launch { offsetX.animateTo(0f, tween(250)) }
                                }
                            },
                        )
                    },
            ) {
                SLIDES.forEachIndexed { index, slide ->
                    SlideContent(
                        slide = slide,
                        colors = colors,
                        modifier = Modifier
                            .fillMaxSize()
                            .graphicsLayer { translationX = (index - page) * widthPx + offsetX.value },
                    )
                }
            }

            // Top: brand wordmark (start) + skip (end). Bottom: dots + primary CTA.
            Row(
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(R.string.app_name),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.pink,
                    fontWeight = androidx.compose.ui.text.font.FontWeight.Black,
                )
                Spacer(Modifier.weight(1f))
                TextButton(onClick = onComplete) {
                    Text(stringResource(R.string.intro_skip), color = colors.inkMuted)
                }
            }

            Column(
                modifier = Modifier
                    .align(Alignment.BottomCenter)
                    .fillMaxWidth()
                    .padding(horizontal = 24.dp, vertical = 28.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Dots(count = SLIDES.size, active = page, colors = colors, onSelect = { goTo(it) })
                Spacer(Modifier.height(20.dp))
                RitmePrimaryButton(
                    text = stringResource(if (page < lastIndex) R.string.intro_next else R.string.intro_start),
                    onClick = { if (page < lastIndex) goTo(page + 1) else onComplete() },
                )
            }
        }
    }
}

/** Advances/rewinds a page by nudging the offset by [delta] to keep the drag continuous, then commits. */
private fun settle(
    scope: kotlinx.coroutines.CoroutineScope,
    offsetX: Animatable<Float, *>,
    delta: Float,
    commitPage: () -> Unit,
) {
    scope.launch {
        commitPage()
        offsetX.snapTo(offsetX.value + delta)
        offsetX.animateTo(0f, tween(280))
    }
}

@Composable
private fun SlideContent(slide: SlideSpec, colors: RitmeColors, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier.padding(horizontal = 28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Spacer(Modifier.height(48.dp))
        IntroIllustration(
            art = slide.art,
            accentFrom = colors.pink,
            accentTo = colors.violetGrad,
            modifier = Modifier.fillMaxWidth(0.8f),
        )
        Spacer(Modifier.height(36.dp))
        // Copy is start-aligned (right in RTL), like the web slides — not centered.
        Text(
            text = stringResource(slide.title),
            modifier = Modifier.fillMaxWidth(),
            style = MaterialTheme.typography.headlineSmall,
            color = colors.ink,
            textAlign = TextAlign.Start,
        )
        Spacer(Modifier.height(14.dp))
        Text(
            text = stringResource(slide.body),
            modifier = Modifier.fillMaxWidth(),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
            textAlign = TextAlign.Start,
        )
        slide.bullets.forEach { bullet ->
            Spacer(Modifier.height(14.dp))
            BulletRow(icon = bullet.icon, text = stringResource(bullet.text), colors = colors)
        }
        slide.disclaimer?.let {
            Spacer(Modifier.height(18.dp))
            Text(
                text = stringResource(it),
                modifier = Modifier.fillMaxWidth(),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
                textAlign = TextAlign.Start,
            )
        }
        slide.hint?.let {
            Spacer(Modifier.height(10.dp))
            Text(
                text = stringResource(it),
                modifier = Modifier.fillMaxWidth(),
                style = MaterialTheme.typography.labelMedium,
                color = colors.pink,
                textAlign = TextAlign.Start,
            )
        }
    }
}

@Composable
private fun BulletRow(@DrawableRes icon: Int, text: String, colors: RitmeColors) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // Icon tile (web renders each feature bullet as a heart/moon tile, not a plain dot).
        Box(
            Modifier
                .size(36.dp)
                .clip(RoundedCornerShape(12.dp))
                .background(colors.pinkContainer),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(icon),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.size(20.dp),
            )
        }
        Spacer(Modifier.width(12.dp))
        Text(
            text = text,
            modifier = Modifier.weight(1f),
            style = MaterialTheme.typography.bodySmall,
            color = colors.ink,
            textAlign = TextAlign.Start,
        )
    }
}

@Composable
private fun Dots(count: Int, active: Int, colors: RitmeColors, onSelect: (Int) -> Unit) {
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        repeat(count) { i ->
            val on = i == active
            Box(
                Modifier
                    .height(8.dp)
                    .width(if (on) 22.dp else 8.dp)
                    .clip(CircleShape)
                    .background(if (on) colors.pink else colors.outline)
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        onClick = { onSelect(i) },
                    ),
            )
        }
    }
}

/** A feature bullet: an icon tile + its copy (web renders these as heart/moon tiles). */
private data class BulletSpec(@DrawableRes val icon: Int, @StringRes val text: Int)

/** One intro slide's content: which artwork, and the string resources for its copy. */
private data class SlideSpec(
    val art: IntroArt,
    @StringRes val title: Int,
    @StringRes val body: Int,
    val bullets: List<BulletSpec> = emptyList(),
    @StringRes val disclaimer: Int? = null,
    @StringRes val hint: Int? = null,
)

private val SLIDES = listOf(
    SlideSpec(IntroArt.RHYTHM, R.string.intro_1_title, R.string.intro_1_body),
    SlideSpec(
        IntroArt.TRACK,
        R.string.intro_2_title,
        R.string.intro_2_body,
        bullets = listOf(
            BulletSpec(R.drawable.ic_heart, R.string.intro_2_bullet_1),
            BulletSpec(R.drawable.ic_moon, R.string.intro_2_bullet_2),
        ),
    ),
    SlideSpec(
        IntroArt.INSIGHT,
        R.string.intro_3_title,
        R.string.intro_3_body,
        disclaimer = R.string.intro_3_disclaimer,
        hint = R.string.intro_3_hint,
    ),
)
